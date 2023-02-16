// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package flows_test

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/projectcalico/calico/libcalico-go/lib/json"

	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/templates"
	"github.com/projectcalico/calico/linseed/pkg/config"

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
	client  lmaelastic.Client
	cache   bapi.Cache
	fb      bapi.FlowBackend
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
	fb = flows.NewFlowBackend(client)
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

// TestListFlows tests running a real elasticsearch query to list flows.
func TestListFlows(t *testing.T) {
	defer setupTest(t)()

	clusterInfo := bapi.ClusterInfo{Cluster: cluster}

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
		WithSourceLabels("bread=rye", "cheese=brie", "wine=none").
		WithPolicies("0|allow-tigera|tigera-system/allow-tigera.cnx-apiserver-access|allow|1")
	expected := populateFlowData(t, ctx, bld, client, clusterInfo.Cluster)

	// Add in the expected policy.
	one := 1
	expected.Policies = []v1.Policy{
		{
			Tier:      "allow-tigera",
			Name:      "cnx-apiserver-access",
			Namespace: "tigera-system",
			Action:    "allow",
			Count:     expected.LogStats.FlowLogCount,
			RuleID:    &one,
		},
	}

	// Set time range so that we capture all of the populated flow logs.
	opts := v1.L3FlowParams{}
	opts.TimeRange = &lmav1.TimeRange{}
	opts.TimeRange.From = time.Now().Add(-5 * time.Minute)
	opts.TimeRange.To = time.Now().Add(5 * time.Minute)

	// Query for flows. There should be a single flow from the populated data.
	r, err := fb.List(ctx, clusterInfo, opts)
	require.NoError(t, err)
	require.Len(t, r.Items, 1)
	require.Nil(t, r.AfterKey)
	require.Empty(t, err)

	// Assert that the flow data is populated correctly.
	require.Equal(t, expected, r.Items[0])
}

// TestMultipleFlows tests that we return multiple flows properly.
func TestMultipleFlows(t *testing.T) {
	defer setupTest(t)()

	// Both flows use the same cluster information.
	clusterInfo := bapi.ClusterInfo{Cluster: cluster}

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
		WithSourceLabels("bread=rye", "cheese=brie", "wine=none")
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
	opts.TimeRange = &lmav1.TimeRange{}
	opts.TimeRange.From = time.Now().Add(-5 * time.Minute)
	opts.TimeRange.To = time.Now().Add(5 * time.Minute)

	// Query for flows. There should be two flows from the populated data.
	r, err := fb.List(ctx, clusterInfo, opts)
	require.NoError(t, err)
	require.Len(t, r.Items, 2)
	require.Nil(t, r.AfterKey)
	require.Empty(t, err)

	// Assert that the flow data is populated correctly.
	require.Equal(t, exp1, r.Items[1])
	require.Equal(t, exp2, r.Items[0])
}

