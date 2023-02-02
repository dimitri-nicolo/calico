// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package flows_test

import (
	"context"
	"fmt"
	"net"
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

// TestListFlows tests running a real elasticsearch query to list flows.
func TestListFlows(t *testing.T) {
	clusterInfo := bapi.ClusterInfo{Cluster: "mycluster"}

	// Create an elasticsearch client to use for the test. For this test, we use a real
	// elasticsearch instance created via "make run-elastic".
	esClient, err := elastic.NewSimpleClient(elastic.SetURL("http://localhost:9200"))
	require.NoError(t, err)
	client := lmaelastic.NewWithClient(esClient)

	// Instantiate a FlowBackend.
	b := flows.NewFlowBackend(client)

	// Timeout the test after 5 seconds.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Put some data into ES so we can query it.
	expected := populateFlowData(t, ctx, client, clusterInfo.Cluster)

	// Set time range so that we capture all of the populated flow logs.
	opts := v1.L3FlowParams{}
	opts.QueryParams = &v1.QueryParams{}
	opts.QueryParams.TimeRange = &lmav1.TimeRange{}
	opts.QueryParams.TimeRange.From = time.Now().Add(-5 * time.Second)
	opts.QueryParams.TimeRange.To = time.Now().Add(5 * time.Second)

	// Query for flows. There should be a single flow from the populated data.
	// r, err := b.List(ctx, clusterInfo, opts)
	// require.NoError(t, err)
	// require.Len(t, r.Items, 1)
	// require.Nil(t, r.AfterKey)
	// require.Empty(t, err)

	// // Assert that the flow data is populated correctly.
	// require.Equal(t, expected, r.Items[0])

	// Query again, but this time with a small page size.
	opts.MaxResults = 1
	r, err := b.List(ctx, clusterInfo, opts)
	require.NoError(t, err)
	require.Len(t, r.Items, 1)
	require.NotNil(t, r.AfterKey)
	require.Empty(t, err)

	// Repeat it, with an after key.
	opts.AfterKey = r.AfterKey
	r, err = b.List(ctx, clusterInfo, opts)
	require.NoError(t, err)
	require.Len(t, r.Items, 1)
	require.NotNil(t, r.AfterKey)
	require.Empty(t, err)

	// Assert that the flow data is populated correctly.
	require.Equal(t, expected, r.Items[0])

	// Clean up after ourselves by deleting the index.
	_, err = esClient.DeleteIndex(fmt.Sprintf("tigera_secure_ee_flows.%s.*", clusterInfo.Cluster)).Do(ctx)
	require.NoError(t, err)
}

// populateFlowData writes a series of flow logs to elasticsearch, and returns the FlowLog that we
// should expect to exist as a result. This can be used to assert round-tripping and aggregation against ES is working correctly.
func populateFlowData(t *testing.T, ctx context.Context, client lmaelastic.Client, cluster string) v1.L3Flow {
	// Clear out any old data first.
	_, _ = client.Backend().DeleteIndex(fmt.Sprintf("tigera_secure_ee_flows.%s.*", cluster)).Do(ctx)

	// Instantiate a FlowBackend.
	b := flows.NewFlowLogBackend(client)

	// The expected flow log - we'll populate fields as we go.
	expected := v1.L3Flow{}
	expected.Key = v1.L3FlowKey{
		Action:   "allowed",
		Reporter: "src",
		Protocol: "udp",
		Source: v1.Endpoint{
			Namespace:      "default",
			Type:           "wep",
			AggregatedName: "my-deployment",
		},
		Destination: v1.Endpoint{
			Namespace:      "kube-system",
			Type:           "wep",
			AggregatedName: "kube-dns-*",
			Port:           53,
		},
	}
	expected.TrafficStats = &v1.TrafficStats{}
	expected.LogStats = &v1.LogStats{
		FlowLogCount: 10,
	}
	expected.Service = &v1.Service{
		Name:      "kube-dns",
		Namespace: "kube-system",
		Port:      53,
		PortName:  "53",
	}
	expected.DestinationLabels = []v1.FlowLabels{{Key: "dest_iteration", Values: []string{}}}
	expected.SourceLabels = []v1.FlowLabels{
		{Key: "bread", Values: []string{"rye"}},
		{Key: "cheese", Values: []string{"brie"}},
		{Key: "wine", Values: []string{"none"}},
	}

	batch := []v1.FlowLog{}

	for i := 0; i < 10; i++ {
		f := v1.FlowLog{
			StartTime:            fmt.Sprintf("%d", time.Now().Unix()),
			EndTime:              fmt.Sprintf("%d", time.Now().Unix()),
			DestType:             "wep",
			DestNamespace:        "kube-system",
			DestNameAggr:         "kube-dns-*",
			DestIP:               net.ParseIP("10.0.0.10"),
			SourceIP:             net.ParseIP("192.168.1.1"),
			DestServiceNamespace: "kube-system",
			DestServiceName:      "kube-dns",
			DestServicePort:      "53",
			DestServicePortNum:   53,
			Protocol:             "udp",
			DestPort:             53,
			SourceType:           "wep",
			SourceNamespace:      "default",
			SourceNameAggr:       "my-deployment",
			ProcessName:          "-",
			Reporter:             "src",
			Action:               "allowed",

			// Flow stats.
			NumFlows:          3,
			NumFlowsStarted:   3,
			NumFlowsCompleted: 1,

			// Packet stats
			PacketsIn:  i,
			PacketsOut: 2 * i,
			BytesIn:    64,
			BytesOut:   128,

			// Add label information.
			SourceLabels: v1.FlowLogLabels{
				Labels: []string{
					"bread=rye",
					"cheese=brie",
					"wine=none",
				},
			},
			DestLabels: v1.FlowLogLabels{
				// We want a variety of label keys and values,
				// so base this one off of the loop variable.
				// Note: We use a nested terms aggregation to get labels, which has an
				// inherent maximum number of buckets of 10. As a result, if a flow has more than
				// 10 labels, not all of them will be shown. We might be able to use a composite aggregation instead,
				// but these are more expensive.
				Labels: []string{fmt.Sprintf("dest_iteration=%d", i)},
			},
		}

		// Add it to the batch
		batch = append(batch, f)

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

	response, err := b.Create(ctx, bapi.ClusterInfo{Cluster: cluster}, batch)
	require.NoError(t, err)
	require.Equal(t, response.Failed, 0)

	// Refresh the index so that data is readily available for the test. Otherwise, we need to wait
	// for the refresh interval to occur.
	index := fmt.Sprintf("tigera_secure_ee_flows.%s.*", cluster)
	err = testutils.RefreshIndex(ctx, client, index)
	require.NoError(t, err)

	return expected
}
