// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package labels_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/libcalico-go/lib/labels"
)

var _ = Describe("lib/labels tests", func() {
	DescribeTable("AddKindandNameLabels",
		func(name string, inputLabels, expectedLabels map[string]string) {
			Expect(labels.AddKindandNameLabels(name, inputLabels)).To(Equal(expectedLabels))
		},

		Entry("Nil labels",
			"my-netset",
			nil,
			map[string]string{apiv3.LabelKind: apiv3.KindNetworkSet, apiv3.LabelName: "my-netset"},
		),
		Entry("Empty labels",
			"my-netset",
			map[string]string{},
			map[string]string{apiv3.LabelKind: apiv3.KindNetworkSet, apiv3.LabelName: "my-netset"},
		),
		Entry("Filled labels",
			"my-netset",
			map[string]string{"a": "b", "projectcalico.org/namespace": "my-namespace"},
			map[string]string{"a": "b", apiv3.LabelKind: apiv3.KindNetworkSet, apiv3.LabelName: "my-netset", "projectcalico.org/namespace": "my-namespace"},
		),
	)

	DescribeTable("ValidateNetworkSetLabels",
		func(name string, inputLabels map[string]string, expectedValidation bool) {
			Expect(labels.ValidateNetworkSetLabels(name, inputLabels)).To(Equal(expectedValidation))
		},

		Entry("Nil labels",
			"my-netset",
			nil,
			false,
		),
		Entry("Empty labels",
			"my-netset",
			map[string]string{},
			false,
		),
		Entry("Filled labels",
			"my-netset",
			map[string]string{"a": "b", apiv3.LabelKind: apiv3.KindNetworkSet, apiv3.LabelName: "my-netset", "projectcalico.org/namespace": "my-namespace"},
			true,
		),
	)
})
