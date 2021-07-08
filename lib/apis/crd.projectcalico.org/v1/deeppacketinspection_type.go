// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +kubebuilder:subresource:status
// +k8s:openapi-gen=true
type DeepPacketInspection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              v3.DeepPacketInspectionSpec   `json:"spec,omitempty"`
	Status            v3.DeepPacketInspectionStatus `json:"status,omitempty"`
}
