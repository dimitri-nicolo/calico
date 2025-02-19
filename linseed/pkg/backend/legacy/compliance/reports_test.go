// Copyright (c) 2023 Tigera All rights reserved.

package compliance_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/compliance"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/index"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/templates"
	backendutils "github.com/projectcalico/calico/linseed/pkg/backend/testutils"
	"github.com/projectcalico/calico/linseed/pkg/config"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

var (
	client   lmaelastic.Client
	cache    bapi.IndexInitializer
	rb       bapi.ReportsBackend
	bb       bapi.BenchmarksBackend
	sb       bapi.SnapshotsBackend
	ctx      context.Context
	cluster1 string
	cluster2 string
	cluster3 string

	// Report, benchmark, and snapshot indexes.
	rIndexGetter bapi.Index
	bIndexGetter bapi.Index
	sIndexGetter bapi.Index
)

// RunAllModes runs the given test function twice, once using the single-index backend, and once using
// the multi-index backend.
func RunAllModes(t *testing.T, name string, testFn func(t *testing.T)) {
	// Run using the multi-index backend.
	t.Run(fmt.Sprintf("%s [legacy]", name), func(t *testing.T) {
		defer setupTest(t, false)()
		testFn(t)
	})

	// Run using the single-index backend.
	t.Run(fmt.Sprintf("%s [singleindex]", name), func(t *testing.T) {
		defer setupTest(t, true)()
		testFn(t)
	})
}

// setupTest runs common logic before each test, and also returns a function to perform teardown
// after each test.
func setupTest(t *testing.T, singleIndex bool) func() {
	// Hook logrus into testing.T
	config.ConfigureLogging("DEBUG")
	logCancel := logutils.RedirectLogrusToTestingT(t)

	// Create an elasticsearch client to use for the test. For this suite, we use a real
	// elasticsearch instance created via "make run-elastic".
	esClient, err := elastic.NewSimpleClient(elastic.SetURL("http://localhost:9200"), elastic.SetInfoLog(logrus.StandardLogger()))

	require.NoError(t, err)
	client = lmaelastic.NewWithClient(esClient)

	cache = templates.NewCachedInitializer(client, 1, 0)

	// Create backends to use.
	if singleIndex {
		rIndexGetter = index.ComplianceReportsIndex()
		bIndexGetter = index.ComplianceBenchmarksIndex()
		sIndexGetter = index.ComplianceSnapshotsIndex()
		rb = compliance.NewSingleIndexReportsBackend(client, cache, 10000)
		bb = compliance.NewSingleIndexBenchmarksBackend(client, cache, 10000)
		sb = compliance.NewSingleIndexSnapshotBackend(client, cache, 10000)
	} else {
		rb = compliance.NewReportsBackend(client, cache, 10000)
		bb = compliance.NewBenchmarksBackend(client, cache, 10000)
		sb = compliance.NewSnapshotBackend(client, cache, 10000)
		rIndexGetter = index.ComplianceReportMultiIndex
		bIndexGetter = index.ComplianceBenchmarkMultiIndex
		sIndexGetter = index.ComplianceSnapshotMultiIndex
	}

	// Create a random cluster name for each test to make sure we don't
	// interfere between tests.
	cluster1 = backendutils.RandomClusterName()
	cluster2 = backendutils.RandomClusterName()
	cluster3 = backendutils.RandomClusterName()

	// Set a timeout for each test.
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)

	// Function contains teardown logic.
	return func() {
		// Cancel the context.
		cancel()

		// Cleanup any data that might left over from a previous run.
		for _, cluster := range []string{cluster1, cluster2, cluster3} {
			for _, indexGetter := range []bapi.Index{rIndexGetter, bIndexGetter, sIndexGetter} {
				err = backendutils.CleanupIndices(context.Background(), esClient, singleIndex, indexGetter, bapi.ClusterInfo{Cluster: cluster})
				require.NoError(t, err)
			}
		}

		// Cancel logging
		logCancel()
	}
}

