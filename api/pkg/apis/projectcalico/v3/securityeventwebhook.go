// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package v3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	KindSecurityEventWebhook     = "SecurityEventWebhook"
	KindSecurityEventWebhookList = "SecurityEventWebhookList"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type SecurityEventWebhook struct {
	metav1.TypeMeta `json:",inline"`
	// standard object metadata
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// status of the SecurityEventWebhook
	Status SecurityEventWebhookStatus
	// specification of the SecurityEventWebhook
	Spec SecurityEventWebhookSpec
}

// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type SecurityEventWebhookList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []SecurityEventWebhook `json:"items"`
}

type SecurityEventWebhookStatus struct {
	// last fetch operation time
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`
	// number of processed security events during the latest fetch operation
	LastTransistionCount uint `json:"lastTransitionCount,omitempty" validate:"omitempty,min=0"`
	// health of the webhook during the latest fetch operation
	Health string `json:"health,omitempty"`
}

type SecurityEventWebhookSpec struct {
	// indicates the SecurityEventWebhook intended consumer, one of: Slack, Jira
	Consumer string `json:"consumer" validate:"required,oneof=Slack Jira"`
	// defines the webhook desired state, one of: Enabled, Disabled or Debug
	State string `json:"state" validate:"required,oneof=Enabled Disabled Debug"`
	// defines the SecurityEventWebhook query to be executed against fields of SecurityEvents
	Query string `json:"query" validate:"required"`
	// contains the SecurityEventWebhook's configuration associated with the intended Consumer
	Config []SecurityEventWebhookConfigVar `json:"config" validate:"required"`
}

type SecurityEventWebhookConfigVar struct {
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// +optional
	Value string `json:"value,omitempty"`
	// +optional
	ValueFrom *SecurityEventWebhookConfigVarSource `json:"valueFrom,omitempty"`
}

type SecurityEventWebhookConfigVarSource struct {
	// +optional
	ConfigMapKeyRef *ConfigMapKeySelector `json:"configMapKeyRef,omitempty" protobuf:"bytes,3,opt,name=configMapKeyRef"`
	// +optional
	SecretKeyRef *SecretKeySelector `json:"secretKeyRef,omitempty" protobuf:"bytes,4,opt,name=secretKeyRef"`
}

type ConfigMapKeySelector struct {
	Namespace string `json:"namespace" validate:"required"`
	Name      string `json:"name" validate:"required"`
	Key       string `json:"key" validate:"required"`
}

type SecretKeySelector struct {
	Namespace string `json:"namespace" validate:"required"`
	Name      string `json:"name" validate:"required"`
	Key       string `json:"key" validate:"required"`
}

// NewSecurityEventWebhook creates a new SecurityEventWebhook struct
// with the TypeMetadata initialized to the current API version.
func NewSecurityEventWebhook() *SecurityEventWebhook {
	return &SecurityEventWebhook{
		TypeMeta: metav1.TypeMeta{
			Kind:       KindSecurityEventWebhook,
			APIVersion: GroupVersionCurrent,
		},
	}
}

// NewSecurityEventWebhookList creates a new SecurityEventWebhookList struct
// with the TypeMetadata initialized to the current API version.
func NewSecurityEventWebhookList() *SecurityEventWebhookList {
	return &SecurityEventWebhookList{
		TypeMeta: metav1.TypeMeta{
			Kind:       KindSecurityEventWebhookList,
			APIVersion: GroupVersionCurrent,
		},
	}
}
