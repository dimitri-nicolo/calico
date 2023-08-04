// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/projectcalico/calico/libcalico-go/lib/json"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"

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

func ingestionSetupAndTeardown(t *testing.T, index string) func() {
	// Hook logrus into testing.T
	config.ConfigureLogging("DEBUG")
	logCancel := logutils.RedirectLogrusToTestingT(t)

	cluster = testutils.RandomClusterName()

	// Create an ES client.
	var err error
	esClient, err = elastic.NewSimpleClient(elastic.SetURL("http://localhost:9200"), elastic.SetInfoLog(logrus.StandardLogger()))
	require.NoError(t, err)
	lmaClient = lmaelastic.NewWithClient(esClient)

	// Instantiate a client.
	cli, err = NewLinseedClient()
	require.NoError(t, err)

	// Get the token to use in HTTP authorization header.
	token, err = os.ReadFile(TokenPath)
	require.NoError(t, err)

	// Set up context with a timeout.
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)

	return func() {
		// Cleanup indices created by the test.
		testutils.CleanupIndices(context.Background(), esClient, cluster)
		logCancel()
		cancel()
	}
}

func TestFV_FlowIngestion(t *testing.T) {
	addr := "https://localhost:8444/api/v1/flows/logs/bulk"
	tenant := "tenant-a"
	expectedResponse := `{"failed":0, "succeeded":25, "total":25}`
	indexPrefix := "tigera_secure_ee_flows.tenant-a."

	t.Run("ingest flow logs via bulk API with production data", func(t *testing.T) {
		defer ingestionSetupAndTeardown(t, indexPrefix)()

		// setup HTTP httpClient and HTTP request
		httpClient := mTLSClient(t)
		spec := xndJSONPostHTTPReqSpec(addr, tenant, cluster, token, []byte(flowLogs))

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
			log = testutils.AssertFlowLogIDAndReset(t, log)
			logStr, err := json.Marshal(log)
			require.NoError(t, err)
			esLogs = append(esLogs, string(logStr))
		}

		assert.Equal(t, flowLogs, strings.Join(esLogs, "\n"))
	})
}

func TestFV_DNSIngestion(t *testing.T) {
	addr := "https://localhost:8444/api/v1/dns/logs/bulk"
	tenant := "tenant-a"
	expectedResponse := `{"failed":0, "succeeded":11, "total":11}`
	indexPrefix := "tigera_secure_ee_dns.tenant-a."

	t.Run("ingest dns logs via bulk API with production data", func(t *testing.T) {
		defer ingestionSetupAndTeardown(t, indexPrefix)()

		// setup HTTP httpClient and HTTP request
		httpClient := mTLSClient(t)
		spec := xndJSONPostHTTPReqSpec(addr, tenant, cluster, token, []byte(dnsLogs))

		// make the request to ingest flows
		res, resBody := doRequest(t, httpClient, spec)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.JSONEq(t, expectedResponse, strings.Trim(string(resBody), "\n"))

		// Force a refresh in order to read the newly ingested data
		index := fmt.Sprintf("%s%s*", indexPrefix, cluster)
		_, err := esClient.Refresh(index).Do(ctx)
		require.NoError(t, err)

		endTime, err := time.Parse(time.RFC3339Nano, "2023-02-22T23:54:02.736970074Z")
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

		require.Equal(t, int64(11), resultList.TotalHits)

		var esLogs []string
		for _, log := range resultList.Items {
			log = testutils.AssertDNSLogIDAndReset(t, log)
			logStr, err := json.Marshal(log)
			require.NoError(t, err)
			esLogs = append(esLogs, string(logStr))
		}

		assert.Equal(t, dnsLogs, strings.Join(esLogs, "\n"))
	})
}

