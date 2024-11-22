// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package runtime_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/index"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/runtime"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/templates"
	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"
	backendutils "github.com/projectcalico/calico/linseed/pkg/backend/testutils"
	"github.com/projectcalico/calico/linseed/pkg/config"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

var (
	client        lmaelastic.Client
	b             bapi.RuntimeBackend
	ctx           context.Context
	cluster       string
	tenant        string
	anotherTenant string
	indexGetter   bapi.Index
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
	esClient, err := backendutils.CreateElasticClient()

	require.NoError(t, err)
	client = lmaelastic.NewWithClient(esClient)
	cache := templates.NewCachedInitializer(client, 1, 0)

	// Instantiate a backend.
	if singleIndex {
		indexGetter = index.RuntimeReportsIndex()
		b = runtime.NewSingleIndexBackend(client, cache, 10000)
	} else {
		b = runtime.NewBackend(client, cache, 10000)
		indexGetter = index.RuntimeReportMultiIndex
	}

	// Create a random cluster name for each test to make sure we don't
	// interfere between tests.
	cluster = testutils.RandomClusterName()
	tenant = testutils.RandomTenantName()
	anotherTenant = testutils.RandomTenantName()

	// Each test should take less than 60 seconds.
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)

	// Function contains teardown logic.
	return func() {
		err = testutils.CleanupIndices(context.Background(), esClient, singleIndex, indexGetter, bapi.ClusterInfo{Cluster: cluster})
		require.NoError(t, err)

		// Cancel the context
		cancel()
		logCancel()
	}
}

// TestCreateRuntimeReport tests running a real elasticsearch query to create a runtime report.
func TestCreateRuntimeReport(t *testing.T) {
	RunAllModes(t, "TestCreateRuntimeReport", func(t *testing.T) {
		clusterInfo := bapi.ClusterInfo{Cluster: cluster}

		startTime := time.Unix(1, 0).UTC()
		endTime := time.Unix(2, 0).UTC()
		generatedTime := time.Unix(3, 0).UTC()
		f := v1.Report{
			// Note, GeneratedTime not specified; Linseed will populate it.
			StartTime:  startTime,
			EndTime:    endTime,
			Host:       "host",
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
					MD5:    "",
					SHA1:   "",
					SHA256: "SHA256",
				},
			},
			FileAccess: v1.FileAccess{},
		}

		// Create the runtime report in ES.
		resp, err := b.Create(ctx, clusterInfo, []v1.Report{f})
		require.NoError(t, err)
		require.Equal(t, []v1.BulkError(nil), resp.Errors)
		require.Equal(t, 1, resp.Total)
		require.Equal(t, 0, resp.Failed)
		require.Equal(t, 1, resp.Succeeded)

		// Refresh the index.
		err = testutils.RefreshIndex(ctx, client, indexGetter.Index(clusterInfo))
		require.NoError(t, err)

		// Query using normal time range.
		opts := &v1.RuntimeReportParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: generatedTime,
					To:   time.Now(),
				},
			},
		}
		results, err := b.List(ctx, clusterInfo, opts)
		require.NoError(t, err)
		require.Equal(t, 1, len(results.Items))
		sanitized := testutils.AssertLogIDAndCopyRuntimeReportsWithoutThem(t, results)
		// Linseed populated the GeneratedTime field.  We can't predict it exactly, so copy it
		// across from the actual to our expected value.
		require.NotNil(t, sanitized[0].Report.GeneratedTime)
		f.GeneratedTime = sanitized[0].Report.GeneratedTime
		require.Equal(t, []v1.RuntimeReport{{Tenant: "", Cluster: cluster, Report: f}}, sanitized)
	})
}

