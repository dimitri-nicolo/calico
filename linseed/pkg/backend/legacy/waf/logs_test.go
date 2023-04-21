// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package waf_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/json"
	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/templates"
	backendutils "github.com/projectcalico/calico/linseed/pkg/backend/testutils"
	"github.com/projectcalico/calico/linseed/pkg/config"

	gojson "encoding/json"

	"github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/require"

	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/waf"
	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

var (
	client  lmaelastic.Client
	b       bapi.WAFBackend
	ctx     context.Context
	cluster string
)

// setupTest runs common logic before each test, and also returns a function to perform teardown
// after each test.
func setupTest(t *testing.T) func() {
	// Hook logrus into testing.T
	config.ConfigureLogging("DEBUG")
	logCancel := logutils.RedirectLogrusToTestingT(t)

	// Create an elasticsearch client to use for the test. For this suite, we use a real
	// elasticsearch instance created via "make run-elastic".
	esClient, err := elastic.NewSimpleClient(elastic.SetURL("http://localhost:9200"), elastic.SetInfoLog(logrus.StandardLogger()))
	require.NoError(t, err)
	client = lmaelastic.NewWithClient(esClient)
	cache := templates.NewTemplateCache(client, 1, 0)

	// Instantiate a backend.
	b = waf.NewBackend(client, cache)

	// Create a random cluster name for each test to make sure we don't
	// interfere between tests.
	cluster = testutils.RandomClusterName()

	// Each test should take less than 60 seconds.
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)

	// Function contains teardown logic.
	return func() {
		err = testutils.CleanupIndices(context.Background(), esClient, cluster)
		require.NoError(t, err)

		// Cancel the context
		cancel()
		logCancel()
	}
}

// TestWAFLogBasic tests running a real elasticsearch query to create a kube waf log.
func TestWAFLogBasic(t *testing.T) {
	for _, tenant := range []string{backendutils.RandomTenantName(), ""} {
		name := fmt.Sprintf("TestCreateWAFLog (tenant=%s)", tenant)
		t.Run(name, func(t *testing.T) {
			defer setupTest(t)()

			clusterInfo := bapi.ClusterInfo{Cluster: cluster, Tenant: tenant}
			logTime := time.Now()
			f := v1.WAFLog{
				Timestamp: logTime,
				Source: &v1.WAFEndpoint{
					IP:       "1.2.3.4",
					PortNum:  789,
					Hostname: "source-hostname",
				},
				Destination: &v1.WAFEndpoint{
					IP:       "4.3.2.1",
					PortNum:  987,
					Hostname: "dest-hostname",
				},
				Path:     "/yellow/brick/road",
				Method:   "GET",
				Protocol: "TCP",
				Msg:      "This is a friendly reminder that nobody knows what is going on",
				RuleInfo: "WAF rules, rule WAF",
			}

			// Create the log in ES.
			resp, err := b.Create(ctx, clusterInfo, []v1.WAFLog{f})
			require.NoError(t, err)
			require.Equal(t, []v1.BulkError(nil), resp.Errors)
			require.Equal(t, 1, resp.Total)
			require.Equal(t, 0, resp.Failed)
			require.Equal(t, 1, resp.Succeeded)

			// Refresh the index.
			index := fmt.Sprintf("tigera_secure_ee_waf.%s.*", cluster)
			if tenant != "" {
				index = fmt.Sprintf("tigera_secure_ee_waf.%s.%s.*", tenant, cluster)
			}
			err = backendutils.RefreshIndex(ctx, client, index)
			require.NoError(t, err)

			params := &v1.WAFLogParams{
				QueryParams: v1.QueryParams{
					TimeRange: &lmav1.TimeRange{
						From: time.Now().Add(-60 * time.Second),
						To:   time.Now().Add(60 * time.Second),
					},
				},
			}

			results, err := b.List(ctx, clusterInfo, params)
			require.NoError(t, err)
			require.Equal(t, 1, len(results.Items))
			require.Equal(t, results.Items[0].Timestamp.Format(time.RFC3339), logTime.Format(time.RFC3339))

			// Timestamps don't equal on read.
			results.Items[0].Timestamp = f.Timestamp
			require.Equal(t, f, results.Items[0])

			// Read again using a dummy tenant - we should get nothing.
			results, err = b.List(ctx, bapi.ClusterInfo{Cluster: cluster, Tenant: "dummy"}, params)
			require.NoError(t, err)
			require.Equal(t, 0, len(results.Items))
		})
	}

	t.Run("no cluster name given on request", func(t *testing.T) {
		defer setupTest(t)()

		// It should reject requests with no cluster name given.
		clusterInfo := bapi.ClusterInfo{}
		_, err := b.Create(ctx, clusterInfo, []v1.WAFLog{})
		require.Error(t, err)

		params := &v1.WAFLogParams{}
		results, err := b.List(ctx, clusterInfo, params)
		require.Error(t, err)
		require.Nil(t, results)
	})

	t.Run("bad startFrom on request", func(t *testing.T) {
		defer setupTest(t)()

		clusterInfo := bapi.ClusterInfo{Cluster: cluster}
		params := &v1.WAFLogParams{
			QueryParams: v1.QueryParams{
				AfterKey: map[string]interface{}{"startFrom": "badvalue"},
			},
		}
		results, err := b.List(ctx, clusterInfo, params)
		require.Error(t, err)
		require.Nil(t, results)
	})
}

