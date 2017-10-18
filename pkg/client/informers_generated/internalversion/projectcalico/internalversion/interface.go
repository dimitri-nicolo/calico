/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// This file was automatically generated by informer-gen

package internalversion

import (
	internalinterfaces "github.com/tigera/calico-k8sapiserver/pkg/client/informers_generated/internalversion/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// GlobalNetworkPolicies returns a GlobalNetworkPolicyInformer.
	GlobalNetworkPolicies() GlobalNetworkPolicyInformer
	// NetworkPolicies returns a NetworkPolicyInformer.
	NetworkPolicies() NetworkPolicyInformer
	// Tiers returns a TierInformer.
	Tiers() TierInformer
}

type version struct {
	internalinterfaces.SharedInformerFactory
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory) Interface {
	return &version{f}
}

// GlobalNetworkPolicies returns a GlobalNetworkPolicyInformer.
func (v *version) GlobalNetworkPolicies() GlobalNetworkPolicyInformer {
	return &globalNetworkPolicyInformer{factory: v.SharedInformerFactory}
}

// NetworkPolicies returns a NetworkPolicyInformer.
func (v *version) NetworkPolicies() NetworkPolicyInformer {
	return &networkPolicyInformer{factory: v.SharedInformerFactory}
}

// Tiers returns a TierInformer.
func (v *version) Tiers() TierInformer {
	return &tierInformer{factory: v.SharedInformerFactory}
}
