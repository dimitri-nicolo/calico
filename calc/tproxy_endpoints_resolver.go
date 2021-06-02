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
const TproxyServicesIPSetV4 = "cali40tproxy-services"
const TproxyServicesIPSetV6 = "cali60tproxy-services"
const TproxyNodePortIpSetV4 = "cali40tproxy-nodeports"

// TproxyEndPointsResolver maintains most up-to-date list of all tproxy enabled endpoints.
//
// To do this, the TproxyEndPointsResolver hooks into the calculation graph
// by handling callbacks for updated local endpoint tier information.
//
// Currently it handles "service" resource that have a predefined annotation for enabling tproxy.
// It emits ipSetUpdateCallbacks when a state change happens, keeping data plane in-sync with datastore.
type TproxyEndPointsResolver struct {
	mutex sync.RWMutex

	suh             *ServiceUpdateHandler
	callbacks       ipSetUpdateCallbacks
	activeServices  map[ipPortProtoKey][]proxy.ServicePortName
	activeNodePorts map[portProtoKey][]proxy.ServicePortName
}

func NewTproxyEndPointsResolver(callbacks ipSetUpdateCallbacks) *TproxyEndPointsResolver {
	tpr := &TproxyEndPointsResolver{
		mutex:           sync.RWMutex{},
		callbacks:       callbacks,
		suh:             NewServiceUpdateHandler(),
		activeServices:  make(map[ipPortProtoKey][]proxy.ServicePortName),
		activeNodePorts: make(map[portProtoKey][]proxy.ServicePortName),
	}
	return tpr
}

func (tpr *TproxyEndPointsResolver) RegisterWith(allUpdateDisp *dispatcher.Dispatcher) {
	log.Infof("registering with all update dispatcher for tproxy service updates")
	allUpdateDisp.Register(model.ResourceKey{}, tpr.OnResourceUpdate)
}

// OnResourceUpdate is the callback method registered with the allUpdates dispatcher. We filter out everything except
// kubernetes services updates (for now). We can add other resources to tproxy in future here.
func (tpr *TproxyEndPointsResolver) OnResourceUpdate(update api.Update) (_ bool) {
	log.Infof("OnResourceUpdate", update)
	switch k := update.Key.(type) {
	case model.ResourceKey:
		switch k.Kind {
		case v3.KindK8sService:
			log.Infof("processing update for service %s", k)
			if update.Value == nil {
				tpr.suh.removeService(k)
				tpr.flush()
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

	serviceUptoDate := reflect.DeepEqual(tpr.activeServices, tpr.suh.ipPortProtoToServices)
	nodePortsUptoDate := reflect.DeepEqual(tpr.activeNodePorts, tpr.suh.nodePortServices)

	if !serviceUptoDate {
		tpr.flushRegularServices()
	}

	if !nodePortsUptoDate {
		tpr.flushNodePorts()
	}

}

func (tpr *TproxyEndPointsResolver) flushRegularServices() {
	// todo: felix maintains a diff of changes. We should use that instead if iterating over entire map
	// remove expired services from active services
	log.Debugf("flush regular services for tproxy")

	for ipPortProto, _ := range tpr.activeServices {
		// if member key exists in up-to-date list, update the value to latest in active service and continue to next
		if latest, ok := tpr.suh.ipPortProtoToServices[ipPortProto]; ok {
			tpr.activeServices[ipPortProto] = latest
			continue
		}

		ipSetId, member := getIpSetMemberFromIpPortProto(ipPortProto)
		// send ip set call back to remove them from tproxy ipset
		tpr.callbacks.OnIPSetMemberRemoved(ipSetId, member)
		delete(tpr.activeServices, ipPortProto)
	}

	// add new items to tproxy from updated list of annotated services
	for ipPortProto, value := range tpr.suh.ipPortProtoToServices {
		// if it already exists in active services skip it
		if existing, ok := tpr.activeServices[ipPortProto]; ok {
			if reflect.DeepEqual(existing, value) {
				continue
			}
		}
		protocol := labelindex.IPSetPortProtocol(ipPortProto.proto)
		// if protocol is not TCP skip it for now
		if protocol != labelindex.ProtocolTCP {
			log.Debugf("IP/Port/Protocol (%v/%d/%d) Protocol not valid for tproxy",
				ipPortProto.ip, protocol, ipPortProto.port)
			continue
		}
		ipSetId, member := getIpSetMemberFromIpPortProto(ipPortProto)
		// send ip set call back for new members and add them to tproxy ipset
		tpr.callbacks.OnIPSetMemberAdded(ipSetId, member)
		tpr.activeServices[ipPortProto] = value
	}
}

func (tpr *TproxyEndPointsResolver) flushNodePorts() {
	// todo: felix maintains a diff of changes. We should use that instead if iterating over entire map
	// remove expired node ports from active
	log.Debugf("flush node ports for tproxy")

	for portProto, _ := range tpr.activeNodePorts {
		// if member key exists in up-to-date list, update the value to latest in active node ports and continue to next
		if latest, ok := tpr.suh.nodePortServices[portProto]; ok {
			tpr.activeNodePorts[portProto] = latest
			continue
		}
		ipSetId, member := getIpSetMemberFromPortProto(portProto)
		// send ip set call back for expired members and delete from tproxy list
		tpr.callbacks.OnIPSetMemberRemoved(ipSetId, member)
		delete(tpr.activeNodePorts, portProto)
	}

	// add new items to active node ports
	for portProto, value := range tpr.suh.nodePortServices {
		// if it already exists in active node ports skip it
		if existing, ok := tpr.activeNodePorts[portProto]; ok {
			if reflect.DeepEqual(existing, value) {
				continue
			}
		}
		protocol := labelindex.IPSetPortProtocol(portProto.proto)
		// if protocol is not TCP skip it for now
		if protocol != labelindex.ProtocolTCP {
			log.Debugf("Port/Protocol (%d/%d) Protocol not valid for tproxy", portProto.port, protocol)
			continue
		}
		ipSetId, member := getIpSetMemberFromPortProto(portProto)
		// send ip set call back for new members and add them to tproxy list
		tpr.callbacks.OnIPSetMemberAdded(ipSetId, member)
		tpr.activeNodePorts[portProto] = value
	}
}

func getIpSetMemberFromIpPortProto(ipPortProto ipPortProtoKey) (string, labelindex.IPSetMember) {
	netIP := net.IP(ipPortProto.ip[:])
	member := labelindex.IPSetMember{
		PortNumber: uint16(ipPortProto.port),
		Protocol:   labelindex.IPSetPortProtocol(ipPortProto.proto),
		CIDR:       ip.FromNetIP(netIP).AsCIDR(),
	}

	ipSetId := TproxyServicesIPSetV4
	if member.CIDR.Version() == 6 {
		ipSetId = TproxyServicesIPSetV6
	}

	return ipSetId, member
}

func getIpSetMemberFromPortProto(portProto portProtoKey) (string, labelindex.IPSetMember) {
	member := labelindex.IPSetMember{
		PortNumber: uint16(portProto.port),
		Protocol:   labelindex.IPSetPortProtocol(portProto.proto),
	}

	return TproxyNodePortIpSetV4, member
}
func hasAnnotation(annotations map[string]string, annotation string) bool {
	if annotations != nil {
		if value, ok := annotations[annotation]; ok {
			return value == "true"
		}
	}

	return false
}
