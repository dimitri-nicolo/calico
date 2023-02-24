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

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
	"github.com/projectcalico/calico/linseed/pkg/config"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

func eventsSetupAndTeardown(t *testing.T) func() {
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
		testutils.CleanupIndices(context.Background(), esClient, fmt.Sprintf("tigera_secure_ee_events.%s", cluster))
		logCancel()
		cancel()
	}
}

func TestFV_Events(t *testing.T) {
	t.Run("should return an empty list if there are no events", func(t *testing.T) {
		defer eventsSetupAndTeardown(t)()

		params := v1.EventParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now(),
				},
			},
		}

		// Perform a query.
		events, err := cli.Events(cluster).List(ctx, &params)
		require.NoError(t, err)
		require.Equal(t, []v1.Event{}, events.Items)
	})

	t.Run("should create and list events", func(t *testing.T) {
		defer eventsSetupAndTeardown(t)()

		// Create a basic event.
		events := []v1.Event{
			{
				Time:        time.Now().Unix(),
				Description: "A rather uneventful evening",
				Origin:      "TODO",
				Severity:    1,
				Type:        "TODO",
			},
		}
		bulk, err := cli.Events(cluster).Create(ctx, events)
		require.NoError(t, err)
		require.Equal(t, bulk.Succeeded, 1, "create event did not succeed")

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_events*")

		// Read it back.
		params := v1.EventParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now().Add(5 * time.Second),
				},
			},
		}
		resp, err := cli.Events(cluster).List(ctx, &params)
		require.NoError(t, err)

		// The ID should be set, but random, so we can't assert on its value.
		require.NotEqual(t, "", resp.Items[0].ID)
		resp.Items[0].ID = ""
		require.Equal(t, events, resp.Items)
	})
}