func TestFV_L7Ingestion(t *testing.T) {
	addr := "https://localhost:8444/api/v1/l7/logs/bulk"
	tenant := "tenant-a"
	expectedResponse := `{"failed":0, "succeeded":15, "total":15}`
	indexPrefix := "tigera_secure_ee_l7.tenant-a."

	t.Run("ingest l7 logs via bulk API with production data", func(t *testing.T) {
		defer ingestionSetupAndTeardown(t, indexPrefix)()

		// setup HTTP httpClient and HTTP request
		httpClient := mTLSClient(t)
		spec := xndJSONPostHTTPReqSpec(addr, tenant, cluster, token, []byte(l7Logs))

		// make the request to ingest flows
		res, resBody := doRequest(t, httpClient, spec)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.JSONEq(t, expectedResponse, strings.Trim(string(resBody), "\n"))

		// Force a refresh in order to read the newly ingested data
		index := fmt.Sprintf("%s%s*", indexPrefix, cluster)
		_, err := esClient.Refresh(index).Do(ctx)
		require.NoError(t, err)

		params := v1.L7LogParams{
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
	tenant := "tenant-a"
	expectedResponse := `{"failed":0, "succeeded":32, "total":32}`
	indexPrefix := "tigera_secure_ee_audit_kube.tenant-a."

	t.Run("ingest kube audit logs via bulk API with production data", func(t *testing.T) {
		defer ingestionSetupAndTeardown(t, indexPrefix)()

		// setup HTTP httpClient and HTTP request
		httpClient := mTLSClient(t)
		spec := xndJSONPostHTTPReqSpec(addr, tenant, cluster, token, []byte(kubeAuditLogs))

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
		require.Equal(t, 32, len(resultList.Items))
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
	tenant := "tenant-a"
	expectedResponse := `{"failed":0, "succeeded":35, "total":35}`
	indexPrefix := "tigera_secure_ee_audit_ee.tenant-a."

	t.Run("ingest ee audit logs via bulk API with production data", func(t *testing.T) {
		defer ingestionSetupAndTeardown(t, indexPrefix)()

		// setup HTTP httpClient and HTTP request
		httpClient := mTLSClient(t)
		spec := xndJSONPostHTTPReqSpec(addr, tenant, cluster, token, []byte(eeAuditLogs))

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

func TestFV_BGPIngestion(t *testing.T) {
	addr := "https://localhost:8444/api/v1/bgp/logs/bulk"
	tenant := "tenant-a"
	expectedResponse := `{"failed":0, "succeeded":4, "total":4}`
	indexPrefix := "tigera_secure_ee_bgp.tenant-a."

	t.Run("ingest bgp logs via bulk API with production data", func(t *testing.T) {
		defer ingestionSetupAndTeardown(t, indexPrefix)()

		// setup HTTP httpClient and HTTP request
		httpClient := mTLSClient(t)
		spec := xndJSONPostHTTPReqSpec(addr, tenant, cluster, token, []byte(bgpLogs))

		// make the request to ingest flows
		res, resBody := doRequest(t, httpClient, spec)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.JSONEq(t, expectedResponse, strings.Trim(string(resBody), "\n"))

		// Force a refresh in order to read the newly ingested data
		index := fmt.Sprintf("%s%s*", indexPrefix, cluster)
		_, err := esClient.Refresh(index).Do(ctx)
		require.NoError(t, err)

		startTime, err := time.Parse(v1.BGPLogTimeFormat, "2023-02-23T00:10:46")
		require.NoError(t, err)
		endTime, err := time.Parse(v1.BGPLogTimeFormat, "2023-02-23T00:15:46")
		require.NoError(t, err)
		params := v1.BGPLogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: startTime,
					To:   endTime,
				},
			},
		}

		resultList, err := cli.BGPLogs(cluster).List(ctx, &params)
		require.NoError(t, err)
		require.NotNil(t, resultList)

		require.Equal(t, int64(4), resultList.TotalHits)

		var esLogs []string
		for _, log := range resultList.Items {
			buffer := &bytes.Buffer{}
			encoder := json.NewEncoder(buffer)
			encoder.SetEscapeHTML(false)
			err := encoder.Encode(log)
			require.NoError(t, err)
			esLogs = append(esLogs, strings.Trim(string(buffer.Bytes()), "\n"))
		}

		assert.Equal(t, bgpLogs, strings.Join(esLogs, "\n"))
	})
}

func TestFV_WAFIngestion(t *testing.T) {
	addr := "https://localhost:8444/api/v1/waf/logs/bulk"
	tenant := "tenant-a"
	expectedResponse := `{"failed":0, "succeeded":2, "total":2}`
	indexPrefix := "tigera_secure_ee_waf.tenant-a."

	t.Run("ingest waf logs via bulk API with production data", func(t *testing.T) {
		defer ingestionSetupAndTeardown(t, indexPrefix)()

		// setup HTTP httpClient and HTTP request
		httpClient := mTLSClient(t)
		spec := xndJSONPostHTTPReqSpec(addr, tenant, cluster, token, []byte(wafLogs))

		// make the request to ingest flows
		res, resBody := doRequest(t, httpClient, spec)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.JSONEq(t, expectedResponse, strings.Trim(string(resBody), "\n"))

		// Force a refresh in order to read the newly ingested data
		index := fmt.Sprintf("%s%s*", indexPrefix, cluster)
		_, err := esClient.Refresh(index).Do(ctx)
		require.NoError(t, err)

		endTime, err := time.Parse(time.RFC3339Nano, "2023-06-22T23:59:59.999999999Z")
		require.NoError(t, err)
		startTime, err := time.Parse(time.RFC3339Nano, "2022-02-11T00:00:00.000000000Z")
		require.NoError(t, err)

		params := v1.WAFLogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: startTime,
					To:   endTime,
				},
			},
		}

		resultList, err := cli.WAFLogs(cluster).List(ctx, &params)
		require.NoError(t, err)
		require.NotNil(t, resultList)

		require.Equal(t, int64(2), resultList.TotalHits)

		var esLogs []string
		for _, log := range resultList.Items {
			logStr, err := json.Marshal(log)
			require.NoError(t, err)
			esLogs = append(esLogs, string(logStr))
		}

		assert.Equal(t, wafLogs, strings.Join(esLogs, "\n"))
	})
}

func TestFV_RuntimeIngestion(t *testing.T) {
	addr := "https://localhost:8444/api/v1/runtime/reports/bulk"
	tenant := "tenant-a"
	expectedResponse := `{"failed":0, "succeeded":29, "total":29}`
	indexPrefix := "tigera_secure_ee_runtime.tenant-a."

	t.Run("ingest runtime reports via bulk API with production data", func(t *testing.T) {
		defer ingestionSetupAndTeardown(t, indexPrefix)()

		// setup HTTP httpClient and HTTP request
		httpClient := mTLSClient(t)
		spec := xndJSONPostHTTPReqSpec(addr, tenant, cluster, token, []byte(runtimeReports))

		// make the request to ingest runtime reports
		res, resBody := doRequest(t, httpClient, spec)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.JSONEq(t, expectedResponse, strings.Trim(string(resBody), "\n"))

		// Force a refresh in order to read the newly ingested data
		index := fmt.Sprintf("%s%s*", indexPrefix, cluster)
		_, err := esClient.Refresh(index).Do(ctx)
		require.NoError(t, err)

		endTime, err := time.Parse(time.RFC3339Nano, "2023-03-14T01:40:59.401474246Z")
		require.NoError(t, err)
		startTime, err := time.Parse(time.RFC3339Nano, "2023-03-14T01:39:41.654053441Z")
		require.NoError(t, err)

		params := v1.RuntimeReportParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: startTime,
					To:   endTime,
				},
			},
			LegacyTimeRange: &lmav1.TimeRange{
				From: startTime,
				To:   endTime,
			},
		}

		resultList, err := cli.RuntimeReports("").List(ctx, &params)
		require.NoError(t, err)
		require.NotNil(t, resultList)

		require.Equal(t, int64(29), resultList.TotalHits)

		var esLogs []string
		for _, log := range resultList.Items {
			// Null out the GeneratedTime field.  Linseed will have populated this - and
			// the following line verifies that - but we can't predict the exact value
			// and hence our runtimeReports fixture does not include it.
			require.NotNil(t, log.Report.GeneratedTime)
			log.Report.GeneratedTime = nil
			logStr, err := json.Marshal(log.Report)
			require.NoError(t, err)
			esLogs = append(esLogs, string(logStr))
		}

		assert.Equal(t, runtimeReports, strings.Join(esLogs, "\n"))
	})
}

