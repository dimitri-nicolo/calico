// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package waf_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/templates"
	"github.com/projectcalico/calico/linseed/pkg/config"

	"github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/require"

	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/waf"
	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

var (
	client  lmaelastic.Client
	b       bapi.WAFBackend
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
	b = waf.NewBackend(client, cache)

	// Create a random cluster name for each test to make sure we don't
	// interfere between tests.
	cluster = testutils.RandomClusterName()

	// Each test should take less than 5 seconds.
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)

	// Function contains teardown logic.
	return func() {
		err = testutils.CleanupIndices(context.Background(), esClient, fmt.Sprintf("tigera_secure_ee_waf.%s", cluster))
		require.NoError(t, err)

		// Cancel the context
		cancel()
		logCancel()
	}
}

// TestCreateWAFLog tests running a real elasticsearch query to create a kube waf log.
func TestCreateWAFLog(t *testing.T) {
	defer setupTest(t)()

	clusterInfo := bapi.ClusterInfo{Cluster: cluster}

	logTime := time.Now()
	f := v1.WAFLog{
		Timestamp: logTime,
		Source: &v1.WAFEndpoint{
			IP:       "1.2.3.4",
			PortNum:  789,
			Hostname: "source-hostname",
		},
		Destination: &v1.WAFEndpoint{
			IP:       "4.3.2.1",
			PortNum:  987,
			Hostname: "dest-hostname",
		},
		Path:     "/yellow/brick/road",
		Method:   "GET",
		Protocol: "TCP",
		Msg:      "This is a friendly reminder that nobody knows what is going on",
		RuleInfo: "WAF rules, rule WAF",
	}

	// Create the log in ES.
	resp, err := b.Create(ctx, clusterInfo, []v1.WAFLog{f})
	require.NoError(t, err)
	require.Equal(t, []v1.BulkError(nil), resp.Errors)
	require.Equal(t, 1, resp.Total)
	require.Equal(t, 0, resp.Failed)
	require.Equal(t, 1, resp.Succeeded)

	// Refresh the index.
	err = testutils.RefreshIndex(ctx, client, "tigera_secure_ee_waf.*")
	require.NoError(t, err)

	results, err := b.List(ctx, clusterInfo, &v1.WAFLogParams{
		QueryParams: v1.QueryParams{
			TimeRange: &lmav1.TimeRange{
				From: time.Now().Add(-10 * time.Second),
				To:   time.Now().Add(10 * time.Second),
			},
		},
	})

	require.NoError(t, err)
	require.Equal(t, 1, len(results.Items))
	require.Equal(t, results.Items[0].Timestamp.Format(time.RFC3339), logTime.Format(time.RFC3339))

	// Timestamps don't equal on read.
	results.Items[0].Timestamp = f.Timestamp

	require.Equal(t, f, results.Items[0])
}
