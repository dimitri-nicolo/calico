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
	"k8s.io/apimachinery/pkg/types"
)

const (
	KindGlobalReportType     = "GlobalReportType"
	KindGlobalReportTypeList = "GlobalReportTypeList"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GlobalReportType contains the configuration for a non-namespaced report type.
type GlobalReportType struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Specification of the GlobalReport.
	Spec ReportTypeSpec `json:"spec,omitempty"`
}

// ReportTypeSpec contains the various templates, and configuration used to render a specific type of report.
type ReportTypeSpec struct {
	// The summary template, explicitly used by the UI to render a summary version of the report. This should render
	// to json containing a sets of widgets that the UI can use to render the summary. The rendered data is returned
	// on the list query of the reports.
	UISummaryTemplate ReportTemplate `json:"uiSummaryTemplate,omitempty" validate:"required"`

	// The complete template, explicitly used by the UI to render a full version of the report. This should render to
	// json containing a set of widgets that the UI can use to render the full report.
	UICompleteTemplate ReportTemplate `json:"uiCompleteTemplate,omitempty" validate:"required"`

	// The set of templates used to render the report for downloads.
	DownloadTemplates []ReportTemplate `json:"downloadTemplates,omitempty"`

	// Whether to include endpoint data in the report. The actual endpoints included may be filtered by the Report,
	// but will otherwise contain the full set of endpoints.
	IncludeEndpointData bool `json:"includeEndpointData,omitempty"`

	// Whether to include endpoint-to-endpoint flow log data in the report.
	IncludeEndpointFlowLogData bool `json:"includeEndpointFlowLogData,omitempty"`

	// What audit log data should be included in the report. If not specified, the report will contain no audit log
	// data. The selection may be further filtered by the Report.
	AuditEventsSelection *AuditEventsSelection `json:"auditEventsSelection,omitempty" validate:"omitempty"`
}

// ReportTemplate defines a template used to render a report into downloadable or UI compatible format.
type ReportTemplate struct {
	// The name of this template. This should be unique across all template names within a ReportType. This will be used
	// by the UI as the suffix of the downloadable file name.
	Name string `json:"name,omitempty" validate:"name,required"`

	// A user-facing description of the template.
	Description string `json:"description,omitempty"`

	// The base-64 encoded go template used to render the report data.
	Template string `json:"template,omitempty" validate:"required"`
}

// AuditEventsSelection defines which set of resources should be audited.
type AuditEventsSelection struct {
	// Resources lists the resources that will be included in the audit logs in the ReportData.  Blank fields in the
	// listed ResourceID structs are treated as wildcards.
	Resources []ResourceID `json:"resources,omitempty" validate:"omitempty"`
}

// ResourceID is used to identify a resource instance in the report data, and is used as a filter for resources in the Report configuration.
//
// When used to identify a resource, all valid fields will be set.
//
// When used as a resource filter, an empty field value indicates a wildcard. For example, if Kind is set to "NetworkPolicy" and all other fields
// are blank then this filter would include all NetworkPolicy resources across all namespaces, including both Calico and Kubernetes resource types.
type ResourceID struct {
	metav1.TypeMeta `json:",inline"`
	Name            string    `json:"name,omitempty" validate:"omitempty"`
	Namespace       string    `json:"namespace,omitempty" validate:"omitempty"`
	UUID            types.UID `json:"uuid,omitempty" validate:"omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GlobalReportTypeList contains a list of GlobalReportType resources.
type GlobalReportTypeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []GlobalReportType `json:"items"`
}

// New GlobalReportType creates a new (zeroed) GlobalReportype struct with the TypeMetadata
// initialized to the current version.
func NewGlobalReportType() *GlobalReportType {
	return &GlobalReportType{
		TypeMeta: metav1.TypeMeta{
			Kind:       KindGlobalReportType,
			APIVersion: GroupVersionCurrent,
		},
	}
}

// NewGlobalReportTypeList creates a new (zeroed) GlobalReportTypeList struct with the TypeMetadata
// initialized to the current version.
func NewGlobalReportTypeList() *GlobalReportTypeList {
	return &GlobalReportTypeList{
		TypeMeta: metav1.TypeMeta{
			Kind:       KindGlobalReportTypeList,
			APIVersion: GroupVersionCurrent,
		},
	}
}
