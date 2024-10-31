// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package v1

import (
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +kubebuilder:subresource:status
// +k8s:openapi-gen=true
// +kubebuilder:resource:scope=Cluster
type PolicyRecommendationScope struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              v3.PolicyRecommendationScopeSpec   `json:"spec,omitempty"`
	Status            v3.PolicyRecommendationScopeStatus `json:"status,omitempty"`
}
