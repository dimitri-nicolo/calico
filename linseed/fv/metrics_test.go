// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/projectcalico/calico/libcalico-go/lib/json"

	"github.com/stretchr/testify/assert"

	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"

	"github.com/google/gopacket/layers"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"

	elastic "github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	"github.com/projectcalico/calico/linseed/pkg/config"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

func metricsSetupAndTeardown(t *testing.T) func() {
	// Hook logrus into testing.T
	config.ConfigureLogging("DEBUG")
	logCancel := logutils.RedirectLogrusToTestingT(t)

	// Create an ES client.
	esClient, err := elastic.NewSimpleClient(elastic.SetURL("http://localhost:9200"), elastic.SetInfoLog(logrus.StandardLogger()))
	require.NoError(t, err)
	lmaClient = lmaelastic.NewWithClient(esClient)

	// Instantiate a client.
	cli, err = NewLinseedClient()
	require.NoError(t, err)

	// Get the token to use in HTTP authorization header.
	token, err = os.ReadFile(TokenPath)
	require.NoError(t, err)

	// Create a random cluster name for each test to make sure we don't
	// interfere between tests.
	cluster = testutils.RandomClusterName()

	// Set up context with a timeout.
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)

	return func() {
		// Cleanup indices created by the test.
		testutils.CleanupIndices(context.Background(), esClient, cluster)
		logCancel()
		cancel()
	}
}

func TestMetrics(t *testing.T) {
	metricsAddr := "localhost:9095"

	t.Run("should provide a metrics endpoint", func(t *testing.T) {
		defer metricsSetupAndTeardown(t)()

		client := mTLSClient(t)
		httpReqSpec := noBodyHTTPReqSpec("GET", fmt.Sprintf("https://%s/metrics", metricsAddr), "", "", token)
		res, _ := doRequest(t, client, httpReqSpec)
		assert.Equal(t, http.StatusOK, res.StatusCode)
	})

	t.Run("should create metrics based on the requests made", func(t *testing.T) {
		defer metricsSetupAndTeardown(t)()
		// Create a basic dns log.
		logs := []v1.DNSLog{
			{
				EndTime: time.Now().UTC(),
				QName:   "service.namespace.svc.cluster.local",
				QClass:  v1.DNSClass(layers.DNSClassIN),
				QType:   v1.DNSType(layers.DNSTypeAAAA),
				RCode:   v1.DNSResponseCode(layers.DNSResponseCodeNXDomain),
				RRSets:  v1.DNSRRSets{},
			},
		}
		bulk, err := cli.DNSLogs(cluster).Create(ctx, logs)
		require.NoError(t, err)
		require.Equal(t, bulk.Succeeded, 1, "create dns log did not succeed")

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_dns*")

		// Read it back.
		params := v1.DNSLogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now().Add(5 * time.Second),
				},
			},
		}
		resp, err := cli.DNSLogs(cluster).List(ctx, &params)
		require.NoError(t, err)
		actualLogs := testutils.AssertLogIDAndCopyDNSLogsWithoutID(t, resp)
		require.Equal(t, logs, actualLogs)

		client := mTLSClient(t)
		httpReqSpec := noBodyHTTPReqSpec("GET", fmt.Sprintf("https://%s/metrics", metricsAddr), "", "", token)
		res, body := doRequest(t, client, httpReqSpec)
		assert.Equal(t, http.StatusOK, res.StatusCode)

		// Check application metrics used for billing
		bytesWritten, err := json.Marshal(logs)
		require.NoError(t, err)
		bytesRead, err := json.Marshal(params)
		require.NoError(t, err)
		bytesReadMetric := fmt.Sprintf(`tigera_linseed_bytes_read{cluster_id="%s",tenant_id=""} %d`, cluster, len(bytesRead))
		bytesWrittenMetric := fmt.Sprintf(`tigera_linseed_bytes_written{cluster_id="%s",tenant_id=""}`, cluster)
		require.Contains(t, string(body), bytesReadMetric, fmt.Sprintf("missing %s from %s", bytesReadMetric, string(body)))
		require.Contains(t, string(body), bytesWrittenMetric, fmt.Sprintf("missing %s from %s", bytesWrittenMetric, string(body)))

		metric := regexp.MustCompile(fmt.Sprintf("%s [\\d]+", bytesWrittenMetric)).Find(body)
		value, err := strconv.Atoi(string(regexp.MustCompile("[1-9][0-9]*").Find(metric)))
		require.NoError(t, err)

		require.InDeltaf(t, len(bytesWritten), value, 3, fmt.Sprintf("expecting %d to be in range of %d", len(bytesWritten), value))
	})
}
