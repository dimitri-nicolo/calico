// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package api

type FeatureID string

const (
	PolicyRec FeatureID = "Policy Recommendation"
)

// Error is the message returned by backend in case any error happens while processing
// a request.
type Error struct {
	Code    int       `json:"code"`
	Message string    `json:"message"`
	Reason  string    `json:"reason"`
	Feature FeatureID `json:"feature"`
}
