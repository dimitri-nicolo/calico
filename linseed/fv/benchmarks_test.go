// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"testing"
	"time"

	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"

	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
)

func TestFV_ComplianceBenchmarks(t *testing.T) {
	t.Run("should return an empty list if there are no benchmarks", func(t *testing.T) {
		defer complianceSetupAndTeardown(t)()

		params := v1.BenchmarksParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now(),
				},
			},
		}

		// Perform a query.
		benchmarks, err := cli.Compliance(cluster).Benchmarks().List(ctx, &params)
		require.NoError(t, err)
		require.Equal(t, []v1.Benchmarks{}, benchmarks.Items)
	})

	t.Run("should create and list benchmarks", func(t *testing.T) {
		defer complianceSetupAndTeardown(t)()

		benchmarks := v1.Benchmarks{
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
		bulk, err := cli.Compliance(cluster).Benchmarks().Create(ctx, []v1.Benchmarks{benchmarks})
		require.NoError(t, err)
		require.Equal(t, bulk.Succeeded, 1, "create did not succeed")

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_benchmark_results*")

		// Read it back, passing an ID query.
		params := v1.BenchmarksParams{ID: benchmarks.UID()}
		resp, err := cli.Compliance(cluster).Benchmarks().List(ctx, &params)
		require.NoError(t, err)

		// The ID should be set.
		require.Len(t, resp.Items, 1)
		require.Equal(t, benchmarks.UID(), resp.Items[0].ID)
		resp.Items[0].ID = ""
		require.Equal(t, benchmarks, resp.Items[0])

		// Read it back, using a time range
		params = v1.BenchmarksParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Unix(0, 0),
					To:   time.Unix(2, 0),
				},
			},
		}
		resp, err = cli.Compliance(cluster).Benchmarks().List(ctx, &params)
		require.NoError(t, err)

		// The ID should be set.
		require.Len(t, resp.Items, 1)
		require.Equal(t, benchmarks.UID(), resp.Items[0].ID)
		resp.Items[0].ID = ""
		require.Equal(t, benchmarks, resp.Items[0])
	})
}
