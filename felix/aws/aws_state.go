// Copyright (c) 2022  All rights reserved.

package aws

import (
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/felix/ip"
	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	"github.com/projectcalico/calico/libcalico-go/lib/set"
)

var rlAWSProblemLogger = logutils.NewRateLimitedLogger()

// awsState captures the current state of the AWS ENIs attached to this host, indexed in various ways.
// We keep this up to date as we make changes to AWS state.
type awsState struct {
	capabilities *NetworkCapabilities

	primaryENI             *eniState
	calicoOwnedENIsByID    map[string]*eniState
	nonCalicoOwnedENIsByID map[string]*eniState
	calicoENIIDsBySubnet   map[string][]string
	eniIDBySecondaryIP     map[ip.Addr]string
	eniIDByPrimaryIP       map[ip.Addr]string
	attachmentIDByENIID    map[string]string

	inUseDeviceIndexes      map[int32]bool
	freeIPv4CapacityByENIID map[string]int
}

func newAWSState(networkCapabilities *NetworkCapabilities) *awsState {
	return &awsState{
		capabilities: networkCapabilities,

		calicoOwnedENIsByID:    map[string]*eniState{},
		nonCalicoOwnedENIsByID: map[string]*eniState{},
		calicoENIIDsBySubnet:   map[string][]string{},
		eniIDBySecondaryIP:     map[ip.Addr]string{},
		eniIDByPrimaryIP:       map[ip.Addr]string{},
		attachmentIDByENIID:    map[string]string{},

		inUseDeviceIndexes:      map[int32]bool{},
		freeIPv4CapacityByENIID: map[string]int{},
	}
}

func (s *awsState) PrimaryENISecurityGroups() []string {
	return s.primaryENI.SecurityGroupIDs
}

func (s *awsState) CalculateUnusedENICapacity(netCaps *NetworkCapabilities) int {
	// For now, only supporting the first network card.
	numPossibleENIs := netCaps.MaxENIsForCard(0)
	numExistingENIs := len(s.inUseDeviceIndexes)
	return numPossibleENIs - numExistingENIs
}

func (s *awsState) FindFreeDeviceIdx() int32 {
	devIdx := int32(0)
	for s.inUseDeviceIndexes[devIdx] {
		devIdx++
	}
	return devIdx
}

func (s *awsState) ClaimDeviceIdx(devIdx int32) {
	s.inUseDeviceIndexes[devIdx] = true
}

func (s *awsState) OnSecondaryIPsAdded(eniID string, addrs []string) {
	eni := s.calicoOwnedENIsByID[eniID]
	for _, addrStr := range addrs {
		addr := ip.FromString(addrStr)
		if addr == nil {
			rlAWSProblemLogger.WithField("rawIP", addrStr).Error(
				"BUG! Successfully added a bad IP to AWS?!")
			// Defensive, record that an IP slot is filled anyway.  This will get refreshed on the next resync.
			eni.numFilteredIPs++
			continue
		}
		eni.IPAddresses = append(eni.IPAddresses, &eniIPAddress{
			PrivateIP: addr,
		})
		s.eniIDBySecondaryIP[addr] = eniID
	}
	s.refreshFreeIPCount(eni)
}

func (s *awsState) OnSecondaryIPsRemoved(eniID string, addrs []string) {
	removedIPs := set.NewBoxed[ip.Addr]()
	for _, addrStr := range addrs {
		addr := ip.FromString(addrStr)
		if addr == nil {
			rlAWSProblemLogger.WithField("rawIP", addrStr).Error(
				"BUG! Successfully removed a bad IP from AWS?!")
			continue
		}
		removedIPs.Add(addr)
		delete(s.eniIDBySecondaryIP, addr)
	}

	eni := s.calicoOwnedENIsByID[eniID]
	newIPs := eni.IPAddresses[:0]
	for _, eniIP := range eni.IPAddresses {
		if removedIPs.Contains(eniIP.PrivateIP) {
			continue
		}
		newIPs = append(newIPs, eniIP)
	}
	eni.IPAddresses = newIPs
	s.refreshFreeIPCount(eni)
}

