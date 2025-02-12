// Copyright (c) 2025 Tigera, Inc. All rights reserved.

// Code generated by lister-gen. DO NOT EDIT.

package v3

import (
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/listers"
	"k8s.io/client-go/tools/cache"
)

// GlobalNetworkPolicyLister helps list GlobalNetworkPolicies.
// All objects returned here must be treated as read-only.
type GlobalNetworkPolicyLister interface {
	// List lists all GlobalNetworkPolicies in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v3.GlobalNetworkPolicy, err error)
	// Get retrieves the GlobalNetworkPolicy from the index for a given name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v3.GlobalNetworkPolicy, error)
	GlobalNetworkPolicyListerExpansion
}

// globalNetworkPolicyLister implements the GlobalNetworkPolicyLister interface.
type globalNetworkPolicyLister struct {
	listers.ResourceIndexer[*v3.GlobalNetworkPolicy]
}

// NewGlobalNetworkPolicyLister returns a new GlobalNetworkPolicyLister.
func NewGlobalNetworkPolicyLister(indexer cache.Indexer) GlobalNetworkPolicyLister {
	return &globalNetworkPolicyLister{listers.New[*v3.GlobalNetworkPolicy](indexer, v3.Resource("globalnetworkpolicy"))}
}
