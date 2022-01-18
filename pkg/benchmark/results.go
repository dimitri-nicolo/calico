// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.
package benchmark

import (
	"github.com/aquasecurity/kube-bench/check"
	"github.com/tigera/lma/pkg/api"
)

// TestsFromKubeBenchControls transforms the kube-bench results into the compliance benchmark structure.
func TestsFromKubeBenchControls(ctrls []*check.Controls) []api.BenchmarkTest {
	var tests []api.BenchmarkTest

	for _, ctrl := range ctrls {
		for _, section := range ctrl.Groups {
			for _, c := range section.Checks {
				test := api.BenchmarkTest{
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
