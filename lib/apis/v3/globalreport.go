// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	KindGlobalReport     = "GlobalReport"
	KindGlobalReportList = "GlobalReportList"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GlobalReport contains the configuration for a non-namespaced Report.
type GlobalReport struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Specification of the GlobalReport.
	Spec   ReportSpec   `json:"spec,omitempty"`
	Status ReportStatus `json:"status,omitempty"`
}

// ReportSpec contains the values of the GlobalReport.
type ReportSpec struct {
	// The name of the report type.
	ReportType string `json:"reportType" validate:"name,required"`

	// EndpointsSelection is used to specify which endpoints are in-scope and stored in the generated report data.
	// Only required if endpoints data is gathered in the report.
	EndpointsSelection EndpointsSelection `json:"endpointsSelection,omitempty" validate:"omitempty,selector"`

	// AuditEventsSelection is used to specify which audit events will be gathered.
	// Only required if audit logs are gathered in the report.
	AuditEventsSelection AuditEventsSelection `json:"auditEventsSelection,omitempty" validate:"omitempty"`

	// The reporting job schedule specified in cron format. This specifies the start time of each report. The reporting
	// interval ends at the start of the next report.
	JobSchedule string `json:"jobSchedule,omitempty" validate:"omitempty,reportschedule"`

	// The node selector used to specify which nodes the report job may be scheduled on.
	JobNodeSelector map[string]string `json:"jobNodeSelector,omitempty" validate:"omitempty"`
}

// ReportStatus contains the status of the automated report generation.
type ReportStatus struct {
	LastSuccessfulReport *ReportCreationStatus `json:"lastSuccessfulReport"`
	ErrorConditions      []ErrorCondition      `json:"errorConditions"`
}

// ReportCreationStatus contains the status of the automated report generation.
type ReportCreationStatus struct {
	// The time the report was generated and archived.
	GenerationTime metav1.Time `json:"generationTime"`

	// The start time of the report.
	Start metav1.Time `json:"start"`

	// The end time of the report.
	End metav1.Time `json:"end"`

	// The ReportType as configured at the time the report was generated.
	ReportType string `json:"reportType"`
}

// EndpointsSelection is a set of selectors used to select the endpoints that are considered to be in-scope for the
// report. An empty selector is equivalent to all(). All three selectors are ANDed together.
type EndpointsSelection struct {
	// Endpoints selector, selecting endpoints by endpoint labels. If omitted, all endpoints are included in the report
	// data.
	EndpointSelector string `json:"endpointSelector,omitempty" validate:"omitempty,selector"`

	// Namespace match restricts endpoint selection to those in the selected namespaces.
	Namespaces *NamesAndLabelsMatch `json:"namespaces,omitempty" validate:"omitempty"`

	// ServiceAccount match restricts endpoint selection to those in the selected service accounts.
	ServiceAccounts *NamesAndLabelsMatch `json:"serviceAccounts,omitempty" validate:"omitempty"`
}

// NamesAndLabelsMatch is used to specify resource matches using both label and name selection.
type NamesAndLabelsMatch struct {
	// Names is an optional field that specifies a set of resources by name.
	Names []string `json:"names,omitempty" validate:"omitempty"`

	// Selector is an optional field that selects a set of resources by label.
	// If both Names and Selector are specified then they are AND'ed.
	Selector string `json:"selector,omitempty" validate:"omitempty,selector"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GlobalReportList contains a list of GlobalReport resources.
type GlobalReportList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []GlobalReport `json:"items"`
}

// NewGlobalReport creates a new (zeroed) GlobalReport struct with the TypeMetadata
// initialized to the current version.
func NewGlobalReport() *GlobalReport {
	return &GlobalReport{
		TypeMeta: metav1.TypeMeta{
			Kind:       KindGlobalReport,
			APIVersion: GroupVersionCurrent,
		},
	}
}

// NewGlobalReportList creates a new (zeroed) GlobalReportList struct with the TypeMetadata
// initialized to the current version.
func NewGlobalReportList() *GlobalReportList {
	return &GlobalReportList{
		TypeMeta: metav1.TypeMeta{
			Kind:       KindGlobalReportList,
			APIVersion: GroupVersionCurrent,
		},
	}
}
