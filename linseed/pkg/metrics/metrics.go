// Copyright (c) 2023 Tigera, Inc. All rights reserved.
//

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	prometheus.MustRegister(HTTPTotalRequests)
	prometheus.MustRegister(HTTPInflightRequests)
	prometheus.MustRegister(HTTPResponseDuration)
	prometheus.MustRegister(HTTPRequestSize)
	prometheus.MustRegister(HTTPResponseSize)
	prometheus.MustRegister(ElasticResponseDuration)
	prometheus.MustRegister(ElasticResponseStatus)
	prometheus.MustRegister(ElasticConnectionErrors)
	prometheus.MustRegister(BytesWrittenPerClusterIDAndTenantID)
	prometheus.MustRegister(BytesReadPerClusterIDAndTenantID)
}

const (
	LabelPath      = "path"
	LabelCode      = "code"
	LabelMethod    = "method"
	LabelClusterID = "cluster_id"
	LabelTenantID  = "tenant_id"
	Source         = "source"
)

var (
	histogramBuckets = []float64{.1, .25, .5, 1, 5, 10}
	sizeBuckets      = prometheus.ExponentialBuckets(1000, 10, 4)

	// HTTPTotalRequests will track the number of HTTP requests
	// across all APIs broken down by code, method and path
	HTTPTotalRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "tigera",
			Subsystem: "linseed",
			Name:      "http_requests_total",
			Help:      "Number of total requests.",
		},
		[]string{LabelPath, LabelMethod, LabelCode, LabelClusterID, LabelTenantID})

	// HTTPInflightRequests will track the number of inflight HTTP requests
	// across all APIs broken down by method and path
	HTTPInflightRequests = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tigera",
		Subsystem: "linseed",
		Name:      "http_requests_inflight",
		Help:      "Number of inflight requests.",
	}, []string{LabelPath, LabelMethod})

	// HTTPResponseDuration will track the duration of Linseed's HTTP response
	// returned by Linseed across all APIs broken down by path and method
	HTTPResponseDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "tigera",
		Subsystem: "linseed",
		Name:      "http_response_time_seconds",
		Help:      "Duration of HTTP response.",
		Buckets:   histogramBuckets,
	}, []string{LabelPath, LabelMethod})

	// HTTPRequestSize will track the size of Linseed's HTTP requests
	// across all APIs broken down by path and method
	HTTPRequestSize = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "tigera",
		Subsystem: "linseed",
		Name:      "http_request_size_bytes",
		Help:      "Size of HTTP request.",
		Buckets:   sizeBuckets,
	}, []string{LabelPath, LabelMethod})

	// HTTPResponseSize will track the size of Linseed's HTTP responses
	// across all APIs broken down by path and method
	HTTPResponseSize = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "tigera",
		Subsystem: "linseed",
		Name:      "http_response_size_bytes",
		Help:      "Size of HTTP response.",
		Buckets:   sizeBuckets,
	}, []string{LabelPath, LabelMethod})

	// ElasticResponseDuration will track the duration of Elastic's HTTP response
	// across all APIs broken down by path and method
	ElasticResponseDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "tigera",
		Subsystem: "linseed",
		Name:      "elastic_response_time_seconds",
		Help:      "Duration of HTTP requests to Elastic.",
		Buckets:   histogramBuckets,
	}, []string{LabelPath, LabelMethod, Source})

	// ElasticResponseStatus will track the duration of Elastic's HTTP response
	// across all APIs broken down by path and method
	ElasticResponseStatus = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "tigera",
			Subsystem: "linseed",
			Name:      "elastic_response_status",
			Help:      "Status of HTTP response to Elastic.",
		},
		[]string{LabelCode, LabelMethod, LabelPath, Source},
	)

	// ElasticConnectionErrors will track the duration of Elastic's HTTP response
	// across all APIs broken down by path and method
	ElasticConnectionErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "tigera",
			Subsystem: "linseed",
			Name:      "elastic_connection_errors",
			Help:      "Number of connection errors from Elastic.",
		},
		[]string{LabelCode, LabelMethod, LabelPath, Source},
	)

	// BytesWrittenPerClusterIDAndTenantID will track how many bytes are written across
	// all bulk APIs broken down by cluster ID; this is an application metric required
	// for billing
	BytesWrittenPerClusterIDAndTenantID = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "tigera",
			Subsystem: "linseed",
			Name:      "bytes_written",
			Help:      "Number of bytes ingested into Linseed broken down per cluster id and tenant id.",
		}, []string{LabelClusterID, LabelTenantID})

	// BytesReadPerClusterIDAndTenantID will track how many bytes are read across all APIs
	// broken down by cluster ID; this is an application metric required for billing
	BytesReadPerClusterIDAndTenantID = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "tigera",
			Subsystem: "linseed",
			Name:      "bytes_read",
			Help:      "Number of bytes read from Linseed broken down per cluster id and tenant id.",
		}, []string{LabelClusterID, LabelTenantID})
)
