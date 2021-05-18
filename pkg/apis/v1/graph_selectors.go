// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package v1

import (
	"encoding/json"
	"strings"
)

const (
	OpAnd = " && "
	OpOr  = " || "
)

// GraphSelectors provides selectors used to asynchronously perform associated queries for an edge or a node.
// These selectors are used in the other raw and service graph APIs to look update additional data for an edge or a
// node. The format of these selectors is the Kibana-style selector.  For example,
//   source_namespace == "namespace1 OR (dest_type == "wep" AND dest_namespace == "namespace2")
type GraphSelectors struct {
	L3Flows GraphSelector
	L7Flows GraphSelector
	DNSLogs GraphSelector
}

// When marshalled to JSON we only include the non-empty values.
func (s GraphSelectors) MarshalJSON() ([]byte, error) {
	val := struct {
		L3Flows string `json:"l3_flows,omitempty"`
		L7Flows string `json:"l7_flows,omitempty"`
		DNSLogs string `json:"dns_logs,omitempty"`
	}{
		L3Flows: s.L3Flows.SelectorString(),
		L7Flows: s.L7Flows.SelectorString(),
		DNSLogs: s.DNSLogs.SelectorString(),
	}
	return json.Marshal(val)
}

// GetEdgeSelectors returns a set of selectors for a graph edge from the supplied node selectors.
func GetEdgeSelectors(source, dest GraphSelectors) GraphSelectors {
	// DNS logs are a little tricky, so for now just exclude from the edge statistics completely.
	// L7 logs require both source and dest selectors. If either is missing, exclude the L7 selector.
	var l7Flows GraphSelector
	if source.L7Flows.Source != "" && dest.L7Flows.Dest != "" {
		l7Flows = GraphSelector{
			Source: source.L7Flows.Source,
			Dest:   dest.L7Flows.Dest,
			isEdge: true,
		}
	}
	return GraphSelectors{
		L3Flows: GraphSelector{
			Source: source.L3Flows.Source,
			Dest:   dest.L3Flows.Dest,
			isEdge: true,
		},
		L7Flows: l7Flows,
	}
}

// And combines two sets of selectors by ANDing them together.
func (s GraphSelectors) And(s2 GraphSelectors) GraphSelectors {
	return GraphSelectors{
		L3Flows: s.L3Flows.And(s2.L3Flows),
		L7Flows: s.L7Flows.And(s2.L7Flows),
		DNSLogs: s.DNSLogs.And(s2.DNSLogs),
	}
}

// Or combines two sets of selectors by ORing them together.
func (s GraphSelectors) Or(s2 GraphSelectors) GraphSelectors {
	return GraphSelectors{
		L3Flows: s.L3Flows.Or(s2.L3Flows),
		L7Flows: s.L7Flows.Or(s2.L7Flows),
		DNSLogs: s.DNSLogs.Or(s2.DNSLogs),
	}
}

// GraphSelector contains the source and dest selector strings for a source a dest either of an edge or of a node.
// By default these encompass a node (and that is what the graph constructor calculates).  The GetEdgeSelections
// function provides a way to create a GraphSelector for an edge from the source and destination node.
type GraphSelector struct {
	// These fields are not exposed directly over JSON, instead they are combined to form a single selector.
	// The Op is set to OR for a node and AND for an edge.
	Source string
	Dest   string
	isEdge bool
}

// When marshalled to JSON, the source and dest selectors are combined to return a single selector string. They are
// either ANDed or ORed depending on whether the selectors are for an Edge or a Node respectively.
func (s GraphSelector) SelectorString() string {
	if s.isEdge {
		return graphSelectorOp(s.Source, s.Dest, OpAnd)
	}
	return graphSelectorOp(s.Source, s.Dest, OpOr)
}

// And combines two selectors by ANDing them together.
func (s GraphSelector) And(s2 GraphSelector) GraphSelector {
	return GraphSelector{
		Source: graphSelectorOp(s.Source, s2.Source, OpAnd),
		Dest:   graphSelectorOp(s.Dest, s2.Dest, OpAnd),
	}
}

// Or combines two selectors by ORing them together.
func (s GraphSelector) Or(s2 GraphSelector) GraphSelector {
	return GraphSelector{
		Source: graphSelectorOp(s.Source, s2.Source, OpOr),
		Dest:   graphSelectorOp(s.Dest, s2.Dest, OpOr),
	}
}

// graphSelectorOp combines two selector strings into a single string using the supplied operator.
func graphSelectorOp(s, s2, op string) string {
	if s == "" {
		return s2
	} else if s2 == "" {
		return s
	}
	return maybeWithParens(s, op) + op + maybeWithParens(s2, op)
}

// maybeWithParens adds parenthesis to a selector string if required depending on the operator.
//
// Note that the graph constructor processing already puts groups of ANDed selectors in parenthesis, so if joining two
// selectors together using OR, it is not necessary to add additional parenthesis.  Otherwise, if joining together using
// AND, it is only necessary to add additional parenthesis if the selector contains an OR operation.
//
// This approximate algorithm works in conjunction with the processing in graphconstructor (in the service graph module)
// and avoids unnecessary nesting of parenthesis. At worst it will over-include parenthesis.
func maybeWithParens(s, op string) string {
	if op == OpOr {
		// We are not enclosing group separated by Ors.
		return s
	}

	// If the string contains an Or, then we'll need to encase in parens.
	if strings.Contains(s, OpOr) {
		return "(" + s + ")"
	}
	return s
}
