// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package v1

import (
	"time"
)

// IPSetThreatFeedParams defines the parameters
// to query threat feeds for malicious IPs
type IPSetThreatFeedParams struct {
	QueryParams `json:",inline" validate:"required"`

	// Match on the ID of the threat feed
	ID string `json:"id"`
}

// DomainNameSetThreatFeedParams defines the parameters
// to query threat feeds for malicious DNS names
type DomainNameSetThreatFeedParams struct {
	QueryParams `json:",inline" validate:"required"`

	// Match on the ID of the threat feed
	ID string `json:"id"`
}

// IPSetThreatFeed defines a threat feed that
// stores malicious IPs
type IPSetThreatFeed struct {
	ID          string `json:"id"`
	SeqNumber   *int64 `json:"seq_number"`
	PrimaryTerm *int64 `json:"primary_term"`

	Data *IPSetThreatFeedData `json:"data"`
}

type IPSetThreatFeedData struct {
	CreatedAt time.Time `json:"created_at"`

	IPs []string `json:"ips"`

	// Cluster is populated by linseed from the request context.
	Cluster string `json:"cluster,omitempty"`
}

// DomainNameSetThreatFeed defines a threat feed that
// stores malicious DNS domains
type DomainNameSetThreatFeed struct {
	ID          string `json:"id"`
	SeqNumber   *int64 `json:"seq_number"`
	PrimaryTerm *int64 `json:"primary_term"`

	Data *DomainNameSetThreatFeedData `json:"data"`
}

type DomainNameSetThreatFeedData struct {
	CreatedAt time.Time `json:"created_at"`

	Domains []string `json:"domains"`

	// Cluster is populated by linseed from the request context.
	Cluster string `json:"cluster,omitempty"`
}
