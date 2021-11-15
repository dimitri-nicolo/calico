// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package clientv3_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	"github.com/projectcalico/libcalico-go/lib/backend"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/testutils"
	"github.com/projectcalico/libcalico-go/lib/watch"
)

var _ = testutils.E2eDatastoreDescribe("UISettings tests", testutils.DatastoreAll, func(config apiconfig.CalicoAPIConfig) {

	ctx := context.Background()
	name1 := "uisettings-1"
	name2 := "uisettings-2"

	focus1 := []apiv3.UIGraphNode{
		{
			ID:        "idf1",
			Type:      "typef1",
			Name:      "f1",
			Namespace: "nsf1",
		},
		{
			ID:        "idf2",
			Type:      "typef2",
			Name:      "f2",
			Namespace: "nsf2",
		},
	}
	expanded1 := []apiv3.UIGraphNode{
		{
			ID:        "idex1",
			Type:      "typeex1",
			Name:      "ex1",
			Namespace: "nsex1",
		},
		{
			ID:        "idex2",
			Type:      "typeex2",
			Name:      "ex2",
			Namespace: "nsex2",
		},
	}
	namedselector1 := []apiv3.NamedSelector{
		{
			Name:     "ns1",
			Selector: "sel1 == '1'",
		},
		{
			Name:     "ns2",
			Selector: "sel2 != '2'",
		},
	}
	followedegress1 := []apiv3.UIGraphNode{
		{
			ID:        "ideg1",
			Type:      "typeeg1",
			Name:      "eg1",
			Namespace: "nseg1",
		},
		{
			ID:        "ideg2",
			Type:      "typeeg2",
			Name:      "eg1",
			Namespace: "nseg2",
		},
	}
	followedingress1 := []apiv3.UIGraphNode{
		{
			ID:        "idin1",
			Type:      "typein1",
			Name:      "in1",
			Namespace: "nsin1",
		},
		{
			ID:        "idin2",
			Type:      "typein2",
			Name:      "in2",
			Namespace: "nsin2",
		},
		{
			ID:        "idin2",
			Type:      "typein2",
			Name:      "in3",
			Namespace: "nsin2",
		},
	}
	position1 := []apiv3.Position{
		{
			ID:   "point1",
			XPos: 10,
			YPos: 100,
			ZPos: 9,
		},
		{
			ID:   "point1",
			XPos: 4,
			YPos: 65,
			ZPos: 87,
		},
	}
	layers1 := []string{
		"l1", "l2", "l3", "l4",
	}

	uigraphview1 := new(apiv3.UIGraphView)
	*uigraphview1 = apiv3.UIGraphView{
		Focus:                     focus1,
		Expanded:                  expanded1,
		ExpandPorts:               true,
		FollowConnectionDirection: false,
		SplitIngressEgress:        true,
		HostAggregationSelectors:  namedselector1,
		FollowedEgress:            followedegress1,
		FollowedIngress:           followedingress1,
		LayoutType:                "standard",
		Positions:                 position1,
		Layers:                    layers1,
	}

	nodes1 := []apiv3.UIGraphNode{
		{ID: "node/n1", Name: "n1"},
		{ID: "node/n2", Name: "n2"},
		{ID: "node/n3", Name: "n3"},
	}

	uigraphlayer1 := new(apiv3.UIGraphLayer)
	*uigraphlayer1 = apiv3.UIGraphLayer{
		Nodes: nodes1,
		Icon:  "smiley face",
	}

	dashboard1 := new(apiv3.UIDashboard)
	*dashboard1 = apiv3.UIDashboard{}

	spec1 := apiv3.UISettingsSpec{
		Group:       "group1",
		Description: "cluster",
		View:        uigraphview1,
		Layer:       nil,
		Dashboard:   nil,
	}
	spec2 := apiv3.UISettingsSpec{
		Group:       "group2",
		Description: "cluster",
		View:        nil,
		Layer:       uigraphlayer1,
		Dashboard:   nil,
	}
	spec3 := apiv3.UISettingsSpec{
		Group:       "group2",
		Description: "cluster",
		View:        nil,
		Layer:       nil,
		Dashboard:   dashboard1,
	}

	DescribeTable("UISettings e2e CRUD tests",
		func(name1, name2 string, spec1, spec2 apiv3.UISettingsSpec) {
			c, err := clientv3.New(config)
			Expect(err).NotTo(HaveOccurred())

			be, err := backend.NewClient(config)
			Expect(err).NotTo(HaveOccurred())
			be.Clean()

			By("Updating the UISettings before it is created")
			_, outError := c.UISettings().Update(ctx, &apiv3.UISettings{
				ObjectMeta: metav1.ObjectMeta{Name: name1, ResourceVersion: "1234", CreationTimestamp: metav1.Now(), UID: "test-fail-uiSettings"},
				Spec:       spec1,
			}, options.SetOptions{})
			Expect(outError).To(HaveOccurred())
			Expect(outError.Error()).To(ContainSubstring("resource does not exist: UISettings(" + name1 + ") with error:"))

			By("Attempting to create a new UISettings with name1/spec1 and a non-empty ResourceVersion")
			_, outError = c.UISettings().Create(ctx, &apiv3.UISettings{
				ObjectMeta: metav1.ObjectMeta{Name: name1, ResourceVersion: "12345"},
				Spec:       spec1,
			}, options.SetOptions{})

			Expect(outError).To(HaveOccurred())
			Expect(outError.Error()).To(Equal("error with field Metadata.ResourceVersion = '12345' (field must not be set for a Create request)"))

			By("Creating a new UISettings with name1/spec1")
			res1, outError := c.UISettings().Create(ctx, &apiv3.UISettings{
				ObjectMeta: metav1.ObjectMeta{Name: name1},
				Spec:       spec1,
			}, options.SetOptions{})
			Expect(outError).NotTo(HaveOccurred())
			Expect(res1).To(MatchResource(apiv3.KindUISettings, testutils.ExpectNoNamespace, name1, spec1))

			// Track the version of the original data for name1.
			rv1_1 := res1.ResourceVersion

			By("Attempting to create the same UISettings with name1, but with spec2")
			_, outError = c.UISettings().Create(ctx, &apiv3.UISettings{
				ObjectMeta: metav1.ObjectMeta{Name: name1},
				Spec:       spec2,
			}, options.SetOptions{})
			Expect(outError).To(HaveOccurred())
			Expect(outError.Error()).To(Equal("resource already exists: UISettings(" + name1 + ")"))

			By("Getting UISettings (name1) and comparing the output against spec1")
			res, outError := c.UISettings().Get(ctx, name1, options.GetOptions{})
			Expect(outError).NotTo(HaveOccurred())
			Expect(res).To(MatchResource(apiv3.KindUISettings, testutils.ExpectNoNamespace, name1, spec1))
			Expect(res.ResourceVersion).To(Equal(res1.ResourceVersion))

			By("Getting UISettings (name2) before it is created")
			_, outError = c.UISettings().Get(ctx, name2, options.GetOptions{})
			Expect(outError).To(HaveOccurred())
			Expect(outError.Error()).To(ContainSubstring("resource does not exist: UISettings(" + name2 + ") with error:"))

			By("Listing all the UISettings, expecting a single result with name1/spec1")
			outList, outError := c.UISettings().List(ctx, options.ListOptions{})
			Expect(outError).NotTo(HaveOccurred())
			Expect(outList.Items).To(ConsistOf(
				testutils.Resource(apiv3.KindUISettings, testutils.ExpectNoNamespace, name1, spec1),
			))

			By("Creating a new UISettings with name2/spec2")
			res2, outError := c.UISettings().Create(ctx, &apiv3.UISettings{
				ObjectMeta: metav1.ObjectMeta{Name: name2},
				Spec:       spec2,
			}, options.SetOptions{})
			Expect(outError).NotTo(HaveOccurred())
			Expect(res2).To(MatchResource(apiv3.KindUISettings, testutils.ExpectNoNamespace, name2, spec2))

			By("Getting UISettings (name2) and comparing the output against spec2")
			res, outError = c.UISettings().Get(ctx, name2, options.GetOptions{})
			Expect(outError).NotTo(HaveOccurred())
			Expect(res2).To(MatchResource(apiv3.KindUISettings, testutils.ExpectNoNamespace, name2, spec2))
			Expect(res.ResourceVersion).To(Equal(res2.ResourceVersion))

			By("Listing all the UISettings, expecting a two results with name1/spec1 and name2/spec2")
			outList, outError = c.UISettings().List(ctx, options.ListOptions{})
			Expect(outError).NotTo(HaveOccurred())
			Expect(outList.Items).To(ConsistOf(
				testutils.Resource(apiv3.KindUISettings, testutils.ExpectNoNamespace, name1, spec1),
				testutils.Resource(apiv3.KindUISettings, testutils.ExpectNoNamespace, name2, spec2),
			))

			By("Updating UISettings name1 with spec2")
			res1.Spec = spec2
			res1, outError = c.UISettings().Update(ctx, res1, options.SetOptions{})
			Expect(outError).NotTo(HaveOccurred())
			Expect(res1).To(MatchResource(apiv3.KindUISettings, testutils.ExpectNoNamespace, name1, spec2))

			By("Attempting to update the UISettings without a Creation Timestamp")
			res, outError = c.UISettings().Update(ctx, &apiv3.UISettings{
				ObjectMeta: metav1.ObjectMeta{Name: name1, ResourceVersion: "1234", UID: "test-fail-uiSettings"},
				Spec:       spec1,
			}, options.SetOptions{})
			Expect(outError).To(HaveOccurred())
			Expect(res).To(BeNil())
			Expect(outError.Error()).To(Equal("error with field Metadata.CreationTimestamp = '0001-01-01 00:00:00 +0000 UTC' (field must be set for an Update request)"))

			By("Attempting to update the UISettings without a UID")
			res, outError = c.UISettings().Update(ctx, &apiv3.UISettings{
				ObjectMeta: metav1.ObjectMeta{Name: name1, ResourceVersion: "1234", CreationTimestamp: metav1.Now()},
				Spec:       spec1,
			}, options.SetOptions{})
			Expect(outError).To(HaveOccurred())
			Expect(res).To(BeNil())
			Expect(outError.Error()).To(Equal("error with field Metadata.UID = '' (field must be set for an Update request)"))

			// Track the version of the updated name1 data.
			rv1_2 := res1.ResourceVersion

			By("Updating UISettings name1 without specifying a resource version")
			res1.Spec = spec1
			res1.ObjectMeta.ResourceVersion = ""
			_, outError = c.UISettings().Update(ctx, res1, options.SetOptions{})
			Expect(outError).To(HaveOccurred())
			Expect(outError.Error()).To(Equal("error with field Metadata.ResourceVersion = '' (field must be set for an Update request)"))

			By("Updating UISettings name1 using the previous resource version")
			res1.Spec = spec1
			res1.ResourceVersion = rv1_1

			_, outError = c.UISettings().Update(ctx, res1, options.SetOptions{})
			Expect(outError).To(HaveOccurred())
			Expect(outError.Error()).To(Equal("update conflict: UISettings(" + name1 + ")"))

			if config.Spec.DatastoreType != apiconfig.Kubernetes {
				By("Getting UISettings (name1) with the original resource version and comparing the output against spec1")
				res, outError = c.UISettings().Get(ctx, name1, options.GetOptions{ResourceVersion: rv1_1})
				Expect(outError).NotTo(HaveOccurred())
				Expect(res).To(MatchResource(apiv3.KindUISettings, testutils.ExpectNoNamespace, name1, spec1))
				Expect(res.ResourceVersion).To(Equal(rv1_1))
			}

			By("Getting UISettings (name1) with the updated resource version and comparing the output against spec2")
			res, outError = c.UISettings().Get(ctx, name1, options.GetOptions{ResourceVersion: rv1_2})
			Expect(outError).NotTo(HaveOccurred())
			Expect(res).To(MatchResource(apiv3.KindUISettings, testutils.ExpectNoNamespace, name1, spec2))
			Expect(res.ResourceVersion).To(Equal(rv1_2))

			if config.Spec.DatastoreType != apiconfig.Kubernetes {
				By("Listing UISettings with the original resource version and checking for a single result with name1/spec1")
				outList, outError = c.UISettings().List(ctx, options.ListOptions{ResourceVersion: rv1_1})
				Expect(outError).NotTo(HaveOccurred())
				Expect(outList.Items).To(ConsistOf(
					testutils.Resource(apiv3.KindUISettings, testutils.ExpectNoNamespace, name1, spec1),
				))
			}

			By("Listing UISettings with the latest resource version and checking for two results with name1/spec2 and name2/spec2")
			outList, outError = c.UISettings().List(ctx, options.ListOptions{})
			Expect(outError).NotTo(HaveOccurred())
			Expect(outList.Items).To(ConsistOf(
				testutils.Resource(apiv3.KindUISettings, testutils.ExpectNoNamespace, name1, spec2),
				testutils.Resource(apiv3.KindUISettings, testutils.ExpectNoNamespace, name2, spec2),
			))

			// Track the version of the updated name1 data.
			rv1_3 := res.ResourceVersion

			if config.Spec.DatastoreType != apiconfig.Kubernetes {
				By("Deleting UISettings (name1) with the old resource version")
				_, outError = c.UISettings().Delete(ctx, name1, options.DeleteOptions{ResourceVersion: rv1_1})
				Expect(outError).To(HaveOccurred())
				Expect(outError.Error()).To(Equal("update conflict: UISettings(" + name1 + ")"))
			}

			By("Deleting UISettings (name1) with the new resource version")
			dres, outError := c.UISettings().Delete(ctx, name1, options.DeleteOptions{ResourceVersion: rv1_3})
			Expect(outError).NotTo(HaveOccurred())
			Expect(dres).To(MatchResource(apiv3.KindUISettings, testutils.ExpectNoNamespace, name1, spec2))

			if config.Spec.DatastoreType != apiconfig.Kubernetes {
				By("Updating UISettings name2 with a 2s TTL and waiting for the entry to be deleted")

				_, outError = c.UISettings().Update(ctx, res2, options.SetOptions{TTL: 2 * time.Second})
				Expect(outError).NotTo(HaveOccurred())
				time.Sleep(1 * time.Second)
				_, outError = c.UISettings().Get(ctx, name2, options.GetOptions{})
				Expect(outError).NotTo(HaveOccurred())
				time.Sleep(2 * time.Second)
				_, outError = c.UISettings().Get(ctx, name2, options.GetOptions{})
				Expect(outError).To(HaveOccurred())
				Expect(outError.Error()).To(ContainSubstring("resource does not exist: UISettings(" + name2 + ") with error:"))

				By("Creating UISettings name2 with a 2s TTL and waiting for the entry to be deleted")
				_, outError = c.UISettings().Create(ctx, &apiv3.UISettings{
					ObjectMeta: metav1.ObjectMeta{Name: name2},
					Spec:       spec2,
				}, options.SetOptions{TTL: 2 * time.Second})
				Expect(outError).NotTo(HaveOccurred())
				time.Sleep(1 * time.Second)
				_, outError = c.UISettings().Get(ctx, name2, options.GetOptions{})
				Expect(outError).NotTo(HaveOccurred())
				time.Sleep(2 * time.Second)
				_, outError = c.UISettings().Get(ctx, name2, options.GetOptions{})
				Expect(outError).To(HaveOccurred())
				Expect(outError.Error()).To(ContainSubstring("resource does not exist: UISettings(" + name2 + ") with error:"))
			}

			if config.Spec.DatastoreType == apiconfig.Kubernetes {
				By("Attempting to delete UISettings (name2) again")
				dres, outError = c.UISettings().Delete(ctx, name2, options.DeleteOptions{})
				Expect(outError).NotTo(HaveOccurred())
				Expect(dres).To(MatchResource(apiv3.KindUISettings, testutils.ExpectNoNamespace, name2, spec2))
			}

			By("Listing all UISettings and expecting no items")
			outList, outError = c.UISettings().List(ctx, options.ListOptions{})
			Expect(outError).NotTo(HaveOccurred())
			Expect(outList.Items).To(HaveLen(0))

			By("Getting UISettings (name2) and expecting an error")
			_, outError = c.UISettings().Get(ctx, name2, options.GetOptions{})
			Expect(outError).To(HaveOccurred())
			Expect(outError.Error()).To(ContainSubstring("resource does not exist: UISettings(" + name2 + ") with error:"))
		},

		// Test 1: Pass two fully populated UISettingsSpecs and expect the series of operations to succeed.
		Entry("Two fully populated UISettingsSpecs", name1, name2, spec1, spec3),
	)

	Describe("UISettings watch functionality", func() {
		It("should handle watch events for different resource versions and event types", func() {
			c, err := clientv3.New(config)
			Expect(err).NotTo(HaveOccurred())

			be, err := backend.NewClient(config)
			Expect(err).NotTo(HaveOccurred())
			be.Clean()

			By("Listing UISettings with the latest resource version and checking for two results with name1/spec2 and name2/spec2")
			outList, outError := c.UISettings().List(ctx, options.ListOptions{})
			Expect(outError).NotTo(HaveOccurred())
			Expect(outList.Items).To(HaveLen(0))
			rev0 := outList.ResourceVersion

			By("Configuring a UISettings name1/spec1 and storing the response")
			outRes1, err := c.UISettings().Create(
				ctx,
				&apiv3.UISettings{
					ObjectMeta: metav1.ObjectMeta{Name: name1},
					Spec:       spec1,
				},
				options.SetOptions{},
			)
			Expect(err).NotTo(HaveOccurred())
			rev1 := outRes1.ResourceVersion

			By("Configuring a UISettings name2/spec2 and storing the response")
			outRes2, err := c.UISettings().Create(
				ctx,
				&apiv3.UISettings{
					ObjectMeta: metav1.ObjectMeta{Name: name2},
					Spec:       spec2,
				},
				options.SetOptions{},
			)
			Expect(err).NotTo(HaveOccurred())

			By("Starting a watcher from revision rev1 - this should skip the first creation")
			w, err := c.UISettings().Watch(ctx, options.ListOptions{ResourceVersion: rev1})
			Expect(err).NotTo(HaveOccurred())
			testWatcher1 := testutils.NewTestResourceWatch(config.Spec.DatastoreType, w)
			defer testWatcher1.Stop()

			By("Deleting res1")
			_, err = c.UISettings().Delete(ctx, name1, options.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Checking for two events, create res2 and delete re1")
			testWatcher1.ExpectEvents(apiv3.KindUISettings, []watch.Event{
				{
					Type:   watch.Added,
					Object: outRes2,
				},
				{
					Type:     watch.Deleted,
					Previous: outRes1,
				},
			})
			testWatcher1.Stop()

			By("Starting a watcher from rev0 - this should get all events")
			w, err = c.UISettings().Watch(ctx, options.ListOptions{ResourceVersion: rev0})
			Expect(err).NotTo(HaveOccurred())
			testWatcher2 := testutils.NewTestResourceWatch(config.Spec.DatastoreType, w)
			defer testWatcher2.Stop()

			By("Modifying res2")
			outRes3, err := c.UISettings().Update(
				ctx,
				&apiv3.UISettings{
					ObjectMeta: outRes2.ObjectMeta,
					Spec:       spec1,
				},
				options.SetOptions{},
			)
			Expect(err).NotTo(HaveOccurred())
			testWatcher2.ExpectEvents(apiv3.KindUISettings, []watch.Event{
				{
					Type:   watch.Added,
					Object: outRes1,
				},
				{
					Type:   watch.Added,
					Object: outRes2,
				},
				{
					Type:     watch.Deleted,
					Previous: outRes1,
				},
				{
					Type:     watch.Modified,
					Previous: outRes2,
					Object:   outRes3,
				},
			})
			testWatcher2.Stop()

			// Only etcdv3 supports watching a specific instance of a resource.
			if config.Spec.DatastoreType == apiconfig.EtcdV3 {
				By("Starting a watcher from rev0 watching name1 - this should get all events for name1")
				w, err = c.UISettings().Watch(ctx, options.ListOptions{Name: name1, ResourceVersion: rev0})
				Expect(err).NotTo(HaveOccurred())
				testWatcher2_1 := testutils.NewTestResourceWatch(config.Spec.DatastoreType, w)
				defer testWatcher2_1.Stop()
				testWatcher2_1.ExpectEvents(apiv3.KindUISettings, []watch.Event{
					{
						Type:   watch.Added,
						Object: outRes1,
					},
					{
						Type:     watch.Deleted,
						Previous: outRes1,
					},
				})
				testWatcher2_1.Stop()
			}

			By("Starting a watcher not specifying a rev - expect the current snapshot")
			w, err = c.UISettings().Watch(ctx, options.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			testWatcher3 := testutils.NewTestResourceWatch(config.Spec.DatastoreType, w)
			defer testWatcher3.Stop()
			testWatcher3.ExpectEvents(apiv3.KindUISettings, []watch.Event{
				{
					Type:   watch.Added,
					Object: outRes3,
				},
			})
			testWatcher3.Stop()

			By("Configuring UISettings name1/spec1 again and storing the response")
			outRes1, _ = c.UISettings().Create(
				ctx,
				&apiv3.UISettings{
					ObjectMeta: metav1.ObjectMeta{Name: name1},
					Spec:       spec1,
				},
				options.SetOptions{},
			)

			By("Starting a watcher not specifying a rev - expect the current snapshot")
			w, err = c.UISettings().Watch(ctx, options.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			testWatcher4 := testutils.NewTestResourceWatch(config.Spec.DatastoreType, w)
			defer testWatcher4.Stop()
			testWatcher4.ExpectEventsAnyOrder(apiv3.KindUISettings, []watch.Event{
				{
					Type:   watch.Added,
					Object: outRes1,
				},
				{
					Type:   watch.Added,
					Object: outRes3,
				},
			})

			By("Cleaning the datastore and expecting deletion events for each configured resource (tests prefix deletes results in individual events for each key)")
			be.Clean()
			testWatcher4.ExpectEvents(apiv3.KindUISettings, []watch.Event{
				{
					Type:     watch.Deleted,
					Previous: outRes1,
				},
				{
					Type:     watch.Deleted,
					Previous: outRes3,
				},
			})
			testWatcher4.Stop()
		})
	})
})
