// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package flows_test

import (
	"testing"
	"time"

	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"

	"github.com/stretchr/testify/require"

	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
)

// TestCreateFlowLog tests running a real elasticsearch query to create a flow log.
func TestCreateFlowLog(t *testing.T) {
	defer setupTest(t)()

	clusterInfo := bapi.ClusterInfo{
		Cluster: "testcluster",
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
	// Use the same cluster information.
	clusterInfo := bapi.ClusterInfo{Cluster: "cluster"}

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

			// Set the time range for the test. We set this per-test
			// so that the time range captures the windows that the logs
			// are created in.
			tr := &lmav1.TimeRange{}
			tr.From = time.Now().Add(-5 * time.Minute)
			tr.To = time.Now().Add(5 * time.Minute)
			testcase.Params.QueryParams.TimeRange = tr

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
				WithReporter("src").WithAction("allowed").
				WithSourceLabels("bread=rye", "cheese=cheddar", "wine=none")

			fl1, err := bld.Build()
			require.NoError(t, err)

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
				WithReporter("src").WithAction("allowed").
				WithSourceLabels("cheese=brie")
			fl2, err := bld2.Build()
			require.NoError(t, err)

			response, err := flb.Create(ctx, clusterInfo, []v1.FlowLog{*fl1, *fl2})
			require.NoError(t, err)
			require.Equal(t, []v1.BulkError(nil), response.Errors)
			require.Equal(t, 0, response.Failed)

			err = testutils.RefreshIndex(ctx, client, "tigera_secure_ee_flows.*")
			require.NoError(t, err)

			// Query for flow logs.
			r, err := flb.List(ctx, clusterInfo, testcase.Params)
			require.NoError(t, err)
			require.Len(t, r.Items, numExpected(testcase))
			require.Nil(t, r.AfterKey)
			require.Empty(t, err)

			if testcase.SkipComparison {
				return
			}

			// Assert that the correct logs are returned.
			if testcase.ExpectLog1 {
				require.Contains(t, r.Items, *fl1)
			}
			if testcase.ExpectLog2 {
				require.Contains(t, r.Items, *fl2)
			}
		})
	}
}
