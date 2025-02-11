// Copyright (c) 2023 Tigera All rights reserved.

package threatfeeds_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	backendutils "github.com/projectcalico/calico/linseed/pkg/backend/testutils"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
)

func TestDomainSetBasic(t *testing.T) {
	RunAllModes(t, "invalid ClusterInfo", func(t *testing.T) {
		f := v1.DomainNameSetThreatFeed{}
		p := v1.DomainNameSetThreatFeedParams{}

		// Empty cluster info.
		empty := bapi.ClusterInfo{}
		_, err := db.Create(ctx, empty, []v1.DomainNameSetThreatFeed{f})
		require.Error(t, err)
		_, err = db.List(ctx, empty, &p)
		require.Error(t, err)

		// Invalid tenant ID in cluster info.
		badTenant := bapi.ClusterInfo{Cluster: cluster1, Tenant: "one,two"}
		_, err = db.Create(ctx, badTenant, []v1.DomainNameSetThreatFeed{f})
		require.Error(t, err)
		_, err = db.List(ctx, badTenant, &p)
		require.Error(t, err)
	})

	// Run each test with a tenant specified, and also without a tenant.
	for _, tenant := range []string{backendutils.RandomTenantName(), ""} {
		name := fmt.Sprintf("create and retrieve reports (tenant=%s)", tenant)
		RunAllModes(t, name, func(t *testing.T) {
			cluster1Info := bapi.ClusterInfo{Cluster: cluster1, Tenant: tenant}
			cluster2Info := bapi.ClusterInfo{Cluster: cluster2, Tenant: tenant}
			cluster3Info := bapi.ClusterInfo{Cluster: cluster3, Tenant: tenant}

			// Create a dummy threat feed.
			feed := v1.DomainNameSetThreatFeedData{
				CreatedAt: time.Unix(0, 0).UTC(),
				Domains:   []string{"a.b.c.d."},
			}
			feedCopy := feed
			f := v1.DomainNameSetThreatFeed{
				ID:   "my-threat-feed",
				Data: &feedCopy, // don't use the original feed, as it will be modified on the backend
			}

			for _, clusterInfo := range []bapi.ClusterInfo{cluster1Info, cluster2Info, cluster3Info} {
				response, err := db.Create(ctx, clusterInfo, []v1.DomainNameSetThreatFeed{f})
				require.NoError(t, err)
				require.Equal(t, []v1.BulkError(nil), response.Errors)
				require.Equal(t, 0, response.Failed)

				err = backendutils.RefreshIndex(ctx, client, domainsIndexGetter.Index(clusterInfo))
				require.NoError(t, err)
			}

			// Read it back and check it matches.
			params := v1.DomainNameSetThreatFeedParams{}

			t.Run("should query single cluster", func(t *testing.T) {
				clusterInfo := cluster1Info
				resp, err := db.List(ctx, clusterInfo, &params)
				require.NoError(t, err)
				require.Len(t, resp.Items, 1)
				require.Equal(t, "my-threat-feed", resp.Items[0].ID)
				backendutils.AssertDomainNameSetThreatFeedClusterAndReset(t, clusterInfo.Cluster, &resp.Items[0])
				require.Equal(t, feed, *resp.Items[0].Data)

				// Attempt to delete it with an invalid tenant ID. It should fail.
				badClusterInfo := bapi.ClusterInfo{Cluster: clusterInfo.Cluster, Tenant: "bad-tenant"}
				bulkResp, err := db.Delete(ctx, badClusterInfo, []v1.DomainNameSetThreatFeed{resp.Items[0]})
				require.NoError(t, err)
				if ipsetIndexGetter.IsSingleIndex() {
					require.Len(t, bulkResp.Errors, 1)
					require.Equal(t, bulkResp.Failed, 1)
				}
			})

			t.Run("should query multiple clusters", func(t *testing.T) {
				selectedClusters := []string{cluster2, cluster3}
				params.SetClusters(selectedClusters)
				resp, err := db.List(ctx, bapi.ClusterInfo{Cluster: v1.QueryMultipleClusters, Tenant: tenant}, &params)
				require.NoError(t, err)
				require.Len(t, resp.Items, 2)
				for _, cluster := range selectedClusters {
					require.Truef(t, backendutils.MatchIn(resp.Items, backendutils.DomainNameSetThreatFeedClusterEquals(cluster)), "cluster %s not found", cluster)
				}
			})

			t.Run("should query all clusters", func(t *testing.T) {
				params.SetAllClusters(true)
				resp, err := db.List(ctx, bapi.ClusterInfo{Cluster: v1.QueryMultipleClusters, Tenant: tenant}, &params)
				require.NoError(t, err)
				for _, cluster := range []string{cluster1, cluster2, cluster3} {
					require.Truef(t, backendutils.MatchIn(resp.Items, backendutils.DomainNameSetThreatFeedClusterEquals(cluster)), "cluster %s not found", cluster)
				}
			})

			t.Run("delete", func(t *testing.T) {
				clusterInfo := cluster1Info
				// Delete it with the correct tenant ID and cluster.
				delResp, err := db.Delete(ctx, clusterInfo, []v1.DomainNameSetThreatFeed{f})
				require.NoError(t, err)
				require.Equal(t, []v1.BulkError(nil), delResp.Errors)
				require.Equal(t, 0, delResp.Failed)

				afterDelete, err := db.List(ctx, clusterInfo, &params)
				require.NoError(t, err)
				require.Len(t, afterDelete.Items, 0)
			})
		})
	}
}

