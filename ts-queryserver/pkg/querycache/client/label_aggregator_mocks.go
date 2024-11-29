// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package client

import (
	"context"

	"github.com/stretchr/testify/mock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/calico/apiserver/pkg/rbac"
	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/calico/libcalico-go/lib/options"
	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/api"
	"github.com/projectcalico/calico/ts-queryserver/queryserver/auth"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

type mockAuthorizer struct{}

type mockPermission struct{}

func (p *mockPermission) IsAuthorized(res api.Resource, tier *string, verbs []rbac.Verb) bool {
	return true
}

func (m *mockAuthorizer) PerformUserAuthorizationReview(context.Context, []v3.AuthorizationReviewResourceAttributes) (auth.Permission, error) {
	permissions := mockPermission{}
	return &permissions, nil
}

type mockClient struct {
	mock.Mock
	clientv3.Interface
}

func newMockCalicoClient() *mockClient {
	calicoClient := &mockClient{}

	return calicoClient
}

func (_m *mockClient) GlobalThreatFeeds() clientv3.GlobalThreatFeedInterface {
	mockResource := &mockGlobalThreatFeed{}
	return mockResource
}

func (_m *mockClient) ManagedClusters() clientv3.ManagedClusterInterface {
	mockResource := &mockManagedCluster{}
	return mockResource
}

type mockGlobalThreatFeed struct {
	mock.Mock
	clientv3.GlobalThreatFeedInterface
}

type mockManagedCluster struct {
	mock.Mock
	clientv3.ManagedClusterInterface
}

func (*mockGlobalThreatFeed) List(ctx context.Context, listOptions options.ListOptions) (*v3.GlobalThreatFeedList, error) {
	return &v3.GlobalThreatFeedList{
		TypeMeta: metav1.TypeMeta{
			Kind:       v3.KindGlobalThreatFeedList,
			APIVersion: v3.GroupVersionCurrent,
		},
		ListMeta: metav1.ListMeta{},
		Items: []v3.GlobalThreatFeed{
			{
				TypeMeta: metav1.TypeMeta{
					Kind:       v3.KindGlobalThreatFeed,
					APIVersion: v3.GroupVersionCurrent,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "global-threatfeed",
					Labels: map[string]string{
						"name": "global-threatfeed",
					},
				},
				Spec: v3.GlobalThreatFeedSpec{},
			},
		},
	}, nil
}

func (*mockManagedCluster) List(ctx context.Context, listOptions options.ListOptions) (*v3.ManagedClusterList, error) {
	return &v3.ManagedClusterList{
		TypeMeta: metav1.TypeMeta{
			Kind:       v3.KindManagedClusterList,
			APIVersion: v3.GroupVersionCurrent,
		},
		ListMeta: metav1.ListMeta{},
		Items: []v3.ManagedCluster{
			{
				TypeMeta: metav1.TypeMeta{
					Kind:       v3.KindManagedCluster,
					APIVersion: v3.GroupVersionCurrent,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "managed-clusters",
					Labels: map[string]string{
						"name": "managed-cluster",
					},
				},
				Spec: v3.ManagedClusterSpec{},
			},
		},
	}, nil
}
