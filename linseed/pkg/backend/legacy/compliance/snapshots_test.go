// Copyright (c) 2023 Tigera All rights reserved.

package compliance_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/calico/libcalico-go/lib/resources"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	backendutils "github.com/projectcalico/calico/linseed/pkg/backend/testutils"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
	"github.com/projectcalico/calico/lma/pkg/list"
)

func TestCreateSnapshots(t *testing.T) {
	type snapshot struct {
		Name         string
		ResourceList resources.ResourceList
	}

	testcases := []snapshot{
		{
			Name: "network policy",
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
		},
		{
			Name: "namespace",
			ResourceList: &corev1.NamespaceList{
				TypeMeta: metav1.TypeMeta{Kind: "Namespace", APIVersion: "v1"},
				ListMeta: metav1.ListMeta{},
				Items: []corev1.Namespace{
					{
						TypeMeta: metav1.TypeMeta{Kind: "Namespace", APIVersion: "v1"},
						ObjectMeta: metav1.ObjectMeta{
							Name: "default",
							Annotations: map[string]string{
								"hymn": "invincible",
							},
							Labels: map[string]string{
								"goblin": "raging",
							},
							UID:               "1234-4321",
							ResourceVersion:   "77777",
							CreationTimestamp: metav1.Time{Time: time.Unix(1, 0)},
							OwnerReferences: []metav1.OwnerReference{
								{
									APIVersion: "v1",
									Kind:       "Owner",
									Name:       "grimace",
								},
							},
						},
					},
				},
			},
		},
	}

	// Run each test with a tenant specified, and also without a tenant.
	for _, tenant := range []string{backendutils.RandomTenantName(), ""} {
		for _, tc := range testcases {
			name := fmt.Sprintf("should write and read %s snapshots (tenant=%s)", tc.Name, tenant)
			RunAllModes(t, name, func(t *testing.T) {
				trl := list.TimestampedResourceList{
					ResourceList:              tc.ResourceList,
					RequestStartedTimestamp:   metav1.Time{Time: time.Unix(1, 0)},
					RequestCompletedTimestamp: metav1.Time{Time: time.Unix(2, 0)},
				}
				f := v1.Snapshot{
					ResourceList: trl,
				}

				for _, cluster := range []string{cluster1, cluster2, cluster3} {
					clusterInfo := bapi.ClusterInfo{Cluster: cluster, Tenant: tenant}

					response, err := sb.Create(ctx, clusterInfo, []v1.Snapshot{f})
					require.NoError(t, err)
					require.Equal(t, []v1.BulkError(nil), response.Errors)
					require.Equal(t, 0, response.Failed)

					err = backendutils.RefreshIndex(ctx, client, sIndexGetter.Index(clusterInfo))
					require.NoError(t, err)
				}

				gvk := tc.ResourceList.GetObjectKind().GroupVersionKind()
				apiVersion := gvk.Version
				if gvk.Group != "" && gvk.Version != "" {
					apiVersion = strings.Join([]string{gvk.Group, gvk.Version}, "/")
				}
				p := v1.SnapshotParams{
					TypeMatch: &metav1.TypeMeta{
						APIVersion: apiVersion,
						Kind:       gvk.Kind,
					},
					Sort: []v1.SearchRequestSortBy{
						{
							Field:      "requestCompletedTimestamp",
							Descending: true,
						},
					},
				}

				t.Run("should query single cluster", func(t *testing.T) {
					clusterInfo := bapi.ClusterInfo{Cluster: cluster1, Tenant: tenant}

					// Read it back and check it matches.
					resp, err := sb.List(ctx, clusterInfo, &p)
					require.NoError(t, err)
					require.Len(t, resp.Items, 1)
					backendutils.AssertSnapshotClusterAndReset(t, clusterInfo.Cluster, &resp.Items[0])
					backendutils.AssertSnapshotGeneratedTimeAndReset(t, &resp.Items[0])
					require.Equal(t, trl, resp.Items[0].ResourceList)
				})

				t.Run("should query multiple clusters", func(t *testing.T) {
					selectedClusters := []string{cluster2, cluster3}
					p.SetClusters(selectedClusters)
					resp, err := sb.List(ctx, bapi.ClusterInfo{Cluster: v1.QueryMultipleClusters, Tenant: tenant}, &p)
					require.NoError(t, err)
					require.Len(t, resp.Items, 2)
					for _, cluster := range selectedClusters {
						require.Truef(t, backendutils.MatchIn(resp.Items, backendutils.SnapshotClusterEquals(cluster)), "expected cluster %s in response", cluster)
					}
				})
			})

			RunAllModes(t, "should ensure data does not overlap", func(t *testing.T) {
				clusterInfo := bapi.ClusterInfo{Cluster: cluster1, Tenant: tenant}
				anotherClusterInfo := bapi.ClusterInfo{Cluster: backendutils.RandomClusterName(), Tenant: tenant}

				trl := list.TimestampedResourceList{
					ResourceList:              tc.ResourceList,
					RequestStartedTimestamp:   metav1.Time{Time: time.Unix(1, 0)},
					RequestCompletedTimestamp: metav1.Time{Time: time.Unix(2, 0)},
				}
				s1 := v1.Snapshot{
					ResourceList: trl,
				}
				s2 := v1.Snapshot{
					ResourceList: trl,
				}

				_, err := sb.Create(ctx, clusterInfo, []v1.Snapshot{s1})
				require.NoError(t, err)

				_, err = sb.Create(ctx, anotherClusterInfo, []v1.Snapshot{s2})
				require.NoError(t, err)

				err = backendutils.RefreshIndex(ctx, client, sIndexGetter.Index(clusterInfo))
				require.NoError(t, err)

				err = backendutils.RefreshIndex(ctx, client, sIndexGetter.Index(anotherClusterInfo))
				require.NoError(t, err)

				// Read back data a managed cluster and check it matches.
				p1 := v1.SnapshotParams{}
				resp, err := sb.List(ctx, clusterInfo, &p1)
				require.NoError(t, err)
				require.Len(t, resp.Items, 1)
				require.NotEmpty(t, resp.Items[0].ID)
				// Overwrite the ID to match the generated one
				s1.ID = s1.ResourceList.String()
				backendutils.AssertSnapshotClusterAndReset(t, clusterInfo.Cluster, &resp.Items[0])
				backendutils.AssertSnapshotGeneratedTimeAndReset(t, &resp.Items[0])
				require.Equal(t, s1, resp.Items[0])

				// Read back data a managed cluster and check it matches.
				p2 := v1.SnapshotParams{}
				resp, err = sb.List(ctx, anotherClusterInfo, &p2)
				require.NoError(t, err)
				require.Len(t, resp.Items, 1)
				require.NotEmpty(t, resp.Items[0].ID)
				// Overwrite the ID to match the generated one
				s2.ID = s2.ResourceList.String()
				backendutils.AssertSnapshotClusterAndReset(t, anotherClusterInfo.Cluster, &resp.Items[0])
				backendutils.AssertSnapshotGeneratedTimeAndReset(t, &resp.Items[0])
				require.Equal(t, s2, resp.Items[0])
			})

		}
	}

	RunAllModes(t, "invalid ClusterInfo", func(t *testing.T) {
		f := v1.Snapshot{}
		p := v1.SnapshotParams{}

		// Empty cluster info.
		empty := bapi.ClusterInfo{}
		_, err := sb.Create(ctx, empty, []v1.Snapshot{f})
		require.Error(t, err)
		_, err = sb.List(ctx, empty, &p)
		require.Error(t, err)

		// Invalid tenant ID in cluster info.
		badTenant := bapi.ClusterInfo{Cluster: cluster1, Tenant: "one,two"}
		_, err = sb.Create(ctx, badTenant, []v1.Snapshot{f})
		require.Error(t, err)
		_, err = sb.List(ctx, badTenant, &p)
		require.Error(t, err)
	})
}