func TestReportDataBasic(t *testing.T) {
	RunAllModes(t, "invalid ClusterInfo", func(t *testing.T) {
		f := v1.ReportData{}
		p := v1.ReportDataParams{}

		// Empty cluster info.
		empty := bapi.ClusterInfo{}
		_, err := rb.Create(ctx, empty, []v1.ReportData{f})
		require.Error(t, err)
		_, err = rb.List(ctx, empty, &p)
		require.Error(t, err)

		// Invalid tenant ID in cluster info.
		badTenant := bapi.ClusterInfo{Cluster: cluster1, Tenant: "one,two"}
		_, err = rb.Create(ctx, badTenant, []v1.ReportData{f})
		require.Error(t, err)
		_, err = rb.List(ctx, badTenant, &p)
		require.Error(t, err)
	})

	// Run each test with a tenant specified, and also without a tenant.
	for _, tenant := range []string{backendutils.RandomTenantName(), ""} {
		name := fmt.Sprintf("create and retrieve reports (tenant=%s)", tenant)
		RunAllModes(t, name, func(t *testing.T) {
			// Create a dummy report.
			report := apiv3.ReportData{
				ReportName:     "test-report",
				ReportTypeName: "my-report-type",
				StartTime:      metav1.Time{Time: time.Unix(1, 0)},
				EndTime:        metav1.Time{Time: time.Unix(2, 0)},
				GenerationTime: metav1.Time{Time: time.Unix(3, 0)},
			}
			f := v1.ReportData{ReportData: &report}
			f.ID = f.UID()

			for _, cluster := range []string{cluster1, cluster2, cluster3} {
				clusterInfo := bapi.ClusterInfo{Cluster: cluster, Tenant: tenant}

				response, err := rb.Create(ctx, clusterInfo, []v1.ReportData{f})
				require.NoError(t, err)
				require.Equal(t, []v1.BulkError(nil), response.Errors)
				require.Equal(t, 0, response.Failed)

				err = backendutils.RefreshIndex(ctx, client, rIndexGetter.Index(clusterInfo))
				require.NoError(t, err)
			}

			params := &v1.ReportDataParams{}

			t.Run("should query single cluster", func(t *testing.T) {
				clusterInfo := bapi.ClusterInfo{Cluster: cluster1, Tenant: tenant}

				// Read it back and check it matches.
				resp, err := rb.List(ctx, clusterInfo, params)
				require.NoError(t, err)
				require.Len(t, resp.Items, 1)
				backendutils.AssertReportDataClusterAndReset(t, clusterInfo.Cluster, &resp.Items[0])
				require.NotEmpty(t, resp.Items[0].ID)
				require.Equal(t, f, resp.Items[0])
			})

			t.Run("should query multiple clusters", func(t *testing.T) {
				selectedClusters := []string{cluster2, cluster3}
				params.SetClusters(selectedClusters)

				resp, err := rb.List(ctx, bapi.ClusterInfo{Cluster: v1.QueryMultipleClusters}, params)
				require.NoError(t, err)

				require.Falsef(t, backendutils.MatchIn(resp.Items, backendutils.ReportDataClusterEquals(cluster1)), "cluster1 should not be in the results")
				for _, cluster := range selectedClusters {
					require.Truef(t, backendutils.MatchIn(resp.Items, backendutils.ReportDataClusterEquals(cluster)), "cluster %s should be in the results", cluster)
				}
			})

			t.Run("should query all clusters", func(t *testing.T) {
				params.SetAllClusters(true)
				resp, err := rb.List(ctx, bapi.ClusterInfo{Cluster: v1.QueryMultipleClusters}, params)
				require.NoError(t, err)
				for _, cluster := range []string{cluster1, cluster2, cluster3} {
					require.Truef(t, backendutils.MatchIn(resp.Items, backendutils.ReportDataClusterEquals(cluster)), "cluster %s should be in the results", cluster)
				}
			})

		})

		RunAllModes(t, "should ensure data does not overlap", func(t *testing.T) {
			clusterInfo := bapi.ClusterInfo{Cluster: cluster1, Tenant: tenant}
			anotherClusterInfo := bapi.ClusterInfo{Cluster: cluster2, Tenant: tenant}

			t1 := time.Unix(100, 0)

			r1 := v1.ReportData{
				ID: "report-id",
				ReportData: &apiv3.ReportData{
					ReportName:     "report-name",
					ReportTypeName: "report-type",
					StartTime:      metav1.Time{Time: t1},
					EndTime:        metav1.Time{Time: t1},
					GenerationTime: metav1.Time{Time: t1},
				},
			}
			r2 := v1.ReportData{
				ID: "report-id",
				ReportData: &apiv3.ReportData{
					ReportName:     "report-name",
					ReportTypeName: "report-type",
					StartTime:      metav1.Time{Time: t1},
					EndTime:        metav1.Time{Time: t1},
					GenerationTime: metav1.Time{Time: t1},
				},
			}

			_, err := rb.Create(ctx, clusterInfo, []v1.ReportData{r1})
			require.NoError(t, err)

			_, err = rb.Create(ctx, anotherClusterInfo, []v1.ReportData{r2})
			require.NoError(t, err)

			err = backendutils.RefreshIndex(ctx, client, rIndexGetter.Index(clusterInfo))
			require.NoError(t, err)

			err = backendutils.RefreshIndex(ctx, client, rIndexGetter.Index(anotherClusterInfo))
			require.NoError(t, err)

			// Read back data a managed cluster and check it matches.
			p1 := v1.ReportDataParams{}
			resp, err := rb.List(ctx, clusterInfo, &p1)
			require.NoError(t, err)
			require.Len(t, resp.Items, 1)
			backendutils.AssertReportDataClusterAndReset(t, clusterInfo.Cluster, &resp.Items[0])
			require.Equal(t, r1, resp.Items[0])

			// Read back data a managed cluster and check it matches.
			p2 := v1.ReportDataParams{}
			resp, err = rb.List(ctx, anotherClusterInfo, &p2)
			require.NoError(t, err)
			require.Len(t, resp.Items, 1)
			backendutils.AssertReportDataClusterAndReset(t, anotherClusterInfo.Cluster, &resp.Items[0])
			require.Equal(t, r2, resp.Items[0])
		})
	}
}

