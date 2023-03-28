// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.
package benchmark

import (
	"github.com/aquasecurity/kube-bench/check"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

// TestsFromKubeBenchControls transforms the kube-bench results into the compliance benchmark structure.
func TestsFromKubeBenchControls(ctrls []*check.Controls) []v1.BenchmarkTest {
	var tests []v1.BenchmarkTest

	for _, ctrl := range ctrls {
		for _, section := range ctrl.Groups {
			for _, c := range section.Checks {
				test := v1.BenchmarkTest{
					Section:     section.ID,
					SectionDesc: section.Text,
					TestNumber:  c.ID,
					TestDesc:    c.Text,
					Status:      string(c.State),
					Scored:      c.Scored,
				}
				if len(c.TestInfo) > 0 {
					test.TestInfo = c.TestInfo[0]
				}
				tests = append(tests, test)
			}
		}
	}
	return tests
}
