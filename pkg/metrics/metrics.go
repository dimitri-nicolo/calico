package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

const (
	labelTenant = "tenant_id"
	labelManagedClusterID = "cluster_id"
)

// NewCollector creates a new instance of a metrics collector.
func NewCollector() (Collector, error) {
	elasticLogChannel := make(chan prometheus.Metric)
	c := &collector{
		elasticLogChannel,
		prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "tigera_elastic_log_bytes_written",
			Help: "Amount of elasticsearch data in bytes ingested broken down per tenant and cluster id",
	}, []string{labelTenant, labelManagedClusterID} )}


	if err := prometheus.Register(c.elasticLogBytes); err != nil {
		return nil, err
	}

	c.elasticLogBytes.Collect(c.elasticLogChannel)

	return c, nil
}

// Collector provides an interface for a prometheus metrics collector.
type Collector interface {

	// Serve runs a server listening on a configurable port to expose the /metrics endpoint.
	Serve(address string) error

	// CollectLogBytes registers number of bytes ingested into Elasticsearch broken down per tenant and cluster.
	CollectLogBytes(tenant string, managedClusterID string, bytes float64) error
}

type collector struct {
	elasticLogChannel chan prometheus.Metric
	elasticLogBytes *prometheus.CounterVec
}

// Serve runs a server listening on a configurable port to expose the /metrics endpoint.
func (c *collector) Serve(address string) error {
	http.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(address, nil)
}

// CollectLogBytes registers number of bytes ingested into Elasticsearch broken down per tenant and cluster.
func (c *collector) CollectLogBytes(tenant, managedClusterID string, bytes float64) error {
	counter, err := c.elasticLogBytes.GetMetricWith(prometheus.Labels{
		labelTenant: tenant,
		labelManagedClusterID: managedClusterID,
	})
	if err != nil {
		return err
	}
	counter.Add(bytes)
	return nil
}