// TestAggregations tests running a real elasticsearch query to get aggregations.
func TestAggregations(t *testing.T) {
	// Run each testcase both as a multi-tenant scenario, as well as a single-tenant case.
	for _, tenant := range []string{backendutils.RandomTenantName(), ""} {
		t.Run(fmt.Sprintf("should return time-series WAFA log aggregation results (tenant=%s)", tenant), func(t *testing.T) {
			defer setupTest(t)()
			clusterInfo := bapi.ClusterInfo{Cluster: cluster, Tenant: tenant}

			numLogs := 5
			timeBetweenLogs := 10 * time.Second
			testStart := time.Unix(0, 0)
			now := testStart.Add(time.Duration(numLogs) * time.Minute)

			// Several dummy logs.
			logs := []v1.WAFLog{}
			start := testStart.Add(1 * time.Second)
			for i := 1; i < numLogs; i++ {
				log := v1.WAFLog{
					Timestamp: start,
					Source: &v1.WAFEndpoint{
						IP:       "1.2.3.4",
						PortNum:  789,
						Hostname: "source-hostname",
					},
					Destination: &v1.WAFEndpoint{
						IP:       "4.3.2.1",
						PortNum:  987,
						Hostname: "dest-hostname",
					},
					Path:     "/yellow/brick/road",
					Method:   "GET",
					Protocol: "TCP",
					Msg:      "This is a friendly reminder that nobody knows what is going on",
					RuleInfo: "WAF rules, rule WAF",
				}
				start = start.Add(timeBetweenLogs)
				logs = append(logs, log)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			resp, err := b.Create(ctx, clusterInfo, logs)
			require.NoError(t, err)
			require.Empty(t, resp.Errors)

			// Refresh.
			index := fmt.Sprintf("tigera_secure_ee_waf.%s.*", cluster)
			if tenant != "" {
				index = fmt.Sprintf("tigera_secure_ee_waf.%s.%s.*", tenant, cluster)
			}
			err = backendutils.RefreshIndex(ctx, client, index)
			require.NoError(t, err)

			params := v1.WAFLogAggregationParams{}
			params.TimeRange = &lmav1.TimeRange{}
			params.TimeRange.From = testStart
			params.TimeRange.To = now
			params.NumBuckets = 4

			// Add a simple aggregation to add up the total instances of each IP.
			agg := elastic.NewTermsAggregation().Field("source.ip")
			src, err := agg.Source()
			require.NoError(t, err)
			bytes, err := json.Marshal(src)
			require.NoError(t, err)
			params.Aggregations = map[string]gojson.RawMessage{"ips": bytes}

			// Use the backend to perform a query.
			aggs, err := b.Aggregations(ctx, clusterInfo, &params)
			require.NoError(t, err)
			require.NotNil(t, aggs)

			ts, ok := aggs.AutoDateHistogram("tb")
			require.True(t, ok)

			// We asked for 4 buckets.
			require.Len(t, ts.Buckets, 4)

			for i, b := range ts.Buckets {
				require.Equal(t, int64(1), b.DocCount, fmt.Sprintf("Bucket %d", i))

				// We asked for a ips agg, which should include a single log
				// in each bucket.
				ips, ok := b.ValueCount("ips")
				require.True(t, ok, "Bucket missing ips agg")
				buckets := string(ips.Aggregations["buckets"])
				require.Equal(t, `[{"key":"1.2.3.4","doc_count":1}]`, buckets)
			}
		})

		t.Run(fmt.Sprintf("should return aggregate stats (tenant=%s)", tenant), func(t *testing.T) {
			defer setupTest(t)()
			clusterInfo := bapi.ClusterInfo{Cluster: cluster, Tenant: tenant}

			// Start the test numLogs minutes in the past.
			numLogs := 5
			timeBetweenLogs := 10 * time.Second
			testStart := time.Unix(0, 0)
			now := testStart.Add(time.Duration(numLogs) * time.Minute)

			// Several dummy logs.
			logs := []v1.WAFLog{}
			start := testStart.Add(1 * time.Second)
			for i := 1; i < numLogs; i++ {
				log := v1.WAFLog{
					Timestamp: start,
					Source: &v1.WAFEndpoint{
						IP:       "1.2.3.4",
						PortNum:  789,
						Hostname: "source-hostname",
					},
					Destination: &v1.WAFEndpoint{
						IP:       "4.3.2.1",
						PortNum:  987,
						Hostname: "dest-hostname",
					},
					Path:     "/yellow/brick/road",
					Method:   "GET",
					Protocol: "TCP",
					Msg:      "This is a friendly reminder that nobody knows what is going on",
					RuleInfo: "WAF rules, rule WAF",
				}
				start = start.Add(timeBetweenLogs)
				logs = append(logs, log)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			resp, err := b.Create(ctx, clusterInfo, logs)
			require.NoError(t, err)
			require.Empty(t, resp.Errors)

			// Refresh.
			index := fmt.Sprintf("tigera_secure_ee_waf.%s.*", cluster)
			if tenant != "" {
				index = fmt.Sprintf("tigera_secure_ee_waf.%s.%s.*", tenant, cluster)
			}
			err = backendutils.RefreshIndex(ctx, client, index)
			require.NoError(t, err)

			params := v1.WAFLogAggregationParams{}
			params.TimeRange = &lmav1.TimeRange{}
			params.TimeRange.From = testStart
			params.TimeRange.To = now
			params.NumBuckets = 0 // Return aggregated stats over the whole time range.

			// Add a simple aggregation to count the instances of an IP.
			agg := elastic.NewTermsAggregation().Field("source.ip")
			src, err := agg.Source()
			require.NoError(t, err)
			bytes, err := json.Marshal(src)
			require.NoError(t, err)
			params.Aggregations = map[string]gojson.RawMessage{"ips": bytes}

			// Use the backend to perform a stats query.
			result, err := b.Aggregations(ctx, clusterInfo, &params)
			require.NoError(t, err)

			// We should get a sum aggregation with all 4 logs.
			ips, ok := result.ValueCount("ips")
			require.True(t, ok)
			buckets := string(ips.Aggregations["buckets"])
			require.Equal(t, `[{"key":"1.2.3.4","doc_count":4}]`, buckets)
		})
	}
}

func TestSorting(t *testing.T) {
	t.Run("should respect sorting", func(t *testing.T) {
		defer setupTest(t)()

		clusterInfo := bapi.ClusterInfo{Cluster: cluster}

		t1 := time.Unix(100, 0)
		t2 := time.Unix(500, 0)

		log1 := v1.WAFLog{
			Timestamp:   t1,
			Source:      &v1.WAFEndpoint{IP: "1.2.3.4", PortNum: 789, Hostname: "source-hostname"},
			Destination: &v1.WAFEndpoint{IP: "4.3.2.1", PortNum: 987, Hostname: "dest-hostname"},
			Path:        "/yellow/brick/road",
			Method:      "GET",
			Protocol:    "TCP",
			Msg:         "This is a friendly reminder that nobody knows what is going on",
			RuleInfo:    "WAF rules, rule WAF",
		}
		log2 := v1.WAFLog{
			Timestamp:   t2,
			Source:      &v1.WAFEndpoint{IP: "1.2.3.4", PortNum: 789, Hostname: "source-hostname"},
			Destination: &v1.WAFEndpoint{IP: "4.3.2.1", PortNum: 987, Hostname: "dest-hostname"},
			Path:        "/red/lobster",
			Method:      "PUT",
			Protocol:    "UDP",
			Msg:         "This is an unreasonable fear of failure",
			RuleInfo:    "Information cannot be found",
		}

		response, err := b.Create(ctx, clusterInfo, []v1.WAFLog{log1, log2})
		require.NoError(t, err)
		require.Equal(t, []v1.BulkError(nil), response.Errors)
		require.Equal(t, 0, response.Failed)

		index := fmt.Sprintf("tigera_secure_ee_waf.%s.*", cluster)
		err = backendutils.RefreshIndex(ctx, client, index)
		require.NoError(t, err)

		// Query for logs without sorting.
		params := v1.WAFLogParams{}
		r, err := b.List(ctx, clusterInfo, &params)
		require.NoError(t, err)
		require.Len(t, r.Items, 2)
		require.Nil(t, r.AfterKey)

		// Assert that the logs are returned in the correct order.
		require.Equal(t, log1, r.Items[0])
		require.Equal(t, log2, r.Items[1])

		// Query again, this time sorting in order to get the logs in reverse order.
		params.Sort = []v1.SearchRequestSortBy{
			{
				Field:      "@timestamp",
				Descending: true,
			},
		}
		r, err = b.List(ctx, clusterInfo, &params)
		require.NoError(t, err)
		require.Len(t, r.Items, 2)
		require.Nil(t, r.AfterKey)
		require.Equal(t, log2, r.Items[0])
		require.Equal(t, log1, r.Items[1])
	})
}
