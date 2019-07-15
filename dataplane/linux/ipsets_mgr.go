// Copyright (c) 2016-2019 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package intdataplane

import (
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/ipsets"
	"github.com/projectcalico/felix/proto"
	"github.com/projectcalico/libcalico-go/lib/set"
)

type ipSetsManagerCallbacks struct {
	addMembersIPSet    *AddMembersIPSetFuncs
	removeMembersIPSet *RemoveMembersIPSetFuncs
	replaceIPSet       *ReplaceIPSetFuncs
	removeIPSet        *RemoveIPSetFuncs
}

func newIPSetsManagerCallbacks(callbacks *callbacks, ipFamily ipsets.IPFamily) ipSetsManagerCallbacks {
	if ipFamily == ipsets.IPFamilyV4 {
		return ipSetsManagerCallbacks{
			addMembersIPSet:    callbacks.AddMembersIPSetV4,
			removeMembersIPSet: callbacks.RemoveMembersIPSetV4,
			replaceIPSet:       callbacks.ReplaceIPSetV4,
			removeIPSet:        callbacks.RemoveIPSetV4,
		}
	} else {
		return ipSetsManagerCallbacks{
			addMembersIPSet:    &AddMembersIPSetFuncs{},
			removeMembersIPSet: &RemoveMembersIPSetFuncs{},
			replaceIPSet:       &ReplaceIPSetFuncs{},
			removeIPSet:        &RemoveIPSetFuncs{},
		}
	}
}

func (c *ipSetsManagerCallbacks) InvokeAddMembersIPSet(setID string, members set.Set) {
	c.addMembersIPSet.Invoke(setID, members)
}

func (c *ipSetsManagerCallbacks) InvokeRemoveMembersIPSet(setID string, members set.Set) {
	c.removeMembersIPSet.Invoke(setID, members)
}

func (c *ipSetsManagerCallbacks) InvokeReplaceIPSet(setID string, members set.Set) {
	c.replaceIPSet.Invoke(setID, members)
}

func (c *ipSetsManagerCallbacks) InvokeRemoveIPSet(setID string) {
	c.removeIPSet.Invoke(setID)
}

// Except for domain IP sets, ipSetsManager simply passes through IP set updates from the datastore
// to the ipsets.IPSets dataplane layer.  For domain IP sets - which hereafter we'll just call
// "domain sets" - ipSetsManager handles the resolution from domain names to expiring IPs.
type ipSetsManager struct {
	ipsetsDataplane ipsetsDataplane
	maxSize         int
	callbacks       ipSetsManagerCallbacks

	// Provider of domain name to IP information.
	domainInfoStore store

	// Map from each active domain set ID to the IPs that are currently programmed for it and
	// why.  The interior map is from each IP to the set of domain names that have resolved to
	// that IP.
	domainSetProgramming map[string]map[string]set.Set

	// Map from each active domain name to the IDs of the domain sets that include that domain
	// name.
	domainSetIds map[string]set.Set
}

type store interface {
	// Get the IPs for a given domain name.
	GetDomainIPs(domain string) []string
}

type DomainInfoChangeHandler interface {
	// Handle a domainInfoChanged message and report if the dataplane needs syncing.
	OnDomainInfoChange(msg *domainInfoChanged) (dataplaneSyncNeeded bool)
}

func newIPSetsManager(ipsets_ ipsetsDataplane, maxIPSetSize int, domainInfoStore store, callbacks *callbacks) *ipSetsManager {
	return &ipSetsManager{
		ipsetsDataplane: ipsets_,
		maxSize:         maxIPSetSize,
		callbacks:       newIPSetsManagerCallbacks(callbacks, ipsets_.GetIPFamily()),
		domainInfoStore: domainInfoStore,

		domainSetProgramming: make(map[string]map[string]set.Set),
		domainSetIds:         make(map[string]set.Set),
	}
}

func (m *ipSetsManager) GetIPSetType(setID string) (ipsets.IPSetType, error) {
	return m.ipsetsDataplane.GetTypeOf(setID)
}

func (m *ipSetsManager) GetIPSetMembers(setID string) (set.Set /*<string>*/, error) {
	return m.ipsetsDataplane.GetMembers(setID)
}

