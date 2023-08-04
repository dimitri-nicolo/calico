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

var anotherCluster string

func runtimeReportsSetupAndTeardown(t *testing.T) func() {
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
	anotherCluster = testutils.RandomClusterName()

	// Set up context with a timeout.
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)

	return func() {
		// Cleanup indices created by the test.
		testutils.CleanupIndices(context.Background(), esClient, cluster)
		testutils.CleanupIndices(context.Background(), esClient, anotherCluster)
		logCancel()
		cancel()
	}
}

func TestFV_RuntimeReports(t *testing.T) {
	t.Run("should return an empty list if there are no runtime reports", func(t *testing.T) {
		defer runtimeReportsSetupAndTeardown(t)()

		params := v1.RuntimeReportParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now(),
				},
			},
		}

		// Perform a query.
		runtimeReports, err := cli.RuntimeReports("").List(ctx, &params)
		require.NoError(t, err)
		require.Equal(t, []v1.RuntimeReport{}, runtimeReports.Items)
	})

	t.Run("should create and list runtime reports using generated time", func(t *testing.T) {
		defer runtimeReportsSetupAndTeardown(t)()

		startTime := time.Unix(1, 0).UTC()
		endTime := time.Unix(1, 0).UTC()
		generatedTime := time.Unix(2, 2).UTC()
		// Create a basic runtime report
		report := v1.Report{
			// Note, Linseed will overwrite GeneratedTime with the current time when
			// Create is called.
			GeneratedTime: &generatedTime,
			StartTime:     startTime,
			EndTime:       endTime,
			Host:          "any-host",
			Count:         1,
			Type:          "ProcessStart",
			ConfigName:    "malware-protection",
			Pod: v1.PodInfo{
				Name:          "app",
				NameAggr:      "app-*",
				Namespace:     "default",
				ContainerName: "app",
			},
			File: v1.File{
				Path:     "/usr/sbin/runc",
				HostPath: "/run/docker/runtime-runc/moby/48f10a5eb9a245e6890433205053ba4e72c8e3bab5c13c2920dc32fadd7290cd/runc.rB3K51",
			},
			ProcessStart: v1.ProcessStart{
				Invocation: "runc --root /var/run/docker/runtime-runc/moby",
				Hashes: v1.ProcessHashes{
					MD5:    "MD5",
					SHA1:   "SHA1",
					SHA256: "SHA256",
				},
			},
			FileAccess: v1.FileAccess{},
		}
		bulk, err := cli.RuntimeReports(cluster).Create(ctx, []v1.Report{report})
		require.NoError(t, err)
		require.Equal(t, bulk.Succeeded, 1, "create runtime reports did not succeed")

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_runtime*")

		// Read it back.
		params := v1.RuntimeReportParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: generatedTime,
					To:   time.Now(),
				},
			},
		}
		resp, err := cli.RuntimeReports("").List(ctx, &params)
		require.NoError(t, err)

		require.Len(t, resp.Items, 1)
		// Linseed overwrote the GeneratedTime field.  We can't predict it exactly, so copy
		// it across from the actual to our expected value.
		require.NotNil(t, resp.Items[0].Report.GeneratedTime)
		report.GeneratedTime = resp.Items[0].Report.GeneratedTime
		require.Equal(t, []v1.RuntimeReport{{Tenant: "tenant-a", Cluster: cluster, Report: report}},
			testutils.AssertLogIDAndCopyRuntimeReportsWithoutThem(t, resp))
	})

	// Since this test case was first written, Linseed now populates the GeneratedTime field whenever it writes a new
	// runtime report into Elasticsearch.  So it is now impossible for Linseed to write a "legacy" report in such a way
	// that it stays "legacy" (i.e. without a GeneratedTime value).  However we must still test Linseed's ability to
	// read any pre-existing reports from ES that were ingested by an older Linseed version (or es-gateway) and that
	// don't have GeneratedTime values.  In order to do that we use the LMA client to write legacy reports.
	t.Run("should create and list runtime reports using legacy and generated start_time", func(t *testing.T) {
		defer runtimeReportsSetupAndTeardown(t)()

		newRuntimeReport := func(startTime time.Time) v1.Report {
			return v1.Report{
				StartTime:  startTime,
				EndTime:    startTime,
				Host:       "any-host",
				Count:      1,
				Type:       "ProcessStart",
				ConfigName: "malware-protection",
				Pod: v1.PodInfo{
					Name:          "app",
					NameAggr:      "app-*",
					Namespace:     "default",
					ContainerName: "app",
				},
				File: v1.File{
					Path:     "/usr/sbin/runc",
					HostPath: "/run/docker/runtime-runc/moby/48f10a5eb9a245e6890433205053ba4e72c8e3bab5c13c2920dc32fadd7290cd/runc.rB3K51",
				},
				ProcessStart: v1.ProcessStart{
					Invocation: "runc --root /var/run/docker/runtime-runc/moby",
					Hashes: v1.ProcessHashes{
						MD5:    "MD5",
						SHA1:   "SHA1",
						SHA256: "SHA256",
					},
				},
				FileAccess: v1.FileAccess{},
			}
		}

		expiredLegacyRuntimeReport := newRuntimeReport(time.Now().Add(-40 * time.Minute).UTC())
		currentLegacyRuntimeReport := newRuntimeReport(time.Now().Add(-20 * time.Minute).UTC())
		expiredRuntimeReport := newRuntimeReport(time.Now().Add(-5 * time.Minute).UTC())
		currentRuntimeReport := newRuntimeReport(time.Now().Add(-30 * time.Second).UTC())

		// Use Linseed to write the first non-legacy report.
		bulk, err := cli.RuntimeReports(cluster).Create(ctx, []v1.Report{expiredRuntimeReport})
		require.NoError(t, err)
		require.Equal(t, bulk.Succeeded, 1, "create runtime reports did not succeed")

		// Wait a second, because ES query times only have per-second precision.  (The ES docs seem to be to indicate
		// per-millisecond precision, but here's what the actual JSON captured with tcpdump says:
		//     ...{"range":{"generated_time":{"from":"2023-08-08T17:19:39Z",...
		time.Sleep((11 * time.Second) / 10)

		// Remember the time now, as we want to use this in the query and expect that it will not return that first
		// non-legacy report.
		queryTime := time.Now().UTC()

		// Use Linseed to write the second non-legacy report.
		bulk, err = cli.RuntimeReports(cluster).Create(ctx, []v1.Report{currentRuntimeReport})
		require.NoError(t, err)
		require.Equal(t, bulk.Succeeded, 1, "create runtime reports did not succeed")

		// Use LMA client to write the legacy reports.
		put, err := lmaClient.Backend().Index().Index("tigera_secure_ee_runtime.tenant-a." + cluster + ".").BodyJson(expiredLegacyRuntimeReport).Do(ctx)
		require.NoError(t, err)
		logrus.Infof("first legacy write: %#v", put)
		put, err = lmaClient.Backend().Index().Index("tigera_secure_ee_runtime.tenant-a." + cluster + ".").BodyJson(currentLegacyRuntimeReport).Do(ctx)
		require.NoError(t, err)
		logrus.Infof("second legacy write: %#v", put)

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_runtime*")

		// Read back, using a query that can read reports using legacy and generated times.
		// We mimic Sasha behaviour by reading legacy reports with a long lookback
		// and GeneratedTime reports with a shorter lookback.
		now := time.Now().UTC()
		resp, err := cli.RuntimeReports("").List(ctx, &v1.RuntimeReportParams{
			// Expect this to return the "current" non-legacy report but not the "expired" one.
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: queryTime,
					To:   now,
				},
			},
			// Expect this to return the "current" legacy report but not the "expired" one.
			LegacyTimeRange: &lmav1.TimeRange{
				From: time.Now().Add(-25 * time.Minute).UTC(),
				To:   now,
			},
		})
		require.NoError(t, err)

		require.Len(t, resp.Items, 2)

		// Linseed populated the GeneratedTime field of the non-legacy report.  We can't predict it exactly, so copy it
		// across from the actual to our expected value.
		nonLegacyIndex := 0
		if resp.Items[0].Report.GeneratedTime == nil {
			nonLegacyIndex = 1
		}
		require.NotNil(t, resp.Items[nonLegacyIndex].Report.GeneratedTime)
		require.Nil(t, resp.Items[1-nonLegacyIndex].Report.GeneratedTime)
		currentRuntimeReport.GeneratedTime = resp.Items[nonLegacyIndex].Report.GeneratedTime

		require.Equal(t, []v1.RuntimeReport{
			{Tenant: "tenant-a", Cluster: cluster, Report: currentLegacyRuntimeReport},
			{Tenant: "tenant-a", Cluster: cluster, Report: currentRuntimeReport},
		},
			testutils.AssertLogIDAndCopyRuntimeReportsWithoutThem(t, resp))
	})

	t.Run("should support pagination", func(t *testing.T) {
		defer runtimeReportsSetupAndTeardown(t)()

		totalItems := 5

		// Create 5 runtime reports.
		referenceTime := time.Unix(1, 0).UTC()
		for i := 0; i < totalItems; i++ {
			reports := []v1.Report{
				{
					Host: fmt.Sprintf("%d", i),
				},
			}
			bulk, err := cli.RuntimeReports(cluster).Create(ctx, reports)
			require.NoError(t, err)
			require.Equal(t, bulk.Succeeded, 1, "create runtime report did not succeed")
		}

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_runtime*")

		// Iterate through the first 4 pages and check they are correct.
		var afterKey map[string]interface{}
		for i := 0; i < totalItems-1; i++ {
			params := v1.RuntimeReportParams{
				QueryParams: v1.QueryParams{
					TimeRange: &lmav1.TimeRange{
						From: referenceTime,
						To:   time.Now(),
					},
					MaxPageSize: 1,
					AfterKey:    afterKey,
				},
			}
			resp, err := cli.RuntimeReports("").List(ctx, &params)
			require.NoError(t, err)
			require.Equal(t, 1, len(resp.Items))

			require.NotNil(t, resp.Items[0].Report.GeneratedTime)
			require.EqualValues(t, []v1.RuntimeReport{
				{
					Cluster: cluster,
					Tenant:  "tenant-a",
					Report: v1.Report{
						GeneratedTime: resp.Items[0].Report.GeneratedTime,
						Host:          fmt.Sprintf("%d", i),
					},
				},
			}, testutils.AssertLogIDAndCopyRuntimeReportsWithoutThem(t, resp),
				fmt.Sprintf("RuntimeReport #%d did not match", i))
			require.NotNil(t, resp.AfterKey)
			require.Contains(t, resp.AfterKey, "startFrom")
			require.Equal(t, resp.AfterKey["startFrom"], float64(i+1))

			// Use the afterKey for the next query.
			afterKey = resp.AfterKey
		}

		// If we query once more, we should get the last page, and no afterkey, since
		// we have paged through all the items.
		lastItem := totalItems - 1
		params := v1.RuntimeReportParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: referenceTime,
					To:   time.Now(),
				},
				MaxPageSize: 1,
				AfterKey:    afterKey,
			},
		}
		resp, err := cli.RuntimeReports("").List(ctx, &params)
		require.NoError(t, err)
		require.Equal(t, 1, len(resp.Items))

		require.NotNil(t, resp.Items[0].Report.GeneratedTime)
		require.EqualValues(t, []v1.RuntimeReport{
			{
				Cluster: cluster,
				Tenant:  "tenant-a",
				Report: v1.Report{
					GeneratedTime: resp.Items[0].Report.GeneratedTime,
					Host:          fmt.Sprintf("%d", lastItem),
				},
			},
		}, testutils.AssertLogIDAndCopyRuntimeReportsWithoutThem(t, resp),
			fmt.Sprintf("RuntimeReport #%d did not match", lastItem))

		// Once we reach the end of the data, we should not receive
		// an afterKey
		require.Nil(t, resp.AfterKey)
	})

	t.Run("should support pagination for items >= 10000 for runtime reports", func(t *testing.T) {
		defer runtimeReportsSetupAndTeardown(t)()

		totalItems := 10001
		// Create > 10K runtime reports.
		referenceTime := time.Now().UTC()
		var reports []v1.Report
		for i := 0; i < totalItems; i++ {
			reports = append(reports, v1.Report{
				Host: fmt.Sprintf("%d", i),
			},
			)
		}
		bulk, err := cli.RuntimeReports(cluster).Create(ctx, reports)
		require.NoError(t, err)
		require.Equal(t, totalItems, bulk.Total, "create reports did not succeed")

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_runtime*")

		// Stream through all the items.
		params := v1.RuntimeReportParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: referenceTime,
					To:   time.Now(),
				},
				MaxPageSize: 1000,
			},
		}

		pager := client.NewListPager[v1.RuntimeReport](&params)
		pages, errors := pager.Stream(ctx, cli.RuntimeReports(cluster).List)

		receivedItems := 0
		for page := range pages {
			receivedItems = receivedItems + len(page.Items)
		}

		if err, ok := <-errors; ok {
			require.NoError(t, err)
		}

		require.Equal(t, receivedItems, totalItems)
	})

	t.Run("should read data for multiple clusters", func(t *testing.T) {
		defer runtimeReportsSetupAndTeardown(t)()

		startTime := time.Unix(1, 0).UTC()
		endTime := time.Unix(1, 0).UTC()

		// Create a basic runtime report
		runtimeReport := v1.Report{
			StartTime:  startTime,
			EndTime:    endTime,
			Host:       "any-host",
			Count:      1,
			Type:       "ProcessStart",
			ConfigName: "malware-protection",
			Pod: v1.PodInfo{
				Name:          "app",
				NameAggr:      "app-*",
				Namespace:     "default",
				ContainerName: "app",
			},
			File: v1.File{
				Path:     "/usr/sbin/runc",
				HostPath: "/run/docker/runtime-runc/moby/48f10a5eb9a245e6890433205053ba4e72c8e3bab5c13c2920dc32fadd7290cd/runc.rB3K51",
			},
			ProcessStart: v1.ProcessStart{
				Invocation: "runc --root /var/run/docker/runtime-runc/moby",
				Hashes: v1.ProcessHashes{
					MD5:    "MD5",
					SHA1:   "SHA1",
					SHA256: "SHA256",
				},
			},
			FileAccess: v1.FileAccess{},
		}
		bulk, err := cli.RuntimeReports(cluster).Create(ctx, []v1.Report{runtimeReport})
		require.NoError(t, err)
		require.Equal(t, bulk.Succeeded, 1, "create runtime reports did not succeed")

		bulk, err = cli.RuntimeReports(anotherCluster).Create(ctx, []v1.Report{runtimeReport})
		require.NoError(t, err)
		require.Equal(t, bulk.Succeeded, 1, "create runtime reports did not succeed")

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_runtime*")

		// Read it back.
		params := v1.RuntimeReportParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: startTime,
					To:   time.Now(),
				},
			},
		}
		resp, err := cli.RuntimeReports("").List(ctx, &params)
		require.NoError(t, err)

		require.Len(t, resp.Items, 2)

		// Validate that the received reports come from two clusters
		var clusters []string
		for _, item := range resp.Items {
			// Validate that the source is populated
			require.NotEmpty(t, item.Cluster)
			clusters = append(clusters, item.Cluster)
			// Validate that the id is populated
			item = testutils.AssertRuntimeReportIDAndReset(t, item)
			// Validate that the rest of the fields are populated
			require.NotNil(t, item.Report.GeneratedTime)
			runtimeReport.GeneratedTime = item.Report.GeneratedTime
			require.Equal(t, runtimeReport, item.Report)
		}

		require.Contains(t, clusters, cluster)
		require.Contains(t, clusters, anotherCluster)
	})
}

