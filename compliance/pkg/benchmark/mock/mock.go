package mock

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/projectcalico/calico/compliance/pkg/api"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

type DB struct {
	storage                     map[string]*v1.Benchmarks
	NStoreCalls, NRetrieveCalls int
}

func NewMockDB() *DB {
	return &DB{storage: map[string]*v1.Benchmarks{}}
}

func (db *DB) StoreBenchmarks(ctx context.Context, b *v1.Benchmarks) error {
	db.NStoreCalls++
	db.storage[b.UID()] = b
	return nil
}

func (db *DB) RetrieveLatestBenchmarks(ctx context.Context, ct v1.BenchmarkType, filters []v1.BenchmarksFilter, start, end time.Time) <-chan api.BenchmarksResult {
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

func (db *DB) GetBenchmarks(ctx context.Context, id string) (*v1.Benchmarks, error) {
	return db.storage[id], nil
}

type Executor struct{}

func (e *Executor) ExecuteBenchmarks(ctx context.Context, ct v1.BenchmarkType, nodename string) (*v1.Benchmarks, error) {
	return &v1.Benchmarks{
		Version:   "1.4",
		Type:      v1.TypeKubernetes,
		NodeName:  nodename,
		Timestamp: metav1.Time{Time: time.Now()},
		Tests: []v1.BenchmarkTest{
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
