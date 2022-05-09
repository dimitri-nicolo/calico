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
	"k8s.io/kubernetes/pkg/proxy"

	"github.com/projectcalico/calico/felix/dispatcher"
	"github.com/projectcalico/calico/felix/labelindex"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
)

const l7LoggingAnnotation = "projectcalico.org/l7-logging"
const TPROXYServicesIPSet = "tproxy-services"
const TPROXYNodePortsTCPIPSet = "tproxy-nodeports-tcp"

// L7FrontEndResolver maintains most up-to-date list of all L7 logging enabled frontends.
//
// To do this, the L7FrontEndResolver hooks into the calculation graph
// by handling callbacks for updated service information.
//
// Currently it handles "service" resource that have a predefined annotation for enabling logging.
// It emits ipSetUpdateCallbacks when a state change happens, keeping data plane in-sync with datastore.
type L7FrontEndResolver struct {
	createdIpSet    bool
	conf            *config.Config
	suh             *ServiceUpdateHandler
	callbacks       ipSetUpdateCallbacks
	activeServices  map[ipPortProtoKey][]proxy.ServicePortName
	activeNodePorts map[portProtoKey][]proxy.ServicePortName
}

func NewL7FrontEndResolver(callbacks ipSetUpdateCallbacks, conf *config.Config) *L7FrontEndResolver {
	tpr := &L7FrontEndResolver{
		conf:            conf,
		createdIpSet:    false,
		callbacks:       callbacks,
		suh:             NewServiceUpdateHandler(),
		activeServices:  make(map[ipPortProtoKey][]proxy.ServicePortName),
		activeNodePorts: make(map[portProtoKey][]proxy.ServicePortName),
	}
	tpr.callbacks.OnIPSetAdded(TPROXYServicesIPSet, proto.IPSetUpdate_IP_AND_PORT)
	tpr.callbacks.OnIPSetAdded(TPROXYNodePortsTCPIPSet, proto.IPSetUpdate_PORTS)

	return tpr
}

func (tpr *L7FrontEndResolver) RegisterWith(allUpdateDisp *dispatcher.Dispatcher) {
	log.Debugf("registering with all update dispatcher for tproxy service updates")
	allUpdateDisp.Register(model.ResourceKey{}, tpr.OnResourceUpdate)
}

