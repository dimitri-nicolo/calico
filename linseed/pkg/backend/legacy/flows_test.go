// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package legacy_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	elastic "github.com/olivere/elastic/v7"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
	"github.com/stretchr/testify/assert"
)

// TestListFlows tests running a real elasticsearch query to list flows.
func TestListFlows(t *testing.T) {
	clusterInfo := bapi.ClusterInfo{Cluster: "mycluster"}

	// Create an elasticsearch client to use for the test. For this test, we use a real
	// elasticsearch instance created via "make run-elastic".
	esClient, err := elastic.NewSimpleClient(elastic.SetURL("http://localhost:9200"))
	assert.NoError(t, err)
	client := lmaelastic.NewWithClient(esClient)

	// Instantiate a FlowBackend.
	b := legacy.NewFlowBackend(client)

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
	r, err := b.List(ctx, clusterInfo, opts)
	assert.NoError(t, err)
	assert.Len(t, r, 1)

	// Assert that the flow data is populated correctly.
	assert.Equal(t, r[0], expected)

	// Clean up after ourselves by deleting the index.
	_, err = esClient.DeleteIndex(fmt.Sprintf("tigera_secure_ee_flows.%s", clusterInfo.Cluster)).Do(ctx)
	assert.NoError(t, err)
}

// populateFlowData writes a series of flow logs to elasticsearch, and returns the FlowLog that we
// should expect to exist as a result. This can be used to assert round-tripping and aggregation against ES is working correctly.
func populateFlowData(t *testing.T, ctx context.Context, client lmaelastic.Client, cluster string) v1.L3Flow {
	// Clear out any old data first.
	_, _ = client.Backend().DeleteIndex(fmt.Sprintf("tigera_secure_ee_flows.%s", cluster)).Do(ctx)

	// Instantiate a FlowBackend.
	b := legacy.NewFlowLogBackend(client)

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
	expected.LogStats = &v1.LogStats{}
	expected.Service = &v1.Service{
		Name:      "kube-dns",
		Namespace: "kube-system",
		Port:      53,
		PortName:  "53",
	}
	// Service:           {Name: "kube-dns", Namespace: "default", Port: 53, PortName: "53"},

	for i := 0; i < 10; i++ {
		f := legacy.FlowLog{
			StartTime:            fmt.Sprintf("%d", time.Now().Unix()),
			EndTime:              fmt.Sprintf("%d", time.Now().Unix()),
			DestType:             "wep",
			DestNamespace:        "kube-system",
			DestNameAggr:         "kube-dns-*",
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
		}

		err := b.Create(ctx, bapi.ClusterInfo{Cluster: cluster}, f)
		assert.NoError(t, err)

		// Increment fields on the expected flow based on the flow log that was
		// just added.
		expected.LogStats.LogCount += int64(f.NumFlows)
		expected.LogStats.Started += int64(f.NumFlowsStarted)
		expected.LogStats.Completed += int64(f.NumFlowsCompleted)
		expected.TrafficStats.BytesIn += int64(f.BytesIn)
		expected.TrafficStats.BytesOut += int64(f.BytesOut)
		expected.TrafficStats.PacketsIn += int64(f.PacketsIn)
		expected.TrafficStats.PacketsOut += int64(f.PacketsOut)
	}

	return expected
}
