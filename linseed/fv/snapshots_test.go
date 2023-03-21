// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"testing"
	"time"

	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"

	"github.com/stretchr/testify/require"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
	"github.com/projectcalico/calico/lma/pkg/list"
)

func TestFV_Snapshots(t *testing.T) {
	t.Run("should return an empty list if there are no snapshots", func(t *testing.T) {
		defer complianceSetupAndTeardown(t)()

		params := v1.SnapshotParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now(),
				},
			},
		}

		// Perform a query.
		snapshots, err := cli.Compliance(cluster).Snapshots().List(ctx, &params)
		require.NoError(t, err)
		require.Equal(t, []v1.Snapshot{}, snapshots.Items)
	})

	t.Run("should create and list snapshots", func(t *testing.T) {
		defer complianceSetupAndTeardown(t)()

		snapshots := v1.Snapshot{
			ResourceList: list.TimestampedResourceList{
				ResourceList: &apiv3.NetworkPolicyList{
					TypeMeta: metav1.TypeMeta{Kind: "NetworkPolicy", APIVersion: "projectcalico.org/v3"},
					ListMeta: metav1.ListMeta{},
					Items: []apiv3.NetworkPolicy{
						{
							TypeMeta: metav1.TypeMeta{Kind: "NetworkPolicy", APIVersion: "projectcalico.org/v3"},
							ObjectMeta: metav1.ObjectMeta{
								Name:      "np1",
								Namespace: "default",
							},
						},
					},
				},
				RequestStartedTimestamp:   metav1.Time{Time: time.Unix(1, 0)},
				RequestCompletedTimestamp: metav1.Time{Time: time.Unix(2, 0)},
			},
		}
		bulk, err := cli.Compliance(cluster).Snapshots().Create(ctx, []v1.Snapshot{snapshots})
		require.NoError(t, err)
		require.Equal(t, bulk.Succeeded, 1, "create did not succeed")

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_snapshots*")

		// Read it back.
		params := v1.SnapshotParams{}
		resp, err := cli.Compliance(cluster).Snapshots().List(ctx, &params)
		require.NoError(t, err)

		// The ID should be set.
		require.Len(t, resp.Items, 1)
		require.NotEqual(t, "", resp.Items[0].ID)
		resp.Items[0].ID = ""
		require.Equal(t, snapshots, resp.Items[0])

		// Read it back, using a time range
		params = v1.SnapshotParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Unix(0, 0),
					To:   time.Unix(2, 0),
				},
			},
		}
		resp, err = cli.Compliance(cluster).Snapshots().List(ctx, &params)
		require.NoError(t, err)

		// The ID should be set.
		require.Len(t, resp.Items, 1)
		require.NotEqual(t, "", resp.Items[0].ID)
		resp.Items[0].ID = ""
		require.Equal(t, snapshots, resp.Items[0])
	})
}
