// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package flows_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/require"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/flows"
	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

var (
	client lmaelastic.Client
	b      bapi.FlowBackend
	ctx    context.Context
)

// beforeEach runs common logic before each test, and also returns a function to perform teardown
// after each test.
func setupSuite(t *testing.T) func() {
	// Create an elasticsearch client to use for the test. For this suite, we use a real
	// elasticsearch instance created via "make run-elastic".
	esClient, err := elastic.NewSimpleClient(elastic.SetURL("http://localhost:9200"))
	require.NoError(t, err)
	client = lmaelastic.NewWithClient(esClient)

	// Cleanup any data that might left over from a previous failed run.
	_, err = client.Backend().DeleteIndex("tigera_secure_ee_flows.*").Do(context.Background())
	require.NoError(t, err)

	// Create a FlowBackend to use.
	b = flows.NewFlowBackend(client)

	// Each test should take less than 5 seconds.
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)

	// Function contains teardown logic.
	return func() {
		// Cancel the context
		cancel()
	}
}

// TestListFlows tests running a real elasticsearch query to list flows.
func TestListFlows(t *testing.T) {
	defer setupSuite(t)()

	clusterInfo := bapi.ClusterInfo{Cluster: "mycluster"}

	// Timeout the test after 5 seconds.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Put some data into ES so we can query it.
	bld := NewFlowLogBuilder()
	bld.WithType("wep").
		WithSourceNamespace("default").
		WithDestNamespace("kube-system").
		WithDestName("kube-dns-*").
		WithDestIP("10.0.0.10").
		WithDestService("kube-dns", 53).
		WithDestPort(53).
		WithProtocol("udp").
		WithSourceName("my-deployment").
		WithSourceIP("192.168.1.1").
		WithRandomFlowStats().WithRandomPacketStats().
		WithReporter("src").WithAction("allowed").
		WithSourceLabels("bread=rye", "cheese=brie", "wine=none")
	expected := populateFlowData(t, ctx, bld, client, clusterInfo.Cluster)

	// Set time range so that we capture all of the populated flow logs.
	opts := v1.L3FlowParams{}
	opts.QueryParams = &v1.QueryParams{}
	opts.QueryParams.TimeRange = &lmav1.TimeRange{}
	opts.QueryParams.TimeRange.From = time.Now().Add(-5 * time.Second)
	opts.QueryParams.TimeRange.To = time.Now().Add(5 * time.Second)

	// Query for flows. There should be a single flow from the populated data.
	r, err := b.List(ctx, clusterInfo, opts)
	require.NoError(t, err)
	require.Len(t, r.Items, 1)
	require.Nil(t, r.AfterKey)
	require.Empty(t, err)

	// Assert that the flow data is populated correctly.
	require.Equal(t, expected, r.Items[0])
}

// TestMultipleFlows tests that we return multiple flows properly.
func TestMultipleFlows(t *testing.T) {
	defer setupSuite(t)()

	// Both flows use the same cluster information.
	clusterInfo := bapi.ClusterInfo{Cluster: "mycluster"}

	// Timeout the test after 5 seconds.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Template for flow #1.
	bld := NewFlowLogBuilder()
	bld.WithType("wep").
		WithSourceNamespace("tigera-operator").
		WithDestNamespace("kube-system").
		WithDestName("kube-dns-*").
		WithDestIP("10.0.0.10").
		WithDestService("kube-dns", 53).
		WithDestPort(53).
		WithProtocol("udp").
		WithSourceName("tigera-operator").
		WithSourceIP("34.15.66.3").
		WithRandomFlowStats().WithRandomPacketStats().
		WithReporter("src").WithAction("allowed").
		WithSourceLabels("bread=rye", "cheese=brie", "wine=none") // TODO
	exp1 := populateFlowData(t, ctx, bld, client, clusterInfo.Cluster)

	// Template for flow #2.
	bld2 := NewFlowLogBuilder()
	bld2.WithType("wep").
		WithSourceNamespace("default").
		WithDestNamespace("kube-system").
		WithDestName("kube-dns-*").
		WithDestIP("10.0.0.10").
		WithDestService("kube-dns", 53).
		WithDestPort(53).
		WithProtocol("udp").
		WithSourceName("my-deployment").
		WithSourceIP("192.168.1.1").
		WithRandomFlowStats().WithRandomPacketStats().
		WithReporter("src").WithAction("allowed").
		WithSourceLabels("bread=rye", "cheese=brie", "wine=none")
	exp2 := populateFlowData(t, ctx, bld2, client, clusterInfo.Cluster)

	// Set time range so that we capture all of the populated flow logs.
	opts := v1.L3FlowParams{}
	opts.QueryParams = &v1.QueryParams{}
	opts.QueryParams.TimeRange = &lmav1.TimeRange{}
	opts.QueryParams.TimeRange.From = time.Now().Add(-5 * time.Second)
	opts.QueryParams.TimeRange.To = time.Now().Add(5 * time.Second)

	// Query for flows. There should be two flows from the populated data.
	r, err := b.List(ctx, clusterInfo, opts)
	require.NoError(t, err)
	require.Len(t, r.Items, 2)
	require.Nil(t, r.AfterKey)
	require.Empty(t, err)

	// Assert that the flow data is populated correctly.
	require.Equal(t, exp1, r.Items[1])
	require.Equal(t, exp2, r.Items[0])
}

