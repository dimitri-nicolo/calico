// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"

	elastic "github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	"github.com/projectcalico/calico/linseed/pkg/client"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
	"github.com/projectcalico/calico/linseed/pkg/config"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

func l7logSetupAndTeardown(t *testing.T) func() {
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
		testutils.CleanupIndices(context.Background(), esClient, fmt.Sprintf("tigera_secure_ee_l7.%s", cluster))
		logCancel()
		cancel()
	}
}

func TestL7_FlowLogs(t *testing.T) {
	t.Run("should return an empty list if there are no l7 logs", func(t *testing.T) {
		defer l7logSetupAndTeardown(t)()

		params := v1.L7LogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now(),
				},
			},
		}

		// Perform a query.
		logs, err := cli.L7Logs(cluster).List(ctx, &params)
		require.NoError(t, err)
		require.Equal(t, []v1.L7Log{}, logs.Items)
	})

	t.Run("should create and list l7 logs", func(t *testing.T) {
		defer l7logSetupAndTeardown(t)()

		// Create a basic flow log.
		logs := []v1.L7Log{
			{
				EndTime:      time.Now().Unix(), // TODO: Add more fields
				ResponseCode: "200",
			},
		}
		bulk, err := cli.L7Logs(cluster).Create(ctx, logs)
		require.NoError(t, err)
		require.Equal(t, bulk.Succeeded, 1, "create l7 log did not succeed")

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_l7*")

		// Read it back.
		params := v1.L7LogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now().Add(5 * time.Second),
				},
			},
		}
		resp, err := cli.L7Logs(cluster).List(ctx, &params)
		require.NoError(t, err)
		require.Equal(t, logs, resp.Items)
	})

	t.Run("should return an empty aggregations if there are no l7 logs", func(t *testing.T) {
		defer l7logSetupAndTeardown(t)()

		params := v1.L7AggregationParams{
			L7LogParams: v1.L7LogParams{
				QueryParams: v1.QueryParams{
					TimeRange: &lmav1.TimeRange{
						From: time.Now().Add(-5 * time.Second),
						To:   time.Now(),
					},
				},
			},
			Aggregations: map[string]json.RawMessage{
				"response_code": []byte(`{"filters":{"other_bucket_key":"other","filters":{"1xx":{"prefix":{"response_code":"1"}},"2xx":{"prefix":{"response_code":"2"}},"3xx":{"prefix":{"response_code":"3"}},"4xx":{"prefix":{"response_code":"4"}},"5xx":{"prefix":{"response_code":"5"}}}},"aggs":{"myDurationMeanHistogram":{"date_histogram":{"field":"start_time","fixed_interval":"60s"},"aggs":{"myDurationMeanAvg":{"avg":{"field":"duration_mean"}}}}}}`),
			},
			NumBuckets: 3,
		}

		// Perform a query.
		aggregations, err := cli.L7Logs(cluster).Aggregations(ctx, &params)
		require.NoError(t, err)
		require.Nil(t, aggregations)
	})

}
