// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"

	elastic "github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
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
	cfg := rest.Config{
		CACertPath:     "cert/RootCA.crt",
		URL:            "https://localhost:8444/",
		ClientCertPath: "cert/localhost.crt",
		ClientKeyPath:  "cert/localhost.key",
	}
	cli, err = client.NewClient("", cfg)
	require.NoError(t, err)

	// Create a random cluster name for each test to make sure we don't
	// interfere between tests.
	cluster = testutils.RandomClusterName()

	// Set up context with a timeout.
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)

	return func() {
		// Cleanup indices created by the test.
		err := testutils.CleanupIndices(context.Background(), esClient, fmt.Sprintf("tigera_secure_ee_compliance_reports.%s", cluster))
		require.NoError(t, err)
		err = testutils.CleanupIndices(context.Background(), esClient, fmt.Sprintf("tigera_secure_ee_benchmark_results.%s", cluster))
		require.NoError(t, err)
		err = testutils.CleanupIndices(context.Background(), esClient, fmt.Sprintf("tigera_secure_ee_snapshots.%s", cluster))
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
}
