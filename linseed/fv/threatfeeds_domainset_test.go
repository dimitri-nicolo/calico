// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/index"
	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"
	"github.com/projectcalico/calico/linseed/pkg/client"
	"github.com/projectcalico/calico/linseed/pkg/config"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
)

// Run runs the given test in all modes.
func RunThreatfeedsDomainTest(t *testing.T, name string, testFn func(*testing.T, bapi.Index)) {
	t.Run(fmt.Sprintf("%s [MultiIndex]", name), func(t *testing.T) {
		args := DefaultLinseedArgs()
		defer setupAndTeardown(t, args, nil, index.ThreatfeedsDomainMultiIndex)()
		testFn(t, index.ThreatfeedsDomainMultiIndex)
	})

	t.Run(fmt.Sprintf("%s [SingleIndex]", name), func(t *testing.T) {
		confArgs := &RunConfigureElasticArgs{
			ThreatFeedsDomainSetBaseIndexName: index.ThreatFeedsDomainSetIndex().Name(bapi.ClusterInfo{}),
			ThreatFeedsDomainSetPolicyName:    index.ThreatFeedsDomainSetIndex().ILMPolicyName(),
		}
		args := DefaultLinseedArgs()
		args.Backend = config.BackendTypeSingleIndex
		defer setupAndTeardown(t, args, confArgs, index.ThreatFeedsDomainSetIndex())()
		testFn(t, index.ThreatFeedsDomainSetIndex())
	})
}

