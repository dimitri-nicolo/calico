// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package client

import (
	"fmt"
	"sort"
	"strings"
)

// compareStringSlice compares a slice of strings. Each item in the slice is compared side by side.
// Returns -1, 0 or 1 depending on the match condition.
func compareStringSlice(m, n []string) int {
	minLen := len(m)
	if lm := len(n); lm < minLen {
		minLen = lm
	}
	for i := 0; i < minLen; i++ {
		diff := strings.Compare(m[i], n[i])
		if diff != 0 {
			return diff
		}
	}
	return len(m) - len(n)
}

// sortNodes sorts the nodes based on the order requirements specified in the Page request.
func sortNodes(items []Node, s *Sort) error {
	ns := nodeSorter{items: items}
	if s != nil {
		for _, sb := range s.SortBy {
			ndf, ok := nodeDiffFuncs[sb]
			if !ok {
				return fmt.Errorf("invalid sort field specified in query: %s", sb)
			}
			ns.diff = append(ns.diff, ndf)
		}
		ns.reverse = s.Reverse
	}
	ns.diff = append(ns.diff, nodeDiffFuncs[nodeDefaultSortField])

	// Sort the entries using the specified sort columns.
	sort.Sort(ns)
	return nil
}

type nodeDiffFunc func(p, q *Node) int

var nodeDiffFuncs = map[string]nodeDiffFunc{
	"name":                 func(p, q *Node) int { return strings.Compare(p.Name, q.Name) },
	"numHostEndpoints":     func(p, q *Node) int { return p.NumHostEndpoints - q.NumHostEndpoints },
	"numWorkloadEndpoints": func(p, q *Node) int { return p.NumWorkloadEndpoints - q.NumWorkloadEndpoints },
	"numEndpoints": func(p, q *Node) int {
		return p.NumWorkloadEndpoints + p.NumHostEndpoints - q.NumHostEndpoints - q.NumWorkloadEndpoints
	},
	"bgpIPAddresses": func(p, q *Node) int { return compareStringSlice(p.BGPIPAddresses, q.BGPIPAddresses) },
}

const nodeDefaultSortField = "name"

type nodeSorter struct {
	items   []Node
	diff    []nodeDiffFunc
	reverse bool
}

func (s nodeSorter) Len() int {
	return len(s.items)
}
func (s nodeSorter) Less(i, j int) bool {
	p, q := &s.items[i], &s.items[j]
	for _, df := range s.diff {
		d := df(p, q)
		if d < 0 {
			return !s.reverse
		} else if d > 0 {
			return s.reverse
		}
	}
	return false
}
func (s nodeSorter) Swap(i, j int) {
	s.items[i], s.items[j] = s.items[j], s.items[i]
}

// sortEndpoints sorts the Endpoints based on the order requirements specified in the Page request.
func sortEndpoints(items []Endpoint, s *Sort) error {
	ns := endpointSorter{items: items}
	if s != nil {
		for _, sb := range s.SortBy {
			ndf, ok := endpointDiffFuncs[sb]
			if !ok {
				return fmt.Errorf("invalid sort field specified in query: %s", sb)
			}
			ns.diff = append(ns.diff, ndf)
		}
		ns.reverse = s.Reverse
	}
	ns.diff = append(ns.diff, endpointDiffFuncs[endpointDefaultSortField1], endpointDiffFuncs[endpointDefaultSortField2])

	// Sort the entries using the specified sort columns.
	sort.Sort(ns)
	return nil
}

type endpointDiffFunc func(p, q *Endpoint) int

