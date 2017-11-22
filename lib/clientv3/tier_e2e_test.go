// Copyright (c) 2017 Tigera, Inc. All rights reserved.

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

package clientv3_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"context"

	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/testutils"
	"github.com/projectcalico/libcalico-go/lib/watch"
)

var _ = testutils.E2eDatastoreDescribe("Tier tests", testutils.DatastoreAll, func(config apiconfig.CalicoAPIConfig) {

	ctx := context.Background()
	order1 := 99.999
	order2 := 22.222
	name1 := "t-1"
	name2 := "t-2"
	defaultName := "default"
	namespace1 := "namespace-1"
	spec1 := apiv3.TierSpec{
		Order: &order1,
	}
	spec2 := apiv3.TierSpec{
		Order: &order2,
	}
	defaultSpec := apiv3.TierSpec{}

	npName1 := name1 + ".networkp-1"
	npSpec1 := apiv3.NetworkPolicySpec{
		Tier:     name1,
		Order:    &order1,
		Ingress:  []apiv3.Rule{testutils.InRule1, testutils.InRule2},
		Egress:   []apiv3.Rule{testutils.EgressRule1, testutils.EgressRule2},
		Selector: "thing == 'value'",
	}

	gnpName1 := name1 + ".globalnetworkp-1"
	gnpSpec1 := apiv3.GlobalNetworkPolicySpec{
		Tier:           name1,
		Order:          &order2,
		Ingress:        []apiv3.Rule{testutils.InRule2, testutils.InRule1},
		Egress:         []apiv3.Rule{testutils.EgressRule2, testutils.EgressRule1},
		Selector:       "thing2 == 'value2'",
		DoNotTrack:     true,
		ApplyOnForward: true,
	}

	DescribeTable("Tier e2e CRUD tests",
		func(name1, name2, npName1, gnpName1, namespace1 string, spec1, spec2 apiv3.TierSpec, npSpec1 apiv3.NetworkPolicySpec, gnpSpec1 apiv3.GlobalNetworkPolicySpec) {
			c, err := clientv3.New(config)
			Expect(err).NotTo(HaveOccurred())

			be, err := backend.NewClient(config)
			Expect(err).NotTo(HaveOccurred())
			be.Clean()

			err = c.EnsureInitialized(ctx, "", "")
			Expect(err).NotTo(HaveOccurred())

			By("Creating the default tier with not nil order")
			res, outError := c.Tiers().Create(ctx, &apiv3.Tier{
				ObjectMeta: metav1.ObjectMeta{Name: defaultName},
				Spec:       spec1,
			}, options.SetOptions{})
			Expect(res).To(BeNil())
			Expect(outError).To(HaveOccurred())
			Expect(outError.Error()).To(Equal("operation Create is not supported on default: Default tier should have nil Order"))

			By("Cannot delete the default Tier")
			_, outError = c.Tiers().Delete(ctx, defaultName, options.DeleteOptions{})
			Expect(outError).To(HaveOccurred())
			Expect(outError.Error()).To(Equal("operation Delete is not supported on default: Cannot delete default tier"))

			By("Getting default Tier")
			defRes, outError := c.Tiers().Get(ctx, defaultName, options.GetOptions{})
			Expect(outError).NotTo(HaveOccurred())
			testutils.ExpectResource(defRes, apiv3.KindTier, testutils.ExpectNoNamespace, defaultName, defaultSpec)

			By("Cannot update the default Tier")
			defRes.Spec = spec2
			_, outError = c.Tiers().Update(ctx, defRes, options.SetOptions{})
			Expect(outError).To(HaveOccurred())
			Expect(outError.Error()).To(Equal("operation Update is not supported on default: Cannot update the order of the default tier"))

			By("Updating the Tier before it is created")
			res, outError = c.Tiers().Update(ctx, &apiv3.Tier{
				ObjectMeta: metav1.ObjectMeta{Name: name1, ResourceVersion: "1234", CreationTimestamp: metav1.Now(), UID: "test-fail-tier"},
				Spec:       spec1,
			}, options.SetOptions{})
			Expect(outError).To(HaveOccurred())
			Expect(res).To(BeNil())
			Expect(outError.Error()).To(Equal("resource does not exist: Tier(" + name1 + ")"))

			By("Attempting to creating a new Tier with name1/spec1 and a non-empty ResourceVersion")
			res, outError = c.Tiers().Create(ctx, &apiv3.Tier{
				ObjectMeta: metav1.ObjectMeta{Name: name1, ResourceVersion: "12345"},
				Spec:       spec1,
			}, options.SetOptions{})
			Expect(res).To(BeNil())
			Expect(outError).To(HaveOccurred())
			Expect(outError.Error()).To(Equal("error with field Metadata.ResourceVersion = '12345' (field must not be set for a Create request)"))

			By("Creating a new Tier with name1/spec1")
			res1, outError := c.Tiers().Create(ctx, &apiv3.Tier{
				ObjectMeta: metav1.ObjectMeta{Name: name1},
				Spec:       spec1,
			}, options.SetOptions{})
			Expect(outError).NotTo(HaveOccurred())
			testutils.ExpectResource(res1, apiv3.KindTier, testutils.ExpectNoNamespace, name1, spec1)

			// Track the version of the original data for name1.
			rv1_1 := res1.ResourceVersion

			By("Attempting to create the same Tier with name1 but with spec2")
			_, outError = c.Tiers().Create(ctx, &apiv3.Tier{
				ObjectMeta: metav1.ObjectMeta{Name: name1},
				Spec:       spec2,
			}, options.SetOptions{})
			Expect(outError).To(HaveOccurred())
			Expect(outError.Error()).To(Equal("resource already exists: Tier(" + name1 + ")"))

			By("Getting Tier (name1) and comparing the output against spec1")
			res, outError = c.Tiers().Get(ctx, name1, options.GetOptions{})
			Expect(outError).NotTo(HaveOccurred())
			testutils.ExpectResource(res, apiv3.KindTier, testutils.ExpectNoNamespace, name1, spec1)
			Expect(res.ResourceVersion).To(Equal(res1.ResourceVersion))

			By("Getting Tier (name2) before it is created")
			_, outError = c.Tiers().Get(ctx, name2, options.GetOptions{})
			Expect(outError).To(HaveOccurred())
			Expect(outError.Error()).To(Equal("resource does not exist: Tier(" + name2 + ")"))

			By("Listing all the Tiers, expecting a single result with name1/spec1")
			outList, outError := c.Tiers().List(ctx, options.ListOptions{})
			Expect(outError).NotTo(HaveOccurred())
			Expect(outList.Items).To(HaveLen(2))
			testutils.ExpectResource(&outList.Items[0], apiv3.KindTier, testutils.ExpectNoNamespace, defaultName, defaultSpec)
			testutils.ExpectResource(&outList.Items[1], apiv3.KindTier, testutils.ExpectNoNamespace, name1, spec1)

			By("Creating a new Tier with name2/spec2")
			res2, outError := c.Tiers().Create(ctx, &apiv3.Tier{
				ObjectMeta: metav1.ObjectMeta{Name: name2},
				Spec:       spec2,
			}, options.SetOptions{})
			Expect(outError).NotTo(HaveOccurred())
			testutils.ExpectResource(res2, apiv3.KindTier, testutils.ExpectNoNamespace, name2, spec2)

			By("Getting Tier (name2) and comparing the output against spec2")
			res, outError = c.Tiers().Get(ctx, name2, options.GetOptions{})
			Expect(outError).NotTo(HaveOccurred())
			testutils.ExpectResource(res2, apiv3.KindTier, testutils.ExpectNoNamespace, name2, spec2)
			Expect(res.ResourceVersion).To(Equal(res2.ResourceVersion))

			By("Listing all the Tiers, expecting a two results with name1/spec1 and name2/spec2")
			outList, outError = c.Tiers().List(ctx, options.ListOptions{})
			Expect(outError).NotTo(HaveOccurred())
			Expect(outList.Items).To(HaveLen(3))
			testutils.ExpectResource(&outList.Items[0], apiv3.KindTier, testutils.ExpectNoNamespace, defaultName, defaultSpec)
			testutils.ExpectResource(&outList.Items[1], apiv3.KindTier, testutils.ExpectNoNamespace, name1, spec1)
			testutils.ExpectResource(&outList.Items[2], apiv3.KindTier, testutils.ExpectNoNamespace, name2, spec2)

			By("Updating Tier name1 with spec2")
			res1.Spec = spec2
			res1, outError = c.Tiers().Update(ctx, res1, options.SetOptions{})
			Expect(outError).NotTo(HaveOccurred())
			testutils.ExpectResource(res1, apiv3.KindTier, testutils.ExpectNoNamespace, name1, spec2)

			// Track the version of the updated name1 data.
			rv1_2 := res1.ResourceVersion

			By("Updating Tier name1 without specifying a resource version")
			res1.Spec = spec1
			res1.ObjectMeta.ResourceVersion = ""
			_, outError = c.Tiers().Update(ctx, res1, options.SetOptions{})
			Expect(outError).To(HaveOccurred())
			Expect(outError.Error()).To(Equal("error with field Metadata.ResourceVersion = '' (field must be set for an Update request)"))

			By("Updating Tier name1 using the previous resource version")
			res1.Spec = spec1
			res1.ResourceVersion = rv1_1
			_, outError = c.Tiers().Update(ctx, res1, options.SetOptions{})
			Expect(outError).To(HaveOccurred())
			Expect(outError.Error()).To(Equal("update conflict: Tier(" + name1 + ")"))

			if config.Spec.DatastoreType != apiconfig.Kubernetes {
				By("Getting Tier (name1) with the original resource version and comparing the output against spec1")
				res, outError = c.Tiers().Get(ctx, name1, options.GetOptions{ResourceVersion: rv1_1})
				Expect(outError).NotTo(HaveOccurred())
				testutils.ExpectResource(res, apiv3.KindTier, testutils.ExpectNoNamespace, name1, spec1)
				Expect(res.ResourceVersion).To(Equal(rv1_1))
			}

			By("Getting Tier (name1) with the updated resource version and comparing the output against spec2")
			res, outError = c.Tiers().Get(ctx, name1, options.GetOptions{ResourceVersion: rv1_2})
			Expect(outError).NotTo(HaveOccurred())
			testutils.ExpectResource(res, apiv3.KindTier, testutils.ExpectNoNamespace, name1, spec2)
			Expect(res.ResourceVersion).To(Equal(rv1_2))

			if config.Spec.DatastoreType != apiconfig.Kubernetes {
				By("Listing Tiers with the original resource version and checking for a single result with name1/spec1")
				outList, outError = c.Tiers().List(ctx, options.ListOptions{ResourceVersion: rv1_1})
				Expect(outError).NotTo(HaveOccurred())
				Expect(outList.Items).To(HaveLen(2))
				testutils.ExpectResource(&outList.Items[0], apiv3.KindTier, testutils.ExpectNoNamespace, defaultName, defaultSpec)
				testutils.ExpectResource(&outList.Items[1], apiv3.KindTier, testutils.ExpectNoNamespace, name1, spec1)
			}

			By("Listing Tiers with the latest resource version and checking for two results with name1/spec2 and name2/spec2")
			outList, outError = c.Tiers().List(ctx, options.ListOptions{})
			Expect(outError).NotTo(HaveOccurred())
			Expect(outList.Items).To(HaveLen(3))
			testutils.ExpectResource(&outList.Items[0], apiv3.KindTier, testutils.ExpectNoNamespace, defaultName, defaultSpec)
			testutils.ExpectResource(&outList.Items[1], apiv3.KindTier, testutils.ExpectNoNamespace, name1, spec2)
			testutils.ExpectResource(&outList.Items[2], apiv3.KindTier, testutils.ExpectNoNamespace, name2, spec2)

			if config.Spec.DatastoreType != apiconfig.Kubernetes {
				By("Deleting Tier (name1) with the old resource version")
				_, outError = c.Tiers().Delete(ctx, name1, options.DeleteOptions{ResourceVersion: rv1_1})
				Expect(outError).To(HaveOccurred())
				Expect(outError.Error()).To(Equal("update conflict: Tier(" + name1 + ")"))
			}

			By("Attempting to delete Tier (name1) when there is a NetworkPolicy in it")
			np1, outError := c.NetworkPolicies().Create(ctx, &apiv3.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{Name: npName1, Namespace: namespace1},
				Spec:       npSpec1,
			}, options.SetOptions{})
			Expect(outError).NotTo(HaveOccurred())
			npSpec1.Types = []apiv3.PolicyType{apiv3.PolicyTypeIngress, apiv3.PolicyTypeEgress}
			testutils.ExpectResource(np1, apiv3.KindNetworkPolicy, namespace1, npName1, npSpec1)
			rv_np := np1.ResourceVersion

			dres, outError := c.Tiers().Delete(ctx, name1, options.DeleteOptions{ResourceVersion: rv1_2})
			Expect(outError).To(HaveOccurred())

			dnp, outError := c.NetworkPolicies().Delete(ctx, namespace1, npName1, options.DeleteOptions{ResourceVersion: rv_np})
			Expect(outError).NotTo(HaveOccurred())
			testutils.ExpectResource(dnp, apiv3.KindNetworkPolicy, namespace1, npName1, npSpec1)

			By("Attempting to delete Tier (name1) when there is a GlobalNetworkPolicy in it")
			gnp1, outError := c.GlobalNetworkPolicies().Create(ctx, &apiv3.GlobalNetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{Name: gnpName1},
				Spec:       gnpSpec1,
			}, options.SetOptions{})
			Expect(outError).NotTo(HaveOccurred())
			gnpSpec1.Types = []apiv3.PolicyType{apiv3.PolicyTypeIngress, apiv3.PolicyTypeEgress}
			testutils.ExpectResource(gnp1, apiv3.KindGlobalNetworkPolicy, testutils.ExpectNoNamespace, gnpName1, gnpSpec1)
			rv_gnp := gnp1.ResourceVersion

			dres, outError = c.Tiers().Delete(ctx, name1, options.DeleteOptions{ResourceVersion: rv1_2})
			Expect(outError).To(HaveOccurred())

			dgnp, outError := c.GlobalNetworkPolicies().Delete(ctx, gnpName1, options.DeleteOptions{ResourceVersion: rv_gnp})
			Expect(outError).NotTo(HaveOccurred())
			testutils.ExpectResource(dgnp, apiv3.KindGlobalNetworkPolicy, testutils.ExpectNoNamespace, gnpName1, gnpSpec1)
			time.Sleep(1 * time.Second)

			By("Deleting Tier (name1) with the new resource version")
			dres, outError = c.Tiers().Delete(ctx, name1, options.DeleteOptions{ResourceVersion: rv1_2})
			Expect(outError).NotTo(HaveOccurred())
			testutils.ExpectResource(dres, apiv3.KindTier, testutils.ExpectNoNamespace, name1, spec2)

			if config.Spec.DatastoreType != apiconfig.Kubernetes {
				By("Updating Tier name2 with a 2s TTL and waiting for the entry to be deleted")
				_, outError = c.Tiers().Update(ctx, res2, options.SetOptions{TTL: 2 * time.Second})
				Expect(outError).NotTo(HaveOccurred())
				time.Sleep(1 * time.Second)
				_, outError = c.Tiers().Get(ctx, name2, options.GetOptions{})
				Expect(outError).NotTo(HaveOccurred())
				time.Sleep(2 * time.Second)
				_, outError = c.Tiers().Get(ctx, name2, options.GetOptions{})
				Expect(outError).To(HaveOccurred())
				Expect(outError.Error()).To(Equal("resource does not exist: Tier(" + name2 + ")"))

				By("Creating Tier name2 with a 2s TTL and waiting for the entry to be deleted")
				_, outError = c.Tiers().Create(ctx, &apiv3.Tier{
					ObjectMeta: metav1.ObjectMeta{Name: name2},
					Spec:       spec2,
				}, options.SetOptions{TTL: 2 * time.Second})
				Expect(outError).NotTo(HaveOccurred())
				time.Sleep(1 * time.Second)
				_, outError = c.Tiers().Get(ctx, name2, options.GetOptions{})
				Expect(outError).NotTo(HaveOccurred())
				time.Sleep(2 * time.Second)
				_, outError = c.Tiers().Get(ctx, name2, options.GetOptions{})
				Expect(outError).To(HaveOccurred())
				Expect(outError.Error()).To(Equal("resource does not exist: Tier(" + name2 + ")"))
			}

			if config.Spec.DatastoreType == apiconfig.Kubernetes {
				By("Deleting Tier (name2)")
				dres, outError = c.Tiers().Delete(ctx, name2, options.DeleteOptions{})
				Expect(outError).NotTo(HaveOccurred())
				testutils.ExpectResource(dres, apiv3.KindTier, testutils.ExpectNoNamespace, name2, spec2)
			}

			By("Attempting to deleting Tier (name2) again")
			_, outError = c.Tiers().Delete(ctx, name2, options.DeleteOptions{})
			Expect(outError).To(HaveOccurred())
			Expect(outError.Error()).To(Equal("resource does not exist: Tier(" + name2 + ")"))

			By("Listing all Tiers and expecting only the default tier")
			outList, outError = c.Tiers().List(ctx, options.ListOptions{})
			Expect(outError).NotTo(HaveOccurred())
			Expect(outList.Items).To(HaveLen(1))
			testutils.ExpectResource(&outList.Items[0], apiv3.KindTier, testutils.ExpectNoNamespace, defaultName, defaultSpec)

			By("Getting Tier (name2) and expecting an error")
			res, outError = c.Tiers().Get(ctx, name2, options.GetOptions{})
			Expect(outError).To(HaveOccurred())
			Expect(outError.Error()).To(Equal("resource does not exist: Tier(" + name2 + ")"))
		},

		// Test 1: Pass two fully populated TierSpecs and expect the series of operations to succeed.
		Entry("Two fully populated TierSpecs", name1, name2, npName1, gnpName1, namespace1, spec1, spec2, npSpec1, gnpSpec1),
	)

	Describe("Tier watch functionality", func() {
		It("should handle watch events for different resource versions and event types", func() {
			c, err := clientv3.New(config)
			Expect(err).NotTo(HaveOccurred())

			be, err := backend.NewClient(config)
			Expect(err).NotTo(HaveOccurred())
			be.Clean()

			By("Listing Tiers with the latest resource version and checking for two results with name1/spec2 and name2/spec2")
			outList, outError := c.Tiers().List(ctx, options.ListOptions{})
			Expect(outError).NotTo(HaveOccurred())
			Expect(outList.Items).To(HaveLen(0))
			rev0 := outList.ResourceVersion

			By("Configuring a Tier name1/spec1 and storing the response")
			outRes1, err := c.Tiers().Create(
				ctx,
				&apiv3.Tier{
					ObjectMeta: metav1.ObjectMeta{Name: name1},
					Spec:       spec1,
				},
				options.SetOptions{},
			)
			rev1 := outRes1.ResourceVersion

			By("Configuring a Tier name2/spec2 and storing the response")
			outRes2, err := c.Tiers().Create(
				ctx,
				&apiv3.Tier{
					ObjectMeta: metav1.ObjectMeta{Name: name2},
					Spec:       spec2,
				},
				options.SetOptions{},
			)

			By("Starting a watcher from revision rev1 - this should skip the first creation")
			w, err := c.Tiers().Watch(ctx, options.ListOptions{ResourceVersion: rev1})
			Expect(err).NotTo(HaveOccurred())
			testWatcher1 := testutils.NewTestResourceWatch(config.Spec.DatastoreType, w)
			defer testWatcher1.Stop()

			By("Deleting res1")
			_, err = c.Tiers().Delete(ctx, name1, options.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Checking for two events, create res2 and delete re1")
			testWatcher1.ExpectEvents(apiv3.KindTier, []watch.Event{
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
			w, err = c.Tiers().Watch(ctx, options.ListOptions{ResourceVersion: rev0})
			Expect(err).NotTo(HaveOccurred())
			testWatcher2 := testutils.NewTestResourceWatch(config.Spec.DatastoreType, w)
			defer testWatcher2.Stop()

			By("Modifying res2")
			outRes3, err := c.Tiers().Update(
				ctx,
				&apiv3.Tier{
					ObjectMeta: outRes2.ObjectMeta,
					Spec:       spec1,
				},
				options.SetOptions{},
			)
			Expect(err).NotTo(HaveOccurred())
			testWatcher2.ExpectEvents(apiv3.KindTier, []watch.Event{
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
				w, err = c.Tiers().Watch(ctx, options.ListOptions{Name: name1, ResourceVersion: rev0})
				Expect(err).NotTo(HaveOccurred())
				testWatcher2_1 := testutils.NewTestResourceWatch(config.Spec.DatastoreType, w)
				defer testWatcher2_1.Stop()
				testWatcher2_1.ExpectEvents(apiv3.KindTier, []watch.Event{
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
			w, err = c.Tiers().Watch(ctx, options.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			testWatcher3 := testutils.NewTestResourceWatch(config.Spec.DatastoreType, w)
			defer testWatcher3.Stop()
			testWatcher3.ExpectEvents(apiv3.KindTier, []watch.Event{
				{
					Type:   watch.Added,
					Object: outRes3,
				},
			})
			testWatcher3.Stop()

			By("Configuring Tier name1/spec1 again and storing the response")
			outRes1, err = c.Tiers().Create(
				ctx,
				&apiv3.Tier{
					ObjectMeta: metav1.ObjectMeta{Name: name1},
					Spec:       spec1,
				},
				options.SetOptions{},
			)

			By("Starting a watcher not specifying a rev - expect the current snapshot")
			w, err = c.Tiers().Watch(ctx, options.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			testWatcher4 := testutils.NewTestResourceWatch(config.Spec.DatastoreType, w)
			defer testWatcher4.Stop()
			testWatcher4.ExpectEvents(apiv3.KindTier, []watch.Event{
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
			testWatcher4.ExpectEvents(apiv3.KindTier, []watch.Event{
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