func (s *awsState) OnCalicoENIAttached(eni *eniState) {
	logCtx := logrus.WithField("id", eni.ID)
	logCtx.Debug("Adding Calico ENI to cached state")
	s.calicoOwnedENIsByID[eni.ID] = eni
	s.calicoENIIDsBySubnet[eni.SubnetID] = append(s.calicoENIIDsBySubnet[eni.SubnetID], eni.ID)
	for _, eniAddr := range eni.IPAddresses {
		if eniAddr.Primary {
			logCtx.WithField("ip", eniAddr.PrivateIP).Debug("Found primary IP on Calico ENI")
			s.eniIDByPrimaryIP[eniAddr.PrivateIP] = eni.ID
		} else {
			logCtx.WithField("ip", eniAddr.PrivateIP).Debug("Found secondary IP on Calico ENI")
			s.eniIDBySecondaryIP[eniAddr.PrivateIP] = eni.ID
		}
	}
	if eni.Attachment != nil {
		s.attachmentIDByENIID[eni.ID] = eni.Attachment.ID
		s.inUseDeviceIndexes[eni.Attachment.DeviceIndex] = true
	} else {
		logCtx.Warn("AWS returned ENI with no attachment (even though it should already be attached)")
	}
	s.refreshFreeIPCount(eni)
}

func (s *awsState) refreshFreeIPCount(eni *eniState) {
	logCtx := logrus.WithField("id", eni.ID)
	s.freeIPv4CapacityByENIID[eni.ID] = s.capabilities.MaxIPv4PerInterface - eni.NumIPs()
	logCtx.WithField("availableIPs", s.freeIPv4CapacityByENIID[eni.ID]).Debug("Calculated available IPs")
	if s.freeIPv4CapacityByENIID[eni.ID] < 0 {
		logCtx.Errorf("ENI appears to have more IPs (%v) that it should (%v)", eni.NumIPs(),
			s.capabilities.MaxIPv4PerInterface)
		s.freeIPv4CapacityByENIID[eni.ID] = 0
	}
}

func (s *awsState) OnCalicoENIDetached(eniID string) {
	eni, ok := s.calicoOwnedENIsByID[eniID]
	if !ok {
		logrus.WithField("eniID", eniID).Warn("BUG: unknown ENI ID, ignoring.")
	}
	delete(s.calicoOwnedENIsByID, eniID)

	newENIsBySubnet := s.calicoENIIDsBySubnet[eni.SubnetID][:0]
	for _, sENIID := range s.calicoENIIDsBySubnet[eni.SubnetID] {
		if sENIID == eniID {
			continue
		}
		newENIsBySubnet = append(newENIsBySubnet, eni.SubnetID)
	}
	s.calicoENIIDsBySubnet[eni.SubnetID] = newENIsBySubnet

	for _, eniAddr := range eni.IPAddresses {
		if eniAddr.Primary {
			delete(s.eniIDByPrimaryIP, eniAddr.PrivateIP)
		} else {
			delete(s.eniIDBySecondaryIP, eniAddr.PrivateIP)
		}
	}
	delete(s.attachmentIDByENIID, eni.ID)
	delete(s.inUseDeviceIndexes, eni.Attachment.DeviceIndex)
	delete(s.freeIPv4CapacityByENIID, eniID)
}

func (s *awsState) OnENIDeleteOnTermUpdated(eniID string, b bool) {
	s.calicoOwnedENIsByID[eniID].Attachment.DeleteOnTermination = b
}

func (s *awsState) OnElasticIPAssociated(eniID string, privIP ip.Addr, assocID, allocID string, pubIP ip.Addr) {
	for _, eniIP := range s.calicoOwnedENIsByID[eniID].IPAddresses {
		if eniIP.PrivateIP == privIP {
			eniIP.Association = &eniAssociation{
				ID:           assocID,
				AllocationID: allocID,
				PublicIP:     pubIP,
			}
			break
		}
	}
}

