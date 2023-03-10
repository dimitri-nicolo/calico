// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package v1

// BulkResponse summarizes the results of a bulk creation operation.
type BulkResponse struct {
	Total     int `json:"total"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`

	Errors []BulkError `json:"errors,omitempty"`

	// Specific items and their status.
	Created []BulkItem `json:"created,omitempty"`
	Deleted []BulkItem `json:"deleted,omitempty"`
	Updated []BulkItem `json:"updated,omitempty"`
}

type BulkItem struct {
	ID     string `json:"id"`
	Status int    `json:"status"`
}
