// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package api

import (
	"context"
	"time"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

// BenchmarksStore is the interface for storing a single set of benchmarks.
type BenchmarksStore interface {
	StoreBenchmarks(ctx context.Context, b *v1.Benchmarks) error
}

// BenchmarksQuery is the interface for querying the latest benchmarks for each node.
type BenchmarksQuery interface {
	RetrieveLatestBenchmarks(ctx context.Context, ct v1.BenchmarkType, filters []v1.BenchmarksFilter, start, end time.Time) <-chan BenchmarksResult
}

// BenchmarksGetter is the interface for getting a specific benchmarks set. Primarily used for testing.
type BenchmarksGetter interface {
	GetBenchmarks(ctx context.Context, id string) (*v1.Benchmarks, error)
}

// BenchmarksExecutor is the interface for executing a specific set of benchmark tests.
type BenchmarksExecutor interface {
	ExecuteBenchmarks(ctx context.Context, ct v1.BenchmarkType, nodename string) (*v1.Benchmarks, error)
}

var AllBenchmarkTypes = []v1.BenchmarkType{v1.TypeKubernetes}

// BenchmarksResult is the result from a Benchmarks query. An error returned in this indicates a terminating error for
// the query.
type BenchmarksResult struct {
	Benchmarks *v1.Benchmarks
	Err        error
}
