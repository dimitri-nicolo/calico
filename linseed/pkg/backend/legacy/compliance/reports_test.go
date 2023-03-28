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
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/templates"
	backendutils "github.com/projectcalico/calico/linseed/pkg/backend/testutils"
	"github.com/projectcalico/calico/linseed/pkg/config"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

var (
	client      lmaelastic.Client
	cache       bapi.Cache
	rb          bapi.ReportsBackend
	bb          bapi.BenchmarksBackend
	sb          bapi.SnapshotsBackend
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
	rb = compliance.NewReportsBackend(client, cache)
	bb = compliance.NewBenchmarksBackend(client, cache)
	sb = compliance.NewSnapshotBackend(client, cache)

	// Create a random cluster name for each test to make sure we don't
	// interfere between tests.
	cluster = backendutils.RandomClusterName()
	clusterInfo = bapi.ClusterInfo{Cluster: cluster}

	// Set a timeout for each test.
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)

	// Function contains teardown logic.
	return func() {
		// Cancel the context.
		cancel()

		// Cleanup any data that might left over from a previous failed run.
		err = backendutils.CleanupIndices(context.Background(), esClient, fmt.Sprintf("tigera_secure_ee_compliance_reports.%s", cluster))
		require.NoError(t, err)
		err = backendutils.CleanupIndices(context.Background(), esClient, fmt.Sprintf("tigera_secure_ee_benchmark_results.%s", cluster))
		require.NoError(t, err)
		err = backendutils.CleanupIndices(context.Background(), esClient, fmt.Sprintf("tigera_secure_ee_snapshots.%s", cluster))
		require.NoError(t, err)

		// Cancel logging
		logCancel()
	}
}

func TestCreateReport(t *testing.T) {
	defer setupTest(t)()

	// Create a dummy report.
	report := apiv3.ReportData{
		ReportName:     "test-report",
		ReportTypeName: "my-report-type",
		StartTime:      metav1.Time{Time: time.Unix(1, 0)},
		EndTime:        metav1.Time{Time: time.Unix(2, 0)},
		GenerationTime: metav1.Time{Time: time.Unix(3, 0)},
	}
	f := v1.ReportData{ReportData: &report}

	response, err := rb.Create(ctx, clusterInfo, []v1.ReportData{f})
	require.NoError(t, err)
	require.Equal(t, []v1.BulkError(nil), response.Errors)
	require.Equal(t, 0, response.Failed)

	err = backendutils.RefreshIndex(ctx, client, "tigera_secure_ee_compliance_reports.*")
	require.NoError(t, err)

	// Read it back and check it matches.
	p := v1.ReportDataParams{}
	resp, err := rb.List(ctx, clusterInfo, &p)
	require.NoError(t, err)
	require.Len(t, resp.Items, 1)
	require.NotEqual(t, "", resp.Items[0].ID)
	resp.Items[0].ID = ""
	require.Equal(t, f, resp.Items[0])
}
