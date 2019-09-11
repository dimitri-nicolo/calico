// Copyright (c) 2019 Tigera, Inc. All rights reserved.

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

package resources_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/projectcalico/libcalico-go/lib/resources"
)

var _ = Describe("types", func() {
	It("should support all the relevant resources", func() {
		var rh ResourceHelper
		var res Resource

		// Pods
		By("creating a Pod instance using NewResource")
		rh = GetResourceHelperByTypeMeta(TypeK8sPods)
		Expect(rh).ToNot(BeNil())
		res = rh.NewResource()
		_, ok := res.(*corev1.Pod)
		Expect(ok).To(BeTrue())
		list := rh.NewResourceList()
		_, ok = list.(*corev1.PodList)
		Expect(ok).To(BeTrue())

		// Namespace
		By("creating a Namespace instance using NewResource")
		rh = GetResourceHelperByTypeMeta(TypeK8sNamespaces)
		Expect(rh).ToNot(BeNil())
		res = rh.NewResource()
		_, ok = res.(*corev1.Namespace)
		Expect(ok).To(BeTrue())
		list = rh.NewResourceList()
		_, ok = list.(*corev1.NamespaceList)
		Expect(ok).To(BeTrue())

		// Service Account
		By("creating a Service Account instance using NewResource")
		rh = GetResourceHelperByTypeMeta(TypeK8sServiceAccounts)
		Expect(rh).ToNot(BeNil())
		res = rh.NewResource()
		_, ok = res.(*corev1.ServiceAccount)
		Expect(ok).To(BeTrue())
		list = rh.NewResourceList()
		_, ok = list.(*corev1.ServiceAccountList)
		Expect(ok).To(BeTrue())

		// Endpoints
		By("creating a Endpoint instance using NewResource")
		rh = GetResourceHelperByTypeMeta(TypeK8sEndpoints)
		Expect(rh).ToNot(BeNil())
		res = rh.NewResource()
		_, ok = res.(*corev1.Endpoints)
		Expect(ok).To(BeTrue())
		list = rh.NewResourceList()
		_, ok = list.(*corev1.EndpointsList)
		Expect(ok).To(BeTrue())

		// Host Endpoints
		By("creating a Host Endpoint instance using NewResource")
		rh = GetResourceHelperByTypeMeta(TypeCalicoHostEndpoints)
		Expect(rh).ToNot(BeNil())
		res = rh.NewResource()
		_, ok = res.(*apiv3.HostEndpoint)
		Expect(ok).To(BeTrue())
		list = rh.NewResourceList()
		_, ok = list.(*apiv3.HostEndpointList)
		Expect(ok).To(BeTrue())

		// Global Network Sets
		By("creating a Global Network Set instance using NewResource")
		rh = GetResourceHelperByTypeMeta(TypeCalicoGlobalNetworkSets)
		Expect(rh).ToNot(BeNil())
		res = rh.NewResource()
		_, ok = res.(*apiv3.GlobalNetworkSet)
		Expect(ok).To(BeTrue())
		list = rh.NewResourceList()
		_, ok = list.(*apiv3.GlobalNetworkSetList)
		Expect(ok).To(BeTrue())

		// Network Policies
		By("creating a Network Policies instance using NewResource")
		rh = GetResourceHelperByTypeMeta(TypeCalicoNetworkPolicies)
		Expect(rh).ToNot(BeNil())
		res = rh.NewResource()
		_, ok = res.(*apiv3.NetworkPolicy)
		Expect(ok).To(BeTrue())
		list = rh.NewResourceList()
		_, ok = list.(*apiv3.NetworkPolicyList)
		Expect(ok).To(BeTrue())

		// Global Network Policies
		By("creating a Global Network Policies instance using NewResource")
		rh = GetResourceHelperByTypeMeta(TypeCalicoGlobalNetworkPolicies)
		Expect(rh).ToNot(BeNil())
		res = rh.NewResource()
		_, ok = res.(*apiv3.GlobalNetworkPolicy)
		Expect(ok).To(BeTrue())
		list = rh.NewResourceList()
		_, ok = list.(*apiv3.GlobalNetworkPolicyList)
		Expect(ok).To(BeTrue())

		// Unknown type
		By("Trying to create unknown resource types")
		unknown := metav1.TypeMeta{
			Kind:       "foo",
			APIVersion: "bar/v1",
		}
		rh = GetResourceHelperByTypeMeta(unknown)
		Expect(rh).To(BeNil())
		res = NewResource(unknown)
		Expect(res).To(BeNil())
		list = NewResourceList(unknown)
		Expect(list).To(BeNil())
	})

	It("should return a valid set of all resources", func() {
		By("getting a copy of all resource helpers")
		rhs := GetAllResourceHelpers()

		By("checking there are >0 helpers and they are not nil")
		Expect(len(rhs)).ToNot(BeZero())
		Expect(rhs[0]).ToNot(BeNil())
	})
})
