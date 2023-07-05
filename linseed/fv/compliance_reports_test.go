// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/projectcalico/calico/linseed/pkg/client"

	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"

	elastic "github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/config"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

func complianceSetupAndTeardown(t *testing.T) func() {
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

func TestFV_ComplianceReports(t *testing.T) {
	t.Run("should return an empty list if there are no reports", func(t *testing.T) {
		defer complianceSetupAndTeardown(t)()

		params := v1.ReportDataParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now(),
				},
			},
		}

		// Perform a query.
		reports, err := cli.Compliance(cluster).ReportData().List(ctx, &params)
		require.NoError(t, err)
		require.Equal(t, []v1.ReportData{}, reports.Items)
	})

	t.Run("should create and list reports", func(t *testing.T) {
		defer complianceSetupAndTeardown(t)()

		// Create a basic report.
		v3r := apiv3.ReportData{
			ReportName:     "test-report",
			ReportTypeName: "my-report-type",
			StartTime:      metav1.Time{Time: time.Unix(1, 0)},
			EndTime:        metav1.Time{Time: time.Unix(2, 0)},
			GenerationTime: metav1.Time{Time: time.Unix(3, 0)},
		}
		report := v1.ReportData{ReportData: &v3r}
		reports := []v1.ReportData{report}
		bulk, err := cli.Compliance(cluster).ReportData().Create(ctx, reports)
		require.NoError(t, err)
		require.Equal(t, bulk.Succeeded, 1, "create did not succeed")

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_compliance_reports*")

		// Read it back.
		params := v1.ReportDataParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Unix(0, 0),
					To:   time.Unix(4, 0),
				},
			},
		}
		resp, err := cli.Compliance(cluster).ReportData().List(ctx, &params)
		require.NoError(t, err)

		// The ID should be set.
		require.Len(t, resp.Items, 1)
		require.Equal(t, report.UID(), resp.Items[0].ID)
		resp.Items[0].ID = ""
		require.Equal(t, reports, resp.Items)
	})

	t.Run("should support pagination", func(t *testing.T) {
		defer complianceSetupAndTeardown(t)()

		totalItems := 5

		// Create 5 Snapshots.
		logTime := time.Unix(100, 0).UTC()
		for i := 0; i < totalItems; i++ {
			reports := []v1.ReportData{
				{
					ReportData: &apiv3.ReportData{
						ReportName:     fmt.Sprintf("test-report-%d", i),
						ReportTypeName: "my-report-type",
						StartTime:      metav1.Time{Time: logTime.Add(time.Duration(i) * time.Second).UTC()},
						EndTime:        metav1.Time{Time: logTime.Add(time.Duration(i+1) * time.Second).UTC()},
						GenerationTime: metav1.Time{Time: logTime.Add(time.Duration(i+2) * time.Second).UTC()},
					},
				},
			}
			bulk, err := cli.Compliance(cluster).ReportData().Create(ctx, reports)
			require.NoError(t, err)
			require.Equal(t, bulk.Succeeded, 1, "create reports did not succeed")
		}

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_compliance_reports*")

		// Iterate through the first 4 pages and check they are correct.
		var afterKey map[string]interface{}
		for i := 0; i < totalItems-1; i++ {
			params := v1.ReportDataParams{
				QueryParams: v1.QueryParams{
					TimeRange: &lmav1.TimeRange{
						From: logTime.Add(-20 * time.Second),
						To:   logTime.Add(20 * time.Second),
					},
					MaxPageSize: 1,
					AfterKey:    afterKey,
				},
			}
			resp, err := cli.Compliance(cluster).ReportData().List(ctx, &params)
			require.NoError(t, err)
			require.Equal(t, 1, len(resp.Items))
			require.Equal(t, []v1.ReportData{
				{
					ReportData: &apiv3.ReportData{
						ReportName:     fmt.Sprintf("test-report-%d", i),
						ReportTypeName: "my-report-type",
						StartTime:      metav1.Time{Time: logTime.Add(time.Duration(i) * time.Second).UTC()},
						EndTime:        metav1.Time{Time: logTime.Add(time.Duration(i+1) * time.Second).UTC()},
						GenerationTime: metav1.Time{Time: logTime.Add(time.Duration(i+2) * time.Second).UTC()},
					},
				},
			}, reportsWithUTCTime(resp), fmt.Sprintf("Reports #%d did not match", i))
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
		params := v1.ReportDataParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: logTime.Add(-20 * time.Second),
					To:   logTime.Add(20 * time.Second),
				},
				MaxPageSize: 1,
				AfterKey:    afterKey,
			},
		}
		resp, err := cli.Compliance(cluster).ReportData().List(ctx, &params)
		require.NoError(t, err)
		require.Equal(t, 1, len(resp.Items))
		require.Equal(t, []v1.ReportData{
			{
				ReportData: &apiv3.ReportData{
					ReportName:     fmt.Sprintf("test-report-%d", lastItem),
					ReportTypeName: "my-report-type",
					StartTime:      metav1.Time{Time: logTime.Add(time.Duration(lastItem) * time.Second).UTC()},
					EndTime:        metav1.Time{Time: logTime.Add(time.Duration(lastItem+1) * time.Second).UTC()},
					GenerationTime: metav1.Time{Time: logTime.Add(time.Duration(lastItem+2) * time.Second).UTC()},
				},
			},
		}, reportsWithUTCTime(resp), fmt.Sprintf("Reports #%d did not match", lastItem))
		require.Equal(t, resp.TotalHits, int64(totalItems))

		// Once we reach the end of the data, we should not receive
		// an afterKey
		require.Nil(t, resp.AfterKey)
	})

	t.Run("should support pagination for items >= 10000 for Reports", func(t *testing.T) {
		defer complianceSetupAndTeardown(t)()

		totalItems := 10001
		// Create > 10K reports.
		logTime := time.Unix(100, 0).UTC()
		var reports []v1.ReportData
		for i := 0; i < totalItems; i++ {
			reports = append(reports,
				v1.ReportData{
					ReportData: &apiv3.ReportData{
						ReportName:     fmt.Sprintf("test-report-%d", i),
						ReportTypeName: "my-report-type",
						StartTime:      metav1.Time{Time: logTime.Add(time.Duration(i) * time.Second).UTC()},
						EndTime:        metav1.Time{Time: logTime.Add(time.Duration(i+1) * time.Second).UTC()},
						GenerationTime: metav1.Time{Time: logTime.Add(time.Duration(i+2) * time.Second).UTC()},
					},
				},
			)
		}
		bulk, err := cli.Compliance(cluster).ReportData().Create(ctx, reports)
		require.NoError(t, err)
		require.Equal(t, totalItems, bulk.Succeeded, "create reports did not succeed")

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_compliance_reports*")

		// Stream through all the items.
		params := v1.ReportDataParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: logTime.Add(-5 * time.Second),
					To:   logTime.Add(time.Duration(totalItems) * time.Second),
				},
				MaxPageSize: 1000,
			},
		}

		pager := client.NewListPager[v1.ReportData](&params)
		pages, errors := pager.Stream(ctx, cli.Compliance(cluster).ReportData().List)

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