// TestPagination tests that we return multiple flows properly using pagination.
func TestPagination(t *testing.T) {
	defer setupSuite(t)()

	// Both flows use the same cluster information.
	clusterInfo := bapi.ClusterInfo{Cluster: "mycluster"}

	// Timeout the test after 5 seconds.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Template for flow #1.
	bld := NewFlowLogBuilder()
	bld.WithType("wep").
		WithSourceNamespace("tigera-operator").
		WithDestNamespace("kube-system").
		WithDestName("kube-dns-*").
		WithDestIP("10.0.0.10").
		WithDestService("kube-dns", 53).
		WithDestPort(53).
		WithProtocol("udp").
		WithSourceName("tigera-operator").
		WithSourceIP("34.15.66.3").
		WithRandomFlowStats().WithRandomPacketStats().
		WithReporter("src").WithAction("allowed").
		WithSourceLabels("bread=rye", "cheese=brie", "wine=none") // TODO
	exp1 := populateFlowData(t, ctx, bld, client, clusterInfo.Cluster)

	// Template for flow #2.
	bld2 := NewFlowLogBuilder()
	bld2.WithType("wep").
		WithSourceNamespace("default").
		WithDestNamespace("kube-system").
		WithDestName("kube-dns-*").
		WithDestIP("10.0.0.10").
		WithDestService("kube-dns", 53).
		WithDestPort(53).
		WithProtocol("udp").
		WithSourceName("my-deployment").
		WithSourceIP("192.168.1.1").
		WithRandomFlowStats().WithRandomPacketStats().
		WithReporter("src").WithAction("allowed").
		WithSourceLabels("bread=rye", "cheese=brie", "wine=none")
	exp2 := populateFlowData(t, ctx, bld2, client, clusterInfo.Cluster)

	// Set time range so that we capture all of the populated flow logs.
	opts := v1.L3FlowParams{}
	opts.QueryParams = &v1.QueryParams{}
	opts.QueryParams.TimeRange = &lmav1.TimeRange{}
	opts.QueryParams.TimeRange.From = time.Now().Add(-5 * time.Second)
	opts.QueryParams.TimeRange.To = time.Now().Add(5 * time.Second)

	// Also set a max results of 1, so that we only get one flow at a time.
	opts.QueryParams.MaxResults = 1

	// Query for flows. There should be a single flow from the populated data.
	r, err := b.List(ctx, clusterInfo, opts)
	require.NoError(t, err)
	require.Len(t, r.Items, 1)
	require.NotNil(t, r.AfterKey)
	require.Empty(t, err)
	require.Equal(t, exp2, r.Items[0])

	// Now, send another request. This time, passing in the pagination key
	// returned from the first. We should get the second flow.
	opts.QueryParams.AfterKey = r.AfterKey
	r, err = b.List(ctx, clusterInfo, opts)
	require.NoError(t, err)
	require.Len(t, r.Items, 1)
	require.NotNil(t, r.AfterKey)
	require.Empty(t, err)
	require.Equal(t, exp1, r.Items[0])
}

// populateFlowData writes a series of flow logs to elasticsearch, and returns the FlowLog that we
// should expect to exist as a result. This can be used to assert round-tripping and aggregation against ES is working correctly.
func populateFlowData(t *testing.T, ctx context.Context, b *flowLogBuilder, client lmaelastic.Client, cluster string) v1.L3Flow {
	batch := []v1.FlowLog{}

	// Initialize the expected output based on the given builder template.
	expected := b.ExpectedFlow()

	// Labels are an aggregation across all flow logs created in the loop below.
	// TODO: This should be calculated by the builder.
	expected.SourceLabels = []v1.FlowLabels{
		{Key: "bread", Values: []string{"rye"}},
		{Key: "cheese", Values: []string{"brie"}},
		{Key: "wine", Values: []string{"none"}},
	}
	expected.DestinationLabels = []v1.FlowLabels{{Key: "dest_iteration", Values: []string{}}}

	for i := 0; i < 10; i++ {
		// We want a variety of label keys and values,
		// so base this one off of the loop variable.
		// Note: We use a nested terms aggregation to get labels, which has an
		// inherent maximum number of buckets of 10. As a result, if a flow has more than
		// 10 labels, not all of them will be shown. We might be able to use a composite aggregation instead,
		// but these are more expensive.
		b2 := b.Copy()
		b2.WithDestLabels(fmt.Sprintf("dest_iteration=%d", i))

		f, err := b2.Build()
		require.NoError(t, err)

		// Add it to the batch
		batch = append(batch, *f)

		// Increment fields on the expected flow based on the flow log that was
		// just added.
		expected.LogStats.LogCount += int64(f.NumFlows)
		expected.LogStats.Started += int64(f.NumFlowsStarted)
		expected.LogStats.Completed += int64(f.NumFlowsCompleted)
		expected.TrafficStats.BytesIn += int64(f.BytesIn)
		expected.TrafficStats.BytesOut += int64(f.BytesOut)
		expected.TrafficStats.PacketsIn += int64(f.PacketsIn)
		expected.TrafficStats.PacketsOut += int64(f.PacketsOut)
		expected.DestinationLabels[0].Values = append(expected.DestinationLabels[0].Values, fmt.Sprintf("%d", i))
	}

	// Instantiate a FlowBackend.
	response, err := flows.NewFlowLogBackend(client).Create(ctx, bapi.ClusterInfo{Cluster: cluster}, batch)
	require.NoError(t, err)
	require.Equal(t, response.Failed, 0)

	// Refresh the index so that data is readily available for the test. Otherwise, we need to wait
	// for the refresh interval to occur.
	index := fmt.Sprintf("tigera_secure_ee_flows.%s.*", cluster)
	err = testutils.RefreshIndex(ctx, client, index)
	require.NoError(t, err)

	return *expected
}
