// Copyright (c) 2023 Tigera All rights reserved.

package threatfeeds_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/threatfeeds"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"

	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/templates"
	backendutils "github.com/projectcalico/calico/linseed/pkg/backend/testutils"
	"github.com/projectcalico/calico/linseed/pkg/config"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

var (
	client      lmaelastic.Client
	cache       bapi.Cache
	ib          bapi.IPSetBackend
	db          bapi.DomainNameSetBackend
	ctx         context.Context
	cluster     string
	clusterInfo bapi.ClusterInfo
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
	cache = templates.NewTemplateCache(client, 1, 0)

	// Create backends to use.
	ib = threatfeeds.NewIPSetBackend(client, cache)
	db = threatfeeds.NewDomainNameSetBackend(client, cache)

	// Create a random cluster name for each test to make sure we don't
	// interfere between tests.
	cluster = backendutils.RandomClusterName()
	clusterInfo = bapi.ClusterInfo{Cluster: cluster}

	// Set a timeout for each test.
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Hour)

	// Function contains teardown logic.
	return func() {
		// Cancel the context.
		cancel()

		// Cleanup any data that might have been left over from a previous failed run.
		err = backendutils.CleanupIndices(context.Background(), esClient, cluster)
		require.NoError(t, err)

		// Cancel logging
		logCancel()
	}
}

func TestIPSetBasic(t *testing.T) {
	t.Run("invalid ClusterInfo", func(t *testing.T) {
		defer setupTest(t)()

		f := v1.IPSetThreatFeed{}
		p := v1.IPSetThreatFeedParams{}

		// Empty cluster info.
		empty := bapi.ClusterInfo{}
		_, err := ib.Create(ctx, empty, []v1.IPSetThreatFeed{f})
		require.Error(t, err)
		_, err = ib.List(ctx, empty, &p)
		require.Error(t, err)

		// Invalid tenant ID in cluster info.
		badTenant := bapi.ClusterInfo{Cluster: cluster, Tenant: "one,two"}
		_, err = ib.Create(ctx, badTenant, []v1.IPSetThreatFeed{f})
		require.Error(t, err)
		_, err = ib.List(ctx, badTenant, &p)
		require.Error(t, err)
	})

	// Run each test with a tenant specified, and also without a tenant.
	for _, tenant := range []string{backendutils.RandomTenantName(), ""} {
		name := fmt.Sprintf("create and retrieve reports (tenant=%s)", tenant)
		t.Run(name, func(t *testing.T) {
			defer setupTest(t)()
			clusterInfo.Tenant = tenant

			// Create a dummy threat feed.
			feed := v1.IPSetThreatFeedData{
				CreatedAt: time.Unix(0, 0).UTC(),
				IPs:       []string{"1.2.3.4/32"},
			}
			f := v1.IPSetThreatFeed{
				ID:   "my-threat-feed",
				Data: &feed,
			}

			response, err := ib.Create(ctx, clusterInfo, []v1.IPSetThreatFeed{f})
			require.NoError(t, err)
			require.Equal(t, []v1.BulkError(nil), response.Errors)
			require.Equal(t, 0, response.Failed)

			err = backendutils.RefreshIndex(ctx, client, "tigera_secure_ee_threatfeeds_ipset.*")
			require.NoError(t, err)

			// Read it back and check it matches.
			p := v1.IPSetThreatFeedParams{}
			resp, err := ib.List(ctx, clusterInfo, &p)
			require.NoError(t, err)
			require.Len(t, resp.Items, 1)
			require.Equal(t, "my-threat-feed", resp.Items[0].ID)
			require.Equal(t, feed, *resp.Items[0].Data)

			delResp, err := ib.Delete(ctx, clusterInfo, []v1.IPSetThreatFeed{f})
			require.NoError(t, err)
			require.Equal(t, []v1.BulkError(nil), delResp.Errors)
			require.Equal(t, 0, delResp.Failed)

			afterDelete, err := ib.List(ctx, clusterInfo, &p)
			require.NoError(t, err)
			require.Len(t, afterDelete.Items, 0)
		})
	}
}

func TestIPSetFiltering(t *testing.T) {
	type testcase struct {
		Name    string
		Params  *v1.IPSetThreatFeedParams
		Expect1 bool
		Expect2 bool
	}

	testcases := []testcase{
		{
			Name: "should filter feeds based on ID",
			Params: &v1.IPSetThreatFeedParams{
				ID: "feed-id-1",
			},
			Expect1: true,
			Expect2: false,
		},
		{
			Name: "should filter feeds based on timestamp range",
			Params: &v1.IPSetThreatFeedParams{
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
			Params: &v1.IPSetThreatFeedParams{
				QueryParams: v1.QueryParams{
					TimeRange: &lmav1.TimeRange{
						To: time.Unix(1000, 0).UTC(),
					},
				},
			},
			Expect1: true,
			Expect2: false,
		},
		{
			Name: "should filter reports based on start time",
			Params: &v1.IPSetThreatFeedParams{
				QueryParams: v1.QueryParams{
					TimeRange: &lmav1.TimeRange{
						From: time.Unix(1000, 0).UTC(),
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
			t.Run(name, func(t *testing.T) {
				defer setupTest(t)()
				clusterInfo.Tenant = tenant

				f1 := v1.IPSetThreatFeed{
					ID: "feed-id-1",
					Data: &v1.IPSetThreatFeedData{
						CreatedAt: time.Unix(100, 0).UTC(),
						IPs:       []string{"1.2.3.4/32"},
					},
				}
				f2 := v1.IPSetThreatFeed{
					ID: "feed-id-2",
					Data: &v1.IPSetThreatFeedData{
						CreatedAt: time.Unix(2000, 0).UTC(),
						IPs:       []string{"3.4.5.6/32"},
					},
				}

				response, err := ib.Create(ctx, clusterInfo, []v1.IPSetThreatFeed{f1, f2})
				require.NoError(t, err)
				require.Equal(t, []v1.BulkError(nil), response.Errors)
				require.Equal(t, 0, response.Failed)

				err = backendutils.RefreshIndex(ctx, client, "tigera_secure_ee_threatfeeds_ipset.*")
				require.NoError(t, err)

				resp, err := ib.List(ctx, clusterInfo, tc.Params)
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
