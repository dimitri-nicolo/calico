// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/projectcalico/calico/linseed/pkg/client"

	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"

	elastic "github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	"github.com/projectcalico/calico/linseed/pkg/config"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

func l7logSetupAndTeardown(t *testing.T) func() {
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

func TestL7_FlowLogs(t *testing.T) {
	t.Run("should return an empty list if there are no l7 logs", func(t *testing.T) {
		defer l7logSetupAndTeardown(t)()

		params := v1.L7LogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now(),
				},
			},
		}

		// Perform a query.
		logs, err := cli.L7Logs(cluster).List(ctx, &params)
		require.NoError(t, err)
		require.Equal(t, []v1.L7Log{}, logs.Items)
	})

	t.Run("should create and list l7 logs", func(t *testing.T) {
		defer l7logSetupAndTeardown(t)()

		// Create a basic flow log.
		logs := []v1.L7Log{
			{
				EndTime:      time.Now().Unix(), // TODO: Add more fields
				ResponseCode: "200",
			},
		}
		bulk, err := cli.L7Logs(cluster).Create(ctx, logs)
		require.NoError(t, err)
		require.Equal(t, bulk.Succeeded, 1, "create l7 log did not succeed")

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_l7*")

		// Read it back.
		params := v1.L7LogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now().Add(5 * time.Second),
				},
			},
		}
		resp, err := cli.L7Logs(cluster).List(ctx, &params)
		require.NoError(t, err)
		require.Equal(t, logs, resp.Items)
	})

	t.Run("should return an empty aggregations if there are no l7 logs", func(t *testing.T) {
		defer l7logSetupAndTeardown(t)()

		params := v1.L7AggregationParams{
			L7LogParams: v1.L7LogParams{
				QueryParams: v1.QueryParams{
					TimeRange: &lmav1.TimeRange{
						From: time.Now().Add(-5 * time.Second),
						To:   time.Now(),
					},
				},
			},
			Aggregations: map[string]json.RawMessage{
				"response_code": []byte(`{"filters":{"other_bucket_key":"other","filters":{"1xx":{"prefix":{"response_code":"1"}},"2xx":{"prefix":{"response_code":"2"}},"3xx":{"prefix":{"response_code":"3"}},"4xx":{"prefix":{"response_code":"4"}},"5xx":{"prefix":{"response_code":"5"}}}},"aggs":{"myDurationMeanHistogram":{"date_histogram":{"field":"start_time","fixed_interval":"60s"},"aggs":{"myDurationMeanAvg":{"avg":{"field":"duration_mean"}}}}}}`),
			},
			NumBuckets: 3,
		}

		// Perform a query.
		aggregations, err := cli.L7Logs(cluster).Aggregations(ctx, &params)
		require.NoError(t, err)
		require.Nil(t, aggregations)
	})

	t.Run("should support pagination", func(t *testing.T) {
		defer l7logSetupAndTeardown(t)()

		totalItems := 5

		// Create 5 flow logs.
		logTime := time.Now().UTC().Unix()
		for i := 0; i < totalItems; i++ {
			logs := []v1.L7Log{
				{
					StartTime: logTime,
					EndTime:   logTime + int64(i), // Make sure logs are ordered.
					Host:      fmt.Sprintf("%d", i),
				},
			}
			bulk, err := cli.L7Logs(cluster).Create(ctx, logs)
			require.NoError(t, err)
			require.Equal(t, bulk.Succeeded, 1, "create L7 log did not succeed")
		}

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_l7*")

		// Iterate through the first 4 pages and check they are correct.
		var afterKey map[string]interface{}
		for i := 0; i < totalItems-1; i++ {
			params := v1.L7LogParams{
				QueryParams: v1.QueryParams{
					TimeRange: &lmav1.TimeRange{
						From: time.Now().Add(-5 * time.Second),
						To:   time.Now().Add(5 * time.Second),
					},
					MaxPageSize: 1,
					AfterKey:    afterKey,
				},
			}
			resp, err := cli.L7Logs(cluster).List(ctx, &params)
			require.NoError(t, err)
			require.Equal(t, 1, len(resp.Items))
			require.Equal(t, []v1.L7Log{
				{
					StartTime: logTime,
					EndTime:   logTime + int64(i),
					Host:      fmt.Sprintf("%d", i),
				},
			}, resp.Items, fmt.Sprintf("L7 #%d did not match", i))
			require.NotNil(t, resp.AfterKey)
			require.Contains(t, resp.AfterKey, "startFrom")
			require.Equal(t, resp.AfterKey["startFrom"], float64(i+1))
			require.Equal(t, resp.TotalHits, int64(totalItems))

			// Use the afterKey for the next query.
			afterKey = resp.AfterKey
		}

		// If we query once more, we should get the last page, and no afterkey, since
		// we have paged through all the items.
		lastItem := totalItems - 1
		params := v1.L7LogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now().Add(5 * time.Second),
				},
				MaxPageSize: 1,
				AfterKey:    afterKey,
			},
		}
		resp, err := cli.L7Logs(cluster).List(ctx, &params)
		require.NoError(t, err)
		require.Equal(t, 1, len(resp.Items))
		require.Equal(t, []v1.L7Log{
			{
				StartTime: logTime,
				EndTime:   logTime + int64(lastItem),
				Host:      fmt.Sprintf("%d", lastItem),
			},
		}, resp.Items, fmt.Sprintf("L7 #%d did not match", lastItem))
		require.Equal(t, resp.TotalHits, int64(totalItems))

		// Once we reach the end of the data, we should not receive
		// an afterKey
		require.Nil(t, resp.AfterKey)
	})

	t.Run("should support pagination for items >= 10000 for l7 logs", func(t *testing.T) {
		defer l7logSetupAndTeardown(t)()

		totalItems := 10001
		// Create > 10K logs.
		logTime := time.Now().UTC().Unix()
		var logs []v1.L7Log
		for i := 0; i < totalItems; i++ {
			logs = append(logs, v1.L7Log{
				StartTime: logTime,
				EndTime:   logTime + int64(i), // Make sure logs are ordered.
				Host:      fmt.Sprintf("%d", i),
			},
			)
		}
		bulk, err := cli.L7Logs(cluster).Create(ctx, logs)
		require.NoError(t, err)
		require.Equal(t, totalItems, bulk.Total, "create logs did not succeed")

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_l7*")

		// Stream through all the items.
		params := v1.L7LogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now().Add(time.Duration(totalItems) * time.Second),
				},
				MaxPageSize: 1000,
			},
		}

		pager := client.NewListPager[v1.L7Log](&params)
		pages, errors := pager.Stream(ctx, cli.L7Logs(cluster).List)

		receivedItems := 0
		for page := range pages {
			receivedItems = receivedItems + len(page.Items)
		}

		if err, ok := <-errors; ok {
			require.NoError(t, err)
		}

		require.Equal(t, receivedItems, totalItems)
	})

}

func TestFV_L7LogsTenancy(t *testing.T) {
	t.Run("should support tenancy restriction", func(t *testing.T) {
		defer l7logSetupAndTeardown(t)()

		// Instantiate a client for an unexpected tenant.
		tenantCLI, err := NewLinseedClientForTenant("bad-tenant")
		require.NoError(t, err)

		// Create a basic log. We expect this to fail, since we're using
		// an unexpected tenant ID on the request.
		logs := []v1.L7Log{
			{
				EndTime:      time.Now().Unix(), // TODO: Add more fields
				ResponseCode: "200",
			},
		}
		bulk, err := tenantCLI.L7Logs(cluster).Create(ctx, logs)
		require.ErrorContains(t, err, "Bad tenant identifier")
		require.Nil(t, bulk)

		// Try a read as well.
		params := v1.L7LogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now().Add(5 * time.Second),
				},
			},
		}
		resp, err := tenantCLI.L7Logs(cluster).List(ctx, &params)
		require.ErrorContains(t, err, "Bad tenant identifier")
		require.Nil(t, resp)
	})
}
