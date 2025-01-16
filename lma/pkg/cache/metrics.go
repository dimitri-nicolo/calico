// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package cache

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	log "github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"k8s.io/component-base/metrics"
)

const labelCacheName = "cacheName"

var (
	hits = metrics.NewCounterVec(&metrics.CounterOpts{
		Name: "tigera_cache_hits_total",
		Help: "The total number of cache hits",
	}, []string{labelCacheName})
	misses = metrics.NewCounterVec(&metrics.CounterOpts{
		Name: "tigera_cache_misses_total",
		Help: "The total number of cache misses",
	}, []string{labelCacheName})
	size = metrics.NewGaugeVec(&metrics.GaugeOpts{
		Name: "tigera_cache_size",
		Help: "The total number of elements in the cache",
	}, []string{labelCacheName})
)

var logger *zap.Logger

func init() {
	// any errors in this package will be repetitive so sample heavily
	loggerCfg := zap.NewProductionConfig()
	loggerCfg.Sampling.Initial = 3
	loggerCfg.Sampling.Thereafter = 0
	root, err := loggerCfg.Build()
	if err != nil {
		log.Fatalf("failed to create zap logger: %v", err)
	}
	logger = root.Named("lma.cache.metrics")

	// eagerly initialize the metrics by registering them with a throwaway registry
	RegisterMetricsWith(metrics.NewKubeRegistry().MustRegister)
}

func RegisterMetricsWith(mustRegister func(...metrics.Registerable)) {
	mustRegister(hits, misses, size)
}

func (c *expiring[K, V]) startMetrics(ctx context.Context, interval time.Duration) {
	go tick(ctx, interval, c.collectGaugeMetrics, c.clearMetrics)
}

func (c *expiring[K, V]) metricsLabels() prometheus.Labels {
	return prometheus.Labels{
		labelCacheName: c.name,
	}
}

func (c *expiring[K, V]) hitsMetric() (int, error) {
	return c.metricCounterValue(hits)
}

func (c *expiring[K, V]) missesMetric() (int, error) {
	return c.metricCounterValue(misses)
}

func (c *expiring[K, V]) sizeMetric() (int, error) {
	return c.metricGaugeValue(size)
}

func (c *expiring[K, V]) registerCacheMiss() {
	c.incMetricsCounter(misses)
}

func (c *expiring[K, V]) registerCacheHit() {
	c.incMetricsCounter(hits)
}

func (c *expiring[K, V]) incMetricsCounter(vec *metrics.CounterVec) {
	counter, err := vec.GetMetricWith(c.metricsLabels())
	if err != nil {
		logger.Info("failed to get counter metric", zap.String("name", vec.Name), zap.Error(err))
	} else {
		counter.Inc()
	}
}

func (c *expiring[K, V]) metricCounterValue(vec *metrics.CounterVec) (int, error) {
	value, err := c.metricsValue(vec.MetricVec)
	if err != nil {
		return 0, err
	}
	return int(value.Counter.GetValue()), nil
}

func (c *expiring[K, V]) metricGaugeValue(vec *metrics.GaugeVec) (int, error) {
	value, err := c.metricsValue(vec.MetricVec)
	if err != nil {
		return 0, err
	}
	return int(value.Gauge.GetValue()), nil
}

func (c *expiring[K, V]) metricsValue(vec *prometheus.MetricVec) (*dto.Metric, error) {
	m, err := vec.GetMetricWith(c.metricsLabels())
	if err != nil {
		return nil, err
	}
	d := dto.Metric{}
	err = m.Write(&d)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (c *expiring[K, V]) collectGaugeMetrics() {
	metric, err := size.GetMetricWith(c.metricsLabels())
	if err != nil {
		logger.Info("failed to get size metric", zap.String("name", size.Name), zap.Error(err))
	} else {
		metric.Set(float64(c.cache.ItemCount()))
	}
}

func (c *expiring[K, V]) clearMetrics() {
	labels := c.metricsLabels()
	hits.Delete(labels)
	misses.Delete(labels)
	size.Delete(labels)
}

func tick(ctx context.Context, interval time.Duration, tickFn func(), stopFn func()) {
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			tickFn()
		case <-ctx.Done():
			stopFn()
			return
		}
	}
}
