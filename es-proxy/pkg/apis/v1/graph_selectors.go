// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package v1

// GraphSelectors provides selectors used to asynchronously perform associated queries for an edge or a node.
// These selectors are used in the other raw and service graph APIs to look update additional data for an edge or a
// node. The format of these selectors is the same as used in GlobalAlerts.  For example,
//
//	source_namespace = "namespace1" || (dest_type = "wep" && dest_namespace != "namespace2")
//
// The JSON formatted output of this is actually a simple set of selector strings for each search option:
//
//	{
//	  "l3_flows": "xx = 'y'",
//	  "l7_flows": "xx = 'y'",
//	  "dns_logs": "xx = 'y'"
//	  "alerts": "_id = 'abcdef'"
//	}
//
// A nil value means the selector is not valid. An empty value indicate select all.
type GraphSelectors struct {
	L3Flows       *string `json:"l3_flows,omitempty"`
	L7Flows       *string `json:"l7_flows,omitempty"`
	DNSLogs       *string `json:"dns_logs,omitempty"`
	Alerts        *string `json:"alerts,omitempty"`
	PacketCapture *string `json:"packet_capture,omitempty"`
}

type GraphSelectorOperator string

const (
	OpIn       GraphSelectorOperator = " IN "
	OpAnd      GraphSelectorOperator = " AND "
	OpOr       GraphSelectorOperator = " OR "
	OpEqual    GraphSelectorOperator = " = "
	OpNotEqual GraphSelectorOperator = " != "

	// Special case internal operator used to indicate an impossible match. This is used to simplify the construction
	// of the selectors.
	OpNoMatch GraphSelectorOperator = " *NOMATCH* "

	// Start and end list and list separator for OpIn.
	OpInListStart = "("
	OpInListSep   = ", "
	OpInListEnd   = ")"
)
