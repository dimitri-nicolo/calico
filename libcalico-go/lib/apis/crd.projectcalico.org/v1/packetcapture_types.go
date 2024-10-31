// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package v1

import (
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
type PacketCapture struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              v3.PacketCaptureSpec   `json:"spec,omitempty"`
	Status            v3.PacketCaptureStatus `json:"status,omitempty"`
}