func TestFlowFiltering(t *testing.T) {
	type testCase struct {
		Name   string
		Params v1.L3FlowParams

		// Configuration for which flows are expected to match.
		ExpectFlow1 bool
		ExpectFlow2 bool

		// Number of logs to create
		NumLogs int

		// Whether to perform an equality comparison on the returned
		// flows. Can be useful for tests where stats differ.
		SkipComparison bool
	}

	numExpected := func(tc testCase) int {
		num := 0
		if tc.ExpectFlow1 {
			num++
		}
		if tc.ExpectFlow2 {
			num++
		}
		return num
	}

	testcases := []testCase{
		{
			Name: "should query a flow based on source type",
			Params: v1.L3FlowParams{
				QueryParams: v1.QueryParams{},
				Source:      &v1.Endpoint{Type: "wep"},
			},
			ExpectFlow1: true,
			ExpectFlow2: false, // Flow 2 is type hep, so won't match.
		},
		{
			Name: "should query a flow based on destination type",
			Params: v1.L3FlowParams{
				QueryParams: v1.QueryParams{},
				Destination: &v1.Endpoint{Type: "wep"},
			},
			ExpectFlow1: true,
			ExpectFlow2: false, // Flow 2 is type hep, so won't match.
		},
		{
			Name: "should query a flow based on source namespace",
			Params: v1.L3FlowParams{
				QueryParams: v1.QueryParams{},
				Source:      &v1.Endpoint{Namespace: "default"},
			},
			ExpectFlow1: false, // Flow 1 has source namespace tigera-operator
			ExpectFlow2: true,
		},
		{
			Name: "should query a flow based on destination namespace",
			Params: v1.L3FlowParams{
				QueryParams: v1.QueryParams{},
				Destination: &v1.Endpoint{Namespace: "kube-system"},
			},
			ExpectFlow1: false, // Flow 1 has dest namespace openshift-system
			ExpectFlow2: true,
		},
		{
			Name: "should query a flow based on source port",
			Params: v1.L3FlowParams{
				QueryParams: v1.QueryParams{},
				Source:      &v1.Endpoint{Port: 1010},
			},
			ExpectFlow1: true,
			ExpectFlow2: false, // Flow 2 has source port 5656
		},
		{
			Name: "should query a flow based on destination port",
			Params: v1.L3FlowParams{
				QueryParams: v1.QueryParams{},
				Destination: &v1.Endpoint{Port: 1053},
			},
			ExpectFlow1: true,
			ExpectFlow2: false, // Flow 2 has dest port 53
		},
		{
			Name: "should query a flow based on source label equal selector",
			Params: v1.L3FlowParams{
				QueryParams: v1.QueryParams{},
				SourceSelectors: []v1.LabelSelector{
					{
						Key:      "bread",
						Operator: "=",
						Values:   []string{"rye"},
					},
				},
			},
			ExpectFlow1: true,
			ExpectFlow2: false, // Flow 2 doesn't have the label
		},
		{
			Name: "should query a flow based on dest label equal selector",
			Params: v1.L3FlowParams{
				QueryParams: v1.QueryParams{},
				DestinationSelectors: []v1.LabelSelector{
					{
						Key:      "dest_iteration",
						Operator: "=",
						Values:   []string{"0"},
					},
				},
			},
			// Both flows have this label set on destination.
			ExpectFlow1: true,
			ExpectFlow2: true,
		},
		{
			Name: "should query a flow based on dest label selector matching none",
			Params: v1.L3FlowParams{
				QueryParams: v1.QueryParams{},
				DestinationSelectors: []v1.LabelSelector{
					{
						Key:      "cranberry",
						Operator: "=",
						Values:   []string{"sauce"},
					},
				},
			},
			// neither flow has this label set on destination.
			ExpectFlow1: false,
			ExpectFlow2: false,
		},
		{
			Name: "should query a flow based on multiple source labels",
			Params: v1.L3FlowParams{
				QueryParams: v1.QueryParams{},
				SourceSelectors: []v1.LabelSelector{
					{
						Key:      "bread",
						Operator: "=",
						Values:   []string{"rye"},
					},
					{
						Key:      "cheese",
						Operator: "=",
						Values:   []string{"cheddar"},
					},
				},
			},
			ExpectFlow1: true,
			ExpectFlow2: false, // Missing both labels
		},
		{
			Name: "should query a flow based on multiple destination values for a single label",
			Params: v1.L3FlowParams{
				QueryParams: v1.QueryParams{},
				DestinationSelectors: []v1.LabelSelector{
					{
						Key:      "dest_iteration",
						Operator: "=",
						Values:   []string{"0", "1"},
					},
				},
			},

			// Both have this label.
			ExpectFlow1: true,
			ExpectFlow2: true,
			NumLogs:     2,
		},
		{
			Name: "should query a flow based on multiple destination values for a single label not comprehensive",
			Params: v1.L3FlowParams{
				QueryParams: v1.QueryParams{},
				DestinationSelectors: []v1.LabelSelector{
					{
						Key:      "dest_iteration",
						Operator: "=",
						Values:   []string{"0", "1"},
					},
				},
			},

			// Both have this label.
			ExpectFlow1: true,
			ExpectFlow2: true,
			NumLogs:     4,

			// Skip comparison on this one, since the returned flows don't match the expected ones
			// due to the filtering and the simplicity of our test modeling of flow logs.
			SkipComparison: true,
		},
		{
			Name: "should query a flow based on action",
			Params: v1.L3FlowParams{
				QueryParams: v1.QueryParams{},
				Action:      testutils.ActionPtr(v1.FlowActionAllow),
			},

			ExpectFlow1: true, // Only the first flow allows.
			ExpectFlow2: false,
		},
	}

	for _, testcase := range testcases {
		// Each testcase creates multiple flows, and then uses
		// different filtering parameters provided in the L3FlowParams
		// to query one or more flows.
		t.Run(testcase.Name, func(t *testing.T) {
			defer setupTest(t)()

			clusterInfo := bapi.ClusterInfo{Cluster: cluster}

			// Set the time range for the test. We set this per-test
			// so that the time range captures the windows that the logs
			// are created in.
			tr := &lmav1.TimeRange{}
			tr.From = time.Now().Add(-5 * time.Minute)
			tr.To = time.Now().Add(5 * time.Minute)
			testcase.Params.QueryParams.TimeRange = tr

			numLogs := testcase.NumLogs
			if numLogs == 0 {
				numLogs = 1
			}

			// Template for flow #1.
			bld := NewFlowLogBuilder()
			bld.WithType("wep").
				WithSourceNamespace("tigera-operator").
				WithDestNamespace("openshift-dns").
				WithDestName("openshift-dns-*").
				WithDestIP("10.0.0.10").
				WithDestService("openshift-dns", 53).
				WithDestPort(1053).
				WithSourcePort(1010).
				WithProtocol("udp").
				WithSourceName("tigera-operator").
				WithSourceIP("34.15.66.3").
				WithRandomFlowStats().WithRandomPacketStats().
				WithReporter("src").WithAction("allow").
				WithSourceLabels("bread=rye", "cheese=cheddar", "wine=none")
			exp1 := populateFlowDataN(t, ctx, bld, client, clusterInfo.Cluster, numLogs)

			// Template for flow #2.
			bld2 := NewFlowLogBuilder()
			bld2.WithType("hep").
				WithSourceNamespace("default").
				WithDestNamespace("kube-system").
				WithDestName("kube-dns-*").
				WithDestIP("10.0.0.10").
				WithDestService("kube-dns", 53).
				WithDestPort(53).
				WithSourcePort(5656).
				WithProtocol("udp").
				WithSourceName("my-deployment").
				WithSourceIP("192.168.1.1").
				WithRandomFlowStats().WithRandomPacketStats().
				WithReporter("src").WithAction("deny").
				WithSourceLabels("cheese=brie")
			exp2 := populateFlowDataN(t, ctx, bld2, client, clusterInfo.Cluster, numLogs)

			// Query for flows.
			r, err := fb.List(ctx, clusterInfo, testcase.Params)
			require.NoError(t, err)
			require.Len(t, r.Items, numExpected(testcase))
			require.Nil(t, r.AfterKey)
			require.Empty(t, err)

			if testcase.SkipComparison {
				return
			}

			// Assert that the correct flows are returned.
			if testcase.ExpectFlow1 {
				require.Contains(t, r.Items, exp1)
			}
			if testcase.ExpectFlow2 {
				require.Contains(t, r.Items, exp2)
			}
		})
	}
}

