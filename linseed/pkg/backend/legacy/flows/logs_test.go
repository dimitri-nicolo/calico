// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package flows_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/require"

	gojson "encoding/json"

	"github.com/projectcalico/calico/libcalico-go/lib/json"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	backendutils "github.com/projectcalico/calico/linseed/pkg/backend/testutils"
	"github.com/projectcalico/calico/linseed/pkg/testutils"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
)

// TestCreateFlowLog tests running a real elasticsearch query to create a flow log.
func TestCreateFlowLog(t *testing.T) {
	defer setupTest(t)()

	clusterInfo := bapi.ClusterInfo{
		Cluster: cluster,
	}

	// Create a dummy flow.
	f := v1.FlowLog{
		StartTime:            time.Now().Unix(),
		EndTime:              time.Now().Unix(),
		DestType:             "wep",
		DestNamespace:        "kube-system",
		DestNameAggr:         "kube-dns-*",
		DestServiceNamespace: "default",
		DestServiceName:      "kube-dns",
		DestServicePortNum:   testutils.Int64Ptr(53),
		DestIP:               testutils.StringPtr("fe80::0"),
		SourceIP:             testutils.StringPtr("fe80::1"),
		Protocol:             "udp",
		DestPort:             testutils.Int64Ptr(53),
		SourceType:           "wep",
		SourceNamespace:      "default",
		SourceNameAggr:       "my-deployment",
		ProcessName:          "-",
		Reporter:             "src",
		Action:               "allowed",
	}

	response, err := flb.Create(ctx, clusterInfo, []v1.FlowLog{f})
	require.NoError(t, err)
	require.Equal(t, []v1.BulkError(nil), response.Errors)
	require.Equal(t, 0, response.Failed)
}

func TestFlowLogFiltering(t *testing.T) {
	type testCase struct {
		Name   string
		Params v1.FlowLogParams

		// Configuration for which logs are expected to match.
		ExpectLog1 bool
		ExpectLog2 bool

		// Whether to perform an equality comparison on the returned
		// logs. Can be useful for tests where stats differ.
		SkipComparison bool
	}

	numExpected := func(tc testCase) int {
		num := 0
		if tc.ExpectLog1 {
			num++
		}
		if tc.ExpectLog2 {
			num++
		}
		return num
	}

	testcases := []testCase{
		{
			Name:       "should query both flow logs",
			Params:     v1.FlowLogParams{},
			ExpectLog1: true,
			ExpectLog2: true,
		},

		{
			Name: "should support selection based on source type",
			Params: v1.FlowLogParams{
				QueryParams: v1.QueryParams{},
				LogSelectionParams: v1.LogSelectionParams{
					Selector: "source_type = wep",
				},
			},
			ExpectLog1: true,
			ExpectLog2: false, // Source is a hep.
		},

		{
			Name: "should support NOT selection based on source type",
			Params: v1.FlowLogParams{
				QueryParams: v1.QueryParams{},
				LogSelectionParams: v1.LogSelectionParams{
					Selector: "source_type != wep",
				},
			},
			ExpectLog1: false,
			ExpectLog2: true,
		},

		{
			Name: "should support combined selectors",
			Params: v1.FlowLogParams{
				QueryParams: v1.QueryParams{},
				LogSelectionParams: v1.LogSelectionParams{
					// This selector matches both.
					Selector: "(source_type != wep AND dest_type != wep) OR proto = udp AND dest_port = 1053",
				},
			},
			ExpectLog1: true,
			ExpectLog2: true,
		},
		{
			Name: "should support NOT with combined selectors",
			Params: v1.FlowLogParams{
				QueryParams: v1.QueryParams{},
				LogSelectionParams: v1.LogSelectionParams{
					// Should match neither.
					Selector: "NOT ((source_type != wep AND dest_type != wep) OR proto = udp AND dest_port = 1053)",
				},
			},
			ExpectLog1: false,
			ExpectLog2: false,
		},
	}

	for _, testcase := range testcases {
		// Each testcase creates multiple flow logs, and then uses
		// different filtering parameters provided in the params
		// to query one or more flow logs.
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

			// Template for flow #1.
			bld := backendutils.NewFlowLogBuilder()
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
				WithReporter("src").WithAction("allowed").
				WithSourceLabels("bread=rye", "cheese=cheddar", "wine=none")

			fl1, err := bld.Build()
			require.NoError(t, err)

			// Template for flow #2.
			bld2 := backendutils.NewFlowLogBuilder()
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
				WithReporter("src").WithAction("allowed").
				WithSourceLabels("cheese=brie")
			fl2, err := bld2.Build()
			require.NoError(t, err)

			response, err := flb.Create(ctx, clusterInfo, []v1.FlowLog{*fl1, *fl2})
			require.NoError(t, err)
			require.Equal(t, []v1.BulkError(nil), response.Errors)
			require.Equal(t, 0, response.Failed)

			err = backendutils.RefreshIndex(ctx, client, "tigera_secure_ee_flows.*")
			require.NoError(t, err)

			// Query for flow logs.
			r, err := flb.List(ctx, clusterInfo, &testcase.Params)
			require.NoError(t, err)
			require.Len(t, r.Items, numExpected(testcase))
			require.Nil(t, r.AfterKey)
			require.Empty(t, err)

			if testcase.SkipComparison {
				return
			}

			copyOfLogs := backendutils.AssertLogIDAndCopyFlowLogsWithoutID(t, r)

			// Assert that the correct logs are returned.
			if testcase.ExpectLog1 {
				require.Contains(t, copyOfLogs, *fl1)
			}
			if testcase.ExpectLog2 {
				require.Contains(t, copyOfLogs, *fl2)
			}
		})
	}
}

