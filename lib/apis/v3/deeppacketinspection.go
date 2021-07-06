// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package v3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	KindDeepPacketInspection     = "DeepPacketInspection"
	KindDeepPacketInspectionList = "DeepPacketInspectionList"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status

type DeepPacketInspection struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Specification of the DeepPacketInspection.
	Spec DeepPacketInspectionSpec `json:"spec,omitempty"`
	// Status of the DeepPacketInspection.
	Status DeepPacketInspectionStatus `json:"status,omitempty"`
}

// DeepPacketInspectionSpec contains the values of the deep packet inspection.
type DeepPacketInspectionSpec struct {
	// The selector is an expression used to pick out the endpoints for which deep packet inspection should
	// be performed on. The selector will only match endpoints in the same namespace as the
	// DeepPacketInspection resource.
	//
	// Selector expressions follow this syntax:
	//
	// 	label == "string_literal"  ->  comparison, e.g. my_label == "foo bar"
	// 	label != "string_literal"   ->  not equal; also matches if label is not present
	// 	label in { "a", "b", "c", ... }  ->  true if the value of label X is one of "a", "b", "c"
	// 	label not in { "a", "b", "c", ... }  ->  true if the value of label X is not one of "a", "b", "c"
	// 	has(label_name)  -> True if that label is present
	// 	! expr -> negation of expr
	// 	expr && expr  -> Short-circuit and
	// 	expr || expr  -> Short-circuit or
	// 	( expr ) -> parens for grouping
	// 	all() or the empty selector -> matches all endpoints.
	//
	// Label names are allowed to contain alphanumerics, -, _ and /. String literals are more permissive
	// but they do not support escape characters.
	//
	// Examples (with made-up labels):
	//
	// 	type == "webserver" && deployment == "prod"
	// 	type in {"frontend", "backend"}
	// 	deployment != "dev"
	// 	! has(label_name)
	Selector string `json:"selector,omitempty" validate:"selector"`
}

// DeepPacketInspectionStatus contains status of DPI in each node.
type DeepPacketInspectionStatus struct {
	ErrorConditions []DPIErrorCondition `json:"errorConditions,omitempty"`
	Active          []DPIActive         `json:"active,omitempty"`
	LastUpdated     []DPILastUpdated    `json:"lastUpdated,omitempty"`
}

type DPIErrorCondition struct {
	// Node identifies with a physical node from the cluster via its hostname.
	Node string `json:"node,omitempty"`
	// Message from DPI error.
	Message string `json:"message,omitempty"`
}

type DPIActive struct {
	// Node identifies with a physical node from the cluster via its hostname.
	Node string `json:"node,omitempty"`
	// Success indicates if DPI is running on all workload endpoint matching the selector.
	Success bool `json:"success,omitempty"`
}

type DPILastUpdated struct {
	// Node identifies with a physical node from the cluster via its hostname.
	Node string `json:"node,omitempty"`
	// Timestamp when DPI was last started or stopped on an endpoint.
	Timestamp *metav1.Time `json:"timestamp,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DeepPacketInspectionList contains list of DeepPacketInspection resource.
type DeepPacketInspectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []DeepPacketInspection `json:"items"`
}

// NewDeepPacketInspection creates a new (zeroed) DeepPacketInspection struct with the TypeMetadata
// initialized to the current version.
func NewDeepPacketInspection() *DeepPacketInspection {
	return &DeepPacketInspection{
		TypeMeta: metav1.TypeMeta{
			Kind:       KindDeepPacketInspection,
			APIVersion: GroupVersionCurrent,
		},
	}
}

// NewDeepPacketInspectionList creates a new zeroed) DeepPacketInspectionList struct with the TypeMetadata
// initialized to the current version.
func NewDeepPacketInspectionList() *DeepPacketInspectionList {
	return &DeepPacketInspectionList{
		TypeMeta: metav1.TypeMeta{
			Kind:       KindDeepPacketInspectionList,
			APIVersion: GroupVersionCurrent,
		},
	}
}