func TestFV_RuntimeReportTenancy(t *testing.T) {
	t.Run("should support tenancy restriction", func(t *testing.T) {
		defer runtimeReportsSetupAndTeardown(t)()

		// Instantiate a client for an unexpected tenant.
		tenantCLI, err := NewLinseedClientForTenant("bad-tenant")
		require.NoError(t, err)

		// Create a basic entry. We expect this to fail, since we're using
		// an unexpected tenant ID on the request.
		startTime := time.Unix(1, 0).UTC()
		endTime := time.Unix(1, 0).UTC()
		generatedTime := time.Unix(2, 2).UTC()
		// Create a basic runtime report
		report := v1.Report{
			GeneratedTime: &generatedTime,
			StartTime:     startTime,
			EndTime:       endTime,
			Host:          "any-host",
			Count:         1,
			Type:          "ProcessStart",
			ConfigName:    "malware-protection",
			Pod: v1.PodInfo{
				Name:          "app",
				NameAggr:      "app-*",
				Namespace:     "default",
				ContainerName: "app",
			},
			File: v1.File{
				Path:     "/usr/sbin/runc",
				HostPath: "/run/docker/runtime-runc/moby/48f10a5eb9a245e6890433205053ba4e72c8e3bab5c13c2920dc32fadd7290cd/runc.rB3K51",
			},
			ProcessStart: v1.ProcessStart{
				Invocation: "runc --root /var/run/docker/runtime-runc/moby",
				Hashes: v1.ProcessHashes{
					MD5:    "MD5",
					SHA1:   "SHA1",
					SHA256: "SHA256",
				},
			},
			FileAccess: v1.FileAccess{},
		}
		bulk, err := tenantCLI.RuntimeReports(cluster).Create(ctx, []v1.Report{report})
		require.ErrorContains(t, err, "Bad tenant identifier")
		require.Nil(t, bulk)

		// Try a read as well.
		params := v1.RuntimeReportParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: generatedTime,
					To:   time.Now(),
				},
			},
		}
		resp, err := tenantCLI.RuntimeReports("").List(ctx, &params)
		require.ErrorContains(t, err, "Bad tenant identifier")
		require.Nil(t, resp)
	})
}
