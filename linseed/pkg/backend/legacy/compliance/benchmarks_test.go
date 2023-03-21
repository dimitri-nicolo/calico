// Copyright (c) 2023 Tigera All rights reserved.

package compliance_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	backendutils "github.com/projectcalico/calico/linseed/pkg/backend/testutils"
)

func TestCreateBenchmarks(t *testing.T) {
	defer setupTest(t)()

	f := v1.Benchmarks{
		Version:           "v1",
		KubernetesVersion: "v1.0",
		Type:              v1.TypeKubernetes,
		NodeName:          "lodestone",
		Timestamp:         metav1.Time{Time: time.Unix(1, 0)},
		Error:             "",
		Tests: []v1.BenchmarkTest{
			{
				Section:     "a.1",
				SectionDesc: "testing the test",
				TestNumber:  "1",
				TestDesc:    "making sure that we're right",
				TestInfo:    "information is fluid",
				Status:      "Just swell",
				Scored:      true,
			},
		},
	}

	response, err := bb.Create(ctx, clusterInfo, []v1.Benchmarks{f})
	require.NoError(t, err)
	require.Equal(t, []v1.BulkError(nil), response.Errors)
	require.Equal(t, 0, response.Failed)

	err = backendutils.RefreshIndex(ctx, client, "tigera_secure_ee_benchmark_results.*")
	require.NoError(t, err)

	// Read it back and check it matches.
	p := v1.BenchmarksParams{}
	resp, err := bb.List(ctx, clusterInfo, &p)
	require.NoError(t, err)
	require.Len(t, resp.Items, 1)
	require.NotEqual(t, "", resp.Items[0].ID)
	resp.Items[0].ID = ""
	require.Equal(t, f, resp.Items[0])
}
