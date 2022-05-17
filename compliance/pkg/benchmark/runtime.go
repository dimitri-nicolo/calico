package benchmark

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/compliance/pkg/config"
	"github.com/projectcalico/calico/lma/pkg/api"
)

const (
	benchmarkFrequency      = 60 * time.Minute
	benchmarkTimeout        = 90 * time.Second
	elasticStorageFrequency = 24 * time.Hour
	elasticQueryCutOff      = 36 * time.Hour
	TickerFrequency         = 120 * time.Second
)

func Run(ctx context.Context, cfg *config.Config, executor api.BenchmarksExecutor, query api.BenchmarksQuery, store api.BenchmarksStore, healthy func(bool)) error {
	return (&benchmarker{
		ctx:                ctx,
		cfg:                cfg,
		executor:           executor,
		query:              query,
		store:              store,
		healthy:            healthy,
		lastBenchmarks:     make(map[api.BenchmarkType]*api.Benchmarks),
		lastExecutionTimes: make(map[api.BenchmarkType]time.Time),
	}).run()
}

type benchmarker struct {
	ctx                context.Context
	cfg                *config.Config
	healthy            func(bool)
	executor           api.BenchmarksExecutor
	query              api.BenchmarksQuery
	store              api.BenchmarksStore
	lastBenchmarks     map[api.BenchmarkType]*api.Benchmarks
	lastExecutionTimes map[api.BenchmarkType]time.Time
}

func (b *benchmarker) run() error {
	log.Infof("executing benchmarks continuously once every %+v", benchmarkFrequency)

	b.healthy(true)
	ticker := time.NewTicker(TickerFrequency)

	for {
		select {
		case <-b.ctx.Done():
			log.Warn("terminating benchmark process")
			ticker.Stop()
			return nil

		case <-ticker.C:
			var errMsgs []string
			for _, bt := range api.AllBenchmarkTypes {
				if err := b.benchmark(b.ctx, bt); err != nil {
					errMsgs = append(errMsgs, err.Error())
				}
			}
			if len(errMsgs) > 0 {
				log.Errorf("error in executing benchmark %+v", errMsgs)
			} else {
				b.healthy(true)
			}
		}
	}
}

// checks if benchmarking is needed for a given benchmark type
func (b *benchmarker) needBenchmark(bt api.BenchmarkType) bool {
	// if there is no last benchmark or if one hour elapsed from time of last benchmark then we need to do it.
	if _, found := b.lastBenchmarks[bt]; !found {
		return true
	}

	if _, exists := b.lastExecutionTimes[bt]; exists {
		return time.Now().After(b.lastExecutionTimes[bt].Add(benchmarkFrequency))
	}

	return true
}

// executes benchmark for a given type, stores the results to elastic search if required.
func (b *benchmarker) executeBenchmark(ctx context.Context, bt api.BenchmarkType) error {
	log.Infof("executing benchmark on node %+v", b.cfg.NodeName)
	current, err := b.executor.ExecuteBenchmarks(ctx, bt, b.cfg.NodeName)

	if err != nil {
		log.WithError(err).Error("failed to execute benchmark")
		return err
	}

	// log the execution error if any occurred, we don't throw error in this case as kube-bench did run successfully.
	if current.Error != "" {
		log.WithField("err", current.Error).Warn("error during benchmark execution")
	}

	b.lastExecutionTimes[bt] = time.Now()
	// don't store results if they are same as previous execution
	if last := b.lastBenchmarks[bt]; last != nil && current.Equal(*last) && last.Timestamp.Time.Add(elasticStorageFrequency).After(time.Now()) {
		log.Info("no change in benchmark results, skip storing to elastic search")
		return nil
	}

	log.Info("storing benchmark results to elastic search")
	if err := b.store.StoreBenchmarks(ctx, current); err != nil {
		log.WithError(err).Error("error in storing benchmark results")
		return err
	}

	b.lastBenchmarks[bt] = current
	return nil
}

// sets lastBenchmarks for a given benchmark type, if lastBenchmarks is not set checks elastic search
func (b *benchmarker) setLastBenchmark(ctx context.Context, bt api.BenchmarkType) error {
	// if last benchmark is already set return.
	if _, found := b.lastBenchmarks[bt]; found {
		return nil
	}
	// if last benchmark is not set, query elastic search for last executed benchmarks.
	result := <-b.query.RetrieveLatestBenchmarks(ctx, bt, []api.BenchmarkFilter{{NodeNames: []string{b.cfg.NodeName}}},
		time.Now().Add(-elasticQueryCutOff), time.Now())

	if result.Err != nil {
		log.WithError(result.Err).Error("failed to retrieve most recent benchmark results")
		return result.Err
	}
	b.lastBenchmarks[bt] = result.Benchmarks
	return nil
}

func (b *benchmarker) benchmark(c context.Context, bt api.BenchmarkType) error {
	ctx, cancel := context.WithTimeout(c, benchmarkTimeout)
	defer cancel()

	rc := make(chan error, 1)
	go func() {
		if err := b.setLastBenchmark(ctx, bt); err != nil {
			rc <- err
			return
		}
		if b.needBenchmark(bt) {
			rc <- b.executeBenchmark(ctx, bt)
			return
		}
		rc <- nil
	}()

	select {
	case err := <-rc:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}

}
