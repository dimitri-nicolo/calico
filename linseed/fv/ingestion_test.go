// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/projectcalico/calico/libcalico-go/lib/json"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"

	"github.com/projectcalico/calico/linseed/pkg/client"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"

	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	"github.com/projectcalico/calico/linseed/pkg/config"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"

	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

var esClient *elastic.Client

func ingestionSetupAndTeardown(t *testing.T, index, cluster string) func() {
	// Hook logrus into testing.T
	config.ConfigureLogging("DEBUG")
	logCancel := logutils.RedirectLogrusToTestingT(t)

	// Create an ES client.
	var err error
	esClient, err = elastic.NewSimpleClient(elastic.SetURL("http://localhost:9200"), elastic.SetInfoLog(logrus.StandardLogger()))
	require.NoError(t, err)
	lmaClient = lmaelastic.NewWithClient(esClient)

	// Instantiate a client.
	cfg := rest.Config{
		CACertPath: "cert/RootCA.crt",
		URL:        "https://localhost:8444/",
	}
	cli, err = client.NewClient("", cfg)
	require.NoError(t, err)

	// Set up context with a timeout.
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)

	return func() {
		// Cleanup indices created by the test.
		testutils.CleanupIndices(context.Background(), esClient, fmt.Sprintf("%s.%s", index, cluster))
		logCancel()
		cancel()
	}
}

func TestFV_FlowIngestion(t *testing.T) {
	addr := "https://localhost:8444/api/v1/flows/logs/bulk"
	tenant := ""
	expectedResponse := `{"failed":0, "succeeded":25, "total":25}`
	indexPrefix := "tigera_secure_ee_flows."

	t.Run("ingest flow logs via bulk API with production data", func(t *testing.T) {
		// Create a random cluster name for each test to make sure we don't
		// interfere between tests.
		cluster = testutils.RandomClusterName()
		defer ingestionSetupAndTeardown(t, indexPrefix, cluster)()

		// setup HTTP httpClient and HTTP request
		httpClient := secureHTTPClient(t)
		spec := xndJSONPostHTTPReqSpec(addr, tenant, cluster, []byte(flowLogs))

		// make the request to ingest flows
		res, resBody := doRequest(t, httpClient, spec)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.JSONEq(t, expectedResponse, strings.Trim(string(resBody), "\n"))

		// Force a refresh in order to read the newly ingested data
		index := fmt.Sprintf("%s%s*", indexPrefix, cluster)
		_, err := esClient.Refresh(index).Do(ctx)
		require.NoError(t, err)

		params := v1.FlowLogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Unix(1675468688, 0),
					To:   time.Unix(1675469001, 0),
				},
			},
		}

		resultList, err := cli.FlowLogs(cluster).List(ctx, &params)
		require.NoError(t, err)
		require.NotNil(t, resultList)

		require.Equal(t, int64(25), resultList.TotalHits)

		var esLogs []string
		for _, log := range resultList.Items {
			logStr, err := json.Marshal(log)
			require.NoError(t, err)
			esLogs = append(esLogs, string(logStr))
		}

		assert.Equal(t, flowLogs, strings.Join(esLogs, "\n"))
	})
}

func TestFV_DNSIngestion(t *testing.T) {
	addr := "https://localhost:8444/api/v1/dns/logs/bulk"
	tenant := ""
	expectedResponse := `{"failed":0, "succeeded":10, "total":10}`
	indexPrefix := "tigera_secure_ee_dns."

	t.Run("ingest dns logs via bulk API with production data", func(t *testing.T) {
		// Create a random cluster name for each test to make sure we don't
		// interfere between tests.
		cluster = testutils.RandomClusterName()
		defer ingestionSetupAndTeardown(t, indexPrefix, cluster)()

		// setup HTTP httpClient and HTTP request
		httpClient := secureHTTPClient(t)
		spec := xndJSONPostHTTPReqSpec(addr, tenant, cluster, []byte(dnsLogs))

		// make the request to ingest flows
		res, resBody := doRequest(t, httpClient, spec)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.JSONEq(t, expectedResponse, strings.Trim(string(resBody), "\n"))

		// Force a refresh in order to read the newly ingested data
		index := fmt.Sprintf("%s%s*", indexPrefix, cluster)
		_, err := esClient.Refresh(index).Do(ctx)
		require.NoError(t, err)

		endTime, err := time.Parse(time.RFC3339Nano, "2023-02-10T01:17:11.310500008Z")
		require.NoError(t, err)
		startTime, err := time.Parse(time.RFC3339Nano, "2023-02-10T01:11:46.413467767Z")
		require.NoError(t, err)

		params := v1.DNSLogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: startTime,
					To:   endTime,
				},
			},
		}

		resultList, err := cli.DNSLogs(cluster).List(ctx, &params)
		require.NoError(t, err)
		require.NotNil(t, resultList)

		require.Equal(t, int64(10), resultList.TotalHits)

		var esLogs []string
		for _, log := range resultList.Items {
			logStr, err := json.Marshal(log)
			require.NoError(t, err)
			esLogs = append(esLogs, string(logStr))
		}

		assert.Equal(t, dnsLogs, strings.Join(esLogs, "\n"))
	})
}