// TestAggregations tests running a real elasticsearch query to get aggregations.
func TestAggregations(t *testing.T) {
	t.Run("should return time-series flow log aggregation results", func(t *testing.T) {
		defer setupTest(t)()
		clusterInfo := bapi.ClusterInfo{Cluster: cluster}

		// Start the test numLogs minutes in the past.
		numLogs := 5
		timeBetweenLogs := 10 * time.Second
		testStart := time.Unix(0, 0)
		now := testStart.Add(time.Duration(numLogs) * time.Minute)

		// Several dummy logs.
		logs := []v1.FlowLog{}
		for i := 1; i < numLogs; i++ {
			start := testStart.Add(time.Duration(i) * time.Second)
			end := start.Add(timeBetweenLogs)
			log := v1.FlowLog{
				StartTime: start.Unix(),
				EndTime:   end.Unix(),
				BytesIn:   1,
			}
			logs = append(logs, log)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		resp, err := flb.Create(ctx, clusterInfo, logs)
		require.NoError(t, err)
		require.Empty(t, resp.Errors)

		// Refresh.
		index := fmt.Sprintf("tigera_secure_ee_flows.%s.", cluster)
		err = backendutils.RefreshIndex(ctx, client, index)
		require.NoError(t, err)

		params := v1.FlowLogAggregationParams{}
		params.TimeRange = &lmav1.TimeRange{}
		params.TimeRange.From = testStart
		params.TimeRange.To = now
		params.NumBuckets = 4

		// Add a simple aggregation to add up the total bytes_in from the logs.
		sumAgg := elastic.NewSumAggregation().Field("bytes_in")
		src, err := sumAgg.Source()
		require.NoError(t, err)
		bytes, err := json.Marshal(src)
		require.NoError(t, err)
		params.Aggregations = map[string]gojson.RawMessage{"count": bytes}

		// Use the backend to perform a query.
		aggs, err := flb.Aggregations(ctx, clusterInfo, &params)
		require.NoError(t, err)
		require.NotNil(t, aggs)

		ts, ok := aggs.AutoDateHistogram("tb")
		require.True(t, ok)

		// We asked for 4 buckets.
		require.Len(t, ts.Buckets, 4)

		times := []string{"11", "12", "13", "14"}

		for i, b := range ts.Buckets {
			require.Equal(t, int64(1), b.DocCount, fmt.Sprintf("Bucket %d", i))

			// We asked for a count agg, which should include a single log
			// in each bucket.
			count, ok := b.Sum("count")
			require.True(t, ok, "Bucket missing count agg")
			require.NotNil(t, count.Value)
			require.Equal(t, float64(1), *count.Value)

			// The key should be the timestamp for the bucket.
			require.NotNil(t, b.KeyAsString)
			require.Equal(t, times[i], *b.KeyAsString)
		}
	})

	t.Run("should return aggregate stats", func(t *testing.T) {
		defer setupTest(t)()
		clusterInfo := bapi.ClusterInfo{Cluster: cluster}

		// Start the test numLogs minutes in the past.
		numLogs := 5
		timeBetweenLogs := 10 * time.Second
		testStart := time.Unix(0, 0)
		now := testStart.Add(time.Duration(numLogs) * time.Minute)

		// Several dummy logs.
		logs := []v1.FlowLog{}
		for i := 1; i < numLogs; i++ {
			start := testStart.Add(time.Duration(i) * time.Second)
			end := start.Add(timeBetweenLogs)
			log := v1.FlowLog{
				StartTime: start.Unix(),
				EndTime:   end.Unix(),
				BytesIn:   1,
			}
			logs = append(logs, log)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		resp, err := flb.Create(ctx, clusterInfo, logs)
		require.NoError(t, err)
		require.Empty(t, resp.Errors)

		// Refresh.
		index := fmt.Sprintf("tigera_secure_ee_flows.%s.", cluster)
		err = backendutils.RefreshIndex(ctx, client, index)
		require.NoError(t, err)

		params := v1.FlowLogAggregationParams{}
		params.TimeRange = &lmav1.TimeRange{}
		params.TimeRange.From = testStart
		params.TimeRange.To = now
		params.NumBuckets = 0 // Return aggregated stats over the whole time range.

		// Add a simple aggregation to add up the total bytes_in from the logs.
		sumAgg := elastic.NewSumAggregation().Field("bytes_in")
		src, err := sumAgg.Source()
		require.NoError(t, err)
		bytes, err := json.Marshal(src)
		require.NoError(t, err)
		params.Aggregations = map[string]gojson.RawMessage{"count": bytes}

		// Use the backend to perform a stats query.
		result, err := flb.Aggregations(ctx, clusterInfo, &params)
		require.NoError(t, err)

		// We should get a sum aggregation with all 4 logs.
		count, ok := result.ValueCount("count")
		require.True(t, ok)
		require.NotNil(t, count.Value)
		require.Equal(t, float64(4), *count.Value)
	})
}
