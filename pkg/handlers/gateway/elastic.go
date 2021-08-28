package gateway

import (
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/tigera/es-gateway/pkg/metrics"
	"github.com/tigera/es-gateway/pkg/middlewares"
)

func ElasticModifyResponseFunc(collector metrics.Collector) func(res *http.Response) error {
	return func(res *http.Response) error {
		req := res.Request
		if res.StatusCode >= 200 && res.StatusCode < 300 {
			ctx := req.Context()
			tenantID := ctx.Value(middlewares.TenantIDKey)
			clusterID := ctx.Value(middlewares.ClusterIDKey)

			// clusterID should always contain a value for authenticated users, while tenantID should be non-nil, but can be empty.
			if clusterID != nil && clusterID.(string) != "" && tenantID != nil {
				if req.ContentLength > 0 {
					if err := collector.CollectLogBytesWritten(tenantID.(string), clusterID.(string), float64(req.ContentLength)); err != nil {
						log.Errorf("Error occurred while collecting CollectLogBytesRead metrics for request to: %v", req.RequestURI)
					}
				}
				if res.ContentLength > 0 {
					if err := collector.CollectLogBytesRead(tenantID.(string), clusterID.(string), float64(res.ContentLength)); err != nil {
						log.Errorf("Error occurred while collecting CollectLogBytesRead metrics for request to: %v", req.RequestURI)
					}
				}
				log.Debugf(
					"Collecting metrics after successful response: %v, url: %v, response: %v, req contentlength: %v, res contentlength: %v, tenant: %v, cluster: %v",
					req.Method,
					req.URL,
					req.Response,
					req.ContentLength,
					res.ContentLength,
					tenantID,
					clusterID,
				)
			}
		}
		return nil
	}
}