func TestReportDataFiltering(t *testing.T) {
	type testcase struct {
		Name    string
		Params  *v1.ReportDataParams
		Expect1 bool
		Expect2 bool
	}

	testcases := []testcase{
		{
			Name: "should filter reports based on ID",
			Params: &v1.ReportDataParams{
				ID: "report-id-1",
			},
			Expect1: true,
			Expect2: false,
		},
		{
			Name: "should filter reports based on name",
			Params: &v1.ReportDataParams{
				ReportMatches: []v1.ReportMatch{
					{
						ReportName: "report-name-1",
					},
				},
			},
			Expect1: true,
			Expect2: false,
		},
		{
			Name: "should filter reports based on type name",
			Params: &v1.ReportDataParams{
				ReportMatches: []v1.ReportMatch{
					{
						ReportTypeName: "report-type-2",
					},
				},
			},
			Expect1: false,
			Expect2: true,
		},
		{
			Name: "should filter reports based on report type and name",
			Params: &v1.ReportDataParams{
				ReportMatches: []v1.ReportMatch{
					{
						ReportName:     "report-name-2",
						ReportTypeName: "report-type-2",
					},
				},
			},
			Expect1: false,
			Expect2: true,
		},
		{
			Name: "should filter reports based on report type and name (no match)",
			Params: &v1.ReportDataParams{
				ReportMatches: []v1.ReportMatch{
					{
						// Should match neither.
						ReportName:     "report-name-1",
						ReportTypeName: "report-type-2",
					},
				},
			},
			Expect1: false,
			Expect2: false,
		},
		{
			Name: "should filter reports based on timestamp range",
			Params: &v1.ReportDataParams{
				QueryParams: v1.QueryParams{
					TimeRange: &lmav1.TimeRange{
						From: time.Unix(1000, 0),
						To:   time.Unix(3000, 0),
					},
				},
			},
			Expect1: false,
			Expect2: true,
		},
		{
			Name: "should filter reports based on end time",
			Params: &v1.ReportDataParams{
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
			Params: &v1.ReportDataParams{
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

				r1 := v1.ReportData{
					ID: "report-id-1",
					ReportData: &apiv3.ReportData{
						ReportName:     "report-name-1",
						ReportTypeName: "report-type-1",
						StartTime:      metav1.Time{Time: time.Unix(100, 0)},
						EndTime:        metav1.Time{Time: time.Unix(100, 0)},
						GenerationTime: metav1.Time{Time: time.Unix(100, 0)},
					},
				}
				r2 := v1.ReportData{
					ID: "report-id-2",
					ReportData: &apiv3.ReportData{
						ReportName:     "report-name-2",
						ReportTypeName: "report-type-2",
						StartTime:      metav1.Time{Time: time.Unix(2000, 0)},
						EndTime:        metav1.Time{Time: time.Unix(2000, 0)},
						GenerationTime: metav1.Time{Time: time.Unix(2000, 0)},
					},
				}

				response, err := rb.Create(ctx, clusterInfo, []v1.ReportData{r1, r2})
				require.NoError(t, err)
				require.Equal(t, []v1.BulkError(nil), response.Errors)
				require.Equal(t, 0, response.Failed)

				err = backendutils.RefreshIndex(ctx, client, rIndexGetter.Index(clusterInfo))
				require.NoError(t, err)

				resp, err := rb.List(ctx, clusterInfo, tc.Params)
				require.NoError(t, err)
				for i := range resp.Items {
					backendutils.AssertReportDataClusterAndReset(t, clusterInfo.Cluster, &resp.Items[i])
				}

				if tc.Expect1 {
					require.Contains(t, resp.Items, r1)
				} else {
					require.NotContains(t, resp.Items, r1)
				}
				if tc.Expect2 {
					require.Contains(t, resp.Items, r2)
				} else {
					require.NotContains(t, resp.Items, r2)
				}
			})
		}
	}
}

