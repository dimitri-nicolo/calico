// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package xrefcache_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	. "github.com/tigera/compliance/internal/testutils"
	"github.com/tigera/compliance/pkg/syncer"
	"github.com/tigera/compliance/pkg/xrefcache"
)

var _ = Describe("xref cache", func() {
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
})

type callbacks struct {
	updated map[apiv3.ResourceID]*xrefcache.CacheEntryEndpoint
}

func (c *callbacks) onUpdate(update syncer.Update) {
	c.updated[update.ResourceID] = update.Resource.(*xrefcache.CacheEntryEndpoint)
}

var _ = Describe("xref cache in-scope callbacks", func() {
	var cb *callbacks
	var tester *XrefCacheTester
	var nsID1 apiv3.ResourceID
	var saID1 apiv3.ResourceID
	var saID2 apiv3.ResourceID
	//var nsID2 apiv3.ResourceID
	var podID1 apiv3.ResourceID
	var podID2 apiv3.ResourceID
	var podID3 apiv3.ResourceID
	var podID4 apiv3.ResourceID

	BeforeEach(func() {
		tester = NewXrefCacheTester()
		cb = &callbacks{
			updated: make(map[apiv3.ResourceID]*xrefcache.CacheEntryEndpoint),
		}
		nsID1 = tester.SetNamespace(Namespace1, Label1)
		tester.SetNamespace(Namespace2, Label2)
		saID1 = tester.SetServiceAccount(Name1, Namespace1, Label1)
		saID2 = tester.SetServiceAccount(Name2, Namespace1, Label2)
		saID1 = tester.SetServiceAccount(Name1, Namespace2, Label1)
		saID2 = tester.SetServiceAccount(Name2, Namespace2, Label2)
		podID1 = tester.SetPod(Name1, Namespace1, Label1, IP1, Name2, NoPodOptions)
		podID2 = tester.SetPod(Name2, Namespace1, Label2, IP1, Name1, NoPodOptions)
		podID3 = tester.SetPod(Name2, Namespace2, Label1, IP1, Name2, NoPodOptions)
		podID4 = tester.SetPod(Name1, Namespace2, Label2, IP1, Name1, NoPodOptions)
		for _, k := range xrefcache.KindsEndpoint {
			tester.RegisterOnUpdateHandler(k, xrefcache.EventInScope, cb.onUpdate)
		}
	})

	It("should flag in-scope endpoints matching endpoint selector", func() {
		tester.RegisterInScopeEndpoints(apiv3.EndpointsSelection{
			EndpointSelector: tester.GetSelector(Select1),
		})
		tester.OnStatusUpdate(syncer.StatusUpdate{
			Type: syncer.StatusTypeInSync,
		})
		Expect(cb.updated).To(HaveLen(2))
		Expect(cb.updated).To(HaveKey(podID1))
		Expect(cb.updated).To(HaveKey(podID3))
	})

	It("should flag in-scope endpoints matching endpoint selector and namespace name", func() {
		tester.RegisterInScopeEndpoints(apiv3.EndpointsSelection{
			EndpointSelector: tester.GetSelector(Select1),
			Namespaces: &apiv3.NamesAndLabelsMatch{
				Names: []string{nsID1.Name},
			},
		})
		tester.OnStatusUpdate(syncer.StatusUpdate{
			Type: syncer.StatusTypeInSync,
		})
		Expect(cb.updated).To(HaveLen(1))
		Expect(cb.updated).To(HaveKey(podID1))
	})

	It("should flag in-scope endpoints matching endpoint selector and namespace selector", func() {
		tester.RegisterInScopeEndpoints(apiv3.EndpointsSelection{
			EndpointSelector: tester.GetSelector(Select1),
			Namespaces: &apiv3.NamesAndLabelsMatch{
				Selector: tester.GetSelector(Select2),
			},
		})
		tester.OnStatusUpdate(syncer.StatusUpdate{
			Type: syncer.StatusTypeInSync,
		})
		Expect(cb.updated).To(HaveLen(1))
		Expect(cb.updated).To(HaveKey(podID3))

		tester.SetNamespace(Namespace1, Label2)
		Expect(cb.updated).To(HaveLen(2))
		Expect(cb.updated).To(HaveKey(podID1))
		Expect(cb.updated).To(HaveKey(podID3))
	})

	It("should flag in-scope endpoints matching endpoint selector and service account name", func() {
		tester.RegisterInScopeEndpoints(apiv3.EndpointsSelection{
			EndpointSelector: tester.GetSelector(Select2),
			ServiceAccounts: &apiv3.NamesAndLabelsMatch{
				Names: []string{saID1.Name},
			},
		})
		tester.OnStatusUpdate(syncer.StatusUpdate{
			Type: syncer.StatusTypeInSync,
		})
		Expect(cb.updated).To(HaveLen(2))
		Expect(cb.updated).To(HaveKey(podID2))
		Expect(cb.updated).To(HaveKey(podID4))
	})

	It("should flag in-scope endpoints by service account selector", func() {
		tester.RegisterInScopeEndpoints(apiv3.EndpointsSelection{
			ServiceAccounts: &apiv3.NamesAndLabelsMatch{
				Selector: tester.GetSelector(Select2),
			},
		})
		tester.OnStatusUpdate(syncer.StatusUpdate{
			Type: syncer.StatusTypeInSync,
		})
		Expect(cb.updated).To(HaveLen(2))
		Expect(cb.updated).To(HaveKey(podID1))
		Expect(cb.updated).To(HaveKey(podID3))

		tester.SetServiceAccount(Name1, Namespace1, Label2)
		Expect(cb.updated).To(HaveLen(3))
		Expect(cb.updated).To(HaveKey(podID1))
		Expect(cb.updated).To(HaveKey(podID2))
		Expect(cb.updated).To(HaveKey(podID3))

		tester.SetServiceAccount(Name1, Namespace2, Label2)
		Expect(cb.updated).To(HaveLen(4))
		Expect(cb.updated).To(HaveKey(podID1))
		Expect(cb.updated).To(HaveKey(podID2))
		Expect(cb.updated).To(HaveKey(podID3))
		Expect(cb.updated).To(HaveKey(podID4))
	})

	It("should flag in-scope endpoints matching endpoint selector and service account selector", func() {
		tester.RegisterInScopeEndpoints(apiv3.EndpointsSelection{
			EndpointSelector: tester.GetSelector(Select1),
			ServiceAccounts: &apiv3.NamesAndLabelsMatch{
				Selector: tester.GetSelector(Select2),
			},
		})
		tester.OnStatusUpdate(syncer.StatusUpdate{
			Type: syncer.StatusTypeInSync,
		})
		Expect(cb.updated).To(HaveLen(2))
		Expect(cb.updated).To(HaveKey(podID1))
		Expect(cb.updated).To(HaveKey(podID3))
	})

	It("should flag in-scope endpoints multiple service account names", func() {
		tester.RegisterInScopeEndpoints(apiv3.EndpointsSelection{
			ServiceAccounts: &apiv3.NamesAndLabelsMatch{
				Names: []string{saID1.Name, saID2.Name},
			},
		})
		tester.OnStatusUpdate(syncer.StatusUpdate{
			Type: syncer.StatusTypeInSync,
		})
		Expect(cb.updated).To(HaveLen(4))
		Expect(cb.updated).To(HaveKey(podID1))
		Expect(cb.updated).To(HaveKey(podID2))
		Expect(cb.updated).To(HaveKey(podID3))
		Expect(cb.updated).To(HaveKey(podID4))
	})
})
