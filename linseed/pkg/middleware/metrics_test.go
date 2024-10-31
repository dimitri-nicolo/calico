// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/projectcalico/calico/linseed/pkg/metrics"
	"github.com/projectcalico/calico/linseed/pkg/middleware"
)

func TestMetrics(t *testing.T) {
	const (
		anyPath    = "/api/v1"
		anyMethod  = "POST"
		anyCode    = "200"
		anyTenant  = "any-tenant"
		anyCluster = "any-cluster"
	)

	t.Run("should track configured metrics", func(t *testing.T) {
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			code, err := strconv.Atoi(anyCode)

			require.NoError(t, err)
			w.WriteHeader(code)
		})

		metricsMiddleware := middleware.Metrics{}.Track()
		req, err := http.NewRequest(anyMethod, anyPath, nil)
		req = req.WithContext(middleware.WithTenantID(req.Context(), anyTenant))
		req = req.WithContext(middleware.WithClusterID(req.Context(), anyCluster))
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		metricsMiddleware(testHandler).ServeHTTP(rec, req)

		// Make sure the metrics was incremented for these labels
		_, err = metrics.HTTPInflightRequests.GetMetricWith(prometheus.Labels{
			metrics.LabelPath:   anyPath,
			metrics.LabelMethod: anyMethod,
		})
		require.NoError(t, err)

		_, err = metrics.HTTPResponseSize.GetMetricWith(prometheus.Labels{
			metrics.LabelPath:   anyPath,
			metrics.LabelMethod: anyMethod,
		})
		require.NoError(t, err)

		_, err = metrics.HTTPRequestSize.GetMetricWith(prometheus.Labels{
			metrics.LabelPath:   anyPath,
			metrics.LabelMethod: anyMethod,
		})
		require.NoError(t, err)

		_, err = metrics.HTTPResponseDuration.GetMetricWith(prometheus.Labels{
			metrics.LabelPath:   anyPath,
			metrics.LabelMethod: anyMethod,
		})
		require.NoError(t, err)

		_, err = metrics.HTTPResponseDuration.GetMetricWith(prometheus.Labels{
			metrics.LabelPath:   anyPath,
			metrics.LabelMethod: anyMethod,
		})
		require.NoError(t, err)

		_, err = metrics.HTTPTotalRequests.GetMetricWith(prometheus.Labels{
			metrics.LabelPath:      anyPath,
			metrics.LabelMethod:    anyMethod,
			metrics.LabelCode:      anyCode,
			metrics.LabelClusterID: anyCluster,
			metrics.LabelTenantID:  anyTenant,
		})
		require.NoError(t, err)

		_, err = metrics.BytesReadPerClusterIDAndTenantID.GetMetricWith(prometheus.Labels{
			metrics.LabelClusterID: anyCluster,
			metrics.LabelTenantID:  anyTenant,
		})
		require.NoError(t, err)

		_, err = metrics.BytesWrittenPerClusterIDAndTenantID.GetMetricWith(prometheus.Labels{
			metrics.LabelClusterID: anyCluster,
			metrics.LabelTenantID:  anyTenant,
		})
		require.NoError(t, err)
	})
}