func TestSnapshotsFiltering(t *testing.T) {
	type testcase struct {
		Name    string
		Params  *v1.SnapshotParams
		Expect1 bool
		Expect2 bool
	}

	testcases := []testcase{
		{
			Name: "should filter snapshots based on timestamp",
			Params: &v1.SnapshotParams{
				QueryParams: v1.QueryParams{
					TimeRange: &lmav1.TimeRange{
						From: time.Unix(1000, 0),
						To:   time.Unix(3000, 0),
					},
				},
			},
			Expect1: false,
			Expect2: true,
		},
	}

	for _, tc := range testcases {
		// Run each test with a tenant specified, and also without a tenant.
		for _, tenant := range []string{backendutils.RandomTenantName(), ""} {
			name := fmt.Sprintf("%s (tenant=%s)", tc.Name, tenant)
			RunAllModes(t, name, func(t *testing.T) {
				clusterInfo := bapi.ClusterInfo{Cluster: cluster1, Tenant: tenant}

				s1 := v1.Snapshot{
					ResourceList: list.TimestampedResourceList{
						ResourceList: &apiv3.GlobalNetworkPolicyList{
							TypeMeta: metav1.TypeMeta{Kind: "GlobalNetworkPolicy", APIVersion: "projectcalico.org/v3"},
							ListMeta: metav1.ListMeta{},
							Items: []apiv3.GlobalNetworkPolicy{
								{
									TypeMeta: metav1.TypeMeta{Kind: "GlobalNetworkPolicy", APIVersion: "projectcalico.org/v3"},
									ObjectMeta: metav1.ObjectMeta{
										Name: "np1",
									},
								},
							},
						},
						RequestStartedTimestamp:   metav1.Time{Time: time.Unix(1, 0)},
						RequestCompletedTimestamp: metav1.Time{Time: time.Unix(2, 0)},
					},
				}
				s1.ID = s1.ResourceList.String()
				s2 := v1.Snapshot{
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
						RequestStartedTimestamp:   metav1.Time{Time: time.Unix(1000, 0)},
						RequestCompletedTimestamp: metav1.Time{Time: time.Unix(2000, 0)},
					},
				}
				s2.ID = s2.ResourceList.String()

				response, err := sb.Create(ctx, clusterInfo, []v1.Snapshot{s1, s2})
				require.NoError(t, err)
				require.Equal(t, []v1.BulkError(nil), response.Errors)
				require.Equal(t, 0, response.Failed)

				err = backendutils.RefreshIndex(ctx, client, sIndexGetter.Index(clusterInfo))
				require.NoError(t, err)

				resp, err := sb.List(ctx, clusterInfo, tc.Params)
				require.NoError(t, err)
				for i := range resp.Items {
					backendutils.AssertSnapshotClusterAndReset(t, clusterInfo.Cluster, &resp.Items[i])
					backendutils.AssertSnapshotGeneratedTimeAndReset(t, &resp.Items[i])
				}

				if tc.Expect1 {
					require.Contains(t, resp.Items, s1)
				} else {
					require.NotContains(t, resp.Items, s1)
				}
				if tc.Expect2 {
					require.Contains(t, resp.Items, s2)
				} else {
					require.NotContains(t, resp.Items, s2)
				}
			})
		}
	}
}

