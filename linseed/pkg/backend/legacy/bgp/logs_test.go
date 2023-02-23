// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package bgp_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"

	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/templates"
	"github.com/projectcalico/calico/linseed/pkg/config"

	"github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/require"

	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/bgp"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

var (
	client  lmaelastic.Client
	b       bapi.BGPBackend
	ctx     context.Context
	cluster string
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
	b = bgp.NewBackend(client, cache)

	// Create a random cluster name for each test to make sure we don't
	// interfere between tests.
	cluster = testutils.RandomClusterName()

	// Each test should take less than 5 seconds.
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)

	// Function contains teardown logic.
	return func() {
		err = testutils.CleanupIndices(context.Background(), esClient, fmt.Sprintf("tigera_secure_ee_bgp.%s", cluster))
		require.NoError(t, err)

		// Cancel the context
		cancel()
		logCancel()
	}
}

// TestCreateBGPLog tests running a real elasticsearch query to create a kube bgp log.
func TestCreateBGPLog(t *testing.T) {
	defer setupTest(t)()

	clusterInfo := bapi.ClusterInfo{Cluster: cluster}

	f := v1.BGPLog{
		// yyyy-MM-dd'T'HH:mm:ss
		LogTime:   "1990-09-15T06:12:32",
		Message:   "BGP is wonderful",
		IPVersion: v1.IPv6BGPLog,
		Host:      "lenox",
	}

	// Create the event in ES.
	resp, err := b.Create(ctx, clusterInfo, []v1.BGPLog{f})
	require.NoError(t, err)
	require.Equal(t, []v1.BulkError(nil), resp.Errors)
	require.Equal(t, 1, resp.Total)
	require.Equal(t, 0, resp.Failed)
	require.Equal(t, 1, resp.Succeeded)

	// Refresh the index.
	err = testutils.RefreshIndex(ctx, client, "tigera_secure_ee_bgp.*")
	require.NoError(t, err)

	// List the log, assert that it matches the one we just wrote.
	start, err := time.Parse(v1.BGPLogTimeFormat, "1990-09-15T06:12:00")
	require.NoError(t, err)

	results, err := b.List(ctx, clusterInfo, v1.BGPLogParams{
		QueryParams: v1.QueryParams{
			// 1990-09-15'T'06:12:32
			TimeRange: &lmav1.TimeRange{
				From: start,
				To:   time.Now(),
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(results.Items))

	require.Equal(t, f, results.Items[0])
}
