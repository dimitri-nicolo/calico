// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package benchmark

import (
	"context"
	"time"
)

// BenchmarksStore is the interface for storing a single set of benchmarks.
type BenchmarksStore interface {
	StoreBenchmarks(ctx context.Context, b *Benchmarks) error
}

// BenchmarksQuery is the interface for querying the latest benchmarks for each node.
type BenchmarksQuery interface {
	RetrieveLatestBenchmarks(ctx context.Context, ct BenchmarkType, filters []Filter, start, end time.Time) <-chan BenchmarksResult
}

// BenchmarksGetter is the interface for getting a specific benchmarks set. Primarily used for testing.
type BenchmarksGetter interface {
	GetBenchmarks(ctx context.Context, id string) (*Benchmarks, error)
}