func TestReportDataSorting(t *testing.T) {
	RunAllModes(t, "should respect sorting", func(t *testing.T) {
		clusterInfo := bapi.ClusterInfo{Cluster: cluster1}

		t1 := time.Unix(100, 0)
		t2 := time.Unix(500, 0)

		r1 := v1.ReportData{
			ID: "report-id-1",
			ReportData: &apiv3.ReportData{
				ReportName:     "report-name-1",
				ReportTypeName: "report-type-1",
				StartTime:      metav1.Time{Time: t1},
				EndTime:        metav1.Time{Time: t1},
				GenerationTime: metav1.Time{Time: t1},
			},
		}
		r2 := v1.ReportData{
			ID: "report-id-2",
			ReportData: &apiv3.ReportData{
				ReportName:     "report-name-2",
				ReportTypeName: "report-type-2",
				StartTime:      metav1.Time{Time: t2},
				EndTime:        metav1.Time{Time: t2},
				GenerationTime: metav1.Time{Time: t2},
			},
		}

		_, err := rb.Create(ctx, clusterInfo, []v1.ReportData{r1, r2})
		require.NoError(t, err)

		err = backendutils.RefreshIndex(ctx, client, rIndexGetter.Index(clusterInfo))
		require.NoError(t, err)

		// Query for flow logs without sorting.
		params := v1.ReportDataParams{}
		r, err := rb.List(ctx, clusterInfo, &params)
		require.NoError(t, err)
		require.Len(t, r.Items, 2)
		require.Nil(t, r.AfterKey)
		for i := range r.Items {
			backendutils.AssertReportDataClusterAndReset(t, clusterInfo.Cluster, &r.Items[i])
		}

		// Assert that the logs are returned in the correct order.
		require.Equal(t, r1, r.Items[0])
		require.Equal(t, r2, r.Items[1])

		// Query again, this time sorting in order to get the logs in reverse order.
		params.Sort = []v1.SearchRequestSortBy{
			{
				Field:      "endTime",
				Descending: true,
			},
		}
		r, err = rb.List(ctx, clusterInfo, &params)
		require.NoError(t, err)
		require.Len(t, r.Items, 2)
		for i := range r.Items {
			backendutils.AssertReportDataClusterAndReset(t, clusterInfo.Cluster, &r.Items[i])
		}
		require.Equal(t, r2, r.Items[0])
		require.Equal(t, r1, r.Items[1])
	})
}
