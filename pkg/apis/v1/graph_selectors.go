// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package v1

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
)

// GraphSelectors provides selectors used to asynchronously perform associated queries for an edge or a node.
// These selectors are used in the other raw and service graph APIs to look update additional data for an edge or a
// node. The format of these selectors is the Kibana-style selector.  For example,
//   source_namespace == "namespace1 || (dest_type == "wep" && dest_namespace == "namespace2")
//
// The JSON formatted output of this is actually a simple set of selector strings for each search option:
// {
//   "l3_flows": "xx == 'y'",
//   "l7_flows": "xx == 'y'",
//   "dns_logs": "xx == 'y'"
// }
type GraphSelectors struct {
	L3Flows *GraphSelector
	L7Flows *GraphSelector
	DNSLogs *GraphSelector
}

// And combines two sets of selectors by ANDing them together.
func (s GraphSelectors) And(s2 GraphSelectors) GraphSelectors {
	return GraphSelectors{
		L3Flows: NewGraphSelector(OpAnd, s.L3Flows, s2.L3Flows),
		L7Flows: NewGraphSelector(OpAnd, s.L7Flows, s2.L7Flows),
		DNSLogs: NewGraphSelector(OpAnd, s.DNSLogs, s2.DNSLogs),
	}
}

// Or combines two sets of selectors by ORing them together.
func (s GraphSelectors) Or(s2 GraphSelectors) GraphSelectors {
	return GraphSelectors{
		L3Flows: NewGraphSelector(OpOr, s.L3Flows, s2.L3Flows),
		L7Flows: NewGraphSelector(OpOr, s.L7Flows, s2.L7Flows),
		DNSLogs: NewGraphSelector(OpOr, s.DNSLogs, s2.DNSLogs),
	}
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

type GraphSelectorOperator string

const (
	OpAll      GraphSelectorOperator = "*"
	OpIn       GraphSelectorOperator = " IN "
	OpAnd      GraphSelectorOperator = " AND "
	OpOr       GraphSelectorOperator = " OR "
	OpEqual    GraphSelectorOperator = " == "
	OpNotEqual GraphSelectorOperator = " != "

	// Special case internal operator used to indicate an impossible match. This is used to simplify the construction
	// of the selectors.
	OpNoMatch GraphSelectorOperator = " *NOMATCH* "

	// Start and end list and list separator for OpIn.
	OpInListStart = "("
	OpInListSep   = ", "
	OpInListEnd   = ")"
)

type GraphSelector struct {
	operator GraphSelectorOperator

	// Valid if operator is && or ||
	selectors []*GraphSelector

	// Valid if operator is == or !=.  Value is an interface to allow string and numerical values.
	key   string
	value interface{}
}

func (s *GraphSelector) SelectorString() string {
	if s == nil {
		return ""
	} else if ss, noMatch := s.selectorString(false); noMatch {
		return ""
	} else if ss == "" {
		return string(OpAll)
	} else {
		return ss
	}
}

func (s *GraphSelector) selectorString(nested bool) (sel string, noMatch bool) {
	if s == nil {
		return "", false
	}

	sb := strings.Builder{}
	switch s.operator {
	case OpAnd, OpOr:
		parts := make(map[string]struct{})
		var foundNoMatch bool
		var ordered []string
		for i := 0; i < len(s.selectors); i++ {
			if ss, noMatch := s.selectors[i].selectorString(true); noMatch {
				// The process selector indicates "not valid". If this is an OpAnd then then entire selector is not
				// valid, otherwise we just skip this
				if s.operator == OpAnd {
					return "", true
				}
				foundNoMatch = true
			} else if _, ok := parts[ss]; !ok {
				parts[ss] = struct{}{}
				ordered = append(ordered, ss)
			}
		}
		sort.Strings(ordered)
		if len(ordered) > 0 {
			if nested {
				sb.WriteString("(")
			}
			for i := 0; i < len(ordered)-1; i++ {
				sb.WriteString(ordered[i])
				sb.WriteString(string(s.operator))
			}
			sb.WriteString(ordered[len(ordered)-1])
			if nested {
				sb.WriteString(")")
			}
		} else if foundNoMatch {
			// We found one or more no matches with no valid selectors, so the full selector is non-matching.
			return "", true
		}
	case OpEqual, OpNotEqual:
		sb.WriteString(s.key)
		sb.WriteString(string(s.operator))
		if _, ok := s.value.(string); ok {
			sb.WriteString(fmt.Sprintf("\"%s\"", s.value))
		} else {
			sb.WriteString(fmt.Sprintf("%v", s.value))
		}
	case OpIn:
		sb.WriteString(s.key)
		sb.WriteString(string(s.operator))
		sb.WriteString(OpInListStart)
		sb.WriteString("\"")
		value := s.value.([]string)
		for i := 0; i < len(value)-1; i++ {
			sb.WriteString(value[i])
			sb.WriteString("\"")
			sb.WriteString(OpInListSep)
			sb.WriteString("\"")
		}
		sb.WriteString(value[len(value)-1])
		sb.WriteString("\"")
		sb.WriteString(OpInListEnd)
	case OpNoMatch:
		return "", true
	}
	return sb.String(), false
}

func NewGraphSelector(op GraphSelectorOperator, parts ...interface{}) *GraphSelector {
	gs := &GraphSelector{
		operator: op,
	}
	switch op {
	case OpNoMatch:
		// Nothing to extract for the no-match operator.
	case OpAnd, OpOr:
		for _, part := range parts {
			egs, ok := part.(*GraphSelector)
			if egs == nil || !ok {
				continue
			}
			if egs.operator == op {
				// If same operand, then expand into this selector to reduce nesting.
				gs.selectors = append(gs.selectors, egs.selectors...)
			} else {
				gs.selectors = append(gs.selectors, egs)
			}
		}

		// Special case if we have zero or 1 expressions.
		if len(gs.selectors) == 0 {
			return nil
		} else if len(gs.selectors) == 1 {
			return gs.selectors[0]
		}
	case OpEqual, OpNotEqual:
		gs.key = parts[0].(string)
		gs.value = parts[1]
	case OpIn:
		gs.key = parts[0].(string)

		// At the moment, the only time we use OpIn is for a slice of strings. This may change in the future, but
		// no point handling other types just yet.
		value := parts[1].([]string)
		if len(value) == 0 {
			return nil
		}
		gs.value = value
	default:
		log.Errorf("Unexpected selector type: %s", op)
	}

	return gs
}
