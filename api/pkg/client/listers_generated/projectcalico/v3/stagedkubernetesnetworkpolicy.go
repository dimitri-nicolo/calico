// Copyright (c) 2025 Tigera, Inc. All rights reserved.

// Code generated by lister-gen. DO NOT EDIT.

package v3

import (
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/listers"
	"k8s.io/client-go/tools/cache"
)

// StagedKubernetesNetworkPolicyLister helps list StagedKubernetesNetworkPolicies.
// All objects returned here must be treated as read-only.
type StagedKubernetesNetworkPolicyLister interface {
	// List lists all StagedKubernetesNetworkPolicies in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v3.StagedKubernetesNetworkPolicy, err error)
	// StagedKubernetesNetworkPolicies returns an object that can list and get StagedKubernetesNetworkPolicies.
	StagedKubernetesNetworkPolicies(namespace string) StagedKubernetesNetworkPolicyNamespaceLister
	StagedKubernetesNetworkPolicyListerExpansion
}

// stagedKubernetesNetworkPolicyLister implements the StagedKubernetesNetworkPolicyLister interface.
type stagedKubernetesNetworkPolicyLister struct {
	listers.ResourceIndexer[*v3.StagedKubernetesNetworkPolicy]
}

// NewStagedKubernetesNetworkPolicyLister returns a new StagedKubernetesNetworkPolicyLister.
func NewStagedKubernetesNetworkPolicyLister(indexer cache.Indexer) StagedKubernetesNetworkPolicyLister {
	return &stagedKubernetesNetworkPolicyLister{listers.New[*v3.StagedKubernetesNetworkPolicy](indexer, v3.Resource("stagedkubernetesnetworkpolicy"))}
}

// StagedKubernetesNetworkPolicies returns an object that can list and get StagedKubernetesNetworkPolicies.
func (s *stagedKubernetesNetworkPolicyLister) StagedKubernetesNetworkPolicies(namespace string) StagedKubernetesNetworkPolicyNamespaceLister {
	return stagedKubernetesNetworkPolicyNamespaceLister{listers.NewNamespaced[*v3.StagedKubernetesNetworkPolicy](s.ResourceIndexer, namespace)}
}

// StagedKubernetesNetworkPolicyNamespaceLister helps list and get StagedKubernetesNetworkPolicies.
// All objects returned here must be treated as read-only.
type StagedKubernetesNetworkPolicyNamespaceLister interface {
	// List lists all StagedKubernetesNetworkPolicies in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v3.StagedKubernetesNetworkPolicy, err error)
	// Get retrieves the StagedKubernetesNetworkPolicy from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v3.StagedKubernetesNetworkPolicy, error)
	StagedKubernetesNetworkPolicyNamespaceListerExpansion
}

// stagedKubernetesNetworkPolicyNamespaceLister implements the StagedKubernetesNetworkPolicyNamespaceLister
// interface.
type stagedKubernetesNetworkPolicyNamespaceLister struct {
	listers.ResourceIndexer[*v3.StagedKubernetesNetworkPolicy]
}
