// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package client

import (
	"context"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/calico/libcalico-go/lib/options"
	"github.com/projectcalico/calico/libcalico-go/lib/set"
	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/api"
	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/cache"
	authhandler "github.com/projectcalico/calico/ts-queryserver/queryserver/auth"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

type LabelAggregator interface {
	GetAllPoliciesLabels(permissions authhandler.Permission, policiesCache cache.PoliciesCache) (*api.LabelsMap, error)
	GetGlobalThreatfeedsLabels(permissions authhandler.Permission, ctx context.Context) (*api.LabelsMap, error)
	GetAllNetworksetsLabels(permissions authhandler.Permission, setsCache cache.NetworkSetsCache) (*api.LabelsMap, error)
	GetManagedClustersLabels(permissions authhandler.Permission, ctx context.Context) (*api.LabelsMap, error)
	GetPodsLabels(permissions authhandler.Permission, cache cache.EndpointsCache) (*api.LabelsMap, error)
	GetNamespacesLabels(permissions authhandler.Permission, ctx context.Context) (*api.LabelsMap, error)
	GetServiceAccountsLabels(permissions authhandler.Permission, ctx context.Context) (*api.LabelsMap, error)
}

type label struct {
	k8sClient    kubernetes.Interface
	calicoClient clientv3.Interface
}

func NewLabelsAggregator(k8sClient kubernetes.Interface, ci clientv3.Interface) LabelAggregator {
	return &label{
		k8sClient:    k8sClient,
		calicoClient: ci,
	}
}

// getAllNetworksetsLabels returns labels for both networksets and globalnetworksets combined in one map.
func (l *label) GetAllNetworksetsLabels(permissions authhandler.Permission, cache cache.NetworkSetsCache) (*api.LabelsMap, error) {
	labelsResponse := &api.LabelsMap{}
	allNetworkSets := cache.GetNetworkSets(set.New[model.Key]())

	for _, ns := range allNetworkSets {
		// check permissions for networkset
		networkSetResource := &apiv3.NetworkSet{
			TypeMeta: v1.TypeMeta{
				Kind:       ns.GetObjectKind().GroupVersionKind().Kind,
				APIVersion: "projectcalico.org/v3",
			},
			ObjectMeta: v1.ObjectMeta{
				Name: ns.GetObjectMeta().GetName(),
			},
		}
		if !permissions.IsAuthorized(networkSetResource, []string{"get", "list"}) {
			continue
		}

		nsLabels := ns.GetObjectMeta().GetLabels()
		for k, v := range nsLabels {
			labelsResponse.SetLabels(k, v)
		}
	}
	return labelsResponse, nil
}

// GetAllPoliciesLabels returns labels for all kinds of policies i.e. kubernetesnetworkpolicies, networkpolicies,
// globalnetworkpolicies, and all the staged ones combined in one map.
func (l *label) GetAllPoliciesLabels(permissions authhandler.Permission, policiesCache cache.PoliciesCache) (*api.LabelsMap, error) {
	labelsResponse := &api.LabelsMap{}

	// GetOrderdedPolicies returns all policies for an empty set of policy keys.
	allPolicies := policiesCache.GetOrderedPolicies(nil)

	for _, tier := range allPolicies {
		// check permission for tier
		if !permissions.IsAuthorized(tier.GetResource(), []string{"list"}) {
			continue
		}
		// iterator over policies within the tier
		policies := tier.GetOrderedPolicies()
		for _, p := range policies {
			if !permissions.IsAuthorized(p.GetResource(), []string{"get", "list"}) {
				continue
			}
			pLabels := p.GetResource().GetObjectMeta().GetLabels()
			for k, v := range pLabels {
				labelsResponse.SetLabels(k, v)
			}
		}
	}
	return labelsResponse, nil
}

func (l *label) GetGlobalThreatfeedsLabels(permissions authhandler.Permission, ctx context.Context) (*api.LabelsMap, error) {
	globalthreadfeeds, err := l.calicoClient.GlobalThreatFeeds().List(ctx, options.ListOptions{})
	if err != nil {
		return nil, err
	}

	allLabels := api.NewLabelsMap()
	for _, resource := range globalthreadfeeds.Items {
		globalthreatfeedresource := &apiv3.GlobalThreatFeed{
			TypeMeta: v1.TypeMeta{
				Kind:       "GlobalThreatFeeds",
				APIVersion: "projectcalico.org/v3",
			},
			ObjectMeta: v1.ObjectMeta{
				Name: resource.Name,
			},
		}

		// check logged-in users permission and add resource labels to allLabels
		if permissions.IsAuthorized(interface{}(globalthreatfeedresource).(api.Resource), []string{"list"}) {
			for key, value := range resource.Labels {
				allLabels.SetLabels(key, value)
			}
		}
	}

	return allLabels, nil
}

func (l *label) GetManagedClustersLabels(permissions authhandler.Permission, ctx context.Context) (*api.LabelsMap, error) {
	// retrieve resource labels from cache
	clusters, err := l.calicoClient.ManagedClusters().List(ctx, options.ListOptions{})
	if err != nil {
		return nil, err
	}

	allLabels := api.NewLabelsMap()
	for _, resource := range clusters.Items {
		managedClusterResource := &apiv3.ManagedCluster{
			TypeMeta: v1.TypeMeta{
				Kind:       "managedclusters",
				APIVersion: "projectcalico.org/v3",
			},
			ObjectMeta: v1.ObjectMeta{
				Name: resource.Name,
			},
		}
		// check logged-in users permission and add resource labels to allLabels
		if permissions.IsAuthorized(interface{}(managedClusterResource).(api.Resource), []string{"list"}) {
			for key, value := range resource.Labels {
				allLabels.SetLabels(key, value)
			}
		}
	}

	return allLabels, nil
}

func (l *label) GetPodsLabels(permissions authhandler.Permission, cache cache.EndpointsCache) (*api.LabelsMap, error) {
	podsList := cache.GetEndpoints([]model.Key{})

	allLabels := api.NewLabelsMap()
	for _, item := range podsList {
		podresource := &corev1.Pod{
			TypeMeta: v1.TypeMeta{
				Kind:       "pods",
				APIVersion: "",
			},
			ObjectMeta: v1.ObjectMeta{
				Name: item.GetResource().GetObjectMeta().GetName(),
			},
		}
		// check logged-in users permission and add resource labels to allLabels
		if permissions.IsAuthorized(interface{}(podresource).(api.Resource), []string{"list"}) {
			for key, value := range item.GetResource().GetObjectMeta().GetLabels() {
				allLabels.SetLabels(key, value)
			}
		}
	}

	return allLabels, nil
}

func (l *label) GetNamespacesLabels(permissions authhandler.Permission, ctx context.Context) (*api.LabelsMap, error) {
	nsList, err := l.k8sClient.CoreV1().Namespaces().List(ctx, v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	allLabels := api.NewLabelsMap()
	for _, item := range nsList.Items {
		namespaceresource := &corev1.Namespace{
			TypeMeta: v1.TypeMeta{
				Kind:       "namespaces",
				APIVersion: "",
			},
			ObjectMeta: v1.ObjectMeta{
				Name: item.Name,
			},
		}
		// check logged-in users permission and add resource labels to allLabels
		if permissions.IsAuthorized(interface{}(namespaceresource).(api.Resource), []string{"list"}) {
			for key, value := range item.Labels {
				allLabels.SetLabels(key, value)
			}
		}
	}

	return allLabels, nil
}

func (l *label) GetServiceAccountsLabels(permissions authhandler.Permission, ctx context.Context) (*api.LabelsMap, error) {
	// @TODO: test to verify that it returns all service accounts
	saList, err := l.k8sClient.CoreV1().ServiceAccounts("").List(ctx, v1.ListOptions{
		TypeMeta: v1.TypeMeta{
			Kind:       "ServiceAccounts",
			APIVersion: "",
		},
	})
	if err != nil {
		if logrus.IsLevelEnabled(logrus.DebugLevel) {
			logrus.Debugf("GetServiceAccountsLabels failed: %v", err)
		}
		return nil, err
	}

	allLabels := api.NewLabelsMap()
	for _, item := range saList.Items {
		serviceaccountresource := &corev1.ServiceAccount{
			TypeMeta: v1.TypeMeta{
				Kind:       "serviceaccounts",
				APIVersion: "",
			},
			ObjectMeta: v1.ObjectMeta{
				Name: item.Name,
			},
		}
		// check logged-in users permission and add resource labels to allLabels
		if permissions.IsAuthorized(interface{}(serviceaccountresource).(api.Resource), []string{"list"}) {
			for key, value := range item.Labels {
				allLabels.SetLabels(key, value)
			}
		}
	}

	return allLabels, nil
}

var LabelsResourceAuthReviewAttrList = []apiv3.AuthorizationReviewResourceAttributes{
	{
		APIGroup: "projectcalico.org",
		Resources: []string{
			"stagednetworkpolicies", "stagedglobalnetworkpolicies", "stagedkubernetesnetworkpolicies",
			"globalnetworkpolicies", "networkpolicies", "networksets", "globalnetworksets",
			"tiers", "managedclusters", "globalthreatfeeds",
		},
		Verbs: []string{"list"},
	},
	{
		APIGroup:  "networking.k8s.io",
		Resources: []string{"networkpolicies"},
		Verbs:     []string{"list"},
	},
	{
		APIGroup: "",
		Resources: []string{
			"pods", "namespaces", "serviceaccounts",
		},
		Verbs: []string{"list"},
	}}
