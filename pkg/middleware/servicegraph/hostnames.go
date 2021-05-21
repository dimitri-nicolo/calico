// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	"context"
	"sync"

	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
)

type HostnameHelper interface {
	// Methods used to modify host name data in flows and events.  These methods may update the helper with additional
	// host names that were found in the logs and events but are no longer in the cluster.
	ProcessL3Flow(f *L3Flow)
	ProcessL7Flow(f *L7Flow)
	ProcessEvent(e *Event)

	// Return the set of host names associated with a host aggregated name.  This method does not update the helper.
	// It returns the final compiled set of hosts associated with the aggregated name.
	GetCompiledHostNamesFromAggregatedName(aggrName string) []string
}

type hostnameHelper struct {
	lock sync.RWMutex

	// If hosts are being aggregated into groups these will be non-nil.
	hostNameToAggrName  map[string]string
	aggrNameToHostnames map[string][]string

	// Host endpoint to host name lookup.
	hepToHostname map[string]string
}

func GetHostnameHelper(ctx context.Context, rd *RequestData) (HostnameHelper, error) {
	hh := &hostnameHelper{}

	wg := sync.WaitGroup{}

	var errHosts, errNodes error
	wg.Add(1)
	go func() {
		defer wg.Done()

		// If the user has specified a set of host aggregation selectors then query the Node resource and determine
		// which selectors match which nodes. The nodes (hosts) matching a selector will be put in a hosts buckets
		// with the name assigned to the selector.
		if len(rd.request.SelectedView.HostAggregationSelectors) > 0 {
			hh.hostNameToAggrName = make(map[string]string)
			hh.aggrNameToHostnames = make(map[string][]string)

			nodes, err := rd.appCluster.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
			if err != nil {
				errNodes = err
				return
			}

		next_node:
			for _, node := range nodes.Items {
				for _, selector := range rd.request.SelectedView.HostAggregationSelectors {
					if selector.Selector.Evaluate(node.Labels) {
						log.Debugf("Host to aggregated name mapping: %s -> %s", node.Name, selector.Name)
						hh.hostNameToAggrName[node.Name] = selector.Name
						hh.aggrNameToHostnames[selector.Name] = append(hh.aggrNameToHostnames[selector.Name], node.Name)
						continue next_node
					}
				}

				log.Debugf("Host to aggregated name mapping: %s -> *", node.Name)
				hh.hostNameToAggrName[node.Name] = "*"
				hh.aggrNameToHostnames["*"] = append(hh.aggrNameToHostnames["*"], node.Name)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		// Get the HostEndpoints to determine a HostEndpoint -> Host name mapping. We use this to correlate events
		// related to HostEndpoint resources with the host or hosts node types.
		hepToHostname := make(map[string]string)
		hostEndpoints, err := rd.appCluster.HostEndpoints().List(ctx, metav1.ListOptions{})
		if err != nil {
			errHosts = err
			return
		}
		for _, hep := range hostEndpoints.Items {
			log.Debugf("Hostendpoint to host name mapping: %s -> %s", hep.Name, hep.Spec.Node)
			hepToHostname[hep.Name] = hep.Spec.Node
		}
		hh.hepToHostname = hepToHostname
	}()

	if errNodes != nil {
		return nil, errNodes
	} else if errHosts != nil {
		return nil, errHosts
	}

	return hh, nil
}

// ProcessL3Flow updates an L3 flow to include additional aggregation details, and will also update the aggregation
// helper to track additional mappings that were not found during instantiation.
func (ah *hostnameHelper) ProcessL3Flow(f *L3Flow) {
	ah.lock.RLock()
	defer ah.lock.RUnlock()

	if ah.hostNameToAggrName == nil {
		return
	}

	// The aggregated name for hosts is actually the full name. Swap over, and apply the calculated aggregated name.
	if f.Edge.Source.Type == v1.GraphNodeTypeHost {
		f.Edge.Source.Name = f.Edge.Source.NameAggr
		if nameAggr := ah.hostNameToAggrName[f.Edge.Source.Name]; nameAggr != "" {
			f.Edge.Source.NameAggr = nameAggr
		} else {
			// The node name in the flow is not currently configured - include in the "*" bucket.
			f.Edge.Source.NameAggr = "*"
			ah.addAdditionalWildcardAggregatedNode(f.Edge.Source.Name)
		}
	}
	if f.Edge.Dest.Type == v1.GraphNodeTypeHost {
		f.Edge.Dest.Name = f.Edge.Dest.NameAggr
		if nameAggr := ah.hostNameToAggrName[f.Edge.Dest.Name]; nameAggr != "" {
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
func (ah *hostnameHelper) ProcessL7Flow(f *L7Flow) {
	ah.lock.RLock()
	defer ah.lock.RUnlock()

	if ah.hostNameToAggrName == nil {
		return
	}

	// The aggregated name for hosts is actually the full name. Swap over, and apply the calculated aggregated name.
	if f.Edge.Source.Type == v1.GraphNodeTypeHost {
		f.Edge.Source.Name = f.Edge.Source.NameAggr
		if nameAggr := ah.hostNameToAggrName[f.Edge.Source.Name]; nameAggr != "" {
			f.Edge.Source.NameAggr = nameAggr
		} else {
			// The node name in the flow is not currently configured - include in the "*" bucket.
			f.Edge.Source.NameAggr = "*"
			ah.addAdditionalWildcardAggregatedNode(f.Edge.Source.Name)
		}
	}
	if f.Edge.Dest.Type == v1.GraphNodeTypeHost {
		f.Edge.Dest.Name = f.Edge.Dest.NameAggr
		if nameAggr := ah.hostNameToAggrName[f.Edge.Dest.Name]; nameAggr != "" {
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
func (ah *hostnameHelper) ProcessEvent(e *Event) {
	ah.lock.RLock()
	defer ah.lock.RUnlock()

	if ah.hostNameToAggrName == nil {
		return
	}

	for i := range e.Endpoints {
		ep := &e.Endpoints[i]
		if ep.Type == v1.GraphNodeTypeHost {
			ep.Name = ep.NameAggr
			if nameAggr := ah.hostNameToAggrName[ep.NameAggr]; nameAggr != "" {
				ep.NameAggr = nameAggr
			} else {
				// The node in the event is not currently configured in the cluster - include in the "*" bucket.
				ep.NameAggr = "*"
				ah.addAdditionalWildcardAggregatedNode(e.Endpoints[i].Name)
			}
		} else if ep.Type == v1.GraphNodeTypeHostEndpoint {
			// We don't expose host endpoints - just hosts - so adjust the event endpoint and include the appropriate
			// aggregated name.
			ep.Type = v1.GraphNodeTypeHost
			if name, ok := ah.hepToHostname[ep.NameAggr]; ok {
				ep.Name = name
				if nameAggr := ah.hostNameToAggrName[ep.NameAggr]; nameAggr != "" {
					ep.NameAggr = nameAggr
				} else {
					// The node name in the event is not currently configured - include in the "*" bucket.
					ep.NameAggr = "*"
					ah.addAdditionalWildcardAggregatedNode(e.Endpoints[i].Name)
				}
			} else {
				// We have a HEP in the logs that we no longer know about. Just keep the name.
				ep.Name = ep.NameAggr
				ep.NameAggr = "*"
			}
		}
	}
}

// GetCompiledHostNamesFromAggregatedName returns the set of host names that correspond to the aggregated name.
// This returns nil if nodes are not aggregated into multiple groups.
func (ah *hostnameHelper) GetCompiledHostNamesFromAggregatedName(aggrName string) []string {
	if len(ah.aggrNameToHostnames) <= 1 {
		return nil
	}

	ah.lock.RLock()
	defer ah.lock.RUnlock()

	return ah.aggrNameToHostnames[aggrName]
}

// addAdditionalWildcardAggregatedNode includes an additional node in the "*" bucket.
// The caller should be holding the read-lock.
func (ah *hostnameHelper) addAdditionalWildcardAggregatedNode(name string) {
	ah.lock.RUnlock()
	ah.lock.Lock()
	ah.hostNameToAggrName[name] = "*"
	ah.aggrNameToHostnames["*"] = append(ah.aggrNameToHostnames["*"], name)
	ah.lock.Unlock()
	ah.lock.RLock()
}
