package benchmark

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/compliance/pkg/config"
	api "github.com/tigera/lma/pkg/api"
)

const (
	Day               = 24 * time.Hour
	DayAndHalf        = 36 * time.Hour
	keepAliveInterval = 120 * time.Second
)

func Run(ctx context.Context, cfg *config.Config, executor api.BenchmarksExecutor, query api.BenchmarksQuery, store api.BenchmarksStore, healthy func(bool)) error {
	return (&benchmarker{
		ctx:      ctx,
		cfg:      cfg,
		executor: executor,
		query:    query,
		store:    store,
		healthy:  healthy,
	}).run()
}

type benchmarker struct {
	ctx           context.Context
	cfg           *config.Config
	healthy       func(bool)
	executor      api.BenchmarksExecutor
	query         api.BenchmarksQuery
	store         api.BenchmarksStore
	lastBenchmark *api.Benchmarks
}

func (b *benchmarker) run() error {
	log.Info("Executing benchmark continuously once every hour")

	// Assume we are initially healthy.
	b.healthy(true)

	// Initialize keep alive ticker.
	keepAliveTicker := time.NewTicker(keepAliveInterval)

	// Run benchmarks infinitely.
	for {
		// Determine if time for benchmark.
		prev, next := timeOfNextBenchmark()

		for _, ct := range api.AllBenchmarkTypes {
			// Execute benchmarks.
			if err := b.maybeDoBenchmark(prev, next, ct); err != nil {
				b.healthy(false)
			}
		}

		select {
		case <-b.ctx.Done():
			log.Info("Process terminating")
			keepAliveTicker.Stop()
			return nil

		case <-keepAliveTicker.C:
			log.Debug("Waking up from keep-alive timer.")
			b.healthy(true)
		}
	}
}

func (b *benchmarker) maybeDoBenchmark(prev, next time.Time, ct api.BenchmarkType) error {
	// If time of last benchmark is not known, then populate from an elastic search query.
	if b.lastBenchmark == nil {
		now := time.Now()
		filter := []api.BenchmarkFilter{{NodeNames: []string{b.cfg.NodeName}}}
		lastBench := <-b.query.RetrieveLatestBenchmarks(b.ctx, ct, filter, now.Add(-DayAndHalf), now)
		if lastBench.Err != nil {
			log.WithError(lastBench.Err).Error("Failed to retrieve most recent benchmark results")
			return lastBench.Err
		} else if lastBench.Benchmarks != nil {
			log.WithField("lastBenchmarkTime", lastBench.Benchmarks.Timestamp.Time).Info("Found most recent benchmark results")
			b.lastBenchmark = lastBench.Benchmarks
		}
	}

	// If time of last benchmark is less than prev then we haven't run a benchmark in this interval. Do it.
	if b.lastBenchmark == nil || b.lastBenchmark.Timestamp.Time.Before(prev) {
		log.Info("Executing benchmark")
		benchmarks, err := b.executor.ExecuteBenchmarks(b.ctx, ct, b.cfg.NodeName)
		if err != nil {
			log.WithError(err).Error("Failed to execute benchmark")
			return err
		}

		// Log the execution error if any occurred.
		if benchmarks.Error != "" {
			log.WithField("err", benchmarks.Error).Warn("Error occurred while executing benchmark, storing anyway.")
		}

		// Only store results if a change has occurred or if it has been more than a day since last test execution.
		if b.lastBenchmark != nil && benchmarks.Equal(*b.lastBenchmark) && b.lastBenchmark.Timestamp.Time.Add(Day).After(time.Now()) {
			log.Info("Benchmark results have not changed, refusing to store results.")
			return nil
		}
		log.Info("Storing benchmark results")
		if err := b.store.StoreBenchmarks(b.ctx, benchmarks); err != nil {
			log.WithError(err).Error("Failed to store benchmark results")
			return err
		}

		b.lastBenchmark = benchmarks
		return nil
	}

	log.WithField("nextBenchmark", next.Sub(b.lastBenchmark.Timestamp.Time)).Debug("Time to next benchmark.")
	return nil
}

func timeOfNextBenchmark() (time.Time, time.Time) {
	now := time.Now()
	year, month, day := now.Date()
	fireTime := time.Date(year, month, day, now.Hour(), 0, 0, 0, now.Location())
	if fireTime.Before(now) {
		return fireTime, fireTime.Add(time.Hour)
	}
	return fireTime.Add(-time.Hour), fireTime
}
