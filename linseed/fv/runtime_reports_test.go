// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"context"
	"fmt"
	"testing"
	"time"

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

		require.Equal(t, []v1.RuntimeReport{{Tenant: "tenant-a", Cluster: cluster, Report: report}},
			testutils.AssertLogIDAndCopyRuntimeReportsWithoutThem(t, resp))
	})

	t.Run("should create and list runtime reports using legacy start_time", func(t *testing.T) {
		defer runtimeReportsSetupAndTeardown(t)()

		startTime := time.Unix(1, 0).UTC()
		endTime := time.Unix(2, 0).UTC()
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

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_runtime*")

		// Read it back.
		params := v1.RuntimeReportParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					// Populate with a bogus time that does not
					// match the report
					From: time.Unix(4, 0).UTC(),
					To:   time.Now(),
				},
			},
			LegacyTimeRange: &lmav1.TimeRange{
				From: startTime,
				To:   endTime,
			},
		}
		resp, err := cli.RuntimeReports("").List(ctx, &params)
		require.NoError(t, err)

		require.Len(t, resp.Items, 1)
		require.Equal(t, []v1.RuntimeReport{{Tenant: "tenant-a", Cluster: cluster, Report: runtimeReport}},
			testutils.AssertLogIDAndCopyRuntimeReportsWithoutThem(t, resp))
	})

	t.Run("should create and list runtime reports using legacy and generated start_time", func(t *testing.T) {
		defer runtimeReportsSetupAndTeardown(t)()

		newRuntimeReport := func(startTime time.Time, generatedTime *time.Time) v1.Report {
			return v1.Report{
				StartTime:     startTime,
				EndTime:       startTime,
				GeneratedTime: generatedTime,
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
		}

		newLegacyRuntimeReport := func(startTime time.Time) v1.Report {
			// A legacy runtime report is a report without generated_time
			return newRuntimeReport(startTime, nil)
		}

		// In this test we want to make sure we can construct a query that can
		// read reports using legacy and generated times.
		// With this query, we typically read legacy reports for a long duration
		// and a shorter one for reports using generate_time.

		now := time.Now().UTC()
		params := v1.RuntimeReportParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-2 * time.Minute).UTC(),
					To:   now,
				},
			},
			LegacyTimeRange: &lmav1.TimeRange{
				From: time.Now().Add(-25 * time.Minute).UTC(),
				To:   now,
			},
		}

		oldLegacyTime := time.Now().Add(-40 * time.Minute).UTC()
		validLegacyTime := time.Now().Add(-20 * time.Minute).UTC()
		oldGeneratedTime := time.Now().Add(-5 * time.Minute).UTC()
		validGeneratedTime := time.Now().Add(-30 * time.Second).UTC()

		oldLegacyRuntimeReport := newLegacyRuntimeReport(oldLegacyTime)
		legacyRuntimeReport := newLegacyRuntimeReport(validLegacyTime)
		oldRuntimeReport := newRuntimeReport(oldGeneratedTime, &oldGeneratedTime)
		runtimeReport := newRuntimeReport(validGeneratedTime, &validGeneratedTime)

		bulk, err := cli.RuntimeReports(cluster).Create(ctx, []v1.Report{oldLegacyRuntimeReport, legacyRuntimeReport, oldRuntimeReport, runtimeReport})
		require.NoError(t, err)
		require.Equal(t, bulk.Succeeded, 4, "create runtime reports did not succeed")

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_runtime*")

		// Read it back.
		resp, err := cli.RuntimeReports("").List(ctx, &params)
		require.NoError(t, err)

		require.Len(t, resp.Items, 2)

		require.Equal(t, []v1.RuntimeReport{
			{Tenant: "tenant-a", Cluster: cluster, Report: legacyRuntimeReport},
			{Tenant: "tenant-a", Cluster: cluster, Report: runtimeReport},
		},
			testutils.AssertLogIDAndCopyRuntimeReportsWithoutThem(t, resp))
	})

	t.Run("should support pagination", func(t *testing.T) {
		defer runtimeReportsSetupAndTeardown(t)()

		totalItems := 5

		// Create 5 runtime reports.
		referenceTime := time.Unix(1, 0).UTC()
		for i := 0; i < totalItems; i++ {
			logTime := referenceTime.Add(time.Duration(i) * time.Second) // Make sure reports are ordered.
			reports := []v1.Report{
				{
					GeneratedTime: &logTime,
					Host:          fmt.Sprintf("%d", i),
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

			logTime := referenceTime.Add(time.Duration(i) * time.Second)
			require.EqualValues(t, []v1.RuntimeReport{
				{
					Cluster: cluster,
					Tenant:  "tenant-a",
					Report: v1.Report{
						GeneratedTime: &logTime,
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

		logTime := referenceTime.Add(time.Duration(lastItem) * time.Second)
		require.EqualValues(t, []v1.RuntimeReport{
			{
				Cluster: cluster,
				Tenant:  "tenant-a",
				Report: v1.Report{
					GeneratedTime: &logTime,
					Host:          fmt.Sprintf("%d", lastItem),
				},
			},
		}, testutils.AssertLogIDAndCopyRuntimeReportsWithoutThem(t, resp),
			fmt.Sprintf("RuntimeReport #%d did not match", lastItem))

		// Once we reach the end of the data, we should not receive
		// an afterKey
		require.Nil(t, resp.AfterKey)
	})

	t.Run("should read data for multiple clusters", func(t *testing.T) {
		defer runtimeReportsSetupAndTeardown(t)()

		startTime := time.Unix(1, 0).UTC()
		endTime := time.Unix(1, 0).UTC()
		generatedTime := time.Unix(2, 0).UTC()
		// Create a basic runtime report
		runtimeReport := v1.Report{
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
					From: generatedTime,
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
