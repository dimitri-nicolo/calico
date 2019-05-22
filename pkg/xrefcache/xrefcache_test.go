// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package xrefcache_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	log "github.com/sirupsen/logrus"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	. "github.com/tigera/compliance/internal/testutils"
	"github.com/tigera/compliance/pkg/resources"
	"github.com/tigera/compliance/pkg/syncer"
	"github.com/tigera/compliance/pkg/xrefcache"
)

var _ = Describe("xref cache", func() {
	// Ensure  the client resource list is in-sync with the resource helper.
	It("should support in-sync and complete with no injected configuration", func() {
		cache := xrefcache.NewXrefCache(func() {
			log.Info("Healthy notification from xref cache")
		})
		cache.OnStatusUpdate(syncer.StatusUpdate{
			Type: syncer.StatusTypeInSync,
		})
		cache.OnStatusUpdate(syncer.StatusUpdate{
			Type: syncer.StatusTypeComplete,
		})
	})
})

type callbacks struct {
	deletes int
	sets    int
	updated map[apiv3.ResourceID]*xrefcache.CacheEntryEndpoint
}

func (c *callbacks) onUpdate(update syncer.Update) {
	if update.Type&xrefcache.EventResourceDeleted != 0 {
		delete(c.updated, update.ResourceID)
		c.deletes++
	} else {
		c.updated[update.ResourceID] = update.Resource.(*xrefcache.CacheEntryEndpoint)
		c.sets++
	}
}

