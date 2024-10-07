// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package v1

// GraphStats contains L3, L7 and process statistics.  If there were no associated statistics recorded then the
// value will be nil and omitted entirely from the JSON response.
type GraphStats struct {
	L3        *GraphL3Stats   `json:"l3,omitempty"`
	L7        *GraphL7Stats   `json:"l7,omitempty"`
	DNS       *GraphDNSStats  `json:"dns,omitempty"`
	Processes *GraphProcesses `json:"processes,omitempty"`
}

// Combine returns a GraphStats that combines the stats from t and t2.
func (s GraphStats) Combine(s2 GraphStats) GraphStats {
	return GraphStats{
		s.L3.Combine(s2.L3),
		s.L7.Combine(s2.L7),
		s.DNS.Combine(s2.DNS),
		s.Processes.Combine(s2.Processes),
	}
}
