// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package datastore_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/compliance/pkg/datastore"
	"github.com/projectcalico/calico/libcalico-go/lib/resources"
)

var _ = Describe("list typemeta", func() {
	It("should fill in list typemeta", func() {
		in := &v3.NetworkPolicyList{
			Items: []v3.NetworkPolicy{
				{}, {},
			},
		}

		err := datastore.SetListTypeMeta(in, resources.TypeCalicoNetworkPolicies)
		Expect(err).NotTo(HaveOccurred())
		Expect(in.TypeMeta).To(Equal(resources.TypeCalicoNetworkPolicies))
		Expect(in.Items[0].TypeMeta).To(Equal(resources.TypeCalicoNetworkPolicies))
		Expect(in.Items[1].TypeMeta).To(Equal(resources.TypeCalicoNetworkPolicies))
	})
})
