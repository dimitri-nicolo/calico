package server

import (
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	jclust "github.com/projectcalico/calico/voltron/internal/pkg/clusters"
	"github.com/projectcalico/calico/voltron/internal/pkg/server/metrics"
	"github.com/projectcalico/calico/voltron/internal/pkg/utils"
)

type InnerHandler interface {
	Handler() http.Handler
}

func NewInnerHandler(t string, c *jclust.ManagedCluster, proxy http.Handler) InnerHandler {
	return &handlerHelper{
		ManagedCluster: c,
		proxy:          proxy,
		tenantID:       t,
	}
}

type handlerHelper struct {
	ManagedCluster *jclust.ManagedCluster
	proxy          http.Handler
	tenantID       string
}

func (h *handlerHelper) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set the cluster and tenant ID headers here. If they are already set,
		// but don't match the expected value for this cluster, return an error.
		clusterID := r.Header.Get(utils.ClusterHeaderField)
		tenantID := r.Header.Get(utils.TenantHeaderField)
		fields := log.Fields{
			"url":                    r.URL,
			utils.ClusterHeaderField: clusterID,
			utils.TenantHeaderField:  tenantID,
		}
		logCtx := log.WithFields(fields)

		// Increment the number of requests in flight and total number of requests received.
		promLabels := []string{h.ManagedCluster.ID, tenantID, r.URL.String()}
		if inflightMetric, err := metrics.InnerRequestsInflight.GetMetricWithLabelValues(promLabels...); err != nil {
			logCtx.WithError(err).Warn("Failed to get inflight metric")
		} else {
			inflightMetric.Inc()
			defer inflightMetric.Dec()
		}
		if totalRequestsMetrics, err := metrics.InnerRequestsTotal.GetMetricWithLabelValues(promLabels...); err != nil {
			logCtx.WithError(err).Warn("Failed to get total requests metric")
		} else {
			totalRequestsMetrics.Inc()
		}

		if clusterID != "" {
			if clusterID != h.ManagedCluster.ID {
				// Cluster ID is set, and it doesn't match what we expect.
				logCtx.Warn("Unexpected cluster ID")
				if metric, err := metrics.InnerRequestBadClusterIDErrors.GetMetricWithLabelValues(promLabels...); err != nil {
					logCtx.WithError(err).Warn("Failed to get bad cluster ID metric")
				} else {
					metric.Inc()
				}
				writeHTTPError(w, unexpectedClusterIDError(clusterID))
				return
			}
		}

		// Set the cluster ID header before forwarding to indicate the originating cluster.
		r.Header.Set(utils.ClusterHeaderField, h.ManagedCluster.ID)

		if h.tenantID != "" {
			// Running in multi-tenant mode. We need to set the tenant ID on
			// any requests received over the tunnel.
			if tenantID != "" && tenantID != h.tenantID {
				// Tenant ID is set, and it doesn't match what we expect.
				logCtx.Warn("Unexpected tenant ID")
				if metric, err := metrics.InnerRequestBadTenantIDErrors.GetMetricWithLabelValues(promLabels...); err != nil {
					logCtx.WithError(err).Warn("Failed to get bad tenant ID metric")
				} else {
					metric.Inc()
				}
				writeHTTPError(w, unexpectedTenantIDError(tenantID))
				return
			}

			// Set the tenant ID before forwarding to indicate the originating tenant.
			r.Header.Set(utils.TenantHeaderField, h.tenantID)
		}

		// Headers have been set properly. Now, proxy the connection
		// using Voltron's own key / cert for mTLS with Linseed.
		logCtx.Debug("Handling connection received over the tunnel")
		start := time.Now()
		h.proxy.ServeHTTP(w, r)

		// Update metrics tracking request duration.
		if requestTimeMetric, err := metrics.InnerRequestTimeSecondsTotal.GetMetricWithLabelValues(promLabels...); err != nil {
			logCtx.WithError(err).Warn("Failed to get request time metric")
		} else {
			requestTimeMetric.Add(time.Since(start).Seconds())
		}
		if requestDurationmetrics, err := metrics.InnerRequestTimeSeconds.GetMetricWithLabelValues(promLabels...); err != nil {
			logCtx.WithError(err).Warn("Failed to get request duration metric")
		} else {
			requestDurationmetrics.Observe(time.Since(start).Seconds())
		}
	})
}
