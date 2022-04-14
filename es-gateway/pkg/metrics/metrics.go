// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

const (
	LabelManagedClusterID = "cluster_id"
)

// NewCollector creates a new instance of a metrics collector.
func NewCollector() (Collector, error) {
	metricLabels := []string{LabelManagedClusterID}
	elasticLogChannel := make(chan prometheus.Metric)
	c := &collector{
		elasticLogChannel,
		prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "tigera_elastic_log_bytes_written",
			Help: "Number of bytes ingested into Elasticsearch broken down per cluster id",
		}, metricLabels),
		prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "tigera_elastic_log_bytes_read",
			Help: "Number of bytes read from into Elasticsearch broken down per cluster id",
		}, metricLabels),
	}

	for _, collector := range []prometheus.Collector{
		c.elasticLogBytesWritten,
		c.elasticLogBytesRead,
	} {
		if err := prometheus.Register(collector); err != nil {
			return nil, err
		}
	}
	c.elasticLogBytesWritten.Collect(c.elasticLogChannel)
	c.elasticLogBytesRead.Collect(c.elasticLogChannel)

	return c, nil
}

// Collector provides an interface for a prometheus metrics collector.
type Collector interface {

	// Serve runs a server listening on a configurable port to expose the /metrics endpoint.
	Serve(address string) error

	// CollectLogBytesWritten registers number of bytes ingested into Elasticsearch broken down per cluster.
	CollectLogBytesWritten(managedClusterID string, bytes float64) error

	// CollectLogBytesRead registers number of bytes ingested into Elasticsearch broken down per cluster.
	CollectLogBytesRead(managedClusterID string, bytes float64) error
}

type collector struct {
	elasticLogChannel      chan prometheus.Metric
	elasticLogBytesWritten *prometheus.CounterVec
	elasticLogBytesRead    *prometheus.CounterVec
}

// Serve runs a server listening on a configurable port to expose the /metrics endpoint.
func (c *collector) Serve(address string) error {
	http.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(address, nil)
}

// CollectLogBytesWritten registers number of bytes ingested into Elasticsearch broken down per cluster.
func (c *collector) CollectLogBytesWritten(clusterID string, bytes float64) error {
	log.Tracef("collecting bytes written for clusterId: %s,bytes: %v", clusterID, bytes)
	labels := prometheus.Labels{
		LabelManagedClusterID: clusterID,
	}
	counter, err := c.elasticLogBytesWritten.GetMetricWith(labels)
	if err != nil {
		return err
	}
	counter.Add(bytes)
	return nil
}

// CollectLogBytesRead registers number of bytes read from Elasticsearch broken down per cluster.
func (c *collector) CollectLogBytesRead(clusterID string, bytes float64) error {
	log.Tracef("collecting bytes read for clusterId: %s,bytes: %v", clusterID, bytes)
	labels := prometheus.Labels{
		LabelManagedClusterID: clusterID,
	}
	counter, err := c.elasticLogBytesRead.GetMetricWith(labels)
	if err != nil {
		return err
	}
	counter.Add(bytes)
	return nil
}
