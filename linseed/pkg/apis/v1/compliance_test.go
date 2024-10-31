// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package v1_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

func TestReportDataUID(t *testing.T) {
	rd := apiv3.ReportData{
		ReportName:     "name",
		ReportTypeName: "type",
		StartTime:      metav1.Time{Time: time.Unix(10, 0).UTC()},
		EndTime:        metav1.Time{Time: time.Unix(20, 0).UTC()},
	}
	rdd := v1.ReportData{&rd, "summary", ""}
	uid := rdd.UID()
	require.Equal(t, "name_type_4a4f4ca3-557c-5b88-b993-84c7fa0828d3", uid)
}

func TestBenchmarksUID(t *testing.T) {
	bm := v1.Benchmarks{}
	bm.Timestamp = metav1.Time{Time: time.Unix(10, 0).UTC()}
	bm.Type = v1.TypeKubernetes
	bm.NodeName = "node-01"
	uid := bm.UID()
	require.Equal(t, "1970-01-01T00:00:10Z::kube::node-01", uid)
}

func TestBenchmarksEquality(t *testing.T) {
	type testcase struct {
		Name  string
		BM1   v1.Benchmarks
		BM2   v1.Benchmarks
		Equal bool
	}

	testcases := []testcase{
		{
			Name:  "two empty benchmarks",
			BM1:   v1.Benchmarks{},
			BM2:   v1.Benchmarks{},
			Equal: true,
		},
		{
			Name:  "two benchmarks with different errors",
			BM1:   v1.Benchmarks{Error: "err1"},
			BM2:   v1.Benchmarks{Error: "err2"},
			Equal: false,
		},
		{
			Name:  "two benchmarks with different versions",
			BM1:   v1.Benchmarks{Version: "v1"},
			BM2:   v1.Benchmarks{Version: "v2"},
			Equal: false,
		},
		{
			Name:  "two benchmarks with different types",
			BM1:   v1.Benchmarks{Type: v1.TypeKubernetes},
			BM2:   v1.Benchmarks{},
			Equal: false,
		},
		{
			Name:  "two benchmarks with different node names",
			BM1:   v1.Benchmarks{NodeName: "node1"},
			BM2:   v1.Benchmarks{NodeName: "node2"},
			Equal: false,
		},
		{
			Name:  "two benchmarks with different tests",
			BM1:   v1.Benchmarks{Tests: []v1.BenchmarkTest{{Section: "sec1"}, {Section: "sec2"}}},
			BM2:   v1.Benchmarks{Tests: []v1.BenchmarkTest{{Section: "sec2"}, {Section: "sec1"}}},
			Equal: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			// Check equality in both directions.
			eq1 := tc.BM1.Equal(tc.BM2)
			eq2 := tc.BM2.Equal(tc.BM1)
			require.Equal(t, tc.Equal, eq1)
			require.Equal(t, tc.Equal, eq2)
		})
	}
}
