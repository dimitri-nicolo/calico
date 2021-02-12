// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	"fmt"
	"sort"

	log "github.com/sirupsen/logrus"
	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/projectcalico/libcalico-go/lib/set"
)

// This file implements a service group organizer and cache. It effectively defines the concept of a service group
// which is a group of services that are related by a common set of endpoints. It is careful with HostEndpoints,
// NetworkSets, GlobalNetworkSets and networks - ensuring that when included in a service, they are unrelated to the
// same endpoint not in a service (i.e. A "pub" Network in service X will be a separate graph node from a "pub"
// Network in service Y, both will be a separate graph node from a "pub" Network not in a service).

// GetServiceGroupFlowEndpointKey returns an aggregated FlowEndpoint associated with the endpoint, protocol and port.
// This is the natural grouping of the endpoint for service groups.
func GetServiceGroupFlowEndpointKey(ep FlowEndpoint) *FlowEndpoint {
	switch ep.Type {
	case v1.GraphNodeTypeWorkload, v1.GraphNodeTypeReplicaSet:
		// All pods within a replica set are part of the same service group, so just use name_aggr for the key and
		// exclude the Port.
		return &FlowEndpoint{
			Type:      v1.GraphNodeTypeReplicaSet,
			Namespace: ep.Namespace,
			NameAggr:  ep.NameAggr,
		}
	case v1.GraphNodeTypeHostEndpoint, v1.GraphNodeTypeNetworkSet, v1.GraphNodeTypeNetwork:
		// For host Endpoints and network sets we also want to match the Port since a host endpoint or a network set
		// do not represent a single service. This allows us to connect inbound connections that go via the service
		// and that go directly. This will not assist in determining outbound connections from such service Endpoints.
		// TODO(rlb): for HEPs we can use the process name rather than the Port - this will allow us to determine
		// outbound connections for these Endpoints.
		return &FlowEndpoint{
			Type:      ep.Type,
			Namespace: ep.Namespace,
			NameAggr:  ep.NameAggr,
			Port:      ep.Port,
			Proto:     ep.Proto,
		}
	}
	return nil
}

type ServiceGroups interface {
	// Methods used to populate the service groups
	AddMapping(svc ServicePort, ep FlowEndpoint)
	FinishMappings()

	// Accessor methods used to lookup service groups.
	Iter(cb func(*ServiceGroup) error) error
	GetByService(svc types.NamespacedName) *ServiceGroup
	GetByEndpoint(ep FlowEndpoint) *ServiceGroup
}

type ServiceGroup struct {
	// The ID for this service group.
	ID string

	// The set of services in this group.
	Services []types.NamespacedName

	// The NameAggr and Namespace for this service group, and the set of underlying services. The name and/or namespace may
	// be set to "*" to indicate it has been aggregated from the set of underlying services.
	Namespace    string
	Name         string
	ServicePorts map[ServicePort]map[FlowEndpoint]struct{}
}

func (s ServiceGroup) String() string {
	return fmt.Sprintf("ServiceGroup(%s/%s)", s.Namespace, s.Name)
}

type serviceGroups struct {
	serviceGroups              set.Set
	serviceGroupsByServiceName map[types.NamespacedName]*ServiceGroup
	serviceGroupsByEndpointKey map[FlowEndpoint]*ServiceGroup
}

func (sgs *serviceGroups) Iter(cb func(*ServiceGroup) error) error {
	var err error
	sgs.serviceGroups.Iter(func(item interface{}) error {
		if err = cb(item.(*ServiceGroup)); err != nil {
			return set.StopIteration
		}
		return nil
	})
	return err
}

func (sgs *serviceGroups) GetByService(svc types.NamespacedName) *ServiceGroup {
	return sgs.serviceGroupsByServiceName[svc]
}

