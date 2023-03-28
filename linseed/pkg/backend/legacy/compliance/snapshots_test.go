// Copyright (c) 2023 Tigera All rights reserved.

package compliance_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	corev1 "k8s.io/api/core/v1"

	"github.com/projectcalico/calico/libcalico-go/lib/resources"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	backendutils "github.com/projectcalico/calico/linseed/pkg/backend/testutils"
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

	for _, tc := range testcases {
		t.Run(fmt.Sprintf("should write and read %s snapshots", tc.Name), func(t *testing.T) {
			defer setupTest(t)()

			trl := list.TimestampedResourceList{
				ResourceList:              tc.ResourceList,
				RequestStartedTimestamp:   metav1.Time{Time: time.Unix(1, 0)},
				RequestCompletedTimestamp: metav1.Time{Time: time.Unix(2, 0)},
			}
			f := v1.Snapshot{
				ResourceList: trl,
			}

			response, err := sb.Create(ctx, clusterInfo, []v1.Snapshot{f})
			require.NoError(t, err)
			require.Equal(t, []v1.BulkError(nil), response.Errors)
			require.Equal(t, 0, response.Failed)

			err = backendutils.RefreshIndex(ctx, client, "tigera_secure_ee_snapshots.*")
			require.NoError(t, err)

			// Read it back and check it matches.
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
			resp, err := sb.List(ctx, clusterInfo, &p)
			require.NoError(t, err)
			require.Len(t, resp.Items, 1)
			require.Equal(t, trl, resp.Items[0].ResourceList)
		})
	}
}
