// Copyright (c) 2022-2024 Tigera, Inc. All rights reserved.
package v1

import lapi "github.com/projectcalico/calico/linseed/pkg/apis/v1"

// BulkEventRequest contains the parameters to perform Elastic bulk operations.
type BulkEventRequest struct {
	// ClusterName defines the name of the cluster.
	ClusterName string `json:"cluster" validate:"omitempty"`

	// Delete defines the delete action and its associated source data.
	Delete *BulkEventRequestData `json:"delete" validate:"omitempty"`

	// Dismiss defines the dismiss action and its associated source data.
	Dismiss *BulkEventRequestData `json:"dismiss" validate:"omitempty"`

	// Restore defines the restore action and its associated source data.
	Restore *BulkEventRequestData `json:"restore" validate:"omitempty"`
}

// BulkEventRequestData contains the associated source data for each bulk operation.
type BulkEventRequestData struct {
	// Items defines an array of items to perform bulk operations.
	Items []BulkEventRequestItem `json:"items" validate:"required"`
}

// BulkEventRequestItem contains the ID of each document to perform bulk operations.
type BulkEventRequestItem struct {
	// Name of the index associated with the ID.
	Index string `json:"index" validate:"required"`

	// The ID of the document.
	ID string `json:"id" validate:"required"`
}

// BulkEventResponse contains the individual results of each operation in the request, returned in the order submitted.
type BulkEventResponse struct {
	// How long, in milliseconds, it took to process the bulk request.
	Took int `json:"took" validate:"omitempty"`

	// If true, one or more of the operations in the bulk request did not complete successfully.
	Errors bool `json:"errors" validate:"omitempty"`

	// Contains the result of each operation in the bulk request, in the order they were submitted.
	Items []BulkEventResponseItem `json:"items" validate:"omitempty"`
}

// BulkEventResponseItem contains the result of each bulk operation in the request, in the order they were submitted.
type BulkEventResponseItem struct {
	// Name of the index associated with the bulk operation.
	Index string `json:"index" validate:"required"`

	// The document ID associated with the bulk operation.
	ID string `json:"id" validate:"required"`

	// Result of the operation. Successful value is deleted or updated.
	Result string `json:"result" validate:"required"`

	// HTTP status code returned for the bulk operation.
	Status int `json:"status" validate:"omitempty"`

	// Contains additional information about the failed operation.
	// The parameter is only returned for failed operations.
	Error *BulkEventErrorDetails `json:"error" validate:"omitempty"`
}

// BulkEventErrorDetails contains additional information about the failed operation.
type BulkEventErrorDetails struct {
	// Error type for the operation.
	Type string `json:"type" validate:"required"`

	// Reason for the failed operation.
	Reason string `json:"reason" validate:"required"`
}

// EventStatisticsParams mirrors linseed's EventStatisticsParams.
// It is redefined here to accommodate ui-apis logic for FieldValuesParam
// to support additional Namespace and MitreTechnique field values.
type EventStatisticsParams struct {
	// EventParams inherits all the normal events selection parameters.
	// However Sort by time is not supported for statistics.
	// Used to specify the subset of events we want to consider when computing statistics.
	lapi.EventParams `json:",inline"`

	// FieldValues defines the event fields we want to compute field values statistics for.
	FieldValues *FieldValuesParam `json:"field_values,omitempty"`

	// SeverityHistograms defines parameters of the severity histograms we want to compute (name and selector for severity range).
	SeverityHistograms []lapi.SeverityHistogramParam `json:"severity_histograms,omitempty"`
}

// FieldValuesParam mirrors linseed's FieldValuesParam.
// It supports the Namespace and MitreTechnique field values.
type FieldValuesParam struct {
	lapi.FieldValuesParam `json:",inline"`
	NamespaceValues       *lapi.FieldValueParam `json:"namespace,omitempty"`
	MitreTechniqueValues  *lapi.FieldValueParam `json:"mitre_technique,omitempty"`
}

// EventStatistics mirrors linseed's EventStatistics.
// It is redefined here to accommodate ui-apis logic for FieldValues
// to support additional Namespace and MitreTechnique field values.
type EventStatistics struct {
	FieldValues        *FieldValues                      `json:"field_values,omitempty"`
	SeverityHistograms map[string][]lapi.HistogramBucket `json:"severity_histograms,omitempty"`
}

// FieldValues mirrors linseed's FieldValues, with
// additional Namespace and MitreTechnique field values.
type FieldValues struct {
	*lapi.FieldValues    `json:",inline"`
	NamespaceValues      []lapi.FieldValue     `json:"namespace,omitempty"`
	MitreTechniqueValues []MitreTechniqueValue `json:"mitre_technique,omitempty"`
}

// MitreTechniqueValue is similar to a FieldValue (Count) except that:
// - The Value include the name of the MITRE technique instead of just its ID.
// - The Url property captures the URL associated with the MITRE technique.
type MitreTechniqueValue struct {
	lapi.FieldValue `json:",inline"`
	Url             string `json:"url"`
}