// TestPagination tests that we return multiple flows properly using pagination.
func TestPagination(t *testing.T) {
	defer setupTest(t)()

	// Both flows use the same cluster information.
	clusterInfo := bapi.ClusterInfo{Cluster: cluster}

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
		WithSourceLabels("bread=rye", "cheese=brie", "wine=none")
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
	opts.TimeRange = &lmav1.TimeRange{}
	opts.TimeRange.From = time.Now().Add(-5 * time.Minute)
	opts.TimeRange.To = time.Now().Add(5 * time.Minute)

	// Also set a max results of 1, so that we only get one flow at a time.
	opts.MaxResults = 1

	// Query for flows. There should be a single flow from the populated data.
	r, err := fb.List(ctx, clusterInfo, opts)
	require.NoError(t, err)
	require.Len(t, r.Items, 1)
	require.NotNil(t, r.AfterKey)
	require.Empty(t, err)
	require.Equal(t, exp2, r.Items[0])

	// Now, send another request. This time, passing in the pagination key
	// returned from the first. We should get the second flow.
	opts.AfterKey = r.AfterKey
	r, err = fb.List(ctx, clusterInfo, opts)
	require.NoError(t, err)
	require.Len(t, r.Items, 1)
	require.NotNil(t, r.AfterKey)
	require.Empty(t, err)
	require.Equal(t, exp1, r.Items[0])
}

// Definitions for search results to be used in the tests below.

//go:embed testdata/elastic_valid_flow.json
var validSingleFlow []byte

