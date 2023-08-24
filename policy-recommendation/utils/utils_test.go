// Copyright (c) 2023 Tigera, Inc. All rights reserved
package utils

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("getRFC1123PolicyName", func() {
	DescribeTable("should return the policy name with suffix if it is valid",
		func(tier, name, suffix, expected string, wantErr bool) {
			result, err := getRFC1123PolicyName(tier, name, suffix)
			if wantErr {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(result).To(Equal(expected))
			}
		},
		Entry("valid name", "tier", "a", "x1ed6", "tier.a-x1ed6", false),
		Entry("valid name with long name", "tier", "a123456789012345678901234567890123456789015432540", "x1ed6", "tier.a123456789012345678901234567890123456789015432540-x1ed6", false),
		Entry("valid name with long name cut down", "tier", "a123456789012345678901234567890123456789015432540243243439385802860348504393405", "x1ed6", "tier.a123456789012345678901234567890123456789015432540243-x1ed6", false),
		Entry("invalid name", "tier", "", "x1ed6", "", true),
		Entry("invalid name", "", "my-name", "x1ed6", "", true),
		Entry("invalid name with long tier name", "tiera12345678901234567000000000000000000000000000000000000000000000000", "a12345678901234567890123456789012345678901543254024324343", "x1ed6", "", true),
	)
})

var _ = Describe("CopyStagedNetworkPolicy", func() {
	DescribeTable("should copy the source policy to the destination policy", func(dest, src v3.StagedNetworkPolicy) {
		CopyStagedNetworkPolicy(&dest, src)

		Expect(dest.ObjectMeta.Name).To(Equal(src.GetObjectMeta().GetName()))
		Expect(dest.ObjectMeta.Namespace).To(Equal(src.GetObjectMeta().GetNamespace()))
		Expect(dest.ObjectMeta.OwnerReferences).To(Equal(src.GetObjectMeta().GetOwnerReferences()))
		Expect(dest.ObjectMeta.Annotations).To(Equal(src.GetObjectMeta().GetAnnotations()))
		Expect(dest.ObjectMeta.Labels).To(Equal(src.GetObjectMeta().GetLabels()))

		Expect(dest.Spec.Selector).To(Equal(src.Spec.Selector))
		Expect(dest.Spec.StagedAction).To(Equal(src.Spec.StagedAction))
		Expect(dest.Spec.Tier).To(Equal(src.Spec.Tier))
		Expect(dest.Spec.Egress).To(Equal(src.Spec.Egress))
		Expect(dest.Spec.Ingress).To(Equal(src.Spec.Ingress))
		Expect(dest.Spec.Types).To(Equal(src.Spec.Types))
	},
		Entry("should copy the source policy to the destination policy",
			v3.StagedNetworkPolicy{},
			v3.StagedNetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-policy",
					Namespace: "default",
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind: "Cluster",
							Name: "test-cluster",
						},
					},
					Annotations: map[string]string{
						"test-annotation": "test-value",
					},
					Labels: map[string]string{
						"test-label": "test-value",
					},
				},
				Spec: v3.StagedNetworkPolicySpec{
					Selector:     "test-selector",
					StagedAction: "Learn",
					Tier:         "test-tier",
					Egress: []v3.Rule{
						{
							Action: v3.Allow,
						},
					},
					Ingress: []v3.Rule{
						{
							Action: v3.Allow,
						},
					},
					Types: []v3.PolicyType{
						"Ingress",
					},
				},
			},
		),
	)
})
