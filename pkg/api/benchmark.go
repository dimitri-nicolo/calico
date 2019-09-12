// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package api

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BenchmarksStore is the interface for storing a single set of benchmarks.
type BenchmarksStore interface {
	StoreBenchmarks(ctx context.Context, b *Benchmarks) error
}

// BenchmarksQuery is the interface for querying the latest benchmarks for each node.
type BenchmarksQuery interface {
	RetrieveLatestBenchmarks(ctx context.Context, ct BenchmarkType, filters []BenchmarkFilter, start, end time.Time) <-chan BenchmarksResult
}

// BenchmarksGetter is the interface for getting a specific benchmarks set. Primarily used for testing.
type BenchmarksGetter interface {
	GetBenchmarks(ctx context.Context, id string) (*Benchmarks, error)
}

// BenchmarksExecutor is the interface for executing a specific set of benchmark tests.
type BenchmarksExecutor interface {
	ExecuteBenchmarks(ctx context.Context, ct BenchmarkType, nodename string) (*Benchmarks, error)
}

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
	Version           string          `json:"version"`
	KubernetesVersion string          `json:"kubernetesVersion"`
	Type              BenchmarkType   `json:"type"`
	NodeName          string          `json:"node_name"`
	Timestamp         metav1.Time     `json:"timestamp"`
	Error             string          `json:"error,omitempty"`
	Tests             []BenchmarkTest `json:"tests,omitempty"`
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
type BenchmarkTest struct {
	Section     string `json:"section"`
	SectionDesc string `json:"section_desc"`
	TestNumber  string `json:"test_number"`
	TestDesc    string `json:"test_desc"`
	TestInfo    string `json:"test_info"`
	Status      string `json:"status"`
	Scored      bool   `json:"scored"`
}

// Filter is the set of filters to limit the returned benchmarks.
type BenchmarkFilter struct {
	Version   string
	NodeNames []string
}

// BenchmarksResult is the result from a Benchmarks query. An error returned in this indicates a terminating error for
// the query.
type BenchmarksResult struct {
	Benchmarks *Benchmarks
	Err        error
}
