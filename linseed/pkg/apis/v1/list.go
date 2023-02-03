// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package v1

// List represents a List response on the API. It contains
// the items returned from the request, as well as additional metadata.
type List[T any] struct {
	// Items are the returned objects from the list request.
	Items []T `json:"items"`

	// AfterKey is an opaque object passed from the server if there
	// are additional items to return. If nil, it means the request
	// was fully satisfied. If non-nil, it can be included on a subsequent
	// request to retrieve the next page of items.
	AfterKey map[string]interface{} `json:"after_key,omitempty"`
}
