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

package dnsresolver_test

import (
	"testing"

	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/felix/bpf/dnsresolver"
	"github.com/projectcalico/calico/felix/bpf/maps"
)

func TestDomainTracker(t *testing.T) {
	RegisterTestingT(t)

	log.SetLevel(log.DebugLevel)

	ids := map[string]uint64{
		"1":    1,
		"2":    2,
		"3":    3,
		"4":    4,
		"5":    5,
		"123":  123,
		"1234": 1234,
		"666":  666,
	}

	tracker, err := dnsresolver.NewDomainTracker(func(s string) uint64 {
		return ids[s]
	})
	Expect(err).NotTo(HaveOccurred())
	defer tracker.Close()

	dnsPfxMap := tracker.Map()
	defer dnsPfxMap.(*maps.PinnedMap).Unpin()

	tracker.Add("ubuntu.com", "123")
	tracker.Add("*.ubuntu.com", "1234")
	tracker.Add("archive.ubuntu.com", "1", "2", "3")
	err = tracker.ApplyAllChanges()
	Expect(err).NotTo(HaveOccurred())

	v, err := dnsPfxMap.Get(dnsresolver.NewPfxKey("ubuntu.com").AsBytes())
	Expect(err).NotTo(HaveOccurred())
	Expect(dnsresolver.DNSPfxValueFromBytes(v).IDs()).To(ContainElements(uint64(123)))

	v, err = dnsPfxMap.Get(dnsresolver.NewPfxKey("*.ubuntu.com").AsBytes())
	Expect(err).NotTo(HaveOccurred())
	Expect(dnsresolver.DNSPfxValueFromBytes(v).IDs()).To(ContainElements(uint64(1234)))

	v, err = dnsPfxMap.Get(dnsresolver.NewPfxKey("archive.ubuntu.com").AsBytes())
	Expect(err).NotTo(HaveOccurred())
	Expect(dnsresolver.DNSPfxValueFromBytes(v).IDs()).To(ContainElements(uint64(1), uint64(2), uint64(3)))

	tracker.Add("archive.ubuntu.com", "1")
	err = tracker.ApplyAllChanges()
	Expect(err).NotTo(HaveOccurred())

	v, err = dnsPfxMap.Get(dnsresolver.NewPfxKey("archive.ubuntu.com").AsBytes())
	Expect(err).NotTo(HaveOccurred())
	Expect(dnsresolver.DNSPfxValueFromBytes(v).IDs()).To(ContainElements(uint64(1), uint64(2), uint64(3)))

	tracker.Del("archive.ubuntu.com", "1", "3")
	err = tracker.ApplyAllChanges()
	Expect(err).NotTo(HaveOccurred())

	v, err = dnsPfxMap.Get(dnsresolver.NewPfxKey("archive.ubuntu.com").AsBytes())
	Expect(err).NotTo(HaveOccurred())
	Expect(dnsresolver.DNSPfxValueFromBytes(v).IDs()).To(ContainElements(uint64(2)))

	tracker.Add("archive.ubuntu.com", "1", "2", "3")
	err = tracker.ApplyAllChanges()
	Expect(err).NotTo(HaveOccurred())

	v, err = dnsPfxMap.Get(dnsresolver.NewPfxKey("archive.ubuntu.com").AsBytes())
	Expect(err).NotTo(HaveOccurred())
	Expect(dnsresolver.DNSPfxValueFromBytes(v).IDs()).To(ContainElements(uint64(1), uint64(2), uint64(3)))

	tracker.Del("archive.ubuntu.com", "5")
	err = tracker.ApplyAllChanges()
	Expect(err).NotTo(HaveOccurred())

	v, err = dnsPfxMap.Get(dnsresolver.NewPfxKey("archive.ubuntu.com").AsBytes())
	Expect(err).NotTo(HaveOccurred())
	Expect(dnsresolver.DNSPfxValueFromBytes(v).IDs()).To(ContainElements(uint64(1), uint64(2), uint64(3)))

	tracker.Del("archive.ubuntu.com", "1", "2", "3")
	err = tracker.ApplyAllChanges()
	Expect(err).NotTo(HaveOccurred())

	v, err = dnsPfxMap.Get(dnsresolver.NewPfxKey("archive.ubuntu.com").AsBytes())
	Expect(err).NotTo(HaveOccurred())
	// matches *.ubuntu.com as archive.ubuntu.com is gone
	Expect(dnsresolver.DNSPfxValueFromBytes(v).IDs()).To(ContainElements(uint64(1234)))

	tracker.Del("ubuntu.com")
	err = tracker.ApplyAllChanges()
	Expect(err).NotTo(HaveOccurred())

	v, err = dnsPfxMap.Get(dnsresolver.NewPfxKey("ubuntu.com").AsBytes())
	Expect(err).NotTo(HaveOccurred())
	Expect(dnsresolver.DNSPfxValueFromBytes(v).IDs()).To(ContainElements(uint64(123)))

	tracker.Del("ubuntu.com", "unknown")
	err = tracker.ApplyAllChanges()
	Expect(err).NotTo(HaveOccurred())

	v, err = dnsPfxMap.Get(dnsresolver.NewPfxKey("ubuntu.com").AsBytes())
	Expect(err).NotTo(HaveOccurred())
	Expect(dnsresolver.DNSPfxValueFromBytes(v).IDs()).To(ContainElements(uint64(123)))

	tracker.Close()

	// Create a new tracker and fill it from DP
	tracker, err = dnsresolver.NewDomainTracker(func(s string) uint64 {
		return ids[s]
	})
	Expect(err).NotTo(HaveOccurred())
	defer tracker.Close()

	tracker.Add("ubuntu.com", "4")
	tracker.Add("*.ubuntu.com", "4")
	tracker.Add("archive.ubuntu.com", "4")
	err = tracker.ApplyAllChanges()
	Expect(err).NotTo(HaveOccurred())

	v, err = dnsPfxMap.Get(dnsresolver.NewPfxKey("ubuntu.com").AsBytes())
	Expect(err).NotTo(HaveOccurred())
	Expect(dnsresolver.DNSPfxValueFromBytes(v).IDs()).To(ContainElements(uint64(123), uint64(4)))

	v, err = dnsPfxMap.Get(dnsresolver.NewPfxKey("*.ubuntu.com").AsBytes())
	Expect(err).NotTo(HaveOccurred())
	Expect(dnsresolver.DNSPfxValueFromBytes(v).IDs()).To(ContainElements(uint64(1234), uint64(4)))

	v, err = dnsPfxMap.Get(dnsresolver.NewPfxKey("archive.ubuntu.com").AsBytes())
	Expect(err).NotTo(HaveOccurred())
	Expect(dnsresolver.DNSPfxValueFromBytes(v).IDs()).To(ContainElements(uint64(1), uint64(2), uint64(3), uint64(4)))
}
