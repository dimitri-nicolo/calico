// Copyright (c) 2022  All rights reserved.

package aws

import (
	"github.com/projectcalico/calico/felix/ip"
	"github.com/projectcalico/calico/libcalico-go/lib/set"
	"github.com/sirupsen/logrus"
)

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

func (s *awsState) calculateUnusedENICapacity(netCaps *NetworkCapabilities) int {
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
			logrus.WithField("rawIP", addrStr).Error("BUG! Successfully added a bad IP to AWS?!")
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
			logrus.WithField("rawIP", addrStr).Error("BUG! Successfully removed a bad IP from AWS?!")
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
