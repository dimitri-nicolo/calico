// Copyright (c) 2016-2021 Tigera, Inc. All rights reserved.

package calc

import (
	"net"
	"reflect"

	"github.com/projectcalico/calico/felix/proto"

	"github.com/projectcalico/calico/felix/config"
	"github.com/projectcalico/calico/felix/ip"

	log "github.com/sirupsen/logrus"
	kapiv1 "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1"

	"github.com/projectcalico/calico/felix/dispatcher"
	"github.com/projectcalico/calico/felix/labelindex"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/calico/libcalico-go/lib/set"
)

const (
	l7LoggingAnnotation     = "projectcalico.org/l7-logging"
	TPROXYServiceIPsIPSet   = "tproxy-svc-ips"
	TPROXYNodePortsTCPIPSet = "tproxy-nodeports-tcp"
)

// L7ServiceIPSetsCalculator maintains most up-to-date list of all L7 logging
// enabled frontends.
//
// To do this, the L7ServiceIPSetsCalculator hooks into the calculation graph.
// by handling callbacks for updated service information.
//
// Currently it handles "service" resource that have a predefined annotation for enabling logging.
// It emits ipSetUpdateCallbacks when a state change happens, keeping data plane in-sync with datastore.
type L7ServiceIPSetsCalculator struct {
	createdIpSet    bool
	conf            *config.Config
	suh             *ServiceUpdateHandler
	esai            *EndpointSliceAddrIndexer
	callbacks       ipSetUpdateCallbacks
	activeEndpoints set.Set[ipPortProtoKey]
	activeNodePorts set.Set[portProtoKey]
}

func NewL7ServiceIPSetsCalculator(callbacks ipSetUpdateCallbacks, conf *config.Config) *L7ServiceIPSetsCalculator {
	tpr := &L7ServiceIPSetsCalculator{
		conf:            conf,
		createdIpSet:    false,
		callbacks:       callbacks,
		suh:             NewServiceUpdateHandler(),
		esai:            NewEndpointSliceAddrIndexer(),
		activeEndpoints: set.New[ipPortProtoKey](),
		activeNodePorts: set.New[portProtoKey](),
	}
	tpr.callbacks.OnIPSetAdded(TPROXYServiceIPsIPSet, proto.IPSetUpdate_IP_AND_PORT)
	tpr.callbacks.OnIPSetAdded(TPROXYNodePortsTCPIPSet, proto.IPSetUpdate_PORTS)

	return tpr
}

func (tpr *L7ServiceIPSetsCalculator) RegisterWith(allUpdateDisp *dispatcher.Dispatcher) {
	log.Debugf("registering with all update dispatcher for tproxy service updates")
	allUpdateDisp.Register(model.ResourceKey{}, tpr.OnResourceUpdate)
}

func (tpr *L7ServiceIPSetsCalculator) isEndpointSliceFromAnnotatedService(
	k model.ResourceKey, v *discovery.EndpointSlice,
) bool {
	serviceKey := model.ResourceKey{
		Namespace: k.Namespace,
		Kind:      model.KindKubernetesService,
	}
	if v == nil {
		edp, ok := tpr.esai.endpointSlices[k]
		if !ok {
			return false
		}
		v = edp
	}
	serviceKey.Name = v.ObjectMeta.Labels["kubernetes.io/service-name"]
	_, ok := tpr.suh.services[serviceKey]
	return ok
}