func (m *ipSetsManager) OnUpdate(msg interface{}) {
	switch msg := msg.(type) {
	// IP set-related messages, these are extremely common.
	case *proto.IPSetDeltaUpdate:
		log.WithField("ipSetId", msg.Id).Debug("IP set delta update")
		if m.domainSetProgramming[msg.Id] != nil {
			// Work needed to resolve domain name deltas against the current ipset
			// programming.
			m.handleDomainIPSetDeltaUpdate(msg.Id, msg.RemovedMembers, msg.AddedMembers)
		} else {
			// Pass deltas directly to the ipsets dataplane layer.
			m.ipsetsDataplane.AddMembers(msg.Id, msg.AddedMembers)
			m.callbacks.InvokeAddMembersIPSet(msg.Id, membersToSet(msg.AddedMembers))
			m.ipsetsDataplane.RemoveMembers(msg.Id, msg.RemovedMembers)
			m.callbacks.InvokeRemoveMembersIPSet(msg.Id, membersToSet(msg.RemovedMembers))
		}
	case *proto.IPSetUpdate:
		log.WithField("ipSetId", msg.Id).Debug("IP set update")
		var setType ipsets.IPSetType
		switch msg.Type {
		case proto.IPSetUpdate_IP:
			setType = ipsets.IPSetTypeHashIP
		case proto.IPSetUpdate_NET:
			setType = ipsets.IPSetTypeHashNet
		case proto.IPSetUpdate_IP_AND_PORT:
			setType = ipsets.IPSetTypeHashIPPort
		case proto.IPSetUpdate_DOMAIN:
			setType = ipsets.IPSetTypeHashIP
		default:
			log.WithField("type", msg.Type).Panic("Unknown IP set type")
		}
		metadata := ipsets.IPSetMetadata{
			Type:    setType,
			SetID:   msg.Id,
			MaxSize: m.maxSize,
		}
		if msg.Type == proto.IPSetUpdate_DOMAIN {
			// Work needed to resolve domain names to expiring IPs.
			m.handleDomainIPSetUpdate(msg, &metadata)
		} else {
			// Pass directly onto the ipsets dataplane layer.
			m.ipsetsDataplane.AddOrReplaceIPSet(metadata, msg.Members)
			m.callbacks.InvokeReplaceIPSet(msg.Id, membersToSet(msg.Members))
		}
	case *proto.IPSetRemove:
		log.WithField("ipSetId", msg.Id).Debug("IP set remove")
		if m.domainSetProgramming[msg.Id] != nil {
			// Remove tracking data for this domain set.
			m.removeDomainIPSetTracking(msg.Id)
		}
		m.ipsetsDataplane.RemoveIPSet(msg.Id)
		if m.domainSetProgramming[msg.Id] == nil {
			// Note: no XDP callbacks for domain IP set removal because XDP is
			// for ingress policy only and domain IP sets are egress only.
			m.callbacks.InvokeRemoveIPSet(msg.Id)
		}
	}
}

func (m *ipSetsManager) CompleteDeferredWork() error {
	// Nothing to do, we don't defer any work.
	return nil
}

func (m *ipSetsManager) domainIncludedInSet(domain string, ipSetId string) {
	if m.domainSetIds[domain] != nil {
		m.domainSetIds[domain].Add(ipSetId)
	} else {
		m.domainSetIds[domain] = set.From(ipSetId)
	}
}

func (m *ipSetsManager) domainRemovedFromSet(domain string, ipSetId string) {
	if m.domainSetIds[domain] != nil {
		m.domainSetIds[domain].Discard(ipSetId)
		if m.domainSetIds[domain].Len() == 0 {
			delete(m.domainSetIds, domain)
		}
	}
}

func (m *ipSetsManager) handleDomainIPSetUpdate(msg *proto.IPSetUpdate, metadata *ipsets.IPSetMetadata) {
	log.Infof("Update whole domain set: msg=%v metadata=%v", msg, metadata)

	if m.domainSetProgramming[msg.Id] != nil {
		log.Info("IPSetUpdate for existing IP set")
		domainsToRemove := set.New()
		domainsToAdd := set.FromArray(msg.Members)
		for domain, domainSetIds := range m.domainSetIds {
			if domainSetIds.Contains(msg.Id) {
				// Domain set previously included this domain name.
				if domainsToAdd.Contains(domain) {
					// And it still should, so don't re-add it.
					domainsToAdd.Discard(domain)
				} else {
					// And now it doesn't, so remove it.
					domainsToRemove.Add(domain)
				}
			}
		}
		m.handleDomainIPSetDeltaUpdate(msg.Id, setToSlice(domainsToRemove), setToSlice(domainsToAdd))
		return
	}

	// Accumulator for the IPs that we need to program for this domain set.
	ipToDomains := make(map[string]set.Set)

	// For each domain name in this set...
	for _, domain := range msg.Members {
		// Update the reverse map that tells us all of the domain sets that include a given
		// domain name.
		m.domainIncludedInSet(domain, msg.Id)

		// Merge the IPs for this domain into the accumulator.
		for _, ip := range m.domainInfoStore.GetDomainIPs(domain) {
			if ipToDomains[ip] == nil {
				ipToDomains[ip] = set.New()
			}
			ipToDomains[ip].Add(domain)
		}
	}

	// Convert that to a list of members for the ipset dataplane layer to program.
	ipMembers := make([]string, 0, len(ipToDomains))
	for ip := range ipToDomains {
		ipMembers = append(ipMembers, ip)
	}
	// Note: no XDP callbacks here because XDP is for ingress policy only and domain
	// IP sets are egress only.
	m.ipsetsDataplane.AddOrReplaceIPSet(*metadata, ipMembers)

	// Record the programming that we've asked the dataplane for.
	m.domainSetProgramming[msg.Id] = ipToDomains
}

