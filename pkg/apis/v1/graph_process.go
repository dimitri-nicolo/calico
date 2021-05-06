package v1

import (
	"encoding/json"
	"math"
	"sort"
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
			MinNumNamesPerFlow: minExcludeZero(existing.MinNumNamesPerFlow, gp.MinNumNamesPerFlow),
			MaxNumNamesPerFlow: math.Max(existing.MaxNumNamesPerFlow, gp.MaxNumNamesPerFlow),
			MinNumIDsPerFlow:   minExcludeZero(existing.MinNumIDsPerFlow, gp.MinNumIDsPerFlow),
			MaxNumIDsPerFlow:   math.Max(existing.MaxNumIDsPerFlow, gp.MaxNumIDsPerFlow),
		}
	}

	return p
}

type GraphEndpointProcess struct {
	// The process name. If aggregated it will be set to "*"
	Name string `json:"name"`

	// The minimum number of process names per flow.
	MinNumNamesPerFlow float64 `json:"min_num_names_per_flow"`

	// The max number of process names per flow.
	MaxNumNamesPerFlow float64 `json:"max_num_names_per_flow"`

	// The minimum number of process IDs per flow.
	MinNumIDsPerFlow float64 `json:"min_num_ids_per_flow"`

	// The max number of process IDs per flow.
	MaxNumIDsPerFlow float64 `json:"max_num_ids_per_flow"`
}

func minExcludeZero(a, b float64) float64 {
	if a == 0 {
		return b
	} else if b == 0 {
		return a
	}
	return math.Min(a, b)
}