// awsNetworkInterfaceToENIState converts an AWS NetworkInterface to our internal model, *eniState.
// *eniState is essentially a trimmed down version of the AWS NetworkInterface, covering only the
// fields that we care about.  To avoid lots of downstream nil checks, we map mandatory fields
// of the NetworkInterface to non-pointers and do validation here to check that the values from AWS
// really aren't null.
//
// For the nested eniAttachment and IPAddresses field, we only include valid versions from AWS.
// so downstream code can assume that, if eniState.Attachment is non-nil then its ID is also filled in.
// In practice, I think it's overwhelmingly likely that we never see AWS returning a NetworkInterface
// with no ID or an attachment with no ID, so we're really just trying to avoid panics and simplify
// downstream code without trying to "handle" those cases in any meaningful way.
func awsNetworkInterfaceToENIState(eni ec2types.NetworkInterface) *eniState {
	if eni.NetworkInterfaceId == nil || eni.SubnetId == nil || eni.MacAddress == nil {
		// This feels like it'd be a bug in the AWS API.
		rlAWSProblemLogger.WithField("eni", eni).Warn(
			"AWS returned ENI with missing NetworkInterfaceId/SubnetId/MacAddress field.")
		return nil
	}
	var ourENI eniState
	ourENI.ID = *eni.NetworkInterfaceId
	ourENI.SubnetID = *eni.SubnetId
	ourENI.MACAddress = *eni.MacAddress

	if eni.Attachment != nil && eni.Attachment.AttachmentId != nil {
		att := eni.Attachment
		if att.DeviceIndex == nil {
			rlAWSProblemLogger.WithField("eni", eni).Warn("AWS returned ENI with missing required field in Attachment.")
			return nil
		}

		if att.NetworkCardIndex != nil && *att.NetworkCardIndex != 0 {
			// Ignore ENIs that aren't on the primary network card.  We only support one network card for now.
			rlAWSProblemLogger.Warnf("Ignoring ENI on non-primary network card: %d.", *eni.Attachment.NetworkCardIndex)
			return nil
		}

		var ourAttachment eniAttachment
		ourAttachment.ID = *att.AttachmentId
		ourAttachment.DeviceIndex = *att.DeviceIndex

		if att.DeleteOnTermination != nil {
			ourAttachment.DeleteOnTermination = *att.DeleteOnTermination
		}
		ourENI.Attachment = &ourAttachment
	}

	for _, addr := range eni.PrivateIpAddresses {
		if addr.PrivateIpAddress == nil {
			continue
		}
		ipAddr := ip.FromString(*addr.PrivateIpAddress)
		if ipAddr == nil {
			rlAWSProblemLogger.WithField("rawIP", *addr.PrivateIpAddress).Warn(
				"AWS Returned malformed Private IP, ignoring.")
			continue
		}

		ourAddr := eniIPAddress{
			PrivateIP: ipAddr,
		}
		if addr.Primary != nil {
			ourAddr.Primary = *addr.Primary
		}

		if addr.Association != nil && addr.Association.AssociationId != nil {
			if addr.Association.PublicIp == nil || addr.Association.AssociationId == nil {
				rlAWSProblemLogger.WithField("association", addr.Association).Warn(
					"AWS Returned malformed association, ignoring.")
				continue
			}
			pubIP := ip.FromString(*addr.Association.PublicIp)
			if pubIP == nil {
				rlAWSProblemLogger.WithField("rawIP", *addr.Association.PublicIp).Warn(
					"AWS Returned malformed Public IP, ignoring.")
				continue
			}
			ourAddr.Association = &eniAssociation{
				ID:           *addr.Association.AssociationId,
				AllocationID: *addr.Association.AllocationId,
				PublicIP:     pubIP,
			}
		}
		ourENI.IPAddresses = append(ourENI.IPAddresses, &ourAddr)
	}
	// Defensive: record the number of IPs we filtered out.
	ourENI.numFilteredIPs = len(eni.PrivateIpAddresses) - len(ourENI.IPAddresses)

	for _, g := range eni.Groups {
		if g.GroupId != nil {
			ourENI.SecurityGroupIDs = append(ourENI.SecurityGroupIDs, *g.GroupId)
		}
	}

	return &ourENI
}

type eniState struct {
	ID               string
	SubnetID         string
	MACAddress       string
	IPAddresses      []*eniIPAddress
	SecurityGroupIDs []string
	Attachment       *eniAttachment
	numFilteredIPs   int
}

func (s *eniState) PrimaryIP() ip.Addr {
	for _, ipInfo := range s.IPAddresses {
		if ipInfo.Primary {
			return ipInfo.PrivateIP
		}
	}
	return nil
}

func (s *eniState) NumIPs() int {
	return len(s.IPAddresses) + s.numFilteredIPs
}

type eniAttachment struct {
	ID                  string
	DeviceIndex         int32
	DeleteOnTermination bool
}

type eniIPAddress struct {
	PrivateIP ip.Addr
	Primary   bool

	Association *eniAssociation
}

type eniAssociation struct {
	ID           string
	AllocationID string
	PublicIP     ip.Addr
}
