// Copyright (c) 2022 Tigera, Inc. All rights reserved.

package v3

import (
	"time"

	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

var _ = DescribeTable("AlertException Validator",
	func(input interface{}, valid bool) {
		if valid {
			Expect(Validate(input)).NotTo(HaveOccurred(),
				"expected value to be valid")
		} else {
			Expect(Validate(input)).To(HaveOccurred(),
				"expected value to be invalid")
		}
	},

	Entry("minimal valid",
		&api.AlertException{
			ObjectMeta: v1.ObjectMeta{Name: "alert-exception"},
			Spec: api.AlertExceptionSpec{
				Description: "alert-exception-desc",
				Selector:    "origin = any",
			},
		},
		true,
	),

	Entry("missing description",
		&api.AlertException{
			ObjectMeta: v1.ObjectMeta{Name: "alert-exception"},
			Spec: api.AlertExceptionSpec{
				Selector: "origin = any",
			},
		},
		false,
	),

	Entry("no selector",
		&api.AlertException{
			ObjectMeta: v1.ObjectMeta{Name: "alert-exception"},
			Spec: api.AlertExceptionSpec{
				Description: "alert-exception-desc",
			},
		},
		false,
	),
	Entry("non parsable selector",
		&api.AlertException{
			ObjectMeta: v1.ObjectMeta{Name: "alert-exception"},
			Spec: api.AlertExceptionSpec{
				Description: "alert-exception-desc",
				Selector:    "origin = ",
			},
		},
		false,
	),
	Entry("invalid selector key",
		&api.AlertException{
			ObjectMeta: v1.ObjectMeta{Name: "alert-exception"},
			Spec: api.AlertExceptionSpec{
				Description: "alert-exception-desc",
				Selector:    "invalid = any",
			},
		},
		false,
	),

	Entry("valid period",
		&api.AlertException{
			ObjectMeta: v1.ObjectMeta{Name: "alert-exception"},
			Spec: api.AlertExceptionSpec{
				Description: "alert-exception-desc",
				Selector:    "origin = any",
				Period:      &v1.Duration{Duration: api.AlertExceptionMinPeriod},
			},
		},
		true,
	),
	Entry("period too short",
		&api.AlertException{
			ObjectMeta: v1.ObjectMeta{Name: "alert-exception"},
			Spec: api.AlertExceptionSpec{
				Description: "alert-exception-desc",
				Selector:    "origin = any",
				Period:      &v1.Duration{Duration: api.GlobalAlertMinPeriod - time.Second},
			},
		},
		false,
	),
)
