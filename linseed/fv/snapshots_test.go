// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"fmt"
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

	t.Run("should support pagination", func(t *testing.T) {
		defer complianceSetupAndTeardown(t)()

		// Create 5 Snapshots.
		logTime := time.Unix(100, 0).UTC()
		for i := 0; i < 5; i++ {
			snapshots := []v1.Snapshot{
				{
					ResourceList: list.TimestampedResourceList{
						ResourceList: &apiv3.NetworkPolicyList{
							TypeMeta: metav1.TypeMeta{Kind: "NetworkPolicy", APIVersion: "projectcalico.org/v3"},
							ListMeta: metav1.ListMeta{},
							Items: []apiv3.NetworkPolicy{
								{
									TypeMeta: metav1.TypeMeta{Kind: "NetworkPolicy", APIVersion: "projectcalico.org/v3"},
									ObjectMeta: metav1.ObjectMeta{
										Name:      fmt.Sprintf("np-%d", i),
										Namespace: "default",
									},
								},
							},
						},
						RequestStartedTimestamp:   metav1.Time{Time: logTime.Add(time.Duration(i) * time.Second)},
						RequestCompletedTimestamp: metav1.Time{Time: logTime.Add(time.Duration(2*i) * time.Second)},
					},
				},
			}
			bulk, err := cli.Compliance(cluster).Snapshots().Create(ctx, snapshots)
			require.NoError(t, err)
			require.Equal(t, bulk.Succeeded, 1, "create snapshots did not succeed")
		}

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_snapshots*")

		// Read them back one at a time.
		var afterKey map[string]interface{}
		for i := 0; i < 5; i++ {
			params := v1.SnapshotParams{
				QueryParams: v1.QueryParams{
					TimeRange: &lmav1.TimeRange{
						From: logTime.Add(-20 * time.Second),
						To:   logTime.Add(20 * time.Second),
					},
					MaxPageSize: 1,
					AfterKey:    afterKey,
				},
				Sort: []v1.SearchRequestSortBy{
					{
						Field: "requestStartedTimestamp",
					},
				},
			}
			resp, err := cli.Compliance(cluster).Snapshots().List(ctx, &params)
			require.NoError(t, err)
			require.Equal(t, 1, len(resp.Items))
			expected := []v1.Snapshot{
				{
					ResourceList: list.TimestampedResourceList{
						ResourceList: &apiv3.NetworkPolicyList{
							TypeMeta: metav1.TypeMeta{Kind: "NetworkPolicy", APIVersion: "projectcalico.org/v3"},
							ListMeta: metav1.ListMeta{},
							Items: []apiv3.NetworkPolicy{
								{
									TypeMeta: metav1.TypeMeta{Kind: "NetworkPolicy", APIVersion: "projectcalico.org/v3"},
									ObjectMeta: metav1.ObjectMeta{
										Name:      fmt.Sprintf("np-%d", i),
										Namespace: "default",
									},
								},
							},
						},
						RequestStartedTimestamp:   metav1.Time{Time: logTime.Add(time.Duration(i) * time.Second)},
						RequestCompletedTimestamp: metav1.Time{Time: logTime.Add(time.Duration(2*i) * time.Second)},
					},
				},
			}
			actual := snapshotsWithUTCTime(resp)
			require.Equal(t, expected, actual, fmt.Sprintf("Snapshot #%d did not match", i))
			require.NotNil(t, resp.AfterKey)
			require.Equal(t, resp.TotalHits, int64(5))

			// Use the afterKey for the next query.
			afterKey = resp.AfterKey
		}

		// If we query once more, we should get no results, and no afterkey, since
		// we have paged through all the items.
		params := v1.SnapshotParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: logTime.Add(-20 * time.Second),
					To:   logTime.Add(20 * time.Second),
				},
				MaxPageSize: 1,
				AfterKey:    afterKey,
			},
		}
		resp, err := cli.Compliance(cluster).Snapshots().List(ctx, &params)
		require.NoError(t, err)
		require.Equal(t, 0, len(resp.Items))
		require.Nil(t, resp.AfterKey)
	})
}

func snapshotsWithUTCTime(resp *v1.List[v1.Snapshot]) []v1.Snapshot {
	for idx, snapshot := range resp.Items {
		utcStartTime := snapshot.ResourceList.RequestStartedTimestamp.UTC()
		utcEndTime := snapshot.ResourceList.RequestCompletedTimestamp.UTC()
		resp.Items[idx].ResourceList.RequestStartedTimestamp = metav1.Time{Time: utcStartTime}
		resp.Items[idx].ResourceList.RequestCompletedTimestamp = metav1.Time{Time: utcEndTime}
		resp.Items[idx].ID = ""
	}
	return resp.Items
}
