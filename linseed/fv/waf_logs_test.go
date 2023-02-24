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
	"github.com/projectcalico/calico/linseed/pkg/client"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
	"github.com/projectcalico/calico/linseed/pkg/config"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

func wafSetupAndTeardown(t *testing.T) func() {
	// Hook logrus into testing.T
	config.ConfigureLogging("DEBUG")
	logCancel := logutils.RedirectLogrusToTestingT(t)

	// Create an ES client.
	esClient, err := elastic.NewSimpleClient(elastic.SetURL("http://localhost:9200"), elastic.SetInfoLog(logrus.StandardLogger()))
	require.NoError(t, err)
	lmaClient = lmaelastic.NewWithClient(esClient)

	// Instantiate a client.
	cfg := rest.Config{
		CACertPath: "cert/RootCA.crt",
		URL:        "https://localhost:8444/",
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
		testutils.CleanupIndices(context.Background(), esClient, fmt.Sprintf("tigera_secure_ee_waf.%s", cluster))
		logCancel()
		cancel()
	}
}

func TestFV_WAF(t *testing.T) {
	t.Run("should return an empty list if there are no WAF logs", func(t *testing.T) {
		defer wafSetupAndTeardown(t)()

		params := v1.WAFLogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now(),
				},
			},
		}

		// Perform a query.
		wafLogs, err := cli.WAFLogs(cluster).List(ctx, &params)
		require.NoError(t, err)
		require.Equal(t, []v1.WAFLog{}, wafLogs.Items)
	})

	t.Run("should create and list waf logs", func(t *testing.T) {
		defer wafSetupAndTeardown(t)()

		reqTime := time.Now()
		// Create a basic waf log
		wafLogs := []v1.WAFLog{
			{
				Timestamp: reqTime,
				Msg:       "any message",
			},
		}
		bulk, err := cli.WAFLogs(cluster).Create(ctx, wafLogs)
		require.NoError(t, err)
		require.Equal(t, bulk.Succeeded, 1, "create waf logs did not succeed")

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_waf*")

		// Read it back.
		params := v1.WAFLogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now().Add(5 * time.Second),
				},
			},
		}
		resp, err := cli.WAFLogs(cluster).List(ctx, &params)
		require.NoError(t, err)

		require.Len(t, resp.Items, 1)
		// Reset the time as it microseconds to not match perfectly
		require.NotEqual(t, "", resp.Items[0].Timestamp)
		resp.Items[0].Timestamp = reqTime

		require.Equal(t, wafLogs, resp.Items)
	})
}
