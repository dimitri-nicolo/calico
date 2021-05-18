// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	"context"
	"sync"

	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
	"github.com/tigera/es-proxy/pkg/middleware/k8s"
)

type AggregationHelper interface {
	ProcessL3Flow(f *L3Flow)
	ProcessL7Flow(f *L7Flow)
	ProcessEvent(e *Event)
	GetHostNamesFromAggregatedName(aggrName string) []string
}

type aggregationHelper struct {
	lock sync.RWMutex

	// If nodes are being aggregated into groups these will be non-nil.
	nodeNameToAggrName  map[string]string
	aggrNodeNameToNodes map[string][]string
}

func GetAggregationHelper(ctx context.Context, client k8s.ClientSet, sgv v1.GraphView) (AggregationHelper, error) {
	af := &aggregationHelper{}

	if len(sgv.HostAggregationSelectors) > 0 {
		af.nodeNameToAggrName = make(map[string]string)
		af.aggrNodeNameToNodes = make(map[string][]string)

		nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, err
		}

	next_node:
		for _, node := range nodes.Items {
			for aggrName, s := range sgv.HostAggregationSelectors {
				if s.Evaluate(node.Labels) {
					log.Debugf("Host to aggregated name mapping: %s -> %s", node.Name, aggrName)
					af.nodeNameToAggrName[node.Name] = aggrName
					af.aggrNodeNameToNodes[aggrName] = append(af.aggrNodeNameToNodes[aggrName], node.Name)
					continue next_node
				}
			}

			log.Debugf("Host to aggregated name mapping: %s -> *", node.Name)
			af.nodeNameToAggrName[node.Name] = "*"
			af.aggrNodeNameToNodes["*"] = append(af.aggrNodeNameToNodes["*"], node.Name)
		}
	}

	return af, nil
}

// ProcessL3Flow updates an L3 flow to include additional aggregation details, and will also update the aggregation
// helper to track additional mappings that were not found during instantiation.
func (ah *aggregationHelper) ProcessL3Flow(f *L3Flow) {
	ah.lock.RLock()
	defer ah.lock.RUnlock()

	if ah.nodeNameToAggrName == nil {
		return
	}

	if f.Edge.Source.Type == v1.GraphNodeTypeHostEndpoint {
		if nameAggr := ah.nodeNameToAggrName[f.Edge.Source.Name]; nameAggr != "" {
			f.Edge.Source.NameAggr = nameAggr
		} else {
			// The node name in the flow is not currently configured - include in the "*" bucket.
			f.Edge.Source.NameAggr = "*"
			ah.addAdditionalWildcardAggregatedNode(f.Edge.Source.Name)
		}
	}
	if f.Edge.Dest.Type == v1.GraphNodeTypeHostEndpoint {
		if nameAggr := ah.nodeNameToAggrName[f.Edge.Dest.Name]; nameAggr != "" {
			f.Edge.Dest.NameAggr = nameAggr
		} else {
			// The node name in the flow is not currently configured - include in the "*" bucket.
			f.Edge.Dest.NameAggr = "*"
			ah.addAdditionalWildcardAggregatedNode(f.Edge.Dest.Name)
		}
	}
}

// ProcessL7Flow updates an L7 flow to include additional aggregation details, and will also update the aggregation
// helper to track additional mappings that were not found during instantiation.
func (ah *aggregationHelper) ProcessL7Flow(f *L7Flow) {
	ah.lock.RLock()
	defer ah.lock.RUnlock()

	if ah.nodeNameToAggrName == nil {
		return
	}

	if f.Edge.Source.Type == v1.GraphNodeTypeHostEndpoint {
		if nameAggr := ah.nodeNameToAggrName[f.Edge.Source.Name]; nameAggr != "" {
			f.Edge.Source.NameAggr = nameAggr
		} else {
			// The node name in the flow is not currently configured - include in the "*" bucket.
			f.Edge.Source.NameAggr = "*"
			ah.addAdditionalWildcardAggregatedNode(f.Edge.Source.Name)
		}
	}
	if f.Edge.Dest.Type == v1.GraphNodeTypeHostEndpoint {
		if nameAggr := ah.nodeNameToAggrName[f.Edge.Dest.Name]; nameAggr != "" {
			f.Edge.Dest.NameAggr = nameAggr
		} else {
			// The node name in the flow is not currently configured - include in the "*" bucket.
			f.Edge.Dest.NameAggr = "*"
			ah.addAdditionalWildcardAggregatedNode(f.Edge.Dest.Name)
		}
	}
}

// ProcessEvent updates an event to include additional aggregation details, and will also update the aggregation
// helper to track additional mappings that were not found during instantiation.
func (ah *aggregationHelper) ProcessEvent(e *Event) {
	ah.lock.RLock()
	defer ah.lock.RUnlock()

	if ah.nodeNameToAggrName == nil {
		return
	}

	for i := range e.EventEndpoints {
		if e.EventEndpoints[i].Type == v1.GraphNodeTypeHostEndpoint {
			if nameAggr := ah.nodeNameToAggrName[e.EventEndpoints[i].Name]; nameAggr != "" {
				e.EventEndpoints[i].NameAggr = nameAggr
			} else {
				// The node name in the event is not currently configured - include in the "*" bucket.
				e.EventEndpoints[i].NameAggr = "*"
				ah.addAdditionalWildcardAggregatedNode(e.EventEndpoints[i].Name)
			}
		}
	}
}

// GetHostNamesFromAggregatedName returns the set of host names that correspond to the aggregated name.
// This returns nil if nodes are not aggregated into multiple groups.
func (ah *aggregationHelper) GetHostNamesFromAggregatedName(aggrName string) []string {
	if len(ah.aggrNodeNameToNodes) <= 1 {
		return nil
	}

	ah.lock.RLock()
	defer ah.lock.RUnlock()

	return ah.aggrNodeNameToNodes[aggrName]
}

// addAdditionalWildcardAggregatedNode includes an additional node in the "*" bucket.
// The caller should be holding the read-lock.
func (ah *aggregationHelper) addAdditionalWildcardAggregatedNode(name string) {
	ah.lock.RUnlock()
	ah.lock.Lock()
	ah.nodeNameToAggrName[name] = "*"
	ah.aggrNodeNameToNodes["*"] = append(ah.aggrNodeNameToNodes["*"], name)
	ah.lock.Unlock()
	ah.lock.RLock()
}