func TestDomainSetFiltering(t *testing.T) {
	type testcase struct {
		Name    string
		Params  *v1.DomainNameSetThreatFeedParams
		Expect1 bool
		Expect2 bool
	}

	testcases := []testcase{
		{
			Name: "should filter feeds based on ID",
			Params: &v1.DomainNameSetThreatFeedParams{
				ID: "feed-id-1",
			},
			Expect1: true,
			Expect2: false,
		},
		{
			Name: "should filter feeds based on timestamp range",
			Params: &v1.DomainNameSetThreatFeedParams{
				QueryParams: v1.QueryParams{
					TimeRange: &lmav1.TimeRange{
						From: time.Unix(1000, 0).UTC(),
						To:   time.Unix(3000, 0).UTC(),
					},
				},
			},
			Expect1: false,
			Expect2: true,
		},
		{
			Name: "should filter feeds based on end time",
			Params: &v1.DomainNameSetThreatFeedParams{
				QueryParams: v1.QueryParams{
					TimeRange: &lmav1.TimeRange{
						To: time.Unix(1000, 0),
					},
				},
			},
			Expect1: true,
			Expect2: false,
		},
		{
			Name: "should filter reports based on start time",
			Params: &v1.DomainNameSetThreatFeedParams{
				QueryParams: v1.QueryParams{
					TimeRange: &lmav1.TimeRange{
						From: time.Unix(1000, 0),
					},
				},
			},
			Expect1: false,
			Expect2: true,
		},
	}

	for _, tc := range testcases {
		// Run each test with a tenant specified, and also without a tenant.
		for _, tenant := range []string{backendutils.RandomTenantName(), ""} {
			name := fmt.Sprintf("%s (tenant=%s)", tc.Name, tenant)
			RunAllModes(t, name, func(t *testing.T) {
				clusterInfo := bapi.ClusterInfo{Cluster: cluster1, Tenant: tenant}
				require.NotEmpty(t, clusterInfo.Cluster)

				f1 := v1.DomainNameSetThreatFeed{
					ID: "feed-id-1",
					Data: &v1.DomainNameSetThreatFeedData{
						CreatedAt: time.Unix(100, 0).UTC(),
						Domains:   []string{"a.b.c.d"},
					},
				}
				f2 := v1.DomainNameSetThreatFeed{
					ID: "feed-id-2",
					Data: &v1.DomainNameSetThreatFeedData{
						CreatedAt: time.Unix(2000, 0).UTC(),
						Domains:   []string{"x.y.z"},
					},
				}

				response, err := db.Create(ctx, clusterInfo, []v1.DomainNameSetThreatFeed{f1, f2})
				require.NoError(t, err)
				require.Equal(t, []v1.BulkError(nil), response.Errors)
				require.Equal(t, 0, response.Failed)

				err = backendutils.RefreshIndex(ctx, client, domainsIndexGetter.Index(clusterInfo))
				require.NoError(t, err)

				resp, err := db.List(ctx, clusterInfo, tc.Params)
				require.NoError(t, err)

				if tc.Expect1 {
					require.Contains(t, resp.Items, f1)
				} else {
					require.NotContains(t, resp.Items, f1)
				}
				if tc.Expect2 {
					require.Contains(t, resp.Items, f2)
				} else {
					require.NotContains(t, resp.Items, f2)
				}
			})
		}
	}
}
