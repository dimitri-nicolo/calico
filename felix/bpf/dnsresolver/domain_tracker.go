//go:build !windows
// +build !windows

// Copyright (c) 2024 Tigera, Inc. All rights reserved.
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

package dnsresolver

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"gopkg.in/tchap/go-patricia.v2/patricia"

	"github.com/projectcalico/calico/felix/bpf/maps"
	"github.com/projectcalico/calico/felix/cachingmap"
	"github.com/projectcalico/calico/felix/idalloc"
	"github.com/projectcalico/calico/libcalico-go/lib/set"
)

type DomainTracker struct {
	mPfx          maps.Map
	mSets         maps.Map
	domainIDAlloc *idalloc.IDAllocator
	pfxMap        *cachingmap.CachingMap[DNSPfxKey, DNSPfxValue]
	setsMap       *cachingmap.CachingMap[DNSSetKey, DNSSetValue]
	setsAcc       *patricia.Trie
	strToUin64    func(string) uint64
}

type saItem struct {
	id       uint64
	wildcard bool
	acc      set.Set[uint64] /* accumulated */
	sets     set.Set[uint64] /* those sets which directly belong to this domain/wildcard */
}

func NewDomainTracker(strToUin64 func(string) uint64) (*DomainTracker, error) {
	mPfx := DNSPrefixMap()
	err := mPfx.EnsureExists()

	if err != nil {
		return nil, fmt.Errorf("could not create BPF map: %w", err)
	}

	mSets := DNSSetMap()
	err = mSets.EnsureExists()

	if err != nil {
		return nil, fmt.Errorf("could not create BPF map: %w", err)
	}

	return NewDomainTrackerWithMaps(strToUin64, mPfx, mSets)
}

func NewDomainTrackerWithMaps(strToUin64 func(string) uint64, mPfx, mSets maps.Map) (*DomainTracker, error) {
	d := &DomainTracker{
		mPfx:          mPfx,
		mSets:         mSets,
		domainIDAlloc: idalloc.New(),
		pfxMap: cachingmap.New[DNSPfxKey, DNSPfxValue](mPfx.GetName(),
			maps.NewTypedMap[DNSPfxKey, DNSPfxValue](
				mPfx.(maps.MapWithExistsCheck), DNSPfxKeyFromBytes, DNSPfxValueFromBytes,
			)),
		setsMap: cachingmap.New[DNSSetKey, DNSSetValue](mSets.GetName(),
			maps.NewTypedMap[DNSSetKey, DNSSetValue](
				mSets.(maps.MapWithExistsCheck), DNSSetKeyFromBytes, DNSSetValueFromBytes,
			)),
		setsAcc:    patricia.NewTrie(),
		strToUin64: strToUin64,
	}

	err := d.pfxMap.LoadCacheFromDataplane()
	if err != nil {
		return nil, fmt.Errorf("could not load data from dataplane: %w", err)
	}

	d.pfxMap.Dataplane().Iter(func(k DNSPfxKey, v DNSPfxValue) {
		domain := k.Domain()
		d.domainIDAlloc.ReserveWellKnownID(domain, uint64(v))
		log.WithFields(log.Fields{
			"domain": domain,
			"id":     uint64(v),
		}).Debug("Reserved id found in dataplane for domain")
	})

	return d, nil
}

