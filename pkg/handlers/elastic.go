// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package handlers

import (
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/tigera/es-gateway/pkg/metrics"
	"github.com/tigera/es-gateway/pkg/middlewares"
)

func ElasticModifyResponseFunc(collector metrics.Collector) func(res *http.Response) error {
	return func(res *http.Response) error {
		req := res.Request
		if collector != nil && res.StatusCode >= http.StatusOK && res.StatusCode < http.StatusMultipleChoices {
			ctx := req.Context()
			clusterID := ctx.Value(middlewares.ClusterIDKey)
			
			if clusterID != nil && clusterID.(string) != "" {
				if req.ContentLength > 0 {
					if err := collector.CollectLogBytesWritten(clusterID.(string), float64(req.ContentLength)); err != nil {
						log.Errorf("Error occurred while collecting CollectLogBytesRead metrics for request to: %v", req.RequestURI)
					}
				}
				if res.ContentLength > 0 {
					if err := collector.CollectLogBytesRead(clusterID.(string), float64(res.ContentLength)); err != nil {
						log.Errorf("Error occurred while collecting CollectLogBytesRead metrics for request to: %v", req.RequestURI)
					}
				}
				log.Tracef(
					"Collecting metrics after successful response: %v, url: %v, response: %v, req contentlength: %v, res contentlength: %v, cluster: %v",
					req.Method,
					req.URL,
					req.Response,
					req.ContentLength,
					res.ContentLength,
					clusterID,
				)
			}
		}
		return nil
	}
}
