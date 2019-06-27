// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package benchmark

import (
	"fmt"

	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BenchmarkType is the type of benchmark.
type BenchmarkType string

const (
	TypeKubernetes BenchmarkType = "kube"
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
