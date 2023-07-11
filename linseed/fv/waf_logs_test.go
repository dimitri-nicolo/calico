// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/projectcalico/calico/linseed/pkg/client"

	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"

	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"

	elastic "github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/config"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

func wafSetupAndTeardown(t *testing.T) func() {
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

func TestFV_WAF(t *testing.T) {
	t.Run("should return an empty list if there are no WAF logs", func(t *testing.T) {
		defer wafSetupAndTeardown(t)()

		params := v1.WAFLogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now(),
				},
			},
		}

		// Perform a query.
		wafLogs, err := cli.WAFLogs(cluster).List(ctx, &params)
		require.NoError(t, err)
		require.Equal(t, []v1.WAFLog{}, wafLogs.Items)
	})

	t.Run("should create and list waf logs", func(t *testing.T) {
		defer wafSetupAndTeardown(t)()

		reqTime := time.Now()
		// Create a basic waf log
		wafLogs := []v1.WAFLog{
			{
				Timestamp: reqTime,
				Msg:       "any message",
			},
		}
		bulk, err := cli.WAFLogs(cluster).Create(ctx, wafLogs)
		require.NoError(t, err)
		require.Equal(t, bulk.Succeeded, 1, "create waf logs did not succeed")

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_waf*")

		// Read it back.
		params := v1.WAFLogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now().Add(5 * time.Second),
				},
			},
		}
		resp, err := cli.WAFLogs(cluster).List(ctx, &params)
		require.NoError(t, err)

		require.Len(t, resp.Items, 1)
		// Reset the time as it microseconds to not match perfectly
		require.NotEqual(t, "", resp.Items[0].Timestamp)
		resp.Items[0].Timestamp = reqTime

		require.Equal(t, wafLogs, resp.Items)
	})

	t.Run("should support pagination", func(t *testing.T) {
		defer wafSetupAndTeardown(t)()

		totalItems := 5

		// Create 5 waf logs.
		logTime := time.Unix(0, 0).UTC()
		for i := 0; i < totalItems; i++ {
			logs := []v1.WAFLog{
				{
					Timestamp: logTime.Add(time.Duration(i) * time.Second), // Make sure logs are ordered.
					Host:      fmt.Sprintf("%d", i),
				},
			}
			bulk, err := cli.WAFLogs(cluster).Create(ctx, logs)
			require.NoError(t, err)
			require.Equal(t, bulk.Succeeded, 1, "create WAF log did not succeed")
		}

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_waf*")

		// Iterate through the first 4 pages and check they are correct.
		var afterKey map[string]interface{}
		for i := 0; i < totalItems-1; i++ {
			params := v1.WAFLogParams{
				QueryParams: v1.QueryParams{
					TimeRange: &lmav1.TimeRange{
						From: time.Unix(0, 0).UTC().Add(-5 * time.Second),
						To:   time.Unix(0, 0).UTC().Add(5 * time.Second),
					},
					MaxPageSize: 1,
					AfterKey:    afterKey,
				},
			}
			resp, err := cli.WAFLogs(cluster).List(ctx, &params)
			require.NoError(t, err)
			require.Equal(t, 1, len(resp.Items))
			require.Equal(t, []v1.WAFLog{
				{
					Timestamp: logTime.Add(time.Duration(i) * time.Second),
					Host:      fmt.Sprintf("%d", i),
				},
			}, resp.Items, fmt.Sprintf("WAF #%d did not match", i))
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
		params := v1.WAFLogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Unix(0, 0).UTC().Add(-5 * time.Second),
					To:   time.Unix(0, 0).UTC().Add(5 * time.Second),
				},
				MaxPageSize: 1,
				AfterKey:    afterKey,
			},
		}
		resp, err := cli.WAFLogs(cluster).List(ctx, &params)
		require.NoError(t, err)
		require.Equal(t, 1, len(resp.Items))
		require.Equal(t, []v1.WAFLog{
			{
				Timestamp: logTime.Add(time.Duration(lastItem) * time.Second),
				Host:      fmt.Sprintf("%d", lastItem),
			},
		}, resp.Items, fmt.Sprintf("WAF #%d did not match", lastItem))
		require.Equal(t, resp.TotalHits, int64(totalItems))

		// Once we reach the end of the data, we should not receive
		// an afterKey
		require.Nil(t, resp.AfterKey)
	})

	t.Run("should support pagination for items >= 10000 for WAF logs", func(t *testing.T) {
		defer wafSetupAndTeardown(t)()

		totalItems := 10001
		// Create > 10K threat logs.
		logTime := time.Unix(0, 0).UTC()
		var logs []v1.WAFLog
		for i := 0; i < totalItems; i++ {
			logs = append(logs, v1.WAFLog{

				Timestamp: logTime.Add(time.Duration(i) * time.Second), // Make sure logs are ordered.
				Host:      fmt.Sprintf("%d", i),
			},
			)
		}
		bulk, err := cli.WAFLogs(cluster).Create(ctx, logs)
		require.NoError(t, err)
		require.Equal(t, totalItems, bulk.Total, "create logs did not succeed")

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_waf*")

		// Stream through all the items.
		params := v1.WAFLogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: logTime.Add(-5 * time.Second),
					To:   logTime.Add(time.Duration(totalItems) * time.Second),
				},
				MaxPageSize: 1000,
			},
		}

		pager := client.NewListPager[v1.WAFLog](&params)
		pages, errors := pager.Stream(ctx, cli.WAFLogs(cluster).List)

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
