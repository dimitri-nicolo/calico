// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package v1

import (
	"encoding/json"
	"sort"

	"github.com/tigera/es-proxy/pkg/math"
)

// GraphProcesses encapsulates the set of processes associated with a particular node or edge in the graph.
// The processes are split into source (opening connections) and dest (listening for connections).
type GraphProcesses struct {
	Source GraphEndpointProcesses `json:"source,omitempty"`
	Dest   GraphEndpointProcesses `json:"dest,omitempty"`
}

func (p *GraphProcesses) Combine(p2 *GraphProcesses) *GraphProcesses {
	if p == nil {
		return p2
	} else if p2 == nil {
		return p
	}

	return &GraphProcesses{
		Source: p.Source.Combine(p2.Source),
		Dest:   p.Dest.Combine(p2.Dest),
	}
}

type GraphEndpointProcesses map[string]GraphEndpointProcess

func (p GraphEndpointProcesses) Copy() GraphEndpointProcesses {
	pcopy := make(GraphEndpointProcesses)
	for n, gep := range p {
		pcopy[n] = gep
	}
	return pcopy
}

func (p GraphEndpointProcesses) MarshalJSON() ([]byte, error) {
	var names []string
	for name := range p {
		names = append(names, name)
	}
	sort.Strings(names)

	processes := make([]GraphEndpointProcess, len(names))
	for i, name := range names {
		processes[i] = p[name]
	}
	return json.Marshal(processes)
}

// Include combines the two sets of process infos.
func (p GraphEndpointProcesses) Combine(p2 GraphEndpointProcesses) GraphEndpointProcesses {
	if len(p) == 0 {
		return p2
	} else if len(p2) == 0 {
		return p
	}

	// Take a copy of p.
	p = p.Copy()

	for _, gp := range p2 {
		existing, ok := p[gp.Name]
		if !ok {
			p[gp.Name] = gp
			continue
		}
		p[gp.Name] = GraphEndpointProcess{
			Name:               gp.Name,
			MinNumNamesPerFlow: math.MinIntGtZero(existing.MinNumNamesPerFlow, gp.MinNumNamesPerFlow),
			MaxNumNamesPerFlow: math.MaxIntGtZero(existing.MaxNumNamesPerFlow, gp.MaxNumNamesPerFlow),
			MinNumIDsPerFlow:   math.MinIntGtZero(existing.MinNumIDsPerFlow, gp.MinNumIDsPerFlow),
			MaxNumIDsPerFlow:   math.MaxIntGtZero(existing.MaxNumIDsPerFlow, gp.MaxNumIDsPerFlow),
		}
	}

	return p
}

type GraphEndpointProcess struct {
	// The process name. If aggregated it will be set to "*"
	Name string `json:"name"`

	// The minimum number of process names per flow.
	MinNumNamesPerFlow int `json:"min_num_names_per_flow"`

	// The max number of process names per flow.
	MaxNumNamesPerFlow int `json:"max_num_names_per_flow"`

	// The minimum number of process IDs per flow.
	MinNumIDsPerFlow int `json:"min_num_ids_per_flow"`

	// The max number of process IDs per flow.
	MaxNumIDsPerFlow int `json:"max_num_ids_per_flow"`
}