func TestRetrieveMostRecentSnapshots(t *testing.T) {
	// Run each testcase both as a multi-tenant scenario, as well as a single-tenant case.
	for _, tenant := range []string{backendutils.RandomTenantName(), ""} {
		name := fmt.Sprintf("TestRetrieveMostRecentSnapshots (tenant=%s)", tenant)
		RunAllModes(t, name, func(t *testing.T) {
			clusterInfo := bapi.ClusterInfo{Tenant: tenant, Cluster: cluster1}

			now := time.Now().UTC()

			t1 := time.Unix(500, 0)
			t2 := time.Unix(400, 0)
			t3 := time.Unix(300, 0)

			s1 := v1.Snapshot{
				ResourceList: list.TimestampedResourceList{
					ResourceList: &apiv3.GlobalNetworkPolicyList{
						TypeMeta: metav1.TypeMeta{Kind: "GlobalNetworkPolicy", APIVersion: "projectcalico.org/v3"},
						ListMeta: metav1.ListMeta{},
						Items: []apiv3.GlobalNetworkPolicy{
							{
								TypeMeta: metav1.TypeMeta{Kind: "GlobalNetworkPolicy", APIVersion: "projectcalico.org/v3"},
								ObjectMeta: metav1.ObjectMeta{
									Name: "np1",
								},
							},
						},
					},
					RequestStartedTimestamp:   metav1.Time{Time: t1},
					RequestCompletedTimestamp: metav1.Time{Time: t1},
				},
			}
			s1.ID = s1.ResourceList.String()
			s2 := v1.Snapshot{
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
					RequestStartedTimestamp:   metav1.Time{Time: t2},
					RequestCompletedTimestamp: metav1.Time{Time: t2},
				},
			}
			s2.ID = s2.ResourceList.String()

			_, err := sb.Create(ctx, clusterInfo, []v1.Snapshot{s1, s2})
			require.NoError(t, err)

			err = backendutils.RefreshIndex(ctx, client, sIndexGetter.Index(clusterInfo))
			require.NoError(t, err)

			// Query for logs
			params := v1.SnapshotParams{
				QueryParams: v1.QueryParams{
					TimeRange: &lmav1.TimeRange{
						Field: lmav1.FieldGeneratedTime,
						From:  now,
					},
				},
				Sort: []v1.SearchRequestSortBy{
					{
						Field: string(lmav1.FieldGeneratedTime),
					},
				},
			}
			r, err := sb.List(ctx, clusterInfo, &params)
			require.NoError(t, err)
			require.Len(t, r.Items, 2)
			require.Nil(t, r.AfterKey)
			lastGeneratedTime := r.Items[1].ResourceList.GeneratedTime
			for i := range r.Items {
				backendutils.AssertSnapshotClusterAndReset(t, clusterInfo.Cluster, &r.Items[i])
				backendutils.AssertSnapshotGeneratedTimeAndReset(t, &r.Items[i])
			}

			// Assert that the logs are returned in the correct order.
			require.Equal(t, s1, r.Items[0])
			require.Equal(t, s2, r.Items[1])

			s3 := v1.Snapshot{
				ResourceList: list.TimestampedResourceList{
					ResourceList: &apiv3.NetworkPolicyList{
						TypeMeta: metav1.TypeMeta{Kind: "NetworkPolicy", APIVersion: "projectcalico.org/v3"},
						ListMeta: metav1.ListMeta{},
						Items: []apiv3.NetworkPolicy{
							{
								TypeMeta: metav1.TypeMeta{Kind: "NetworkPolicy", APIVersion: "projectcalico.org/v3"},
								ObjectMeta: metav1.ObjectMeta{
									Name:      "np3",
									Namespace: "default",
								},
							},
						},
					},
					RequestStartedTimestamp:   metav1.Time{Time: t3},
					RequestCompletedTimestamp: metav1.Time{Time: t3},
				},
			}
			s3.ID = s3.ResourceList.String()
			_, err = sb.Create(ctx, clusterInfo, []v1.Snapshot{s3})
			require.NoError(t, err)

			err = backendutils.RefreshIndex(ctx, client, sIndexGetter.Index(clusterInfo))
			require.NoError(t, err)

			// Query the last ingested log
			params.QueryParams = v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					Field: lmav1.FieldGeneratedTime,
					From:  lastGeneratedTime.UTC(),
				},
			}

			r, err = sb.List(ctx, clusterInfo, &params)
			require.NoError(t, err)
			require.Len(t, r.Items, 1)
			require.Nil(t, r.AfterKey)
			for i := range r.Items {
				backendutils.AssertSnapshotClusterAndReset(t, clusterInfo.Cluster, &r.Items[i])
				backendutils.AssertSnapshotGeneratedTimeAndReset(t, &r.Items[i])
			}

			// Assert that the logs are returned in the correct order.
			require.Equal(t, s3, r.Items[0])
		})
	}
}
