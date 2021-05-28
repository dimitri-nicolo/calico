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
	"reflect"
	"sync"

	"github.com/projectcalico/felix/ip"

	"github.com/projectcalico/felix/dispatcher"
	"github.com/projectcalico/felix/labelindex"
	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	log "github.com/sirupsen/logrus"
	kapiv1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/proxy"
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

	suh              *ServiceUpdateHandler
	callbacks        PipelineCallbacks
	regularServices  map[ipPortProtoKey][]proxy.ServicePortName
	nodePortServices map[portProtoKey][]proxy.ServicePortName
}

func NewTproxyEndPointsResolver(callbacks PipelineCallbacks) *TproxyEndPointsResolver {
	tpr := &TproxyEndPointsResolver{
		mutex:     sync.RWMutex{},
		callbacks: callbacks,
		suh:       NewServiceUpdateHandler(),
	}
	return tpr
}

func (tpr *TproxyEndPointsResolver) RegisterWith(allUpdateDisp *dispatcher.Dispatcher) {
	allUpdateDisp.Register(model.ResourceKey{}, tpr.OnResourceUpdate)
}

// OnResourceUpdate is the callback method registered with the allUpdates dispatcher. We filter out everything except
// kubernetes services updates (for now). We can add other resources to tproxy in future here.
func (tpr *TproxyEndPointsResolver) OnResourceUpdate(update api.Update) (_ bool) {
	switch k := update.Key.(type) {
	case model.ResourceKey:
		switch k.Kind {
		case v3.KindK8sService:
			if update.Value == nil {
				tpr.suh.removeService(k)
				tpr.flush()
			} else {
				service := update.Value.(*kapiv1.Service)
				annotations := service.ObjectMeta.Annotations
				// only services annotated with l7 are of interest for us
				if hasAnnotation(annotations, l7LoggingAnnotation) {
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

	flushRegular := reflect.DeepEqual(tpr.regularServices, tpr.suh.ipPortProtoToServices)
	flushNodePorts := reflect.DeepEqual(tpr.nodePortServices, tpr.suh.nodePortServices)

	if flushRegular {
		tpr.flushRegularServices()
	}

	if flushNodePorts {
		tpr.flushNodePorts()
	}

}

func (tpr *TproxyEndPointsResolver) flushRegularServices() {
	// todo: felix maintains a diff of changes. We should use that instead if iterating over entire map
	// remove expired services from tproxy
	for key, _ := range tpr.regularServices {
		// skip deleting if member exists in updated list of annotated services
		if _, ok := tpr.suh.ipPortProtoToServices[key]; ok {
			continue
		}
		ipaddr := ip.V6Addr(key.ip)
		ipSetId := TproxyServicesIPSetV4
		if ipaddr.AsNetIP().To4() == nil {
			ipSetId = TproxyServicesIPSetV6
		}
		member := labelindex.IPSetMember{
			PortNumber: uint16(key.port),
			Protocol:   labelindex.ProtocolTCP,
			CIDR:       ipaddr.AsCIDR(),
		}
		// send ip set call back for expired members and delete from tproxy list
		tpr.callbacks.OnIPSetMemberRemoved(ipSetId, member)
		delete(tpr.regularServices, key)
	}

	// add new items to tproxy from updated list of annotated services
	for ipPortProto, value := range tpr.suh.ipPortProtoToServices {
		// if it already exists in resolved service skip it
		if existing, ok := tpr.regularServices[ipPortProto]; ok {
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

		ipaddr := ip.V6Addr(ipPortProto.ip)
		ipSetId := TproxyServicesIPSetV4
		if ipaddr.AsNetIP().To4() == nil {
			ipSetId = TproxyServicesIPSetV6
		}
		member := labelindex.IPSetMember{
			PortNumber: uint16(ipPortProto.port),
			Protocol:   labelindex.ProtocolTCP,
			CIDR:       ipaddr.AsCIDR(),
		}
		// send ip set call back for new members and add them to tproxy list
		tpr.callbacks.OnIPSetMemberAdded(ipSetId, member)
		tpr.regularServices[ipPortProto] = value
	}
}

func (tpr *TproxyEndPointsResolver) flushNodePorts() {
	// todo: felix maintains a diff of changes. We should use that instead if iterating over entire map
	// remove old items in updated list
	for key, _ := range tpr.nodePortServices {
		// skip deleting if member exists in updated list of annotated services
		if _, ok := tpr.suh.nodePortServices[key]; ok {
			continue
		}
		member := labelindex.IPSetMember{
			PortNumber: uint16(key.port),
			Protocol:   labelindex.ProtocolTCP,
		}
		// send ip set call back for expired members and delete from tproxy list
		tpr.callbacks.OnIPSetMemberRemoved(TproxyNodePortIpSetV4, member)
		delete(tpr.nodePortServices, key)
	}

	// add new items to tproxy from updated list of annotated services
	for ipPortProto, value := range tpr.suh.nodePortServices {
		// if it already exists in resolved service skip it
		if existing, ok := tpr.nodePortServices[ipPortProto]; ok {
			if reflect.DeepEqual(existing, value) {
				continue
			}
		}
		protocol := labelindex.IPSetPortProtocol(ipPortProto.proto)
		// if protocol is not TCP skip it for now
		if protocol != labelindex.ProtocolTCP {
			log.Debugf("Port/Protocol (%v/%d/%d) Protocol not valid for tproxy", protocol, ipPortProto.port)
			continue
		}
		member := labelindex.IPSetMember{
			PortNumber: uint16(ipPortProto.port),
			Protocol:   labelindex.ProtocolTCP,
		}
		// send ip set call back for new members and add them to tproxy list
		tpr.callbacks.OnIPSetMemberAdded(TproxyNodePortIpSetV4, member)
		tpr.nodePortServices[ipPortProto] = value
	}
}

func hasAnnotation(annotations map[string]string, annotation string) bool {
	if annotations != nil {
		if value, ok := annotations[annotation]; ok {
			return value == "true"
		}
	}

	return false
}