// TestCreateRuntimeReport tests running a real elasticsearch query to create a runtime report.
func TestCreateRuntimeReportForMultipleTenants(t *testing.T) {
	RunAllModes(t, "TestCreateRuntimeReportForMultipleTenants", func(t *testing.T) {
		startTime := time.Unix(1, 0).UTC()
		endTime := time.Unix(2, 0).UTC()
		generatedTime := time.Unix(3, 0).UTC()
		f := v1.Report{
			// Note, GeneratedTime not specified; Linseed will populate it.
			StartTime:  startTime,
			EndTime:    endTime,
			Host:       "host",
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
					MD5:    "",
					SHA1:   "",
					SHA256: "SHA256",
				},
			},
			FileAccess: v1.FileAccess{},
		}

		// Create the runtime report in ES.
		clusterInfoA := bapi.ClusterInfo{Cluster: cluster, Tenant: tenant}
		resp, err := b.Create(ctx, clusterInfoA, []v1.Report{f})
		require.NoError(t, err)
		require.Equal(t, []v1.BulkError(nil), resp.Errors)
		require.Equal(t, 1, resp.Total)
		require.Equal(t, 0, resp.Failed)
		require.Equal(t, 1, resp.Succeeded)

		clusterInfoB := bapi.ClusterInfo{Cluster: cluster, Tenant: anotherTenant}
		resp, err = b.Create(ctx, clusterInfoB, []v1.Report{f})
		require.NoError(t, err)
		require.Equal(t, []v1.BulkError(nil), resp.Errors)
		require.Equal(t, 1, resp.Total)
		require.Equal(t, 0, resp.Failed)
		require.Equal(t, 1, resp.Succeeded)

		// Refresh the index.
		err = testutils.RefreshIndex(ctx, client, indexGetter.Index(clusterInfoA))
		require.NoError(t, err)
		err = testutils.RefreshIndex(ctx, client, indexGetter.Index(clusterInfoB))
		require.NoError(t, err)

		// Read data and verify for tenant A
		results, err := b.List(ctx, clusterInfoA, &v1.RuntimeReportParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: generatedTime,
					To:   time.Now(),
				},
			},
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(results.Items))
		// Linseed populated the GeneratedTime field.  We can't predict it exactly, so copy it
		// across from the actual to our expected value.
		require.NotNil(t, results.Items[0].Report.GeneratedTime)
		f.GeneratedTime = results.Items[0].Report.GeneratedTime
		require.Equal(t, []v1.RuntimeReport{{Tenant: tenant, Cluster: cluster, Report: f}},
			testutils.AssertLogIDAndCopyRuntimeReportsWithoutThem(t, results))

		// Read data and verify for tenant B
		results, err = b.List(ctx, clusterInfoB, &v1.RuntimeReportParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: generatedTime,
					To:   time.Now(),
				},
			},
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(results.Items))
		// Linseed populated the GeneratedTime field.  We can't predict it exactly, so copy it
		// across from the actual to our expected value.
		require.NotNil(t, results.Items[0].Report.GeneratedTime)
		f.GeneratedTime = results.Items[0].Report.GeneratedTime
		require.Equal(t, []v1.RuntimeReport{{Tenant: anotherTenant, Cluster: cluster, Report: f}},
			testutils.AssertLogIDAndCopyRuntimeReportsWithoutThem(t, results))
	})
}