func (sgs *serviceGroups) GetByEndpoint(ep FlowEndpoint) *ServiceGroup {
	if key := GetServiceGroupFlowEndpointKey(ep); key != nil {
		return sgs.serviceGroupsByEndpointKey[*key]
	}
	return nil
}

func NewServiceGroups() ServiceGroups {
	// Create a ServiceGroups helper.
	sd := &serviceGroups{
		serviceGroups:              set.New(),
		serviceGroupsByServiceName: make(map[types.NamespacedName]*ServiceGroup),
		serviceGroupsByEndpointKey: make(map[FlowEndpoint]*ServiceGroup),
	}

	return sd
}

func (sd *serviceGroups) FinishMappings() {
	// Calculate the service groups name and namespace.
	sd.serviceGroups.Iter(func(item interface{}) error {
		sg := item.(*ServiceGroup)
		names := &nameCalculator{}
		namespaces := &nameCalculator{}
		for svcKey := range sg.ServicePorts {
			names.add(svcKey.Name)
			namespaces.add(svcKey.Namespace)
		}
		sg.Name = names.calc()
		sg.Namespace = namespaces.calc()

		return nil
	})

	// Trace out the service groups if the log level is debug.
	if log.IsLevelEnabled(log.DebugLevel) {
		log.Debug("=== Service groups ===")
		sd.serviceGroups.Iter(func(item interface{}) error {
			sg := item.(*ServiceGroup)
			log.Debugf("%s ->", sg)
			for sk, svc := range sg.ServicePorts {
				log.Debugf("  %s ->", sk)
				for ep := range svc {
					log.Debugf("    o %s", ep)
				}
			}
			return nil
		})
		log.Debug("=== Endpoint key to service group ===")
		for ep, sg := range sd.serviceGroupsByEndpointKey {
			log.Debugf("%s -> %s", ep, sg)
		}
		log.Debug("=== Service name to service group ===")
		for svc, sg := range sd.serviceGroupsByServiceName {
			log.Debugf("%s -> %s", svc, sg)
		}
	}

	// Update the set of services in each service group. It is easiest to do this at the end when each service has
	// the correct service group assigned to it.
	for sn, sg := range sd.serviceGroupsByServiceName {
		sg.Services = append(sg.Services, sn)
	}

	// Update the ID for each group, and simplify the groups to use the replica set instead of the workload if the
	// port is common across replicas.
	f := IDInfo{}
	sd.serviceGroups.Iter(func(item interface{}) error {
		sg := item.(*ServiceGroup)

		// Sort the services for easier testing.
		sort.Sort(sortableServices(sg.Services))

		// Construct the id using the IDInfo.
		f.Services = sg.Services
		sg.ID = f.GetServiceGroupID()

		// Update the service group to not include the full name if the port/proto is fixed across all endpoints in the
		// replica set for a given service port.
		aggrs := make(map[FlowEndpoint]map[ServicePort][]FlowEndpoint)
		for sp, eps := range sg.ServicePorts {
			for ep := range eps {
				ae := FlowEndpoint{
					Type:      ConvertEndpointTypeToAggrEndpointType(ep.Type),
					Namespace: ep.Namespace,
					NameAggr:  ep.NameAggr,
					Proto:     ep.Proto,
					Port:      ep.Port,
				}
				m := aggrs[ae]
				if m == nil {
					m = make(map[ServicePort][]FlowEndpoint)
					aggrs[ae] = m
				}
				m[sp] = append(m[sp], ep)
			}
		}
		for aep, sps := range aggrs {
			if len(sps) > 1 {
				// There are multiple service ports associated with this aggregated endpoint port, so do not aggregate
				// the data in the service group.
				continue
			}
			// There is only one entry, but only way to get it is to iterate.
			for sp, eps := range sps {
				for _, ep := range eps {
					delete(sg.ServicePorts[sp], ep)
				}

				// Replace with a single aggregated-endpoint-name/port/proto. Use the aggregated form of the type
				// when storing in the service group.
				sg.ServicePorts[sp][aep] = struct{}{}
			}
		}

		return nil
	})
}