func TestFV_ComplianceReportsTenancy(t *testing.T) {
	t.Run("should support tenancy restriction", func(t *testing.T) {
		defer complianceSetupAndTeardown(t)()

		// Instantiate a client for an unexpected tenant.
		tenantCLI, err := NewLinseedClientForTenant("bad-tenant")
		require.NoError(t, err)

		// Create a basic log. We expect this to fail, since we're using
		// an unexpected tenant ID on the request.
		v3r := apiv3.ReportData{
			ReportName:     "test-report",
			ReportTypeName: "my-report-type",
			StartTime:      metav1.Time{Time: time.Unix(1, 0)},
			EndTime:        metav1.Time{Time: time.Unix(2, 0)},
			GenerationTime: metav1.Time{Time: time.Unix(3, 0)},
		}
		report := v1.ReportData{ReportData: &v3r}
		reports := []v1.ReportData{report}
		bulk, err := tenantCLI.Compliance(cluster).ReportData().Create(ctx, reports)
		require.ErrorContains(t, err, "Bad tenant identifier")
		require.Nil(t, bulk)

		// Try a read as well.
		params := v1.ReportDataParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Unix(0, 0),
					To:   time.Unix(4, 0),
				},
			},
		}
		resp, err := tenantCLI.Compliance(cluster).ReportData().List(ctx, &params)
		require.ErrorContains(t, err, "Bad tenant identifier")
		require.Nil(t, resp)
	})
}

func reportsWithUTCTime(resp *v1.List[v1.ReportData]) []v1.ReportData {
	for idx, report := range resp.Items {
		utcStartTime := report.StartTime.UTC()
		utcEndTime := report.EndTime.UTC()
		utcGenTime := report.GenerationTime.UTC()
		resp.Items[idx].StartTime = metav1.Time{Time: utcStartTime}
		resp.Items[idx].EndTime = metav1.Time{Time: utcEndTime}
		resp.Items[idx].GenerationTime = metav1.Time{Time: utcGenTime}
		resp.Items[idx].ID = ""
	}
	return resp.Items
}
