// Copyright (c) 2022  All rights reserved.

package aws

import (
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

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
	eniIDsBySubnet         map[string][]string
	eniIDByIP              map[ip.Addr]string
	eniIDByPrimaryIP       map[ip.Addr]string
	attachmentIDByENIID    map[string]string

	inUseDeviceIndexes      map[int32]bool
	freeIPv4CapacityByENIID map[string]int
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

func (s *awsState) OnPrivateIPsAdded(eniID string, addrs []string) {
	eni := s.calicoOwnedENIsByID[eniID]
	for _, addrStr := range addrs {
		addr := ip.FromString(addrStr)
		if addr == nil {
			rlAWSProblemLogger.WithField("rawIP", addrStr).Error(
				"BUG! Successfully added a bad IP to AWS?!")
			continue
		}
		eni.IPAddresses = append(eni.IPAddresses, &eniIPAddress{
			PrivateIP: addr,
		})
		s.eniIDByIP[addr] = eniID
	}
	s.freeIPv4CapacityByENIID[eni.ID] = s.capabilities.MaxIPv4PerInterface - len(eni.IPAddresses)
}

func (s *awsState) OnPrivateIPsRemoved(eniID string, addrs []string) {
	removedIPs := set.New()
	for _, addrStr := range addrs {
		addr := ip.FromString(addrStr)
		if addr == nil {
			rlAWSProblemLogger.WithField("rawIP", addrStr).Error(
				"BUG! Successfully removed a bad IP from AWS?!")
			continue
		}
		removedIPs.Add(addr)
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
	s.freeIPv4CapacityByENIID[eni.ID] = s.capabilities.MaxIPv4PerInterface - len(eni.IPAddresses)
}

func (s *awsState) OnCalicoENIAttached(eni *eniState) {
	s.calicoOwnedENIsByID[eni.ID] = eni
	s.eniIDsBySubnet[eni.SubnetID] = append(s.eniIDsBySubnet[eni.SubnetID], eni.ID)
	for _, eniAddr := range eni.IPAddresses {
		s.eniIDByIP[eniAddr.PrivateIP] = eni.ID
		if eniAddr.Primary {
			s.eniIDByPrimaryIP[eniAddr.PrivateIP] = eni.ID
		}
	}
	s.attachmentIDByENIID[eni.Attachment.ID] = eni.ID
	s.inUseDeviceIndexes[eni.Attachment.DeviceIndex] = true
	s.freeIPv4CapacityByENIID[eni.ID] = s.capabilities.MaxIPv4PerInterface - len(eni.IPAddresses)
}

func (s *awsState) OnCalicoENIDetached(eniID string) {
	eni := s.calicoOwnedENIsByID[eniID]
	delete(s.calicoOwnedENIsByID, eniID)

	newENIsBySubnet := s.eniIDsBySubnet[eni.SubnetID][:0]
	for _, sENIID := range s.eniIDsBySubnet[eni.SubnetID] {
		if sENIID == eniID {
			continue
		}
		newENIsBySubnet = append(newENIsBySubnet, eni.SubnetID)
	}
	s.eniIDsBySubnet[eni.SubnetID] = newENIsBySubnet

	for _, eniAddr := range eni.IPAddresses {
		delete(s.eniIDByIP, eniAddr.PrivateIP)
		if eniAddr.Primary {
			delete(s.eniIDByPrimaryIP, eniAddr.PrivateIP)
		}
	}
	delete(s.attachmentIDByENIID, eni.Attachment.ID)
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