func (s *serviceGroups) AddMapping(svc ServicePort, ep FlowEndpoint) {
	// If there is an existing service group either by service or endpoint then apply updates to that service
	// group, otherwise create a new service group.
	var sg, sge *ServiceGroup

	// Get the existing service groups associated with the endpoint and the service.
	epKey := GetServiceGroupFlowEndpointKey(ep)
	if epKey != nil {
		sge = s.serviceGroupsByEndpointKey[*epKey]
	}
	sgs := s.serviceGroupsByServiceName[svc.NamespacedName]

	if sge != nil && sgs != nil {
		// There is an entry by service and endpoint. If they are the same ServiceGroup then nothing to do, if they are
		// different then combine the two ServiceGroups.
		if sge != sgs {
			// The ServiceGroup referenced by service is different to the one referenced by the endpoint. Migrate the
			// references from the service SG to the endpoint SG. Copy across the data - since the endpoint SG will not
			// already have the service, it's possible to copy across the endpoint map in full.
			log.Debugf("Merging ServiceGroup for %s into ServiceGroup for %s", ep, svc)
			s.migrateReferences(sgs, sge)
		}
		sg = sge
	} else if sge != nil {
		// No entry by service, but there is by endpoint - use that.
		log.Debugf("Including %s into ServiceGroup for %s", svc, ep)
		sg = sge
	} else if sgs != nil {
		// No entry by endpoint, but there is by service - use that.
		log.Debugf("Including %s into ServiceGroup for %s", ep, svc)
		sg = sgs
	} else {
		// No existing entry by endpoint or service, so create a new service group.
		log.Debugf("Creating new ServiceGroup containing %s and %s", svc, ep)
		sg = &ServiceGroup{
			ServicePorts: make(map[ServicePort]map[FlowEndpoint]struct{}),
		}
		s.serviceGroups.Add(sg)
	}

	// Set references.
	s.serviceGroupsByServiceName[svc.NamespacedName] = sg
	if epKey != nil {
		s.serviceGroupsByEndpointKey[*epKey] = sg
	}

	// Update service group data to include the endpoint.
	if sg.ServicePorts[svc] == nil {
		sg.ServicePorts[svc] = map[FlowEndpoint]struct{}{
			ep: {},
		}
	} else {
		sg.ServicePorts[svc][ep] = struct{}{}
	}
}

func (s *serviceGroups) migrateReferences(from, to *ServiceGroup) {
	// Update the mappings.
	for svc, eps := range from.ServicePorts {
		s.serviceGroupsByServiceName[svc.NamespacedName] = to

		for ep := range eps {
			if epKey := GetServiceGroupFlowEndpointKey(ep); epKey != nil {
				s.serviceGroupsByEndpointKey[*epKey] = to
			}
		}

		// Copy across the service ports.
		to.ServicePorts[svc] = eps
	}

	// Remote the old grouop.
	s.serviceGroups.Discard(from)
}

type sortableServices []types.NamespacedName

func (s sortableServices) Len() int {
	return len(s)
}
func (s sortableServices) Less(i, j int) bool {
	if s[i].Namespace < s[j].Namespace {
		return true
	} else if s[i].Namespace == s[j].Namespace && s[i].Name < s[j].Name {
		return true
	}
	return false
}
func (s sortableServices) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// nameCalculator is used to track names underpinning a group of resources, and to create an aggregated name from the
// set of names.
// At the moment this simply returns a "*" if there are multiple distinct names, but in future we could look for
// common name segment prefixes.
type nameCalculator struct {
	name string
}

func (nc *nameCalculator) add(name string) {
	if nc.name == "" {
		nc.name = name
	} else if nc.name != name {
		nc.name = "*"
	}
}

func (nc *nameCalculator) calc() string {
	return nc.name
}
