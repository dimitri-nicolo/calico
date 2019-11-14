// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package benchmark

import (
	"github.com/aquasecurity/kube-bench/check"

	api "github.com/tigera/lma/pkg/api"
)

// TestsFromKubeBenchControls transforms the kube-bench results into the compliance benchmark structure.
func TestsFromKubeBenchControls(ctrls []*check.Controls) []api.BenchmarkTest {
	tests := []api.BenchmarkTest{}
	for _, ctrl := range ctrls {
		for _, section := range ctrl.Groups {
			for _, check := range section.Checks {
				test := api.BenchmarkTest{
					Section:     section.ID,
					SectionDesc: section.Text,
					TestNumber:  check.ID,
					TestDesc:    check.Text,
					Status:      string(check.State),
					Scored:      check.Scored,
				}
				if len(check.TestInfo) > 0 {
					test.TestInfo = check.TestInfo[0]
				}
				tests = append(tests, test)
			}
		}
	}
	return tests
}
