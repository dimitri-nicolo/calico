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

	"github.com/projectcalico/calico/felix/bpf/maps"
	"github.com/projectcalico/calico/felix/cachingmap"
)

type DomainTracker struct {
	m          maps.Map
	pfxMap     *cachingmap.CachingMap[DNSPfxKey, DNSPfxValue]
	strToUin64 func(string) uint64
}

func NewDomainTracker(strToUin64 func(string) uint64) (*DomainTracker, error) {
	m := DNSPrefixMap()
	err := m.EnsureExists()

	if err != nil {
		return nil, fmt.Errorf("could not create BPF map: %w", err)
	}

	d := &DomainTracker{
		m: m,
		pfxMap: cachingmap.New[DNSPfxKey, DNSPfxValue](m.GetName(),
			maps.NewTypedMap[DNSPfxKey, DNSPfxValue](
				m.(maps.MapWithExistsCheck), DNSPfxKeyFromBytes, DNSPfxValueFromBytes,
			)),
		strToUin64: strToUin64,
	}

	err = d.pfxMap.LoadCacheFromDataplane()
	if err != nil {
		return nil, fmt.Errorf("could not load data from dataplane: %w", err)
	}

	return d, nil
}

func (d *DomainTracker) Add(domain string, setIDs ...string) {
	if len(setIDs) == 0 {
		return
	}

	k := NewPfxKey(domain)

	exists := make(map[uint64]struct{})

	v, ok := d.pfxMap.Desired().Get(k)
	if !ok {
		v, _ = d.pfxMap.Dataplane().Get(k)
	}

	ids64 := v.IDs()
	for _, id := range ids64 {
		exists[id] = struct{}{}
	}

	for _, id := range setIDs {
		id64 := d.strToUin64(id)
		if id64 != 0 {
			exists[id64] = struct{}{}
		}
	}

	ids64 = make([]uint64, 0, len(exists))
	for id := range exists {
		ids64 = append(ids64, id)
	}

	v = NewPfxValue(ids64...)
	d.pfxMap.Desired().Set(k, v)
}

func (d *DomainTracker) Del(domain string, setIDs ...string) {
	if len(setIDs) == 0 {
		return
	}

	k := NewPfxKey(domain)
	v, ok := d.pfxMap.Desired().Get(k)
	if !ok {
		return
	}

	exists := make(map[uint64]struct{})
	ids64 := v.IDs()
	for _, id := range ids64 {
		exists[id] = struct{}{}
	}

	for _, id := range setIDs {
		id64 := d.strToUin64(id)
		delete(exists, id64)
	}

	ids64 = make([]uint64, 0, len(exists))
	for id := range exists {
		ids64 = append(ids64, id)
	}

	if len(ids64) > 0 {
		v := NewPfxValue(ids64...)
		d.pfxMap.Desired().Set(k, v)
	} else {
		d.pfxMap.Desired().Delete(k)
	}
}

func (d *DomainTracker) ApplyAllChanges() error {
	return d.pfxMap.ApplyAllChanges()
}

func (d *DomainTracker) Close() {
	d.m.Close()
}

func (d *DomainTracker) Map() maps.Map {
	return d.m
}