var endpointDiffFuncs = map[string]endpointDiffFunc{
	"kind":                     func(p, q *Endpoint) int { return strings.Compare(p.Kind, q.Kind) },
	"name":                     func(p, q *Endpoint) int { return strings.Compare(p.Name, q.Name) },
	"namespace":                func(p, q *Endpoint) int { return strings.Compare(p.Namespace, q.Namespace) },
	"node":                     func(p, q *Endpoint) int { return strings.Compare(p.Node, q.Node) },
	"workload":                 func(p, q *Endpoint) int { return strings.Compare(p.Workload, q.Workload) },
	"orchestrator":             func(p, q *Endpoint) int { return strings.Compare(p.Orchestrator, q.Orchestrator) },
	"pod":                      func(p, q *Endpoint) int { return strings.Compare(p.Pod, q.Pod) },
	"interfaceName":            func(p, q *Endpoint) int { return strings.Compare(p.InterfaceName, q.InterfaceName) },
	"ipNetworks":               func(p, q *Endpoint) int { return compareStringSlice(p.IPNetworks, q.IPNetworks) },
	"numGlobalNetworkPolicies": func(p, q *Endpoint) int { return p.NumGlobalNetworkPolicies - q.NumGlobalNetworkPolicies },
	"numNetworkPolicies":       func(p, q *Endpoint) int { return p.NumNetworkPolicies - q.NumNetworkPolicies },
	"numPolicies": func(p, q *Endpoint) int {
		return p.NumGlobalNetworkPolicies + p.NumNetworkPolicies - q.NumGlobalNetworkPolicies - q.NumNetworkPolicies
	},
}

const endpointDefaultSortField1 = "name"
const endpointDefaultSortField2 = "namespace"

type endpointSorter struct {
	items   []Endpoint
	diff    []endpointDiffFunc
	reverse bool
}

func (s endpointSorter) Len() int {
	return len(s.items)
}
func (s endpointSorter) Less(i, j int) bool {
	p, q := &s.items[i], &s.items[j]
	for _, df := range s.diff {
		d := df(p, q)
		if d < 0 {
			return !s.reverse
		} else if d > 0 {
			return s.reverse
		}
	}
	return false
}
func (s endpointSorter) Swap(i, j int) {
	s.items[i], s.items[j] = s.items[j], s.items[i]
}

// sortPolicies sorts the Policies based on the order requirements specified in the Page request.
func sortPolicies(items []Policy, s *Sort) error {
	ns := policySorter{items: items}
	if s != nil {
		for _, sb := range s.SortBy {
			ndf, ok := policyDiffFuncs[sb]
			if !ok {
				return fmt.Errorf("invalid sort field specified in query: %s", sb)
			}
			ns.diff = append(ns.diff, ndf)
		}
		ns.reverse = s.Reverse
	}
	ns.diff = append(ns.diff, policyDiffFuncs[policyDefaultSortField])

	// Sort the entries using the specified sort columns.
	sort.Sort(ns)
	return nil
}

type policyDiffFunc func(p, q *Policy) int

var policyDiffFuncs = map[string]policyDiffFunc{
	"index":                func(p, q *Policy) int { return p.Index - q.Index },
	"kind":                 func(p, q *Policy) int { return strings.Compare(p.Kind, q.Kind) },
	"name":                 func(p, q *Policy) int { return strings.Compare(p.Name, q.Name) },
	"namespace":            func(p, q *Policy) int { return strings.Compare(p.Namespace, q.Namespace) },
	"tier":                 func(p, q *Policy) int { return strings.Compare(p.Tier, q.Tier) },
	"numHostEndpoints":     func(p, q *Policy) int { return p.NumHostEndpoints - q.NumHostEndpoints },
	"numWorkloadEndpoints": func(p, q *Policy) int { return p.NumWorkloadEndpoints - q.NumWorkloadEndpoints },
	"numEndpoints": func(p, q *Policy) int {
		return p.NumWorkloadEndpoints + p.NumHostEndpoints - q.NumHostEndpoints - q.NumWorkloadEndpoints
	},
}

const policyDefaultSortField = "index"

type policySorter struct {
	items   []Policy
	diff    []policyDiffFunc
	reverse bool
}

func (s policySorter) Len() int {
	return len(s.items)
}
func (s policySorter) Less(i, j int) bool {
	p, q := &s.items[i], &s.items[j]
	for _, df := range s.diff {
		d := df(p, q)
		if d < 0 {
			return !s.reverse
		} else if d > 0 {
			return s.reverse
		}
	}
	return false
}
func (s policySorter) Swap(i, j int) {
	s.items[i], s.items[j] = s.items[j], s.items[i]
}