func setToSlice(setOfThings set.Set) []string {
	slice := make([]string, 0, setOfThings.Len())
	setOfThings.Iter(func(item interface{}) error {
		slice = append(slice, item.(string))
		return nil
	})
	return slice
}

func (m *ipSetsManager) handleDomainIPSetDeltaUpdate(ipSetId string, domainsRemoved []string, domainsAdded []string) {
	log.Infof("Domain set delta update: id=%v removed=%v added=%v", ipSetId, domainsRemoved, domainsAdded)

	// Get the current programming for this domain set.
	ipToDomains := m.domainSetProgramming[ipSetId]
	if ipToDomains == nil {
		log.Panic("Got IPSetDeltaUpdate for an unknown IP set")
	}

	// Accumulators for the IPs that we need to remove and add.  Do remove processing first, so
	// that it works to process a domain info change by calling this function with the same
	// domain name being removed and then added again.
	ipsToRemove := set.New()
	ipsToAdd := set.New()

	// For each removed domain name...
	for _, domain := range domainsRemoved {
		// Update the reverse map that tells us all of the domain sets that include a given
		// domain name.
		m.domainRemovedFromSet(domain, ipSetId)
	}

	// For each programmed IP...
	for ip, domains := range ipToDomains {
		// Remove the removed domains.
		for _, domain := range domainsRemoved {
			domains.Discard(domain)
		}
		if domains.Len() == 0 {
			// We should remove this IP now.
			ipsToRemove.Add(ip)
			delete(ipToDomains, ip)
		}
	}

	// For each new domain name...
	for _, domain := range domainsAdded {
		// Update the reverse map that tells us all of the domain sets that include a given
		// domain name.
		m.domainIncludedInSet(domain, ipSetId)

		// Get the IPs and expiry times for this domain, then merge those into the current
		// programming, noting any updates that we need to send to the dataplane.
		for _, ip := range m.domainInfoStore.GetDomainIPs(domain) {
			if ipToDomains[ip] == nil {
				ipToDomains[ip] = set.New()
				ipsToAdd.Add(ip)
			}
			ipToDomains[ip].Add(domain)
		}
	}

	// If there are any IPs that are now in both ipsToRemove and ipsToAdd, we don't need either
	// to add or remove those IPs.
	ipsToRemove.Iter(func(item interface{}) error {
		if ipsToAdd.Contains(item) {
			ipsToAdd.Discard(item)
			return set.RemoveItem
		}
		return nil
	})

	// Pass IP deltas onto the ipsets dataplane layer.  Note: no XDP callbacks here
	// because XDP is for ingress policy only and domain IP sets are egress only.
	m.ipsetsDataplane.RemoveMembers(ipSetId, setToSlice(ipsToRemove))
	m.ipsetsDataplane.AddMembers(ipSetId, setToSlice(ipsToAdd))
}

func (m *ipSetsManager) removeDomainIPSetTracking(ipSetId string) {
	log.Infof("Domain set removed: id=%v", ipSetId)
	for domain, _ := range m.domainSetIds {
		m.domainRemovedFromSet(domain, ipSetId)
	}
	delete(m.domainSetProgramming, ipSetId)
}

func (m *ipSetsManager) OnDomainInfoChange(msg *domainInfoChanged) (dataplaneSyncNeeded bool) {
	log.WithFields(log.Fields{"domain": msg.domain, "reason": msg.reason}).Info("Domain info changed")

	// Find the affected domain sets.
	domainSetIds := m.domainSetIds[msg.domain]
	if domainSetIds != nil {
		// This is a domain name of active interest, so report that a dataplane sync will be
		// needed.
		dataplaneSyncNeeded = true

		// Tell each domain set that includes this domain name to requery the IPs for the
		// domain name and adjust its overall IP set accordingly.
		domainSetIds.Iter(func(item interface{}) error {
			// Handle as a delta update where the same domain name is removed and then re-added.
			m.handleDomainIPSetDeltaUpdate(item.(string), []string{msg.domain}, []string{msg.domain})
			return nil
		})
	}

	return
}

func membersToSet(members []string) set.Set /*string*/ {
	membersSet := set.New()
	for _, m := range members {
		membersSet.Add(m)
	}

	return membersSet
}