// OnResourceUpdate is the callback method registered with the allUpdates dispatcher. We filter out everything except
// kubernetes services updates (for now). We can add other resources to L7 in future here.
func (tpr *L7FrontEndResolver) OnResourceUpdate(update api.Update) (_ bool) {
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
// members list maintained by ServiceUpdateHandler to the list maintained by L7FrontEndResolver.
func (tpr *L7FrontEndResolver) flush() {

	addedSvs, removedSvs := tpr.resolveRegularServices()

	if len(addedSvs) > 0 || len(removedSvs) > 0 {
		tpr.flushRegularService(addedSvs, removedSvs)
	}

	addedNPs, removedNPs := tpr.resolveNodePorts()

	if len(addedNPs) > 0 || len(removedNPs) > 0 {
		tpr.flushNodePorts(addedNPs, removedNPs)
	}
}

func (tpr *L7FrontEndResolver) resolveRegularServices() (map[ipPortProtoKey][]proxy.ServicePortName,
	map[ipPortProtoKey]struct{}) {
	// todo: felix maintains a diff of changes. We should use that instead if iterating over entire map
	log.Infof("flush regular services for tproxy")

	added := make(map[ipPortProtoKey][]proxy.ServicePortName)
	removed := make(map[ipPortProtoKey]struct{})

	for ipPortProto := range tpr.activeServices {
		// if member key exists in up-to-date list, else add to removed
		if latest, ok := tpr.suh.ipPortProtoToServices[ipPortProto]; ok {
			tpr.activeServices[ipPortProto] = latest
		} else {
			removed[ipPortProto] = struct{}{}
		}
	}
	// add new items to tproxy from updated list of annotated services
	for ipPortProto, value := range tpr.suh.ipPortProtoToServices {
		// if it already exists in active services skip it
		if _, ok := tpr.activeServices[ipPortProto]; ok {
			continue
		}
		// if protocol is not TCP skip it for now
		protocol := labelindex.IPSetPortProtocol(ipPortProto.proto)
		if protocol != labelindex.ProtocolTCP {
			log.Infof("IP/Port/Protocol (%v/%d/%d) Protocol not valid for l7 logging",
				ipPortProto.ip, protocol, ipPortProto.port)
			continue
		}
		added[ipPortProto] = value
	}

	return added, removed
}

func (tpr *L7FrontEndResolver) flushRegularService(added map[ipPortProtoKey][]proxy.ServicePortName,
	removed map[ipPortProtoKey]struct{}) {

	for ipPortProto := range removed {
		member := getIpSetMemberFromIpPortProto(ipPortProto)
		tpr.callbacks.OnIPSetMemberRemoved(TPROXYServicesIPSet, member)
		delete(tpr.activeServices, ipPortProto)
	}

	for ipPortProto, value := range added {
		member := getIpSetMemberFromIpPortProto(ipPortProto)
		tpr.callbacks.OnIPSetMemberAdded(TPROXYServicesIPSet, member)
		tpr.activeServices[ipPortProto] = value
	}

}

func (tpr *L7FrontEndResolver) resolveNodePorts() (map[portProtoKey][]proxy.ServicePortName,
	map[portProtoKey]struct{}) {
	// todo: felix maintains a diff of changes. We should use that instead if iterating over entire map
	log.Infof("flush node ports for tproxy")

	added := make(map[portProtoKey][]proxy.ServicePortName)
	removed := make(map[portProtoKey]struct{})

	for portProto := range tpr.activeNodePorts {
		// if member key exists in up-to-date list, update the value to latest in active node ports and continue to next
		if latest, ok := tpr.suh.nodePortServices[portProto]; ok {
			tpr.activeNodePorts[portProto] = latest
			continue
		}
		removed[portProto] = struct{}{}
	}

	for portProto, value := range tpr.suh.nodePortServices {
		// if it already exists in active node ports skip it
		if _, ok := tpr.activeNodePorts[portProto]; ok {
			continue
		}
		protocol := labelindex.IPSetPortProtocol(portProto.proto)
		// skip non tcp for now
		if protocol != labelindex.ProtocolTCP {
			log.Infof("Port/Protocol (%d/%d) Protocol not valid for tproxy", portProto.port, protocol)
			continue
		}
		// if node port is zero skip callbacks
		if portProto.port == 0 {
			continue
		}
		added[portProto] = value
		log.Debugf("Added Port/Protocol (%d/%d).", portProto.port, protocol)
	}
	return added, removed
}

func (tpr *L7FrontEndResolver) flushNodePorts(added map[portProtoKey][]proxy.ServicePortName,
	removed map[portProtoKey]struct{}) {

	for portProto := range removed {
		if labelindex.IPSetPortProtocol(portProto.proto) == labelindex.ProtocolTCP {
			member := getIpSetPortMemberFromPortProto(portProto)
			member.Family = 4
			tpr.callbacks.OnIPSetMemberRemoved(TPROXYNodePortsTCPIPSet, member)
			member.Family = 6
			tpr.callbacks.OnIPSetMemberRemoved(TPROXYNodePortsTCPIPSet, member)
			delete(tpr.activeNodePorts, portProto)
		}
	}

	for portProto, value := range added {
		if labelindex.IPSetPortProtocol(portProto.proto) == labelindex.ProtocolTCP {
			member := getIpSetPortMemberFromPortProto(portProto)
			member.Family = 4
			tpr.callbacks.OnIPSetMemberAdded(TPROXYNodePortsTCPIPSet, member)
			member.Family = 6
			tpr.callbacks.OnIPSetMemberAdded(TPROXYNodePortsTCPIPSet, member)
			tpr.activeNodePorts[portProto] = value
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
