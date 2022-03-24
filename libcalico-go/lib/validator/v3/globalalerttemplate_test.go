package v3

import (
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	api "github.com/tigera/api/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = DescribeTable("GlobalAlertTemplate Validator",
	func(input interface{}, valid bool) {
		if valid {
			Expect(Validate(input)).NotTo(HaveOccurred(),
				"expected value to be valid")
		} else {
			Expect(Validate(input)).To(HaveOccurred(),
				"expected value to be invalid")
		}
	},
	Entry("valid GlobalAlertTemplate with name that relate to detector for ADJobs",
		&api.GlobalAlertTemplate{
			ObjectMeta: v1.ObjectMeta{Name: "tigera.io.detector.port-scan"},
			Spec: api.GlobalAlertSpec{
				Type:        api.GlobalAlertTypeAnomalyDetection,
				Description: "test",
				Detector: &api.DetectorParams{
					Name: "port_scan",
				},
				Severity: 100,
			},
		},
		true,
	),
	Entry("invalid GlobalAlertTemplate with uancceptable name for detector",
		&api.GlobalAlertTemplate{
			ObjectMeta: v1.ObjectMeta{Name: "sandwiches"},
			Spec: api.GlobalAlertSpec{
				Type:        api.GlobalAlertTypeAnomalyDetection,
				Description: "test",
				Detector: &api.DetectorParams{
					Name: "port_scan",
				},
				Severity: 100,
			},
		},
		false,
	),
)
