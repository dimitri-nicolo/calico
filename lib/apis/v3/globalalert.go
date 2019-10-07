// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package v3

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	KindGlobalAlert     = "GlobalAlert"
	KindGlobalAlertList = "GlobalAlertList"

	GlobalAlertDataSetAudit = "audit"
	GlobalAlertDataSetDNS   = "dns"
	GlobalAlertDataSetFlows = "flows"

	GlobalAlertMetricAvg   = "avg"
	GlobalAlertMetricMax   = "max"
	GlobalAlertMetrixMin   = "min"
	GlobalAlertMetricSum   = "sum"
	GlobalAlertMetricCount = "count"

	GlobalAlertMinPeriod   = time.Minute
	GlobalAlertMinLookback = GlobalAlertMinPeriod
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type GlobalAlert struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Specification of the GlobalAlert.
	Spec   GlobalAlertSpec   `json:"spec,omitempty"`
	Status GlobalAlertStatus `json:"status,omitempty"`
}

type GlobalAlertSpec struct {
	Description string          `json:"description" validate:"required"`
	Severity    int             `json:"severity" validate:"required,min=1,max=100"`
	Period      metav1.Duration `json:"period"`
	Lookback    metav1.Duration `json:"lookback"`
	DataSet     string          `json:"dataSet" validate:"required,oneof=flows dns audit"`
	Query       string          `json:"query"`
	AggregateBy []string        `json:"aggregateBy"`
	Field       string          `json:"field"`
	Metric      string          `json:"metric" validate:"omitempty,oneof=avg max min sum count"`
	Condition   string          `json:"condition" validate:"omitempty,oneof=eq not_eq gt gte lt lte"`
	Threshold   float64         `json:"threshold"`
}

type GlobalAlertStatus struct {
	LastUpdate      metav1.Time      `json:"lastUpdate,omitempty"`
	Active          bool             `json:"active"`
	ExecutionState  string           `json:"executionState"`
	LastFired       metav1.Time      `json:"lastFired"`
	LastTriggered   metav1.Time      `json:"lastTriggered"`
	ErrorConditions []ErrorCondition `json:"errorConditions"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GlobalAlertList contains a list of GlobalAlert resources.
type GlobalAlertList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []GlobalAlert `json:"items"`
}

// NewGlobalAlert creates a new (zeroed) GlobalAlert struct with the TypeMetadata
// initialized to the current version.
func NewGlobalAlert() *GlobalAlert {
	return &GlobalAlert{
		TypeMeta: metav1.TypeMeta{
			Kind:       KindGlobalAlert,
			APIVersion: GroupVersionCurrent,
		},
	}
}

// NewGlobalAlertList creates a new (zeroed) GlobalAlertList struct with the TypeMetadata
// initialized to the current version.
func NewGlobalAlertList() *GlobalAlertList {
	return &GlobalAlertList{
		TypeMeta: metav1.TypeMeta{
			Kind:       KindGlobalAlertList,
			APIVersion: GroupVersionCurrent,
		},
	}
}
