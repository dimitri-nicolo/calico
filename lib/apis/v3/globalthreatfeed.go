// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package v3

import (
	"time"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	KindGlobalThreatFeed     = "GlobalThreatFeed"
	KindGlobalThreatFeedList = "GlobalThreatFeedList"
	DefaultPullPeriod        = 24 * time.Hour
	MinPullPeriod            = 5 * time.Minute
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GlobalThreatFeed is a source of intel for possible threats to the cluster. This
// object configures how Tigera components communicate with the feed and update
// detection jobs or policy based on the intel.
// +kubebuilder:subresource:status
type GlobalThreatFeed struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Specification of the GlobalThreatFeed.
	Spec   GlobalThreatFeedSpec   `json:"spec,omitempty"`
	Status GlobalThreatFeedStatus `json:"status,omitempty"`
}

// GlobalThreatFeedSpec contains the specification of a GlobalThreatFeed resource.
type GlobalThreatFeedSpec struct {
	// Content describes the kind of data the data feed provides.
	Content          ThreatFeedContent     `json:"content,omitempty" validate:"omitempty,oneof=IPSet DomainNameSet"`
	GlobalNetworkSet *GlobalNetworkSetSync `json:"globalNetworkSet,omitempty"`
	Pull             *Pull                 `json:"pull,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GlobalThreatFeedList contains a list of GlobalThreatFeed resources.
type GlobalThreatFeedList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []GlobalThreatFeed `json:"items"`
}

type ThreatFeedContent string

const (
	ThreatFeedContentIPset         ThreatFeedContent = "IPSet"
	ThreatFeedContentDomainNameSet ThreatFeedContent = "DomainNameSet"
)

var ThreatFeedContentDefault = ThreatFeedContentIPset

type GlobalNetworkSetSync struct {
	Labels map[string]string `json:"labels,omitempty" validate:"labels"`
}

type Pull struct {
	Period string    `json:"period,omitempty"`
	HTTP   *HTTPPull `json:"http" validate:"required"`
}

type HTTPPull struct {
	Format  ThreatFeedFormat `json:"format,omitempty" validate:"omitempty,eq=NewlineDelimited"`
	URL     string           `json:"url" validate:"required,url"`
	Headers []HTTPHeader     `json:"headers,omitempty" validate:"dive"`
}

type ThreatFeedFormat string

const (
	ThreatFeedFormatNewlineDelimited ThreatFeedFormat = "NewlineDelimited"
)

var ThreatFeedFormatDefault = ThreatFeedFormatNewlineDelimited

type HTTPHeader struct {
	Name      string            `json:"name" validate:"printascii"`
	Value     string            `json:"value,omitempty"`
	ValueFrom *HTTPHeaderSource `json:"valueFrom,omitempty"`
}

type HTTPHeaderSource struct {
	// Selects a key of a ConfigMap.
	// +optional
	ConfigMapKeyRef *k8sv1.ConfigMapKeySelector `json:"configMapKeyRef,omitempty"`
	// Selects a key of a secret in the pod's namespace
	// +optional
	SecretKeyRef *k8sv1.SecretKeySelector `json:"secretKeyRef,omitempty"`
}

type GlobalThreatFeedStatus struct {
	LastSuccessfulSync   metav1.Time      `json:"lastSuccessfulSync"`
	LastSuccessfulSearch metav1.Time      `json:"lastSuccessfulSearch"`
	ErrorConditions      []ErrorCondition `json:"errorConditions"`
}

type ErrorCondition struct {
	Type    string `json:"type" validate:"required"`
	Message string `json:"message" validate:"required"`
}

// NewGlobalThreatFeed creates a new (zeroed) GlobalThreatFeed struct with the TypeMetadata initialised to the current
// version.
func NewGlobalThreatFeed() *GlobalThreatFeed {
	return &GlobalThreatFeed{
		TypeMeta: metav1.TypeMeta{
			Kind:       KindGlobalThreatFeed,
			APIVersion: GroupVersionCurrent,
		},
	}
}

// NewGlobalThreatFeedList creates a new (zeroed) GlobalThreatFeedList struct with the TypeMetadata initialised to the current
// version.
func NewGlobalThreatFeedList() *GlobalThreatFeedList {
	return &GlobalThreatFeedList{
		TypeMeta: metav1.TypeMeta{
			Kind:       KindGlobalThreatFeedList,
			APIVersion: GroupVersionCurrent,
		},
	}
}
