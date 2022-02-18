// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package v1

// BulkRequest contains the parameters to perform Elastic bulk operations.
type BulkRequest struct {
	// ClusterName defines the name of the cluster.
	ClusterName string `json:"cluster" validate:"omitempty"`

	// Delete defines the delete action and its associated source data.
	Delete *BulkRequestData `json:"delete" validate:"omitempty"`

	// Delete defines the dismiss action and its associated source data.
	Dismiss *BulkRequestData `json:"dismiss" validate:"omitempty"`
}

// BulkRequestData contains the associated source data for each bulk operation.
type BulkRequestData struct {
	// Items defines an array of items to perform bulk operations.
	Items []BulkRequestItem `json:"items" validate:"required"`
}

// BulkRequestItem contains the ID of each document to perform bulk operations.
type BulkRequestItem struct {
	// The ID of the document.
	ID string `json:"id" validate:"required"`
}

// BulkResponse contains the individual results of each operation in the request, returned in the order submitted.
type BulkResponse struct {
	// How long, in milliseconds, it took to process the bulk request.
	Took int `json:"took" validate:"omitempty"`

	// If true, one or more of the operations in the bulk request did not complete successfully.
	Errors bool `json:"errors" validate:"omitempty"`

	// Contains the result of each operation in the bulk request, in the order they were submitted.
	Items []BulkResponseItem `json:"items" validate:"omitempty"`
}

// BulkResponseItem contains the result of each bulk operation in the request, in the order they were submitted.
type BulkResponseItem struct {
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
	Error *BulkErrorDetails `json:"error" validate:"omitempty"`
}

// BulkErrorDetails contains additional information about the failed operation.
type BulkErrorDetails struct {
	// Error type for the operation.
	Type string `json:"type" validate:"required"`

	// Reason for the failed operation.
	Reason string `json:"reason" validate:"required"`
}
