// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package xrefcache_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/tigera/compliance/internal/testutils"
	"github.com/tigera/compliance/pkg/syncer"
	"github.com/tigera/compliance/pkg/xrefcache"
)

var _ = Describe("xref cache", func() {
	var tester *XRefCacheTester

	BeforeEach(func() {
		tester = NewXrefCacheTester()
	})

	// Ensure  the client resource list is in-sync with the resource helper.
	It("should support in-sync and complete with no injected configuration", func() {
		cache := xrefcache.NewXrefCache()
		cache.OnStatusUpdate(syncer.StatusUpdate{
			Type: syncer.StatusTypeInSync,
		})
		cache.OnStatusUpdate(syncer.StatusUpdate{
			Type: syncer.StatusTypeComplete,
		})
	})

	// Ensure  the client resource list is in-sync with the resource helper.
	It("should identify a network set with internet exposed", func() {
		By("applying a network set with no nets")
		tester.SetGlobalNetworkSet(1, NoLabels, nil)

		By("checking the cache settings")
		ns := tester.GetGlobalNetworkSet(1)
		Expect(ns).ToNot(BeNil())
		Expect(ns.Flags).To(BeZero())

		By("applying a network set with one public net")
		tester.SetGlobalNetworkSet(1, Label1, []string{"1.1.1.1/32"})

		By("checking the cache settings")
		ns = tester.GetGlobalNetworkSet(1)
		Expect(ns).ToNot(BeNil())
		Expect(ns.Flags).To(Equal(xrefcache.EventInternetAddressExposed))

		By("applying a network set with one private net")
		tester.SetGlobalNetworkSet(1, Label1, []string{"10.0.0.0/16"})

		By("checking the cache settings")
		ns = tester.GetGlobalNetworkSet(1)
		Expect(ns).ToNot(BeNil())
		Expect(ns.Flags).To(BeZero())

		By("applying a network set with one private and one public net")
		tester.SetGlobalNetworkSet(1, Label1, []string{"10.0.0.0/16", "1.1.1.1/32", "10.10.0.0/16"})

		By("checking the cache settings")
		ns = tester.GetGlobalNetworkSet(1)
		Expect(ns).ToNot(BeNil())
		Expect(ns.Flags).To(Equal(xrefcache.EventInternetAddressExposed))

		By("deleting the network set")
		tester.SetGlobalNetworkSet(1, Label1, []string{"10.0.0.0/16"})

		By("checking the cache settings")
		ns = tester.GetGlobalNetworkSet(1)
		Expect(ns).To(BeNil())
	})
})
