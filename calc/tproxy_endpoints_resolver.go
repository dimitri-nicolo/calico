// Copyright (c) 2016-2021 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package calc

import (
	"net"
	"reflect"
	"sync"

	"github.com/projectcalico/felix/proto"

	"github.com/projectcalico/felix/ip"

	log "github.com/sirupsen/logrus"
	kapiv1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/proxy"

	"github.com/projectcalico/felix/dispatcher"
	"github.com/projectcalico/felix/labelindex"
	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

const l7LoggingAnnotation = "projectcalico.org/l7-logging"
const TPROXYServicesIPSet = "tproxy-services"
const TRPOXYNodePortsIPSet = "tproxy-nodeports"

// TproxyEndPointsResolver maintains most up-to-date list of all tproxy enabled endpoints.
//
// To do this, the TproxyEndPointsResolver hooks into the calculation graph
// by handling callbacks for updated local endpoint tier information.
//
// Currently it handles "service" resource that have a predefined annotation for enabling tproxy.
// It emits ipSetUpdateCallbacks when a state change happens, keeping data plane in-sync with datastore.
type TproxyEndPointsResolver struct {
	mutex sync.RWMutex

	createIpSet     bool
	suh             *ServiceUpdateHandler
	callbacks       ipSetUpdateCallbacks
	activeServices  map[ipPortProtoKey][]proxy.ServicePortName
	activeNodePorts map[portProtoKey][]proxy.ServicePortName
}

func NewTproxyEndPointsResolver(callbacks ipSetUpdateCallbacks) *TproxyEndPointsResolver {
	tpr := &TproxyEndPointsResolver{
		mutex:           sync.RWMutex{},
		createIpSet:     false,
		callbacks:       callbacks,
		suh:             NewServiceUpdateHandler(),
		activeServices:  make(map[ipPortProtoKey][]proxy.ServicePortName),
		activeNodePorts: make(map[portProtoKey][]proxy.ServicePortName),
	}
	return tpr
}

func (tpr *TproxyEndPointsResolver) RegisterWith(allUpdateDisp *dispatcher.Dispatcher) {
	log.Debugf("registering with all update dispatcher for tproxy service updates")
	allUpdateDisp.Register(model.ResourceKey{}, tpr.OnResourceUpdate)
}

// OnResourceUpdate is the callback method registered with the allUpdates dispatcher. We filter out everything except
// kubernetes services updates (for now). We can add other resources to tproxy in future here.
func (tpr *TproxyEndPointsResolver) OnResourceUpdate(update api.Update) (_ bool) {
	switch k := update.Key.(type) {
	case model.ResourceKey:
		switch k.Kind {
		case v3.KindK8sService:
			log.Debugf("processing update for service %s", k)
			if update.Value == nil {
				if _, ok := tpr.suh.services[k]; ok {
					tpr.suh.removeService(k)
					tpr.flush()
				}
			} else {
				service := update.Value.(*kapiv1.Service)
				annotations := service.ObjectMeta.Annotations
				// only services annotated with l7 are of interest for us
				if hasAnnotation(annotations, l7LoggingAnnotation) {
					log.Infof("processing update for tproxy annotated service %s", k)
					tpr.suh.addOrUpdateService(k, service)
					tpr.flush()
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

// flush emits ipSetUpdateCallbacks (OnIPSetMemberAdded, OnIPSetMemberRemoved) when endpoints for tproxy traffic
// selection changes. It detects the change in state by comparing most up to data members list maintained by
// ServiceUpdateHandler to the list maintained by TproxyEndPointsResolver.
// It's also responsible for setting the appropriate IpSetID when emitting callbacks.
func (tpr *TproxyEndPointsResolver) flush() {
	tpr.mutex.Lock()
	defer tpr.mutex.Unlock()

	addedSvs, removedSvs := tpr.resolveRegularServices()

	if len(addedSvs) > 0 || len(removedSvs) > 0 {
		tpr.flushRegularService(addedSvs, removedSvs)
	}

	// TODO: current release doesn't intend to support nodeports
	//addedNPs, removedNps := tpr.resolveNodePorts()
	//if len(addedNPs) > 0 || len(removedNps) > 0 {
	//	tpr.flushNodePorts(addedNPs, removedNps)
	//}
}

func (tpr *TproxyEndPointsResolver) resolveRegularServices() (map[ipPortProtoKey][]proxy.ServicePortName,
	map[ipPortProtoKey]struct{}) {
	// todo: felix maintains a diff of changes. We should use that instead if iterating over entire map
	log.Debugf("flush regular services for tproxy")

	added := make(map[ipPortProtoKey][]proxy.ServicePortName)
	removed := make(map[ipPortProtoKey]struct{})

	for ipPortProto := range tpr.activeServices {
		// if member key exists in up-to-date list, update the value to latest in active service and continue to next
		if latest, ok := tpr.suh.ipPortProtoToServices[ipPortProto]; ok {
			tpr.activeServices[ipPortProto] = latest
			continue
		}
		removed[ipPortProto] = struct{}{}
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

func (tpr *TproxyEndPointsResolver) flushRegularService(added map[ipPortProtoKey][]proxy.ServicePortName,
	removed map[ipPortProtoKey]struct{}) {

	// before sending member callbacks create ipset if doesn't exists
	//if len(tpr.activeServices) == 0 {
	//	tpr.callbacks.OnIPSetAdded(TPROXYServicesIPSet, proto.IPSetUpdate_IP_AND_PORT)
	//}

	if !tpr.createIpSet {
		tpr.callbacks.OnIPSetAdded(TPROXYServicesIPSet, proto.IPSetUpdate_IP_AND_PORT)
		tpr.createIpSet = true
	}

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

	// after sending member callbacks delete ipset if no active services present
	// TODO: This can't be done since the deletion of ipset fails with message
	// Set cannot be destroyed: it is in use by a kernel component
	//if len(tpr.activeServices) == 0 {
	//	tpr.callbacks.OnIPSetRemoved(TPROXYServicesIPSet)
	//}

}

func (tpr *TproxyEndPointsResolver) resolveNodePorts() (map[portProtoKey][]proxy.ServicePortName,
	map[portProtoKey]struct{}) {
	// todo: felix maintains a diff of changes. We should use that instead if iterating over entire map
	log.Debugf("flush node ports for tproxy")

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
	}
	return added, removed
}

func (tpr *TproxyEndPointsResolver) flushNodePorts(added map[portProtoKey][]proxy.ServicePortName,
	removed map[portProtoKey]struct{}) {

	// before sending member callbacks create ipset if doesn't exists
	//if len(tpr.activeNodePorts) == 0 {
	//	tpr.callbacks.OnIPSetAdded(TRPOXYNodePortsIPSet, proto.IPSetUpdate_IP_AND_PORT)
	//}

	for portProto := range removed {
		member := getIpSetMemberFromPortProto(portProto)
		tpr.callbacks.OnIPSetMemberRemoved(TRPOXYNodePortsIPSet, member)
		delete(tpr.activeNodePorts, portProto)
	}

	for portProto, value := range added {
		member := getIpSetMemberFromPortProto(portProto)
		tpr.callbacks.OnIPSetMemberAdded(TRPOXYNodePortsIPSet, member)
		tpr.activeNodePorts[portProto] = value
	}

	// after sending member callbacks create ipset if doesn't exists
	//if len(tpr.activeNodePorts) == 0 {
	//	tpr.callbacks.OnIPSetRemoved(TRPOXYNodePortsIPSet)
	//}
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

func getIpSetMemberFromPortProto(portProto portProtoKey) labelindex.IPSetMember {
	member := labelindex.IPSetMember{
		PortNumber: uint16(portProto.port),
		Protocol:   labelindex.IPSetPortProtocol(portProto.proto),
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