func TestFV_L7Ingestion(t *testing.T) {
	addr := "https://localhost:8444/api/v1/l7/logs/bulk"
	tenant := ""
	expectedResponse := `{"failed":0, "succeeded":15, "total":15}`
	indexPrefix := "tigera_secure_ee_l7."

	t.Run("ingest l7 logs via bulk API with production data", func(t *testing.T) {
		// Create a random cluster name for each test to make sure we don't
		// interfere between tests.
		cluster = testutils.RandomClusterName()
		defer ingestionSetupAndTeardown(t, indexPrefix, cluster)()

		// setup HTTP httpClient and HTTP request
		httpClient := secureHTTPClient(t)
		spec := xndJSONPostHTTPReqSpec(addr, tenant, cluster, []byte(l7Logs))

		// make the request to ingest flows
		res, resBody := doRequest(t, httpClient, spec)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.JSONEq(t, expectedResponse, strings.Trim(string(resBody), "\n"))

		// Force a refresh in order to read the newly ingested data
		index := fmt.Sprintf("%s%s*", indexPrefix, cluster)
		_, err := esClient.Refresh(index).Do(ctx)
		require.NoError(t, err)

		params := v1.DNSLogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Unix(1676062496, 0),
					To:   time.Unix(1676067134, 0),
				},
			},
		}

		resultList, err := cli.L7Logs(cluster).List(ctx, &params)
		require.NoError(t, err)
		require.NotNil(t, resultList)

		require.Equal(t, int64(15), resultList.TotalHits)

		var esLogs []string
		for _, log := range resultList.Items {
			logStr, err := json.Marshal(log)
			require.NoError(t, err)
			esLogs = append(esLogs, string(logStr))
		}

		assert.Equal(t, l7Logs, strings.Join(esLogs, "\n"))
	})
}

func TestFV_KubeAuditIngestion(t *testing.T) {
	addr := "https://localhost:8444/api/v1/audit/logs/kube/bulk"
	tenant := ""
	expectedResponse := `{"failed":0, "succeeded":32, "total":32}`
	indexPrefix := "tigera_secure_ee_audit_kube."

	t.Run("ingest kube audit logs via bulk API with production data", func(t *testing.T) {
		// Create a random cluster name for each test to make sure we don't
		// interfere between tests.
		cluster = testutils.RandomClusterName()
		defer ingestionSetupAndTeardown(t, indexPrefix, cluster)()

		// setup HTTP httpClient and HTTP request
		httpClient := secureHTTPClient(t)
		spec := xndJSONPostHTTPReqSpec(addr, tenant, cluster, []byte(kubeAuditLogs))

		// make the request to ingest flows
		res, resBody := doRequest(t, httpClient, spec)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.JSONEq(t, expectedResponse, strings.Trim(string(resBody), "\n"))

		// Force a refresh in order to read the newly ingested data
		index := fmt.Sprintf("%s%s*", indexPrefix, cluster)
		_, err := esClient.Refresh(index).Do(ctx)
		require.NoError(t, err)

		startTime, err := time.Parse(time.RFC3339, "2023-02-10T01:15:20.855601Z")
		require.NoError(t, err)
		endTime, err := time.Parse(time.RFC3339, "2023-02-14T00:08:47.590948Z")
		require.NoError(t, err)
		params := v1.AuditLogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: startTime,
					To:   endTime,
				},
			},
			Type: v1.AuditLogTypeKube,
		}

		resultList, err := cli.AuditLogs(cluster).List(ctx, &params)
		require.NoError(t, err)
		require.NotNil(t, resultList)

		require.Equal(t, int64(32), resultList.TotalHits)

		var esLogs []string
		for _, log := range resultList.Items {
			logStr, err := log.MarshalJSON()
			require.NoError(t, err)
			esLogs = append(esLogs, string(logStr))
		}

		assert.Equal(t, kubeAuditLogs, strings.Join(esLogs, "\n"))
	})
}

func TestFV_EEAuditIngestion(t *testing.T) {
	addr := "https://localhost:8444/api/v1/audit/logs/ee/bulk"
	tenant := ""
	expectedResponse := `{"failed":0, "succeeded":35, "total":35}`
	indexPrefix := "tigera_secure_ee_audit_ee."

	t.Run("ingest ee audit logs via bulk API with production data", func(t *testing.T) {
		// Create a random cluster name for each test to make sure we don't
		// interfere between tests.
		cluster = testutils.RandomClusterName()
		defer ingestionSetupAndTeardown(t, indexPrefix, cluster)()

		// setup HTTP httpClient and HTTP request
		httpClient := secureHTTPClient(t)
		spec := xndJSONPostHTTPReqSpec(addr, tenant, cluster, []byte(eeAuditLogs))

		// make the request to ingest flows
		res, resBody := doRequest(t, httpClient, spec)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.JSONEq(t, expectedResponse, strings.Trim(string(resBody), "\n"))

		// Force a refresh in order to read the newly ingested data
		index := fmt.Sprintf("%s%s*", indexPrefix, cluster)
		_, err := esClient.Refresh(index).Do(ctx)
		require.NoError(t, err)

		startTime, err := time.Parse(time.RFC3339, "2023-02-10T21:40:58.476376Z")
		require.NoError(t, err)
		endTime, err := time.Parse(time.RFC3339, "2023-02-10T21:42:03.168059Z")
		require.NoError(t, err)
		params := v1.AuditLogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: startTime,
					To:   endTime,
				},
			},
			Type: v1.AuditLogTypeEE,
		}

		resultList, err := cli.AuditLogs(cluster).List(ctx, &params)
		require.NoError(t, err)
		require.NotNil(t, resultList)

		require.Equal(t, int64(35), resultList.TotalHits)

		var esLogs []string
		for _, log := range resultList.Items {
			logStr, err := log.MarshalJSON()
			require.NoError(t, err)
			esLogs = append(esLogs, string(logStr))
		}

		assert.Equal(t, eeAuditLogs, strings.Join(esLogs, "\n"))
	})
}