// OnResourceUpdate is the callback method registered with the allUpdates dispatcher. We filter out everything except
// kubernetes services updates (for now). We can add other resources to L7 in future here.
func (tpr *L7ServiceIPSetsCalculator) OnResourceUpdate(update api.Update) (_ bool) {
	switch k := update.Key.(type) {
	case model.ResourceKey:
		switch k.Kind {
		case model.KindKubernetesService:
			log.Debugf("processing update for service %s", k)
			if update.Value == nil {
				if _, ok := tpr.suh.services[k]; ok {
					tpr.suh.RemoveService(k)
					tpr.flush()
				}
			} else {
				service := update.Value.(*kapiv1.Service)
				annotations := service.ObjectMeta.Annotations
				// process services annotated with l7 or all service when in EnabledAllServices mode
				if hasAnnotation(annotations, l7LoggingAnnotation) || tpr.conf.TPROXYModeEnabledAllServices() {
					log.Infof("processing update for tproxy annotated service %s", k)
					tpr.suh.AddOrUpdateService(k, service)
					tpr.flush()
				} else {
					// case when service is present in services and no longer has annotation
					if _, ok := tpr.suh.services[k]; ok {
						log.Infof("removing unannotated service from ipset %s", k)
						tpr.suh.RemoveService(k)
						tpr.flush()
					}
				}
			}
		case model.KindKubernetesEndpointSlice:
			log.Debugf("processing update for endpointslice %s", k)
			needFlush := false
			if update.Value == nil {
				if tpr.isEndpointSliceFromAnnotatedService(k, nil) {
					needFlush = true
				}
				tpr.esai.RemoveEndpointSlice(k)
			} else {
				endpointSlice := update.Value.(*discovery.EndpointSlice)
				tpr.esai.AddOrUpdateEndpointSlice(k, endpointSlice)
				if tpr.isEndpointSliceFromAnnotatedService(k, endpointSlice) {
					needFlush = true
				}
			}

			if needFlush {
				tpr.flush()
			}
		default:
			log.Debugf("Ignoring update for resource: %s", k)
		}
	default:
		log.Errorf("Ignoring unexpected update: %v %#v",
			reflect.TypeOf(update.Key), update)
	}
	return
}

// flush emits ipSetUpdateCallbacks (OnIPSetAdded, OnIPSetMemberAdded, OnIPSetMemberRemoved) when endpoints
// for tproxy traffic selection changes. It detects the change in state by comparing most up to date
// members list maintained by UpdateHandlers to the list maintained by
// L7ServiceIPSetsCalculator.
func (tpr *L7ServiceIPSetsCalculator) flush() {

	addedSvs, removedSvs := tpr.resolveRegularEndpoints()

	if len(addedSvs) > 0 || len(removedSvs) > 0 {
		tpr.flushRegularEndpoints(addedSvs, removedSvs)
	}

	addedNPs, removedNPs := tpr.resolveNodePorts()

	if len(addedNPs) > 0 || len(removedNPs) > 0 {
		tpr.flushNodePorts(addedNPs, removedNPs)
	}
}

func isTCP(ipPortProto ipPortProtoKey) bool {
	protocol := labelindex.IPSetPortProtocol(ipPortProto.proto)
	if protocol != labelindex.ProtocolTCP {
		log.Warningf("IP/Port/Protocol (%v/%d/%d) Protocol not valid for l7 logging",
			ipPortProto.ip, protocol, ipPortProto.port)
		return false
	}

	return true
}

func (tpr *L7ServiceIPSetsCalculator) resolveRegularEndpoints() ([]ipPortProtoKey,
	[]ipPortProtoKey) {
	// todo: felix maintains a diff of changes. We should use that instead if iterating over entire map
	log.Infof("flush regular services for tproxy")

	var added, removed []ipPortProtoKey

	// Get all ipPortProtos from endpointSlice update handler
	allSvcKeys := make([]model.ResourceKey, len(tpr.suh.services))
	for k := range tpr.suh.services {
		allSvcKeys = append(allSvcKeys, k)
	}
	esaiIPPortProtos := tpr.esai.IPPortProtosByService(allSvcKeys...)

	tpr.activeEndpoints.Iter(func(ipPortProto ipPortProtoKey) error {
		// if member key exists in up-to-date list, else add to removed
		_, ok := tpr.suh.ipPortProtoToServices[ipPortProto]
		if ok {
			return nil
		}

		if esaiIPPortProtos.Contains(ipPortProto) {
			return nil
		}
		removed = append(removed, ipPortProto)
		return nil
	})

	// add new items to tproxy from updated list of annotated endpoints
	for ipPortProto, _ := range tpr.suh.ipPortProtoToServices {
		// if it already exists in active endpoints skip it
		if tpr.activeEndpoints.Contains(ipPortProto) {
			continue
		}
		// if protocol is not TCP skip it for now
		if !isTCP(ipPortProto) {
			continue
		}
		added = append(added, ipPortProto)
	}

	esaiIPPortProtos.Iter(func(ipPortProto ipPortProtoKey) error {
		// if it already exists in active endpoints skip it
		if tpr.activeEndpoints.Contains(ipPortProto) {
			return nil
		}
		// if protocol is not TCP skip it for now
		if !isTCP(ipPortProto) {
			return nil
		}
		added = append(added, ipPortProto)
		return nil
	})

	return added, removed
}