func (d *DomainTracker) Add(domain string, setIDs ...string) {
	log.WithFields(log.Fields{
		"domain": domain,
		"setIDs": setIDs,
	}).Debug("Add")

	if len(setIDs) == 0 {
		return
	}

	wildcard := domain == "" || domain[0] == '*'

	k := NewPfxKey(domain)

	isKnown := false

	domainID := d.domainIDAlloc.GetNoAlloc(domain)
	if domainID != 0 {
		isKnown = true
	} else {
		domainID = d.domainIDAlloc.GetOrAlloc(domain)
	}

	v := NewPfxValue(domainID)
	d.pfxMap.Desired().Set(k, v)

	kb := k.LPMDomain()

	var current *saItem

	c := d.setsAcc.Get(kb)
	if c == nil {
		current = &saItem{
			id:       domainID,
			wildcard: wildcard,
			acc:      set.New[uint64](),
			sets:     set.New[uint64](),
		}
		d.setsAcc.Set(kb, current)
	} else {
		current = c.(*saItem)
	}

	for _, si := range setIDs {
		id64 := d.strToUin64(si)
		if id64 == 0 {
			log.Debugf("No uint64 id for domain %s string set id '%s'", domain, si)
			continue
		}

		if current.sets.Contains(id64) {
			log.Debugf("Set %s (0x%x) alredy belongs to domain %s", si, id64, string(kb))
			continue
		}
		log.Debugf("current %p", current)
		log.Debugf("current.sets %v", current.sets)

		current.acc.Add(id64)
		current.sets.Add(id64)

		log.Debugf("Adding set %s (0x%x) to domain %s", si, id64, string(kb))
		d.setsMap.Desired().Set(NewDNSSetKey(domainID, id64), DNSSetValueVoid)

		if wildcard {
			_ = d.setsAcc.VisitPrefixes(kb, func(pfx patricia.Prefix, item patricia.Item) error {
				if item.(*saItem).wildcard {
					log.Debugf("Adding set %s (0x%x) to wildcard prefix %s", si, id64, string(pfx))
					i := item.(*saItem)
					i.acc.Add(id64)
					d.setsMap.Desired().Set(NewDNSSetKey(i.id, id64), DNSSetValueVoid)
				}
				return nil
			})
			_ = d.setsAcc.VisitSubtree(kb, func(dom patricia.Prefix, item patricia.Item) error {
				log.Debugf("Adding set %s (0x%x) to domain %s", si, id64, string(dom))
				i := item.(*saItem)
				i.acc.Add(id64)
				d.setsMap.Desired().Set(NewDNSSetKey(i.id, id64), DNSSetValueVoid)
				return nil
			})
		}
	}

	if !isKnown && !wildcard {
		_ = d.setsAcc.VisitPrefixes(kb, func(pfx patricia.Prefix, item patricia.Item) error {
			i := item.(*saItem)
			if i.wildcard {
				log.Debugf("Adding wildcard %s set %s to domain %s", pfx, i.sets, domain)
				i.acc.AddSet(current.sets)

				i.sets.Iter(func(setid uint64) error {
					d.setsMap.Desired().Set(NewDNSSetKey(domainID, setid), DNSSetValueVoid)
					return nil
				})
			}
			return nil
		})
	}
}

func (d *DomainTracker) Del(domain string, setIDs ...string) {
	log.WithFields(log.Fields{
		"domain": domain,
		"setIDs": setIDs,
	}).Debug("Del")

	wildcard := domain == "" || domain[0] == '*'

	k := NewPfxKey(domain)

	kb := k.LPMDomain()

	c := d.setsAcc.Get(kb)
	if c == nil {
		return
	}
	current := c.(*saItem)

	domainID := d.domainIDAlloc.GetNoAlloc(domain)
	if domainID == 0 {
		return
	}

	for _, si := range setIDs {
		id64 := d.strToUin64(si)
		if id64 == 0 {
			log.Debugf("No uint64 id for domain %s string set id '%s'", domain, si)
			continue
		}

		if wildcard {
			_ = d.setsAcc.VisitSubtree(kb, func(dom patricia.Prefix, item patricia.Item) error {
				log.Debugf("Removing set %s (0x%x) from domain %s", si, id64, string(dom))
				i := item.(*saItem)
				i.acc.Discard(id64)
				d.setsMap.Desired().Delete(NewDNSSetKey(i.id, id64))
				return nil
			})
		}

		current.sets.Discard(id64)
		current.acc.Discard(id64)

		log.Debugf("Removing set %s (0x%x) from domain %s", si, id64, domain)
		d.setsMap.Desired().Delete(NewDNSSetKey(domainID, id64))
	}

	if current.sets.Len() == 0 {
		log.Debugf("Removing domain %s without sets", domain)
		d.setsAcc.Delete(kb)
		d.pfxMap.Desired().Delete(k)
	}
}

func (d *DomainTracker) ApplyAllChanges() error {
	if err := d.setsMap.ApplyAllChanges(); err != nil {
		return fmt.Errorf("ApplyAllChanges to DNS sets map: %w", err)
	}

	if err := d.pfxMap.ApplyAllChanges(); err != nil {
		return fmt.Errorf("ApplyAllChanges to DNS prefix map: %w", err)
	}

	return nil
}

func (d *DomainTracker) Close() {
	d.mPfx.Close()
	d.mSets.Close()
}

func (d *DomainTracker) Maps() []maps.Map {
	return []maps.Map{d.mPfx, d.mSets}
}