func TestFV_ThreatFeedsDomainSet(t *testing.T) {
	RunThreatfeedsDomainTest(t, "should return an empty list if there are no threat feeds", func(t *testing.T, idx bapi.Index) {
		params := v1.DomainNameSetThreatFeedParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now(),
				},
			},
		}

		// Perform a query.
		feeds, err := cli.ThreatFeeds(cluster).DomainNameSet().List(ctx, &params)
		require.NoError(t, err)
		require.Equal(t, []v1.DomainNameSetThreatFeed{}, feeds.Items)
	})

	RunThreatfeedsDomainTest(t, "should create and list threat feeds", func(t *testing.T, idx bapi.Index) {
		feed := v1.DomainNameSetThreatFeed{
			ID: "feed-a",
			Data: &v1.DomainNameSetThreatFeedData{
				CreatedAt: time.Unix(0, 0).UTC(),
				Domains:   []string{"a.b.c.d"},
			},
		}
		bulk, err := cli.ThreatFeeds(cluster).DomainNameSet().Create(ctx, []v1.DomainNameSetThreatFeed{feed})
		require.NoError(t, err)
		require.Equal(t, bulk.Succeeded, 1, "create did not succeed")

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, idx.Index(clusterInfo))

		// Read it back, passing an ID query.
		params := v1.DomainNameSetThreatFeedParams{ID: "feed-a"}
		resp, err := cli.ThreatFeeds(cluster).DomainNameSet().List(ctx, &params)
		require.NoError(t, err)

		// The ID should be set.
		require.Len(t, resp.Items, 1)
		require.Equal(t, feed.ID, resp.Items[0].ID)
		require.Equal(t, feed, resp.Items[0])

		// Delete the feed
		bulkDelete, err := cli.ThreatFeeds(cluster).DomainNameSet().Delete(ctx, []v1.DomainNameSetThreatFeed{feed})
		require.NoError(t, err)
		require.Equal(t, bulkDelete.Succeeded, 1, "delete did not succeed")

		// Read after delete
		afterDelete, err := cli.ThreatFeeds(cluster).DomainNameSet().List(ctx, &params)
		require.NoError(t, err)
		require.Empty(t, afterDelete.Items)
	})

	RunThreatfeedsDomainTest(t, "should create and list threat feeds, from 2 separate managed clusters", func(t *testing.T, idx bapi.Index) {
		feed := v1.DomainNameSetThreatFeed{
			ID: "my-built-in-feed",
			Data: &v1.DomainNameSetThreatFeedData{
				CreatedAt: time.Unix(0, 0).UTC(),
				Domains:   []string{"a.b.c.d"},
			},
		}
		// Let's imagine that we have 2 managed clusters in the same tenant
		clusterA := testutils.RandomClusterName()
		clusterAInfo := bapi.ClusterInfo{Cluster: clusterA, Tenant: clusterInfo.Tenant}
		clusterB := testutils.RandomClusterName()
		clusterBInfo := bapi.ClusterInfo{Cluster: clusterB, Tenant: clusterInfo.Tenant}

		// clusterA and clusterB both use the same feed (like the built-in alienvault feed in CC)
		bulk, err := cli.ThreatFeeds(clusterA).DomainNameSet().Create(ctx, []v1.DomainNameSetThreatFeed{feed})
		require.NoError(t, err)
		require.Equal(t, bulk.Succeeded, 1, "create did not succeed")
		bulk, err = cli.ThreatFeeds(clusterB).DomainNameSet().Create(ctx, []v1.DomainNameSetThreatFeed{feed})
		require.NoError(t, err)
		require.Equal(t, bulk.Succeeded, 1, "create did not succeed")

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, idx.Index(clusterAInfo))
		testutils.RefreshIndex(ctx, lmaClient, idx.Index(clusterBInfo))

		// Read it back, not using any ID to get all feeds (like IDC does)
		params := v1.DomainNameSetThreatFeedParams{}
		resp, err := cli.ThreatFeeds(clusterB).DomainNameSet().List(ctx, &params)
		require.NoError(t, err)

		// The ID should be set.
		require.Len(t, resp.Items, 1)
		require.Equal(t, feed.ID, resp.Items[0].ID)
		require.Equal(t, feed, resp.Items[0])

		// Read back the other one, which was created first
		resp, err = cli.ThreatFeeds(clusterA).DomainNameSet().List(ctx, &params)
		require.NoError(t, err)

		// The ID should be set.
		require.Len(t, resp.Items, 1)
		require.Equal(t, feed.ID, resp.Items[0].ID)
		require.Equal(t, feed, resp.Items[0])
	})

	RunThreatfeedsDomainTest(t, "should support pagination", func(t *testing.T, idx bapi.Index) {
		totalItems := 5

		// Create 5 Feeds.
		createdAtTime := time.Unix(0, 0).UTC()
		for i := 0; i < totalItems; i++ {
			feed := []v1.DomainNameSetThreatFeed{
				{
					ID: strconv.Itoa(i),
					Data: &v1.DomainNameSetThreatFeedData{
						CreatedAt: createdAtTime.Add(time.Duration(i) * time.Second),
					},
				},
			}
			bulk, err := cli.ThreatFeeds(cluster).DomainNameSet().Create(ctx, feed)
			require.NoError(t, err)
			require.Equal(t, bulk.Succeeded, 1, "create feeds did not succeed")
		}

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, idx.Index(clusterInfo))

		// Iterate through the first 4 pages and check they are correct.
		var afterKey map[string]interface{}
		for i := 0; i < totalItems-1; i++ {
			params := v1.DomainNameSetThreatFeedParams{
				QueryParams: v1.QueryParams{
					TimeRange: &lmav1.TimeRange{
						From: createdAtTime.Add(-5 * time.Second),
						To:   createdAtTime.Add(5 * time.Second),
					},
					MaxPageSize: 1,
					AfterKey:    afterKey,
				},
			}
			resp, err := cli.ThreatFeeds(cluster).DomainNameSet().List(ctx, &params)
			require.NoError(t, err)
			require.Equal(t, 1, len(resp.Items))
			require.Equal(t, []v1.DomainNameSetThreatFeed{
				{
					ID: strconv.Itoa(i),
					Data: &v1.DomainNameSetThreatFeedData{
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
		params := v1.DomainNameSetThreatFeedParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: createdAtTime.Add(-5 * time.Second),
					To:   createdAtTime.Add(5 * time.Second),
				},
				MaxPageSize: 1,
				AfterKey:    afterKey,
			},
		}
		resp, err := cli.ThreatFeeds(cluster).DomainNameSet().List(ctx, &params)
		require.NoError(t, err)
		require.Equal(t, 1, len(resp.Items))
		require.Equal(t, []v1.DomainNameSetThreatFeed{
			{
				ID: strconv.Itoa(lastItem),
				Data: &v1.DomainNameSetThreatFeedData{
					CreatedAt: createdAtTime.Add(time.Duration(lastItem) * time.Second),
				},
			},
		}, resp.Items, fmt.Sprintf("Feeds #%d did not match", lastItem))
		require.Equal(t, resp.TotalHits, int64(totalItems))

		// Once we reach the end of the data, we should not receive
		// an afterKey
		require.Nil(t, resp.AfterKey)
	})

	RunThreatfeedsDomainTest(t, "should support pagination for items >= 10000 for threat feeds", func(t *testing.T, idx bapi.Index) {
		totalItems := 10001
		// Create > 10K threat feeds.
		createdAtTime := time.Unix(0, 0).UTC()
		var feeds []v1.DomainNameSetThreatFeed
		for i := 0; i < totalItems; i++ {
			feeds = append(feeds, v1.DomainNameSetThreatFeed{
				ID: strconv.Itoa(i),
				Data: &v1.DomainNameSetThreatFeedData{
					CreatedAt: createdAtTime.Add(time.Duration(i) * time.Second),
				},
			},
			)
		}
		bulk, err := cli.ThreatFeeds(cluster).DomainNameSet().Create(ctx, feeds)
		require.NoError(t, err)
		require.Equal(t, totalItems, bulk.Total, "create feeds did not succeed")

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, idx.Index(clusterInfo))

		// Stream through all the items.
		params := v1.DomainNameSetThreatFeedParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: createdAtTime.Add(-5 * time.Second),
					To:   createdAtTime.Add(time.Duration(totalItems) * time.Second),
				},
				MaxPageSize: 1000,
			},
		}

		pager := client.NewListPager[v1.DomainNameSetThreatFeed](&params)
		pages, errors := pager.Stream(ctx, cli.ThreatFeeds(cluster).DomainNameSet().List)

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

func TestFV_DomainNameSetTenancy(t *testing.T) {
	RunThreatfeedsDomainTest(t, "should support tenancy restriction", func(t *testing.T, idx bapi.Index) {
		// Instantiate a client for an unexpected tenant.
		args := DefaultLinseedArgs()
		args.TenantID = "bad-tenant"
		tenantCLI, err := NewLinseedClient(args)
		require.NoError(t, err)

		// Create a basic feed. We expect this to fail, since we're using
		// an unexpected tenant ID on the request.
		feed := v1.DomainNameSetThreatFeed{
			ID: "feed-a",
			Data: &v1.DomainNameSetThreatFeedData{
				CreatedAt: time.Unix(0, 0).UTC(),
				Domains:   []string{"a.b.c.d"},
			},
		}
		bulk, err := tenantCLI.ThreatFeeds(cluster).DomainNameSet().Create(ctx, []v1.DomainNameSetThreatFeed{feed})
		require.ErrorContains(t, err, "Bad tenant identifier")
		require.Nil(t, bulk)

		// Try a read as well.
		params := v1.DomainNameSetThreatFeedParams{ID: "feed-a"}
		resp, err := tenantCLI.ThreatFeeds(cluster).DomainNameSet().List(ctx, &params)
		require.ErrorContains(t, err, "Bad tenant identifier")
		require.Nil(t, resp)
	})
}