func (tpr *L7ServiceIPSetsCalculator) flushRegularEndpoints(added []ipPortProtoKey,
	removed []ipPortProtoKey) {

	for _, ipPortProto := range removed {
		member := getIpSetMemberFromIpPortProto(ipPortProto)
		tpr.callbacks.OnIPSetMemberRemoved(TPROXYServiceIPsIPSet, member)
		tpr.activeEndpoints.Discard(ipPortProto)
	}

	for _, ipPortProto := range added {
		member := getIpSetMemberFromIpPortProto(ipPortProto)
		tpr.callbacks.OnIPSetMemberAdded(TPROXYServiceIPsIPSet, member)
		tpr.activeEndpoints.Add(ipPortProto)
	}

}

func (tpr *L7ServiceIPSetsCalculator) resolveNodePorts() ([]portProtoKey,
	[]portProtoKey) {
	// todo: felix maintains a diff of changes. We should use that instead if iterating over entire map
	log.Infof("flush node ports for tproxy")

	var added, removed []portProtoKey

	tpr.activeNodePorts.Iter(func(portProto portProtoKey) error {
		// if member key exists in up-to-date list, update the value to latest in active node ports and continue to next
		if _, ok := tpr.suh.nodePortServices[portProto]; ok {
			return nil
		}
		removed = append(removed, portProto)
		return nil
	})

	for portProto := range tpr.suh.nodePortServices {
		// if it already exists in active node ports skip it
		if tpr.activeNodePorts.Contains(portProto) {
			continue
		}
		protocol := labelindex.IPSetPortProtocol(portProto.proto)
		// skip non tcp for now
		if protocol != labelindex.ProtocolTCP {
			log.Warningf("Port/Protocol (%d/%d) Protocol not valid for tproxy", portProto.port, protocol)
			continue
		}
		// if node port is zero skip callbacks
		if portProto.port == 0 {
			continue
		}
		added = append(added, portProto)
		log.Debugf("Added Port/Protocol (%d/%d).", portProto.port, protocol)
	}
	return added, removed
}

func (tpr *L7ServiceIPSetsCalculator) flushNodePorts(added []portProtoKey,
	removed []portProtoKey) {

	for _, portProto := range removed {
		if labelindex.IPSetPortProtocol(portProto.proto) == labelindex.ProtocolTCP {
			member := getIpSetPortMemberFromPortProto(portProto)
			member.Family = 4
			tpr.callbacks.OnIPSetMemberRemoved(TPROXYNodePortsTCPIPSet, member)
			member.Family = 6
			tpr.callbacks.OnIPSetMemberRemoved(TPROXYNodePortsTCPIPSet, member)
			tpr.activeNodePorts.Discard(portProto)
		}
	}

	for _, portProto := range added {
		if labelindex.IPSetPortProtocol(portProto.proto) == labelindex.ProtocolTCP {
			member := getIpSetPortMemberFromPortProto(portProto)
			member.Family = 4
			tpr.callbacks.OnIPSetMemberAdded(TPROXYNodePortsTCPIPSet, member)
			member.Family = 6
			tpr.callbacks.OnIPSetMemberAdded(TPROXYNodePortsTCPIPSet, member)
			tpr.activeNodePorts.Add(portProto)
		}
	}

}

func getIpSetMemberFromIpPortProto(ipPortProto ipPortProtoKey) labelindex.IPSetMember {
	netIP := net.IP(ipPortProto.ip[:])
	member := labelindex.IPSetMember{
		PortNumber: uint16(ipPortProto.port),
		Protocol:   labelindex.IPSetPortProtocol(ipPortProto.proto),
		CIDR:       ip.FromNetIP(netIP).AsCIDR(),
	}

	return member
}

func getIpSetPortMemberFromPortProto(portProto portProtoKey) labelindex.IPSetMember {
	member := labelindex.IPSetMember{
		PortNumber: uint16(portProto.port),
	}

	return member
}

func hasAnnotation(annotations map[string]string, annotation string) bool {
	if annotations != nil {
		if value, ok := annotations[annotation]; ok {
			return value == "true"
		}
	}

	return false
}