var _ = Describe("xref cache in-scope callbacks", func() {
	var cb *callbacks
	var tester *XrefCacheTester
	var nsID1 apiv3.ResourceID
	var saName1 string
	var saName2 string
	var podID1 apiv3.ResourceID
	var podID2 apiv3.ResourceID
	var podID3 apiv3.ResourceID
	var podID4 apiv3.ResourceID

	BeforeEach(func() {
		tester = NewXrefCacheTester()
		cb = &callbacks{
			updated: make(map[apiv3.ResourceID]*xrefcache.CacheEntryEndpoint),
		}
		ns := tester.SetNamespace(Namespace1, Label1)
		nsID1 = resources.GetResourceID(ns)
		tester.SetNamespace(Namespace2, Label2)
		sa := tester.SetServiceAccount(Name1, Namespace1, Label1)
		saName1 = sa.GetObjectMeta().GetName()
		sa = tester.SetServiceAccount(Name2, Namespace1, Label2)
		saName2 = sa.GetObjectMeta().GetName()
		tester.SetServiceAccount(Name1, Namespace2, Label1)
		tester.SetServiceAccount(Name2, Namespace2, Label2)
		pod := tester.SetPod(Name1, Namespace1, Label1, IP1, Name2, NoPodOptions)
		podID1 = resources.GetResourceID(pod)
		pod = tester.SetPod(Name2, Namespace1, Label2, IP1, Name1, NoPodOptions)
		podID2 = resources.GetResourceID(pod)
		pod = tester.SetPod(Name2, Namespace2, Label1, IP1, Name2, NoPodOptions)
		podID3 = resources.GetResourceID(pod)
		pod = tester.SetPod(Name1, Namespace2, Label2, IP1, Name1, NoPodOptions)
		podID4 = resources.GetResourceID(pod)
		for _, k := range xrefcache.KindsEndpoint {
			tester.RegisterOnUpdateHandler(k, xrefcache.EventInScope, cb.onUpdate)
		}
	})

	It("should flag in-scope endpoints with no endpoint selector", func() {
		tester.RegisterInScopeEndpoints(nil)
		Expect(cb.updated).To(HaveLen(0))
		tester.OnStatusUpdate(syncer.StatusUpdate{
			Type: syncer.StatusTypeInSync,
		})
		Expect(cb.updated).To(HaveLen(4))
		Expect(cb.updated).To(HaveKey(podID1))
		Expect(cb.updated).To(HaveKey(podID2))
		Expect(cb.updated).To(HaveKey(podID3))
		Expect(cb.updated).To(HaveKey(podID4))
	})

	It("should flag in-scope endpoints matching endpoint selector", func() {
		tester.RegisterInScopeEndpoints(&apiv3.EndpointsSelection{
			Selector: tester.GetSelector(Select1),
		})
		Expect(cb.updated).To(HaveLen(0))
		tester.OnStatusUpdate(syncer.StatusUpdate{
			Type: syncer.StatusTypeInSync,
		})
		Expect(cb.updated).To(HaveLen(2))
		Expect(cb.updated).To(HaveKey(podID1))
		Expect(cb.updated).To(HaveKey(podID3))
	})

	It("should flag in-scope endpoints matching endpoint selector and namespace name", func() {
		tester.RegisterInScopeEndpoints(&apiv3.EndpointsSelection{
			Selector: tester.GetSelector(Select1),
			Namespaces: &apiv3.NamesAndLabelsMatch{
				Names: []string{nsID1.Name},
			},
		})
		Expect(cb.updated).To(HaveLen(0))
		tester.OnStatusUpdate(syncer.StatusUpdate{
			Type: syncer.StatusTypeInSync,
		})
		Expect(cb.updated).To(HaveLen(1))
		Expect(cb.updated).To(HaveKey(podID1))
	})

	It("should flag in-scope endpoints matching endpoint selector and namespace selector", func() {
		tester.RegisterInScopeEndpoints(&apiv3.EndpointsSelection{
			Selector: tester.GetSelector(Select1),
			Namespaces: &apiv3.NamesAndLabelsMatch{
				Selector: tester.GetSelector(Select2),
			},
		})
		Expect(cb.updated).To(HaveLen(0))
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
		tester.RegisterInScopeEndpoints(&apiv3.EndpointsSelection{
			Selector: tester.GetSelector(Select2),
			ServiceAccounts: &apiv3.NamesAndLabelsMatch{
				Names: []string{saName1},
			},
		})
		Expect(cb.updated).To(HaveLen(0))
		tester.OnStatusUpdate(syncer.StatusUpdate{
			Type: syncer.StatusTypeInSync,
		})
		Expect(cb.updated).To(HaveLen(2))
		Expect(cb.updated).To(HaveKey(podID2))
		Expect(cb.updated).To(HaveKey(podID4))
	})

	It("should flag in-scope endpoints by service account selector", func() {
		tester.RegisterInScopeEndpoints(&apiv3.EndpointsSelection{
			ServiceAccounts: &apiv3.NamesAndLabelsMatch{
				Selector: tester.GetSelector(Select2),
			},
		})
		Expect(cb.updated).To(HaveLen(0))
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
		tester.RegisterInScopeEndpoints(&apiv3.EndpointsSelection{
			Selector: tester.GetSelector(Select1),
			ServiceAccounts: &apiv3.NamesAndLabelsMatch{
				Selector: tester.GetSelector(Select2),
			},
		})
		Expect(cb.updated).To(HaveLen(0))
		tester.OnStatusUpdate(syncer.StatusUpdate{
			Type: syncer.StatusTypeInSync,
		})
		Expect(cb.updated).To(HaveLen(2))
		Expect(cb.updated).To(HaveKey(podID1))
		Expect(cb.updated).To(HaveKey(podID3))
	})

	It("should flag in-scope endpoints multiple service account names", func() {
		tester.RegisterInScopeEndpoints(&apiv3.EndpointsSelection{
			ServiceAccounts: &apiv3.NamesAndLabelsMatch{
				Names: []string{saName1, saName2},
			},
		})
		Expect(cb.updated).To(HaveLen(0))
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

var _ = Describe("xref cache multiple update transactions", func() {
	It("should handle an update transaction containing create and delete of the same resource when in-sync", func() {
		tester := NewXrefCacheTester()
		By("Setting in-sync")
		tester.OnStatusUpdate(syncer.StatusUpdate{
			Type: syncer.StatusTypeInSync,
		})
		cb := &callbacks{
			updated: make(map[apiv3.ResourceID]*xrefcache.CacheEntryEndpoint),
		}

		By("Registering all endpoints as in-scope and registering for inscope events")
		tester.RegisterInScopeEndpoints(&apiv3.EndpointsSelection{})
		for _, k := range xrefcache.KindsEndpoint {
			tester.RegisterOnUpdateHandler(k, xrefcache.EventInScope, cb.onUpdate)
		}

		By("Setting tester to accumlate updates and creating a pod and service account")
		tester.AccumlateUpdates = true
		tester.SetNamespace(Namespace1, Label1)
		tester.SetPod(Name1, Namespace1, Label1, IP1, Name2, NoPodOptions)

		By("Setting tester to no longer accumlate updates and deleting the pod")
		tester.AccumlateUpdates = false
		tester.DeletePod(Name1, Namespace1)

		By("Checking both updates")
		Expect(cb.updated).To(HaveLen(0))
		Expect(cb.sets).To(Equal(1))
		Expect(cb.deletes).To(Equal(1))
	})

	It("should handle an update transaction containing create, delete, recreate of the same resource when in-sync", func() {
		tester := NewXrefCacheTester()
		By("Setting in-sync")
		tester.OnStatusUpdate(syncer.StatusUpdate{
			Type: syncer.StatusTypeInSync,
		})
		cb := &callbacks{
			updated: make(map[apiv3.ResourceID]*xrefcache.CacheEntryEndpoint),
		}

		By("Registering all endpoints as in-scope and registering for inscope events")
		tester.RegisterInScopeEndpoints(&apiv3.EndpointsSelection{})
		for _, k := range xrefcache.KindsEndpoint {
			tester.RegisterOnUpdateHandler(k, xrefcache.EventInScope, cb.onUpdate)
		}

		By("Setting tester to accumlate updates and creating a pod and service account")
		tester.AccumlateUpdates = true
		tester.SetNamespace(Namespace1, Label1)
		tester.SetPod(Name1, Namespace1, Label1, IP1, Name2, NoPodOptions)

		By("Deleting the pod")
		tester.DeletePod(Name1, Namespace1)

		By("Setting tester to no longer accumlate updates and recreating the pod")
		tester.AccumlateUpdates = false
		pod := tester.SetPod(Name1, Namespace1, Label1, IP1, Name2, NoPodOptions)
		podID1 := resources.GetResourceID(pod)

		By("Checking for all three updates and newly created pod")
		Expect(cb.updated).To(HaveLen(1))
		Expect(cb.updated).To(HaveKey(podID1))
		Expect(cb.sets).To(Equal(2))
		Expect(cb.deletes).To(Equal(1))
	})

	It("should handle an update transaction containing create and delete of the same resource before being in-sync", func() {
		tester := NewXrefCacheTester()
		cb := &callbacks{
			updated: make(map[apiv3.ResourceID]*xrefcache.CacheEntryEndpoint),
		}

		By("Registering all endpoints as in-scope and registering for inscope events")
		tester.RegisterInScopeEndpoints(&apiv3.EndpointsSelection{})
		for _, k := range xrefcache.KindsEndpoint {
			tester.RegisterOnUpdateHandler(k, xrefcache.EventInScope, cb.onUpdate)
		}

		By("Setting tester to accumlate updates and creating a pod and service account")
		tester.AccumlateUpdates = true
		tester.SetNamespace(Namespace1, Label1)
		tester.SetPod(Name1, Namespace1, Label1, IP1, Name2, NoPodOptions)

		By("Setting tester to no longer accumlate updates and deleting the pod")
		tester.AccumlateUpdates = false
		tester.DeletePod(Name1, Namespace1)

		By("Setting in-sync")
		tester.OnStatusUpdate(syncer.StatusUpdate{
			Type: syncer.StatusTypeInSync,
		})

		By("Checking for no updates")
		Expect(cb.updated).To(HaveLen(0))
		Expect(cb.sets).To(Equal(0))
		Expect(cb.deletes).To(Equal(0))
	})

	It("should handle an update transaction containing create, delete, recreate of the same resource before being in-sync", func() {
		tester := NewXrefCacheTester()
		cb := &callbacks{
			updated: make(map[apiv3.ResourceID]*xrefcache.CacheEntryEndpoint),
		}

		By("Registering all endpoints as in-scope and registering for inscope events")
		tester.RegisterInScopeEndpoints(&apiv3.EndpointsSelection{})
		for _, k := range xrefcache.KindsEndpoint {
			tester.RegisterOnUpdateHandler(k, xrefcache.EventInScope, cb.onUpdate)
		}

		By("Setting tester to accumlate updates and creating a pod and service account")
		tester.AccumlateUpdates = true
		tester.SetNamespace(Namespace1, Label1)
		tester.SetPod(Name1, Namespace1, Label1, IP1, Name2, NoPodOptions)

		By("Deleting the pod")
		tester.DeletePod(Name1, Namespace1)

		By("Setting tester to no longer accumlate updates and recreating the pod")
		tester.AccumlateUpdates = false
		pod := tester.SetPod(Name1, Namespace1, Label1, IP1, Name2, NoPodOptions)
		podID1 := resources.GetResourceID(pod)

		By("Checking for no updates")
		Expect(cb.updated).To(HaveLen(0))
		Expect(cb.sets).To(Equal(0))
		Expect(cb.deletes).To(Equal(0))

		By("Setting in-sync")
		tester.OnStatusUpdate(syncer.StatusUpdate{
			Type: syncer.StatusTypeInSync,
		})

		By("Checking for just a single update for the newly created pod")
		Expect(cb.updated).To(HaveLen(1))
		Expect(cb.updated).To(HaveKey(podID1))
		Expect(cb.deletes).To(Equal(0))
		Expect(cb.sets).To(Equal(1))
	})
})
