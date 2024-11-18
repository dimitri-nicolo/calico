// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package client

import (
	"context"
	"strings"

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
	GetAllPoliciesLabels(permissions authhandler.Permission, policiesCache cache.PoliciesCache) (*api.LabelsMap, []string, error)
	GetGlobalThreatfeedsLabels(permissions authhandler.Permission, ctx context.Context) (*api.LabelsMap, []string, error)
	GetAllNetworksetsLabels(permissions authhandler.Permission, setsCache cache.NetworkSetsCache) (*api.LabelsMap, []string, error)
	GetManagedClustersLabels(permissions authhandler.Permission, ctx context.Context) (*api.LabelsMap, []string, error)
	GetPodsLabels(permissions authhandler.Permission, cache cache.EndpointsCache) (*api.LabelsMap, []string, error)
	GetNamespacesLabels(permissions authhandler.Permission, ctx context.Context) (*api.LabelsMap, []string, error)
	GetServiceAccountsLabels(permissions authhandler.Permission, ctx context.Context) (*api.LabelsMap, []string, error)
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
func (l *label) GetAllNetworksetsLabels(permissions authhandler.Permission, cache cache.NetworkSetsCache) (*api.LabelsMap, []string, error) {
	labelsResponse := &api.LabelsMap{}
	allNetworkSets := cache.GetNetworkSets(set.New[model.Key]())

	missingPermissionGlobalNetworkset, missingPermissionNetworkset := false, false

	for _, ns := range allNetworkSets {
		// check permissions for networkset
		networkSetResource := &apiv3.NetworkSet{
			TypeMeta: v1.TypeMeta{
				Kind:       ns.GetObjectKind().GroupVersionKind().Kind,
				APIVersion: "projectcalico.org/v3",
			},
			ObjectMeta: v1.ObjectMeta{
				Namespace: ns.GetObjectMeta().GetNamespace(),
			},
		}
		if !permissions.IsAuthorized(networkSetResource, nil, []string{"list"}) {
			if strings.ToLower(ns.GetObjectKind().GroupVersionKind().Kind) == "globalnetworkset" {
				missingPermissionGlobalNetworkset = true
			} else if strings.ToLower(ns.GetObjectKind().GroupVersionKind().Kind) == "networkset" {
				missingPermissionNetworkset = true
			}
			continue
		}

		nsLabels := ns.GetObjectMeta().GetLabels()
		for k, v := range nsLabels {
			labelsResponse.SetLabels(k, v)
		}
	}

	var warning []string
	if missingPermissionGlobalNetworkset {
		warning = []string{"missing \"list\" RBAC to globalnetworksets."}
	}
	if missingPermissionNetworkset {
		if warning == nil {
			warning = []string{}
		}
		warning = append(warning, "missing \"list\" RBAC to networksets.")
	}

	return labelsResponse, warning, nil
}

// GetAllPoliciesLabels returns labels for all kinds of policies i.e. kubernetesnetworkpolicies, networkpolicies,
// globalnetworkpolicies, and all the staged ones combined in one map.
func (l *label) GetAllPoliciesLabels(permissions authhandler.Permission, policiesCache cache.PoliciesCache) (*api.LabelsMap, []string, error) {
	labelsResponse := &api.LabelsMap{}

	missingPermissionTier, missingPermisionPolicy := false, false

	// GetOrderdedPolicies returns all policies for an empty set of policy keys.
	allPolicies := policiesCache.GetOrderedPolicies(nil)

	tierresource := &apiv3.Tier{
		TypeMeta: v1.TypeMeta{
			Kind:       "Tier",
			APIVersion: "projectcalico.org/v3",
		},
	}
	// check permission for tier
	if !permissions.IsAuthorized(tierresource, nil, []string{"list"}) {
		missingPermissionTier = true
	}

	for _, tier := range allPolicies {
		// iterator over policies within the tier
		policies := tier.GetOrderedPolicies()
		for _, p := range policies {
			tier := p.GetTier()
			// check permission to tier for non-kubernetes policies
			if !p.IsKubernetesType() && missingPermissionTier {
				continue
			}

			// check permission to policy type
			if !permissions.IsAuthorized(p.GetResourceType(), &tier, []string{"list"}) {
				missingPermisionPolicy = true
				continue
			}
			pLabels := p.GetResource().GetObjectMeta().GetLabels()
			for k, v := range pLabels {
				labelsResponse.SetLabels(k, v)
			}
		}
	}

	var warning []string
	if missingPermissionTier {
		warning = []string{"missing \"list\" RBAC to tiers."}
	}
	if missingPermisionPolicy {
		if warning == nil {
			warning = []string{}
		}
		warning = append(warning, "missing \"list\" RBAC to some policy types.")
	}
	return labelsResponse, warning, nil
}

func (l *label) GetGlobalThreatfeedsLabels(permissions authhandler.Permission, ctx context.Context) (*api.LabelsMap, []string, error) {
	globalthreadfeeds, err := l.calicoClient.GlobalThreatFeeds().List(ctx, options.ListOptions{})
	if err != nil {
		return nil, nil, err
	}

	globalthreatfeedresource := &apiv3.GlobalThreatFeed{
		TypeMeta: v1.TypeMeta{
			Kind:       "GlobalThreatFeeds",
			APIVersion: "projectcalico.org/v3",
		},
	}
	// check logged-in users permission to GlobalThreatFeeds
	if !permissions.IsAuthorized(interface{}(globalthreatfeedresource).(api.Resource), nil, []string{"list"}) {
		warning := []string{"missing \"list\" RBAC to globalthreatfeeds."}
		return nil, warning, nil
	}

	allLabels := api.NewLabelsMap()
	for _, resource := range globalthreadfeeds.Items {
		for key, value := range resource.Labels {
			allLabels.SetLabels(key, value)
		}
	}

	return allLabels, nil, nil
}

func (l *label) GetManagedClustersLabels(permissions authhandler.Permission, ctx context.Context) (*api.LabelsMap, []string, error) {
	// retrieve resource labels from cache
	clusters, err := l.calicoClient.ManagedClusters().List(ctx, options.ListOptions{})
	if err != nil {
		if logrus.IsLevelEnabled(logrus.DebugLevel) {
			logrus.Debugf("Get ManagedClusters failed: %v", err)
		}
		return nil, nil, err
	}

	allLabels := api.NewLabelsMap()
	managedClusterResource := &apiv3.ManagedCluster{
		TypeMeta: v1.TypeMeta{
			Kind:       "managedclusters",
			APIVersion: "projectcalico.org/v3",
		},
	}
	// check logged-in users permission to ManagedClusters
	if !permissions.IsAuthorized(interface{}(managedClusterResource).(api.Resource), nil, []string{"list"}) {
		warning := []string{"missing \"list\" RBAC to managedclusters."}
		return nil, warning, nil
	}
	for _, resource := range clusters.Items {
		for key, value := range resource.Labels {
			allLabels.SetLabels(key, value)
		}
	}

	return allLabels, nil, nil
}

func (l *label) GetPodsLabels(permissions authhandler.Permission, cache cache.EndpointsCache) (*api.LabelsMap, []string, error) {
	podsList := cache.GetEndpoints([]model.Key{})
	missingPermission := false

	allLabels := api.NewLabelsMap()
	for _, item := range podsList {
		podresource := &corev1.Pod{
			TypeMeta: v1.TypeMeta{
				Kind:       "pod",
				APIVersion: "",
			},
			ObjectMeta: v1.ObjectMeta{
				Namespace: item.GetResource().GetObjectMeta().GetNamespace(),
			},
		}
		// check logged-in users permission and add resource labels to allLabels
		if permissions.IsAuthorized(interface{}(podresource).(api.Resource), nil, []string{"list"}) {
			for key, value := range item.GetResource().GetObjectMeta().GetLabels() {
				allLabels.SetLabels(key, value)
			}
		} else {
			missingPermission = true
		}
	}

	if missingPermission {
		warning := []string{"missing \"list\" RBAC to some pods / namespaces."}
		return allLabels, warning, nil
	}
	return allLabels, nil, nil
}

func (l *label) GetNamespacesLabels(permissions authhandler.Permission, ctx context.Context) (*api.LabelsMap, []string, error) {
	nsList, err := l.k8sClient.CoreV1().Namespaces().List(ctx, v1.ListOptions{})
	if err != nil {
		if logrus.IsLevelEnabled(logrus.DebugLevel) {
			logrus.Debugf("Get Namespaces failed: %v", err)
		}
		return nil, nil, err
	}

	missingPermissionNamespace := false
	allLabels := api.NewLabelsMap()
	for _, item := range nsList.Items {
		namespaceresource := &corev1.Namespace{
			TypeMeta: v1.TypeMeta{
				Kind:       "namespace",
				APIVersion: "",
			},
			ObjectMeta: v1.ObjectMeta{
				Name: item.Name,
			},
		}
		// check logged-in users permission and add resource labels to allLabels
		if permissions.IsAuthorized(namespaceresource, nil, []string{"list"}) {
			for key, value := range item.Labels {
				allLabels.SetLabels(key, value)
			}
		} else {
			missingPermissionNamespace = true
		}
	}

	if missingPermissionNamespace {
		warning := []string{"missing \"list\" RBAC to some namespaces."}
		return allLabels, warning, nil
	}

	return allLabels, nil, nil
}

func (l *label) GetServiceAccountsLabels(permissions authhandler.Permission, ctx context.Context) (*api.LabelsMap, []string, error) {
	saList, err := l.k8sClient.CoreV1().ServiceAccounts("").List(ctx, v1.ListOptions{
		TypeMeta: v1.TypeMeta{
			Kind:       "ServiceAccounts",
			APIVersion: "",
		},
	})
	if err != nil {
		if logrus.IsLevelEnabled(logrus.DebugLevel) {
			logrus.Debugf("Get ServiceAccounts failed: %v", err)
		}
		return nil, nil, err
	}

	missingPermissionServiceAccount := false

	allLabels := api.NewLabelsMap()
	for _, item := range saList.Items {
		serviceaccountresource := &corev1.ServiceAccount{
			TypeMeta: v1.TypeMeta{
				Kind:       "serviceaccount",
				APIVersion: "",
			},
			ObjectMeta: v1.ObjectMeta{
				Namespace: item.Namespace,
			},
		}
		// check logged-in users permission and add resource labels to allLabels
		if permissions.IsAuthorized(interface{}(serviceaccountresource).(api.Resource), nil, []string{"list"}) {
			for key, value := range item.Labels {
				allLabels.SetLabels(key, value)
			}
		} else {
			missingPermissionServiceAccount = true
		}
	}

	if missingPermissionServiceAccount {
		warning := []string{"missing \"list\" RBAC to some serviceaccounts / namespaces."}
		return allLabels, warning, nil

	}
	return allLabels, nil, nil
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