// Test the handling of various responses from elastic. This suite of tests uses a mock http server
// to return custom responses from elastic without the need for running a real elastic server.
// This can be useful for simulating strange or malformed responses from Elasticsearch.
func TestElasticResponses(t *testing.T) {
	// Set elasticResponse in each test to mock out a given response from Elastic.
	var server *httptest.Server
	var ctx context.Context
	var opts v1.L3FlowParams
	var clusterInfo bapi.ClusterInfo

	// setupAndTeardown initializes and tears down each test.
	setupAndTeardown := func(t *testing.T, elasticResponse []byte) func() {
		// Hook logrus into testing.T
		config.ConfigureLogging("DEBUG")
		logCancel := logutils.RedirectLogrusToTestingT(t)

		// Create a mock server to return elastic responses.
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			_, err := w.Write(elasticResponse)
			require.NoError(t, err)
		}))

		// Configure the elastic client to use the URL of our test server.
		esClient, err := elastic.NewSimpleClient(elastic.SetURL(server.URL))
		require.NoError(t, err)
		client = lmaelastic.NewWithClient(esClient)

		// Create a FlowBackend using the client.
		fb = flows.NewFlowBackend(client)

		// Basic parameters for each test.
		clusterInfo.Cluster = cluster
		opts.TimeRange = &lmav1.TimeRange{}
		opts.TimeRange.From = time.Now().Add(-5 * time.Minute)
		opts.TimeRange.To = time.Now().Add(5 * time.Minute)

		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 1*time.Minute)

		// Teardown goes within this returned func.
		return func() {
			cancel()
			logCancel()
		}
	}

	type testCase struct {
		// Name of the test
		name string

		// Response from elastic to be returned by the mock server.
		response interface{}

		// Expected error
		err bool
	}

	// Define the list of testcases to run
	testCases := []testCase{
		{
			name:     "empty json",
			response: []byte("{}"),
			err:      false,
		},
		{
			name:     "malformed json",
			response: []byte("{"),
			err:      true,
		},
		{
			name:     "timeout",
			response: elastic.SearchResult{TimedOut: true},
			err:      true,
		},
		{
			name:     "valid single flow",
			response: validSingleFlow,
			err:      false,
		},
	}

	for _, testcase := range testCases {
		t.Run(testcase.name, func(t *testing.T) {
			// We allow either raw byte arrays, or structures to be passed
			// as input. If it's a struct, serialize it first.
			var err error
			bs, ok := testcase.response.([]byte)
			if !ok {
				bs, err = json.Marshal(testcase.response)
				require.NoError(t, err)
			}
			defer setupAndTeardown(t, bs)()

			// Query for flows.
			_, err = fb.List(ctx, clusterInfo, opts)
			if testcase.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// populateFlowData writes a series of flow logs to elasticsearch, and returns the FlowLog that we
// should expect to exist as a result. This can be used to assert round-tripping and aggregation against ES is working correctly.
func populateFlowData(t *testing.T, ctx context.Context, b *flowLogBuilder, client lmaelastic.Client, cluster string) v1.L3Flow {
	return populateFlowDataN(t, ctx, b, client, cluster, 10)
}

func populateFlowDataN(t *testing.T, ctx context.Context, b *flowLogBuilder, client lmaelastic.Client, cluster string, n int) v1.L3Flow {
	batch := []v1.FlowLog{}

	// Initialize the expected output based on the given builder template.
	expected := b.ExpectedFlow()

	expected.DestinationLabels = []v1.FlowLabels{
		// A different dest_iteration is applied to each log in the flow.
		{Key: "dest_iteration", Values: []string{}},
	}
	expected.LogStats.FlowLogCount = int64(n)

	for i := 0; i < n; i++ {
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

	// Create the batch.
	response, err := flb.Create(ctx, bapi.ClusterInfo{Cluster: cluster}, batch)
	require.NoError(t, err)
	require.Equal(t, []v1.BulkError(nil), response.Errors)
	require.Equal(t, 0, response.Failed)

	// Refresh the index so that data is readily available for the test. Otherwise, we need to wait
	// for the refresh interval to occur. Lookup the actual index name that was created to
	// perform the refresh.
	indices, err := client.Backend().CatIndices().Do(ctx)
	require.NoError(t, err)
	prefix := fmt.Sprintf("tigera_secure_ee_flows.%s.", cluster)
	var index string
	for _, idx := range indices {
		if strings.HasPrefix(idx.Index, prefix) {
			// Match
			index = idx.Index
			break
		}
	}
	require.NotEqual(t, "", index)
	err = testutils.RefreshIndex(ctx, client, index)
	require.NoError(t, err)

	return *expected
}
