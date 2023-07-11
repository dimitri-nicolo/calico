// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/projectcalico/calico/linseed/pkg/client"

	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"

	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"
	"github.com/projectcalico/calico/linseed/pkg/config"

	"github.com/stretchr/testify/require"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
)

func threatFeedsSetupAndTeardown(t *testing.T) func() {
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
		err := testutils.CleanupIndices(context.Background(), esClient, cluster)
		require.NoError(t, err)
		logCancel()
		cancel()
	}
}

func TestFV_ThreatFeedsIPSet(t *testing.T) {
	t.Run("should return an empty list if there are no threat feeds", func(t *testing.T) {
		defer threatFeedsSetupAndTeardown(t)()

		params := v1.IPSetThreatFeedParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now(),
				},
			},
		}

		// Perform a query.
		feeds, err := cli.ThreatFeeds(cluster).IPSet().List(ctx, &params)
		require.NoError(t, err)
		require.Equal(t, []v1.IPSetThreatFeed{}, feeds.Items)
	})

	t.Run("should create and list threat feeds", func(t *testing.T) {
		defer threatFeedsSetupAndTeardown(t)()

		feeds := v1.IPSetThreatFeed{
			ID: "feed-a",
			Data: &v1.IPSetThreatFeedData{
				CreatedAt: time.Unix(0, 0).UTC(),
				IPs:       []string{"1.2.3.4/32"},
			},
		}
		bulk, err := cli.ThreatFeeds(cluster).IPSet().Create(ctx, []v1.IPSetThreatFeed{feeds})
		require.NoError(t, err)
		require.Equal(t, bulk.Succeeded, 1, "create did not succeed")

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_threatfeeds_ipset*")

		// Read it back, passing an ID query.
		params := v1.IPSetThreatFeedParams{ID: "feed-a"}
		resp, err := cli.ThreatFeeds(cluster).IPSet().List(ctx, &params)
		require.NoError(t, err)

		// The ID should be set.
		require.Len(t, resp.Items, 1)
		require.Equal(t, feeds.ID, resp.Items[0].ID)
		require.Equal(t, feeds, resp.Items[0])

		// Delete the feed
		bulkDelete, err := cli.ThreatFeeds(cluster).IPSet().Delete(ctx, []v1.IPSetThreatFeed{feeds})
		require.NoError(t, err)
		require.Equal(t, bulkDelete.Succeeded, 1, "delete did not succeed")

		// Read after delete
		afterDelete, err := cli.ThreatFeeds(cluster).IPSet().List(ctx, &params)
		require.NoError(t, err)
		require.Empty(t, afterDelete.Items)
	})

	t.Run("should support pagination", func(t *testing.T) {
		defer threatFeedsSetupAndTeardown(t)()

		totalItems := 5

		// Create 5 Feeds.
		createdAtTime := time.Unix(0, 0).UTC()
		for i := 0; i < totalItems; i++ {
			feeds := []v1.IPSetThreatFeed{
				{
					ID: strconv.Itoa(i),
					Data: &v1.IPSetThreatFeedData{
						CreatedAt: createdAtTime.Add(time.Duration(i) * time.Second),
					},
				},
			}
			bulk, err := cli.ThreatFeeds(cluster).IPSet().Create(ctx, feeds)
			require.NoError(t, err)
			require.Equal(t, bulk.Succeeded, 1, "create feeds did not succeed")
		}

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_threatfeeds_ipset*")

		// Iterate through the first 4 pages and check they are correct.
		var afterKey map[string]interface{}
		for i := 0; i < totalItems-1; i++ {
			params := v1.IPSetThreatFeedParams{
				QueryParams: v1.QueryParams{
					TimeRange: &lmav1.TimeRange{
						From: createdAtTime.Add(-5 * time.Second),
						To:   createdAtTime.Add(5 * time.Second),
					},
					MaxPageSize: 1,
					AfterKey:    afterKey,
				},
			}
			resp, err := cli.ThreatFeeds(cluster).IPSet().List(ctx, &params)
			require.NoError(t, err)
			require.Equal(t, 1, len(resp.Items))
			require.Equal(t, []v1.IPSetThreatFeed{
				{
					ID: strconv.Itoa(i),
					Data: &v1.IPSetThreatFeedData{
						CreatedAt: createdAtTime.Add(time.Duration(i) * time.Second),
					},
				},
			}, resp.Items, fmt.Sprintf("Threat Feed #%d did not match", i))
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
		params := v1.IPSetThreatFeedParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: createdAtTime.Add(-5 * time.Second),
					To:   createdAtTime.Add(5 * time.Second),
				},
				MaxPageSize: 1,
				AfterKey:    afterKey,
			},
		}
		resp, err := cli.ThreatFeeds(cluster).IPSet().List(ctx, &params)
		require.NoError(t, err)
		require.Equal(t, 1, len(resp.Items))
		require.Equal(t, []v1.IPSetThreatFeed{
			{
				ID: strconv.Itoa(lastItem),
				Data: &v1.IPSetThreatFeedData{
					CreatedAt: createdAtTime.Add(time.Duration(lastItem) * time.Second),
				},
			},
		}, resp.Items, fmt.Sprintf("Feeds #%d did not match", lastItem))
		require.Equal(t, resp.TotalHits, int64(totalItems))

		// Once we reach the end of the data, we should not receive
		// an afterKey
		require.Nil(t, resp.AfterKey)
	})

	t.Run("should support pagination for items >= 10000 for threat feeds", func(t *testing.T) {
		defer threatFeedsSetupAndTeardown(t)()

		totalItems := 10001
		// Create > 10K threat feeds.
		createdAtTime := time.Unix(0, 0).UTC()
		var feeds []v1.IPSetThreatFeed
		for i := 0; i < totalItems; i++ {
			feeds = append(feeds, v1.IPSetThreatFeed{
				ID: strconv.Itoa(i),
				Data: &v1.IPSetThreatFeedData{
					CreatedAt: createdAtTime.Add(time.Duration(i) * time.Second),
				},
			},
			)
		}
		bulk, err := cli.ThreatFeeds(cluster).IPSet().Create(ctx, feeds)
		require.NoError(t, err)
		require.Equal(t, totalItems, bulk.Total, "create feeds did not succeed")

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_threatfeeds_ipset*")

		// Stream through all the items.
		params := v1.IPSetThreatFeedParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: createdAtTime.Add(-5 * time.Second),
					To:   createdAtTime.Add(time.Duration(totalItems) * time.Second),
				},
				MaxPageSize: 1000,
			},
		}

		pager := client.NewListPager[v1.IPSetThreatFeed](&params)
		pages, errors := pager.Stream(ctx, cli.ThreatFeeds(cluster).IPSet().List)

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

func TestFV_IPSetTenancy(t *testing.T) {
	t.Run("should support tenancy restriction", func(t *testing.T) {
		defer threatFeedsSetupAndTeardown(t)()

		// Instantiate a client for an unexpected tenant.
		tenantCLI, err := NewLinseedClientForTenant("bad-tenant")
		require.NoError(t, err)

		// Create a basic feed. We expect this to fail, since we're using
		// an unexpected tenant ID on the request.
		feeds := v1.IPSetThreatFeed{
			ID: "feed-a",
			Data: &v1.IPSetThreatFeedData{
				CreatedAt: time.Unix(0, 0),
				IPs:       []string{"1.2.3.4/32"},
			},
		}
		bulk, err := tenantCLI.ThreatFeeds(cluster).IPSet().Create(ctx, []v1.IPSetThreatFeed{feeds})
		require.ErrorContains(t, err, "Bad tenant identifier")
		require.Nil(t, bulk)

		// Try a read as well.
		params := v1.IPSetThreatFeedParams{ID: "feed-a"}
		resp, err := tenantCLI.ThreatFeeds(cluster).IPSet().List(ctx, &params)
		require.ErrorContains(t, err, "Bad tenant identifier")
		require.Nil(t, resp)
	})
}
