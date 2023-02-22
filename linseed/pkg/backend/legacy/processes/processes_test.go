// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package processes_test

import (
	"context"
	_ "embed"
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/templates"
	"github.com/projectcalico/calico/linseed/pkg/config"

	"github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/require"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/flows"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/processes"
	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

var (
	client  lmaelastic.Client
	cache   bapi.Cache
	pb      bapi.ProcessBackend
	flb     bapi.FlowLogBackend
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
	cache = templates.NewTemplateCache(client, 1, 0)

	// Create backends to use.
	pb = processes.NewBackend(client)
	flb = flows.NewFlowLogBackend(client, cache)

	// Create a random cluster name for each test to make sure we don't
	// interfere between tests.
	cluster = testutils.RandomClusterName()

	// Set a timeout for each test.
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)

	// Function contains teardown logic.
	return func() {
		// Cancel the context.
		cancel()

		// Cleanup any data that might left over from a previous failed run.
		err = testutils.CleanupIndices(context.Background(), esClient, fmt.Sprintf("tigera_secure_ee_flows.%s", cluster))
		require.NoError(t, err)

		// Cancel logging
		logCancel()
	}
}

// TestListProcesses tests running a real elasticsearch query to list processes.
func TestListProcesses(t *testing.T) {
	defer setupTest(t)()

	clusterInfo := bapi.ClusterInfo{Cluster: cluster}

	// Put some data into ES so we can query it.
	// Build the same flow, reported by the source and the dest.
	bld := testutils.NewFlowLogBuilder()
	bld.WithType("wep").
		WithSourceNamespace("default").
		WithDestNamespace("kube-system").
		WithDestName("kube-dns-*").
		WithProtocol("udp").
		WithSourceName("my-deployment-*").
		WithSourceIP("192.168.1.1").
		WithRandomFlowStats().WithRandomPacketStats().
		WithReporter("src").WithAction("allow").
		WithProcessName("/bin/curl")
	srcLog, err := bld.Build()
	require.NoError(t, err)
	bld.WithReporter("dst")
	dstLog, err := bld.Build()

	response, err := flb.Create(ctx, clusterInfo, []v1.FlowLog{*srcLog, *dstLog})
	require.NoError(t, err)
	require.Equal(t, []v1.BulkError(nil), response.Errors)
	require.Equal(t, 0, response.Failed)

	// Set time range so that we capture all of the populated logs.
	opts := v1.ProcessParams{}
	opts.TimeRange = &lmav1.TimeRange{}
	opts.TimeRange.From = time.Now().Add(-5 * time.Minute)
	opts.TimeRange.To = time.Now().Add(5 * time.Minute)

	err = testutils.RefreshIndex(ctx, client, "tigera_secure_ee_flows.*")

	// Query for process info. There should be a single entry from the populated data.
	r, err := pb.List(ctx, clusterInfo, &opts)
	require.NoError(t, err)
	require.Len(t, r.Items, 1)
	require.Nil(t, r.AfterKey)
	require.Empty(t, err)

	// Assert that the process data is populated correctly.
	expected := v1.ProcessInfo{
		Name:     "/bin/curl",
		Endpoint: "my-deployment-*",
		Count:    1,
	}
	require.Equal(t, expected, r.Items[0])
}