func TestRuntimeSelection(t *testing.T) {
	timeA := time.Unix(9564, 0).UTC()
	timeB := timeA.Add(time.Minute)
	reports := []v1.Report{
		{
			StartTime: timeA,
			EndTime:   timeB,
			Host:      "host",
			Count:     2,
			Type:      "ProcessStart",
			Pod: v1.PodInfo{
				Name:          "drax",
				NameAggr:      "drax-*",
				Namespace:     "moonraker",
				ContainerName: "diamond",
			},
			File: v1.File{
				Path:     "/usr/sbin/runc",
				HostPath: "/run/docker/runtime-runc/moby/48f10a5eb9a245e6890433205053ba4e72c8e3bab5c13c2920dc32fadd7290cd/runc.rB3K51",
			},
			ProcessStart: v1.ProcessStart{
				Invocation: "runc --root /var/run/docker/runtime-runc/moby",
				Hashes: v1.ProcessHashes{
					MD5:    "",
					SHA1:   "",
					SHA256: "SHA256",
				},
			},
		},
		{
			StartTime: timeA,
			EndTime:   timeB,
			Host:      "host",
			Count:     10,
			Type:      "ProcessStart",
			Pod: v1.PodInfo{
				Name:          "goldfinger",
				NameAggr:      "goldfinger-*",
				Namespace:     "fortknox",
				ContainerName: "gold",
			},
			File: v1.File{
				Path:     "/usr/sbin/laser",
				HostPath: "/run/docker/runtime-runc/moby/48f10a5eb9a245e6890433205053ba4e72c8e3bab5c13c2920dc32fadd7290cd/laser.rB3K51",
			},
			ProcessStart: v1.ProcessStart{
				Invocation: "runc --root /var/run/docker/runtime-runc/laser",
				Hashes: v1.ProcessHashes{
					MD5:    "",
					SHA1:   "",
					SHA256: "Laser-SHA256",
				},
			},
		},
		{
			StartTime: timeA,
			EndTime:   timeB,
			Host:      "host",
			Count:     1,
			Type:      "ProcessStart",
			Pod: v1.PodInfo{
				Name:          "blofeld",
				NameAggr:      "blofeld-*",
				Namespace:     "spectre",
				ContainerName: "fur",
			},
			File: v1.File{
				Path:     "/usr/sbin/ski",
				HostPath: "/run/docker/runtime-runc/moby/48f10a5eb9a245e6890433205053ba4e72c8e3bab5c13c2920dc32fadd7290cd/ski.rB3K51",
			},
			ProcessStart: v1.ProcessStart{
				Invocation: "runc --root /var/run/docker/runtime-runc/ski",
				Hashes: v1.ProcessHashes{
					MD5:    "",
					SHA1:   "",
					SHA256: "Ski-SHA256",
				},
			},
		},
	}

	testSelection := func(t *testing.T, selector string, expectedReports []v1.Report) {
		clusterInfo := bapi.ClusterInfo{Cluster: cluster}

		// Create the event in ES.
		resp, err := b.Create(ctx, clusterInfo, reports)
		require.NoError(t, err)
		require.Equal(t, 0, len(resp.Errors))
		require.Equal(t, len(reports), resp.Total)
		require.Equal(t, 0, resp.Failed)
		require.Equal(t, len(reports), resp.Succeeded)

		// Refresh the index.
		err = backendutils.RefreshIndex(ctx, client, indexGetter.Index(clusterInfo))
		require.NoError(t, err)

		r, e := b.List(ctx, clusterInfo, &v1.RuntimeReportParams{
			Selector: selector,
		})
		require.NoError(t, e)
		require.Equal(t, len(expectedReports), len(r.Items))
		for i := range expectedReports {
			expectedReports[i].GeneratedTime = r.Items[i].Report.GeneratedTime
			require.Equal(t, expectedReports[i], backendutils.AssertRuntimeReportIDAndReset(t, r.Items[i]).Report)
		}
	}

	tests := []struct {
		selector        string
		expectedReports []v1.Report
	}{
		{"'pod.name'=\"goldfinger\"", []v1.Report{reports[1]}},
		{"'pod.name' IN {\"goldfinger\",\"blofeld\"}", []v1.Report{reports[1], reports[2]}},
		{"count > 5", []v1.Report{reports[1]}},
		{"count <= 2", []v1.Report{reports[0], reports[2]}},
		{"type = FutureFeature", []v1.Report{}},
		{"type = ProcessStart", []v1.Report{reports[0], reports[1], reports[2]}},
		{"'pod.namespace'='moonraker'", []v1.Report{reports[0]}},
		{"'pod.name_aggr' != 'blofeld-*'", []v1.Report{reports[0], reports[1]}},
		{"'pod.container_name' NOTIN {gold}", []v1.Report{reports[0], reports[2]}},
		{"'pod.ready' = false", []v1.Report{reports[0], reports[1], reports[2]}},
		{"'pod.ready' = true", []v1.Report{}},
		{"'file.path' IN {'*laser*'}", []v1.Report{reports[1]}},
		{"'file.host_path' IN {'*laser*'}", []v1.Report{reports[1]}},
		{"'process_start.invocation' NOTIN {'*laser*'}", []v1.Report{reports[0], reports[2]}},
		{"'process_start.hashes.md5' != \"\"", []v1.Report{}},
		{"'process_start.hashes.sha256' IN {\"*Laser*\"}", []v1.Report{reports[1]}},
		{"host = host", []v1.Report{reports[0], reports[1], reports[2]}},
	}

	for _, tt := range tests {
		name := fmt.Sprintf("TestReportSelection: %s", tt.selector)
		RunAllModes(t, name, func(t *testing.T) {
			testSelection(t, tt.selector, tt.expectedReports)
		})
	}
}
