package mock

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/tigera/lma/pkg/api"
)

type DB struct {
	storage                     map[string]*api.Benchmarks
	NStoreCalls, NRetrieveCalls int
}

func NewMockDB() *DB {
	return &DB{storage: map[string]*api.Benchmarks{}}
}

func (db *DB) StoreBenchmarks(ctx context.Context, b *api.Benchmarks) error {
	db.NStoreCalls++
	db.storage[b.UID()] = b
	return nil
}

func (db *DB) RetrieveLatestBenchmarks(ctx context.Context, ct api.BenchmarkType, filters []api.BenchmarkFilter, start, end time.Time) <-chan api.BenchmarksResult {
	ch := make(chan api.BenchmarksResult, 1)
	go func() {
		defer close(ch)

		for _, b := range db.storage {
			ch <- api.BenchmarksResult{Benchmarks: b}
		}
	}()
	db.NRetrieveCalls++
	return ch
}

func (db *DB) GetBenchmarks(ctx context.Context, id string) (*api.Benchmarks, error) {
	return db.storage[id], nil
}

type Executor struct {
}

func (e *Executor) ExecuteBenchmarks(ctx context.Context, ct api.BenchmarkType, nodename string) (*api.Benchmarks, error) {
	return &api.Benchmarks{
		Version:   "1.4",
		Type:      api.TypeKubernetes,
		NodeName:  nodename,
		Timestamp: metav1.Time{Time: time.Now()},
		Tests: []api.BenchmarkTest{
			{
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
