// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package v3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	KindPacketCapture     = "PacketCapture"
	KindPacketCaptureList = "PacketCaptureList"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PacketCapture contains the configuration for any packet capture.
type PacketCapture struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Specification of the PacketCapture.
	Spec PacketCaptureSpec `json:"spec,omitempty"`
}

// PacketCaptureSpec contains the values of the packet capture.
type PacketCaptureSpec struct {
	// The selector is an expression used to pick pick out the endpoints that the policy should
	// be applied to.
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

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PacketCaptureList contains a list of PacketCapture resources.
type PacketCaptureList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []PacketCapture `json:"items"`
}

// NewPacketCapture creates a new (zeroed) PacketCapture struct with the TypeMetadata initialised to the current
// version.
func NewPacketCapture() *PacketCapture {
	return &PacketCapture{
		TypeMeta: metav1.TypeMeta{
			Kind:       KindPacketCapture,
			APIVersion: GroupVersionCurrent,
		},
	}
}

// NewPacketCaptureList creates a new (zeroed) PacketCaptureList struct with the TypeMetadata initialised to the current
// version.
func NewPacketCaptureList() *PacketCaptureList {
	return &PacketCaptureList{
		TypeMeta: metav1.TypeMeta{
			Kind:       KindPacketCaptureList,
			APIVersion: GroupVersionCurrent,
		},
	}
}
