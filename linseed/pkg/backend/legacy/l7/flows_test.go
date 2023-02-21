// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package l7_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/templates"
	"github.com/projectcalico/calico/linseed/pkg/config"

	elastic "github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/require"
	kapiv1 "k8s.io/apimachinery/pkg/types"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/l7"
	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

var (
	client  lmaelastic.Client
	cache   bapi.Cache
	b       bapi.L7FlowBackend
	lb      bapi.L7LogBackend
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
	b = l7.NewL7FlowBackend(client)
	lb = l7.NewL7LogBackend(client, cache)

	// Create a random cluster name for each test to make sure we don't
	// interfere between tests.
	cluster = testutils.RandomClusterName()

	// Each test should take less than 5 seconds.
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)

	// Function contains teardown logic.
	return func() {
		// Cancel the context.
		cancel()

		// Cleanup any data that might left over from a previous failed run.
		err = testutils.CleanupIndices(context.Background(), esClient, fmt.Sprintf("tigera_secure_ee_l7.%s", cluster))
		require.NoError(t, err)

		// Cancel logging
		logCancel()
	}
}

// TestListL7Flows tests running a real elasticsearch query to list L7 flows.
func TestListL7Flows(t *testing.T) {
	defer setupTest(t)()

	clusterInfo := bapi.ClusterInfo{Cluster: cluster}

	// Put some data into ES so we can query it.
	expected := populateL7FlowData(t, ctx, client, cache, clusterInfo.Cluster)

	// Set time range so that we capture all of the populated flow logs.
	opts := v1.L7FlowParams{}
	opts.TimeRange = &lmav1.TimeRange{}
	opts.TimeRange.From = time.Now().Add(-5 * time.Second)
	opts.TimeRange.To = time.Now().Add(5 * time.Second)

	// Query for flows. There should be a single flow from the populated data.
	r, err := b.List(ctx, clusterInfo, opts)
	require.NoError(t, err)
	require.Len(t, r.Items, 1)

	// Assert that the flow data is populated correctly.
	require.Equal(t, expected, r.Items[0])
}

// populateFlowData writes a series of flow logs to elasticsearch, and returns the FlowLog that we
// should expect to exist as a result. This can be used to assert round-tripping and aggregation against ES is working correctly.
func populateL7FlowData(t *testing.T, ctx context.Context, client lmaelastic.Client, cache bapi.Cache, cluster string) v1.L7Flow {
	// The expected flow log - we'll populate fields as we go.
	expected := v1.L7Flow{}
	expected.Key = v1.L7FlowKey{
		Protocol: "tcp",
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
		DestinationService: v1.ServicePort{
			Service: kapiv1.NamespacedName{
				Name:      "kube-dns",
				Namespace: "kube-system",
			},
			PortName: "dns",
			Port:     53,
		},
	}
	expected.Stats = &v1.L7Stats{}
	expected.Code = 200

	// Used to track the total DurationMean across all L7 logs we create.
	var durationMeanTotal int64 = 0

	numFlows := 10

	batch := []v1.L7Log{}
	for i := 0; i < numFlows; i++ {
		f := v1.L7Log{
			StartTime: time.Now().Unix(),
			EndTime:   time.Now().Unix(),

			ResponseCode: "200",
			URL:          "http://example.com",
			UserAgent:    "test-user",
			Method:       "GET",
			Latency:      5,

			SourceType:      "wep",
			SourceNamespace: "default",
			SourceNameAggr:  "my-deployment",
			SourcePortNum:   1234,

			DestType:             "wep",
			DestNamespace:        "kube-system",
			DestNameAggr:         "kube-dns-*",
			DestServiceNamespace: "kube-system",
			DestServiceName:      "kube-dns",
			DestServicePortName:  "dns",
			DestPortNum:          53,
			DestServicePort:      53,

			DurationMax:  int64(2 * i),
			DurationMean: int64(i),
			BytesIn:      64,
			BytesOut:     128,
			Count:        int64(i),
		}

		// Increment fields on the expected flow based on the flow log that was
		// just added.
		expected.Stats.BytesIn += f.BytesIn
		expected.Stats.BytesOut += f.BytesOut
		expected.LogCount += f.Count
		durationMeanTotal += f.DurationMean

		// Add it to the batch.
		batch = append(batch, f)
	}

	// MinDuration is the smallest recorded value for DurationMean
	// amongst L7 logs used to generate this flow. Since DurationMean for each log
	// is calculated based on the loop variable, we know this must be 0.
	expected.Stats.MinDuration = 0

	// MaxDuration is the largest recorded value for DurationMax
	// amongst L7 logs used to generate this flow. DurationMax for each log
	// is calculated based on the loop variable.
	expected.Stats.MaxDuration = int64((numFlows - 1) * 2)

	// MeanDuration is the average value for DurationMean among L7 logs used to generate
	// this flow.
	expected.Stats.MeanDuration = durationMeanTotal / int64(numFlows)

	// Create the batch all at once.
	response, err := lb.Create(ctx, bapi.ClusterInfo{Cluster: cluster}, batch)
	require.NoError(t, err)
	require.Equal(t, response.Failed, 0)

	// Refresh the index so that data is readily available for the test. Otherwise, we need to wait
	// for the refresh interval to occur.
	index := fmt.Sprintf("tigera_secure_ee_l7.%s.*", cluster)
	err = testutils.RefreshIndex(ctx, client, index)
	require.NoError(t, err)

	return expected
}
