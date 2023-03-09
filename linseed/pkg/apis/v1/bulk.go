// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package v1

// BulkResponse summarizes the results of a bulk creation operation.
type BulkResponse struct {
	Total     int `json:"total"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`

	Errors []BulkError `json:"errors,omitempty"`

	// Specific items and their status.
	Created []BulkItem `json:"created"`
	Deleted []BulkItem `json:"deleted"`
	Updated []BulkItem `json:"updated"`
}

type BulkItemStatus string

const (
	StatusOK     BulkItemStatus = "OK"
	StatusFailed BulkItemStatus = "Failed"
)

type BulkItem struct {
	ID     string         `json:"id"`
	Status BulkItemStatus `json:"status"`
}
