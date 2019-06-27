package mock

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tigera/compliance/pkg/benchmark"
)

type DB struct {
	storage                     map[string]*benchmark.Benchmarks
	NStoreCalls, NRetrieveCalls int
}

func NewMockDB() *DB {
	return &DB{storage: map[string]*benchmark.Benchmarks{}}
}

func (db *DB) StoreBenchmarks(ctx context.Context, b *benchmark.Benchmarks) error {
	db.NStoreCalls++
	db.storage[b.UID()] = b
	return nil
}

func (db *DB) RetrieveLatestBenchmarks(ctx context.Context, ct benchmark.BenchmarkType, filters []benchmark.Filter, start, end time.Time) <-chan benchmark.BenchmarksResult {
	ch := make(chan benchmark.BenchmarksResult, 1)
	go func() {
		defer close(ch)

		for _, b := range db.storage {
			ch <- benchmark.BenchmarksResult{Benchmarks: b}
		}
	}()
	db.NRetrieveCalls++
	return ch
}

func (db *DB) GetBenchmarks(ctx context.Context, id string) (*benchmark.Benchmarks, error) {
	return db.storage[id], nil
}

type Executor struct {
}

func (e *Executor) ExecuteBenchmarks(ctx context.Context, ct benchmark.BenchmarkType, nodename string) (*benchmark.Benchmarks, error) {
	return &benchmark.Benchmarks{
		Version:   "1.4",
		Type:      benchmark.TypeKubernetes,
		NodeName:  nodename,
		Timestamp: metav1.Time{time.Now()},
		Tests: []benchmark.Test{
			benchmark.Test{
				Section:     "1.1",
				SectionDesc: "API Server",
				TestNumber:  "1.1.1",
				TestDesc:    "Ensure that --annoymous-auth is not enabled",
				TestInfo:    "Remove the line on the kube-apiserver command",
				Status:      "PASS",
				Scored:      true,
			},
		},
	}, nil
}
