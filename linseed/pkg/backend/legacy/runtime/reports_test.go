// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package runtime_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"

	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"

	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/templates"
	"github.com/projectcalico/calico/linseed/pkg/config"

	"github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/require"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/runtime"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

var (
	client        lmaelastic.Client
	b             bapi.RuntimeBackend
	ctx           context.Context
	cluster       string
	tenant        string
	anotherTenant string
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
	cache := templates.NewTemplateCache(client, 1, 0)

	// Instantiate a backend.
	b = runtime.NewBackend(client, cache)

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
		err = testutils.CleanupIndices(context.Background(), esClient, fmt.Sprintf("tigera_secure_ee_runtime.%s", cluster))
		require.NoError(t, err)
		err = testutils.CleanupIndices(context.Background(), esClient, fmt.Sprintf("tigera_secure_ee_runtime.%s", tenant))
		require.NoError(t, err)
		err = testutils.CleanupIndices(context.Background(), esClient, fmt.Sprintf("tigera_secure_ee_runtime.%s", anotherTenant))
		require.NoError(t, err)

		// Cancel the context
		cancel()
		logCancel()
	}
}

// TestCreateRuntimeReport tests running a real elasticsearch query to create a runtime report.
func TestCreateRuntimeReport(t *testing.T) {
	defer setupTest(t)()

	clusterInfo := bapi.ClusterInfo{Cluster: cluster}

	startTime := time.Unix(1, 0).UTC()
	endTime := time.Unix(2, 0).UTC()
	generatedTime := time.Unix(3, 0).UTC()
	f := v1.Report{
		GeneratedTime: &generatedTime,
		StartTime:     startTime,
		EndTime:       endTime,
		Host:          "host",
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
	err = testutils.RefreshIndex(ctx, client, fmt.Sprintf("tigera_secure_ee_runtime.%s.*", cluster))
	require.NoError(t, err)

	results, err := b.List(ctx, clusterInfo, &v1.RuntimeReportParams{
		QueryParams: v1.QueryParams{
			TimeRange: &lmav1.TimeRange{
				From: generatedTime,
				To:   time.Now(),
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(results.Items))

	require.Equal(t, []v1.RuntimeReport{{Tenant: "", Cluster: cluster, Report: f}},
		testutils.AssertLogIDAndCopyRuntimeReportsWithoutThem(t, results))
}

// TestCreateRuntimeReport tests running a real elasticsearch query to create a runtime report.
func TestCreateRuntimeReportForMultipleTenants(t *testing.T) {
	defer setupTest(t)()

	startTime := time.Unix(1, 0).UTC()
	endTime := time.Unix(2, 0).UTC()
	generatedTime := time.Unix(3, 0).UTC()
	f := v1.Report{
		GeneratedTime: &generatedTime,
		StartTime:     startTime,
		EndTime:       endTime,
		Host:          "host",
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
	err = testutils.RefreshIndex(ctx, client, "tigera_secure_ee_runtime.*")
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
	require.Equal(t, []v1.RuntimeReport{{Tenant: anotherTenant, Cluster: cluster, Report: f}},
		testutils.AssertLogIDAndCopyRuntimeReportsWithoutThem(t, results))
}
