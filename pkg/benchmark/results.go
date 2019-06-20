// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package benchmark

import (
	"fmt"

	"time"

	"github.com/aquasecurity/kube-bench/check"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BenchmarkType is the type of benchmark.
type BenchmarkType string

const (
	TypeKubernetes BenchmarkType = "kube"
)

var (
	AllBenchmarkTypes = []BenchmarkType{TypeKubernetes}
)

// Benchmarks is a set of benchmarks for a given node.
type Benchmarks struct {
	Version           string        `json:"version"`
	KubernetesVersion string        `json:"kubernetesVersion"`
	Type              BenchmarkType `json:"type"`
	NodeName          string        `json:"node_name"`
	Timestamp         metav1.Time   `json:"timestamp"`
	Error             string        `json:"error,omitempty"`
	Tests             []Test        `json:"tests,omitempty"`
}

// UID is a unique identifier for a set of benchmarks.
func (b Benchmarks) UID() string {
	return fmt.Sprintf("%s::%s::%s", b.Timestamp.Format(time.RFC3339), b.Type, b.NodeName)
}

// Equal computes equality between benchmark results. Does not include Timestamp in the calculation.
func (b Benchmarks) Equal(other Benchmarks) bool {
	// First check the error field.
	if b.Error != "" {
		return b.Error == other.Error
	}

	// Initial equality determined by metadata fields.
	if b.Version != other.Version || b.Type != other.Type || b.NodeName != other.NodeName {
		return false
	}

	// Finally, check the tests. Assumes that the tests in both structures are positioned in the same order.
	for i, test := range b.Tests {
		if test != other.Tests[i] {
			return false
		}
	}
	return true
}

// Test is a given test within a set of benchmarks.
type Test struct {
	Section     string `json:"section"`
	SectionDesc string `json:"section_desc"`
	TestNumber  string `json:"test_number"`
	TestDesc    string `json:"test_desc"`
	TestInfo    string `json:"test_info"`
	Status      string `json:"status"`
	Scored      bool   `json:"scored"`
}

// Filter is the set of filters to limit the returned benchmarks.
type Filter struct {
	Version   string
	NodeNames []string
}

// BenchmarksResult is the result from a Benchmarks query. An error returned in this indicates a terminating error for
// the query.
type BenchmarksResult struct {
	Benchmarks *Benchmarks
	Err        error
}

// TestsFromKubeBenchControls transforms the kube-bench results into the compliance benchmark structure.
func TestsFromKubeBenchControls(ctrls []*check.Controls) []Test {
	tests := []Test{}
	for _, ctrl := range ctrls {
		for _, section := range ctrl.Groups {
			for _, check := range section.Checks {
				test := Test{
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
