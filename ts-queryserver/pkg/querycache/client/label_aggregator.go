// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package client

import (
	"context"
	"strings"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/projectcalico/calico/apiserver/pkg/rbac"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/calico/libcalico-go/lib/options"
	"github.com/projectcalico/calico/libcalico-go/lib/set"
	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/api"
	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/cache"
	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/utils"
	authhandler "github.com/projectcalico/calico/ts-queryserver/queryserver/auth"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

type LabelAggregator interface {
	GetAllPoliciesLabels(ctx context.Context, permissions authhandler.Permission, policiesCache cache.PoliciesCache) (*api.LabelsMap, []string, error)
	GetGlobalThreatfeedsLabels(ctx context.Context, permissions authhandler.Permission) (*api.LabelsMap, []string, error)
	GetAllNetworkSetsLabels(ctx context.Context, permissions authhandler.Permission, setsCache cache.NetworkSetsCache) (*api.LabelsMap, []string, error)
	GetManagedClustersLabels(ctx context.Context, permissions authhandler.Permission) (*api.LabelsMap, []string, error)
	GetPodsLabels(ctx context.Context, permissions authhandler.Permission, cache cache.EndpointsCache) (*api.LabelsMap, []string, error)
	GetNamespacesLabels(ctx context.Context, permissions authhandler.Permission) (*api.LabelsMap, []string, error)
	GetServiceAccountsLabels(ctx context.Context, permissions authhandler.Permission) (*api.LabelsMap, []string, error)
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

// GetAllNetworkSetsLabels returns networkSets and globalNetworkSets from the cache. Labels are then extracted for both
// types and returned in one map.
func (l *label) GetAllNetworkSetsLabels(ctx context.Context, permissions authhandler.Permission, cache cache.NetworkSetsCache) (*api.LabelsMap, []string, error) {
	labelsResponse := &api.LabelsMap{}
	allNetworkSets := cache.GetNetworkSets(set.New[model.Key]())

	missingPermissionGlobalNetworkSet, missingPermissionNetworkSet := false, false

	for _, ns := range allNetworkSets {
		// check permissions for networkSets
		kind := ns.GetObjectKind().GroupVersionKind().Kind

		// There is no namespace for globalNetworkSets, but there is also no harm is setting the namespace for the
		// benefit of same code path for both networkSets and globalNetworkSets
		networkSetResource := &apiv3.NetworkSet{
			TypeMeta: v1.TypeMeta{
				Kind:       kind,
				APIVersion: apiv3.GroupVersionCurrent,
			},
			ObjectMeta: v1.ObjectMeta{
				Namespace: ns.GetObjectMeta().GetNamespace(),
			},
		}
		if !permissions.IsAuthorized(networkSetResource, nil, []rbac.Verb{rbac.VerbList}) {
			if strings.ToLower(kind) == "globalnetworkset" {
				missingPermissionGlobalNetworkSet = true
			} else if strings.ToLower(kind) == "networkset" {
				missingPermissionNetworkSet = true
			}
			continue
		}

		nsLabels := ns.GetObjectMeta().GetLabels()
		for k, v := range nsLabels {
			labelsResponse.SetLabels(k, v)
		}
	}

	var warning []string
	if missingPermissionGlobalNetworkSet {
		warning = append(warning, "missing \"list\" RBAC to globalnetworksets.")
	}
	if missingPermissionNetworkSet {
		warning = append(warning, "missing \"list\" RBAC to networksets.")
	}

	return labelsResponse, warning, nil
}

// GetAllPoliciesLabels returns labels for all kinds of policies i.e. kubernetesnetworkpolicies, networkpolicies,
// globalnetworkpolicies, and all the staged ones combined in one map.
func (l *label) GetAllPoliciesLabels(ctx context.Context, permissions authhandler.Permission, policiesCache cache.PoliciesCache) (*api.LabelsMap, []string, error) {
	missingPermissionPolicy := false

	// Retriever policies form cache
	allPolicies := policiesCache.GetOrderedPolicies(nil)

	allLabels := &api.LabelsMap{}
	for _, tier := range allPolicies {
		// Iterator over policies within the tier
		policies := tier.GetOrderedPolicies()
		for _, p := range policies {
			// Check permission to policy type & tier
			resource, policyTier := utils.GetActualResourceAndTierFromCachedPolicyForRBAC(p)
			if !permissions.IsAuthorized(resource, &policyTier, []rbac.Verb{rbac.VerbGet}) {
				missingPermissionPolicy = true
				continue
			}

			// Add labels to allLabels
			pLabels := p.GetResource().GetObjectMeta().GetLabels()
			for k, v := range pLabels {
				allLabels.SetLabels(k, v)
			}
		}
	}

	var warning []string
	if missingPermissionPolicy {
		warning = append(warning, "missing \"get\" RBAC to some policy types.")
	}
	return allLabels, warning, nil
}

// GetGlobalThreatfeedsLabels gets GlobalThreatfeeds data directly from the datastore since they are not cached in
// queryserver cache.
// TODO: introduce caching layer for all objects to avoid direct calls to the datastore.
func (l *label) GetGlobalThreatfeedsLabels(ctx context.Context, permissions authhandler.Permission) (*api.LabelsMap, []string, error) {
	// Check logged-in users permission to GlobalThreatFeeds
	globalThreatFeedResource := &apiv3.GlobalThreatFeed{
		TypeMeta: v1.TypeMeta{
			Kind:       apiv3.KindGlobalThreatFeed,
			APIVersion: apiv3.GroupVersionCurrent,
		},
	}

	if !permissions.IsAuthorized(globalThreatFeedResource, nil, []rbac.Verb{rbac.VerbList}) {
		warning := []string{"missing \"list\" RBAC to globalthreatfeeds."}
		return nil, warning, nil
	}

	// Retrieve globalthreatfeeds from datastore
	globalthreadfeeds, err := l.calicoClient.GlobalThreatFeeds().List(ctx, options.ListOptions{})
	if err != nil {
		return nil, nil, err
	}

	// Populate allLabels
	allLabels := api.NewLabelsMap()
	for _, resource := range globalthreadfeeds.Items {
		for key, value := range resource.Labels {
			allLabels.SetLabels(key, value)
		}
	}

	return allLabels, nil, nil
}

// GetManagedClustersLabels get ManagedCluster data directly from the datastore since they are not cached in
// queryserver cache.
// TODO: introduce caching layer for all objects to avoid direct calls to datastore
func (l *label) GetManagedClustersLabels(ctx context.Context, permissions authhandler.Permission) (*api.LabelsMap, []string, error) {
	// Check logged-in users permission to ManagedClusters
	managedClusterResource := &apiv3.ManagedCluster{
		TypeMeta: v1.TypeMeta{
			Kind:       apiv3.KindManagedCluster,
			APIVersion: apiv3.GroupVersionCurrent,
		},
	}
	if !permissions.IsAuthorized(managedClusterResource, nil, []rbac.Verb{rbac.VerbList}) {
		warning := []string{"missing \"list\" RBAC to managedclusters."}
		return nil, warning, nil
	}

	// Retrieve ManagedClusters
	clusters, err := l.calicoClient.ManagedClusters().List(ctx, options.ListOptions{})
	if err != nil {
		if logrus.IsLevelEnabled(logrus.DebugLevel) {
			logrus.Debugf("Get ManagedClusters failed: %v", err)
		}
		return nil, nil, err
	}

	// Populate all labels
	allLabels := api.NewLabelsMap()
	for _, resource := range clusters.Items {
		for key, value := range resource.Labels {
			allLabels.SetLabels(key, value)
		}
	}

	return allLabels, nil, nil
}

func (l *label) GetPodsLabels(ctx context.Context, permissions authhandler.Permission, cache cache.EndpointsCache) (*api.LabelsMap, []string, error) {
	// Retrieve pods from the cache
	podsList := cache.GetEndpoints([]model.Key{})
	missingPermission := false

	allLabels := api.NewLabelsMap()
	for _, item := range podsList {
		// Check logged-in users permission
		podResource := &corev1.Pod{
			TypeMeta: v1.TypeMeta{
				Kind:       apiv3.KindK8sPod,
				APIVersion: "",
			},
			ObjectMeta: v1.ObjectMeta{
				Namespace: item.GetResource().GetObjectMeta().GetNamespace(),
			},
		}
		if !permissions.IsAuthorized(podResource, nil, []rbac.Verb{rbac.VerbList}) {
			missingPermission = true
			continue
		}

		// Add labels to allLabels
		for key, value := range item.GetResource().GetObjectMeta().GetLabels() {
			allLabels.SetLabels(key, value)
		}
	}

	if missingPermission {
		warning := []string{"missing \"list\" RBAC to some pods."}
		return allLabels, warning, nil
	}
	return allLabels, nil, nil
}

// GetNamespacesLabels returns labels for namespaces directly from the datastore since there is no cache storing namespaces
// in the queryserver.
// It returns labels for all namespaces if user has "list" RBAC to "namespace" resource without any resourceNames
// defined in a clusterRole, otherwise it returns a warning that user does not have the required RBAC for this operation.
// TODO: caching to be implemented
func (l *label) GetNamespacesLabels(ctx context.Context, permissions authhandler.Permission) (*api.LabelsMap, []string, error) {
	// Check logged-in users permission
	namespaceResource := &corev1.Namespace{
		TypeMeta: v1.TypeMeta{
			Kind:       apiv3.KindK8sNamespace,
			APIVersion: "",
		},
	}
	// check logged-in users permission and add resource labels to allLabels
	if !permissions.IsAuthorized(namespaceResource, nil, []rbac.Verb{rbac.VerbList}) {
		warning := []string{"missing \"list\" RBAC to namespaces."}
		return nil, warning, nil
	}

	// Retrieve namespaces from the datastore
	nsList, err := l.k8sClient.CoreV1().Namespaces().List(ctx, v1.ListOptions{})
	if err != nil {
		if logrus.IsLevelEnabled(logrus.DebugLevel) {
			logrus.Debugf("Get Namespaces failed: %v", err)
		}
		return nil, nil, err
	}

	allLabels := api.NewLabelsMap()
	for _, item := range nsList.Items {
		for key, value := range item.Labels {
			allLabels.SetLabels(key, value)
		}
	}
	return allLabels, nil, nil
}

// GetServiceAccountsLabels returns labels for serviceAccounts directly from the datastore since htere is no cache storing
// serviceAccounts in the queryserver.
// TODO: caching to be implemented
func (l *label) GetServiceAccountsLabels(ctx context.Context, permissions authhandler.Permission) (*api.LabelsMap, []string, error) {
	// Retrieve serviceAccounts from the datastore
	saList, err := l.k8sClient.CoreV1().ServiceAccounts("").List(ctx, v1.ListOptions{
		TypeMeta: v1.TypeMeta{
			Kind:       apiv3.KindK8sServiceAccount,
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
		// Check logged-in users permission
		serviceAccountResource := &corev1.ServiceAccount{
			TypeMeta: v1.TypeMeta{
				Kind:       authhandler.ResourceServiceAccounts,
				APIVersion: "",
			},
			ObjectMeta: v1.ObjectMeta{
				Namespace: item.Namespace,
			},
		}
		if !permissions.IsAuthorized(serviceAccountResource, nil, []rbac.Verb{rbac.VerbList}) {
			missingPermissionServiceAccount = true
			continue
		}

		// Add serviceAccount labels to allLabels
		for key, value := range item.Labels {
			allLabels.SetLabels(key, value)
		}
	}

	if missingPermissionServiceAccount {
		warning := []string{"missing \"list\" RBAC to some serviceAccounts."}
		return allLabels, warning, nil

	}
	return allLabels, nil, nil
}

var LabelsResourceAuthReviewAttrList = []apiv3.AuthorizationReviewResourceAttributes{
	{
		APIGroup: apiv3.Group,
		Resources: []string{
			authhandler.ResourceStageNetworkPolicies, authhandler.ResourceStagedGlobalNetworkPolicies,
			authhandler.ResourceStagedKubernetesNetworkPolicies, authhandler.ResourceGlobalNetworkPolicies,
			authhandler.ResourceNetworkPolicies, authhandler.ResourceNetworkSets, authhandler.ResourceGlobalNetworkSets,
			authhandler.ResourceTiers, authhandler.ResourceManagedClusters, authhandler.ResourceGlobalThreatFeeds,
		},
		Verbs: []string{string(rbac.VerbGet), string(rbac.VerbList)},
	},
	{
		APIGroup:  authhandler.ApiGroupK8sNetworking,
		Resources: []string{authhandler.ResourceNetworkPolicies},
		Verbs:     []string{string(rbac.VerbGet), string(rbac.VerbList)},
	},
	{
		APIGroup: "",
		Resources: []string{
			authhandler.ResourcePods, authhandler.ResourceNamespaces, authhandler.ResourceServiceAccounts,
		},
		Verbs: []string{string(rbac.VerbGet), string(rbac.VerbList)},
	}}