func TestFV_Ingestion(t *testing.T) {
	addr := "https://localhost:8444/api/v1/audit/logs/ee/bulk"
	tenant := "tenant-a"
	expectedResponse := `{"Msg":"http: request body too large", "Status":400}`
	indexPrefix := "tigera_secure_ee_audit_ee.tenant-a."

	t.Run("cannot ingest arequest bigger than 2Gb", func(t *testing.T) {
		t.Skip()

		defer ingestionSetupAndTeardown(t, indexPrefix)()

		// setup HTTP httpClient and HTTP request
		httpClient := mTLSClient(t)
		var largeBody []byte
		for float64(len(largeBody)) < 2*math.Pow(1024, 3)+10 {
			largeBody = append(largeBody, []byte(eeAuditLogs)...)
		}

		spec := xndJSONPostHTTPReqSpec(addr, tenant, cluster, token, largeBody)

		// make the request to ingest flows
		res, resBody := doRequest(t, httpClient, spec)
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		assert.JSONEq(t, expectedResponse, strings.Trim(string(resBody), "\n"))
	})
}

func TestFV_AnomalyDetectionEventsIngestion(t *testing.T) {
	addr := "https://localhost:8444/api/v1/events/bulk"
	tenant := "tenant-a"
	expectedResponse := `{"failed":0, "succeeded":1, "total":1}`
	indexPrefix := "tigera_secure_ee_events.tenant-a."

	t.Run("ingest anomaly detection events via bulk API with production data", func(t *testing.T) {
		defer ingestionSetupAndTeardown(t, indexPrefix)()

		spec := xndJSONPostHTTPReqSpec(addr, tenant, cluster, token, []byte(anomalyDetectionEvent))
		httpClient := mTLSClient(t)

		// make the request to ingest anomaly detection alerts
		res, resBody := doRequest(t, httpClient, spec)
		require.Equal(t, http.StatusOK, res.StatusCode)
		require.JSONEq(t, expectedResponse, strings.Trim(string(resBody), "\n"))

		// Force a refresh in order to read the newly ingested data
		index := fmt.Sprintf("%s%s*", indexPrefix, cluster)
		_, err := esClient.Refresh(index).Do(ctx)
		require.NoError(t, err)

		endTime, err := time.Parse(time.RFC3339, "2023-04-28T19:38:14+00:00")
		require.NoError(t, err)
		startTime, err := time.Parse(time.RFC3339, "2023-04-28T19:37:14+00:00")
		require.NoError(t, err)

		params := v1.EventParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: startTime,
					To:   endTime,
				},
			},
		}

		resultList, err := cli.Events(cluster).List(ctx, &params)
		require.NoError(t, err)
		require.NotNil(t, resultList)

		require.Equal(t, int64(1), resultList.TotalHits)
		expectedEvent := v1.Event{}
		err = json.Unmarshal([]byte(anomalyDetectionEvent), &expectedEvent)
		require.NoError(t, err)
		assert.Equal(t, []v1.Event{expectedEvent}, testutils.AssertLogIDAndCopyEventsWithoutID(t, resultList))
	})
}
