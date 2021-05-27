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

	"github.com/projectcalico/felix/dispatcher"
	"github.com/projectcalico/felix/ip"
	"github.com/projectcalico/felix/labelindex"
	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	log "github.com/sirupsen/logrus"
	kapiv1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/proxy"
)

const l7LoggingAnnotation = "projectcalico.org/l7-logging"
const TproxyServicesIPSetV4 = "tproxy-services"
const TproxyServicesIPSetV6 = "tproxy-services"
const TproxyNodePortIpSetV4 = "tproxy-nodeports"
const TproxyNodePortIpSetV6 = "tproxy-nodeports"

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
	activeServices   map[ipPortProtoKey][]proxy.ServicePortName
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
				// only services annotated with l7 are of interest for us
				if containsL7Annotation(service) {
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

	sendActive := reflect.DeepEqual(tpr.activeServices, tpr.suh.ipPortProtoToServices)
	sendNodePorts := reflect.DeepEqual(tpr.nodePortServices, tpr.suh.nodePortServices)

	// nothing to do if handler is insync with tproxy resolver
	if !sendActive && !sendNodePorts {
		return
	}

	if sendActive {
		// remove old items in updated list
		for key, _ := range tpr.activeServices {
			// if the member already exists in active service skip over it
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
				Protocol:   labelindex.ProtocolTCP, // for now its TCP
				CIDR:       ipaddr.AsCIDR(),
			}
			// send call back for expired member and delete from active
			tpr.callbacks.OnIPSetMemberRemoved(ipSetId, member)
			delete(tpr.activeServices, key)
		}

		// check for new items in updated list
		for key, value := range tpr.suh.ipPortProtoToServices {
			// if the member already exists in active service skip over it
			if existing, ok := tpr.activeServices[key]; ok {
				if reflect.DeepEqual(existing, value) {
					continue
				}
			}
			ipaddr := ip.V6Addr(key.ip)
			ipSetId := TproxyServicesIPSetV4
			if ipaddr.AsNetIP().To4() == nil {
				ipSetId = TproxyServicesIPSetV6
			}
			// have to check if the IP is v6 or v4
			member := labelindex.IPSetMember{
				PortNumber: uint16(key.port),
				Protocol:   labelindex.ProtocolTCP, // for now its TCP
				CIDR:       ipaddr.AsCIDR(),
			}
			// send callback for new member and add it to active
			tpr.callbacks.OnIPSetMemberAdded(ipSetId, member)
			tpr.activeServices[key] = value
		}
	}

}

func containsL7Annotation(svc *kapiv1.Service) bool {
	annotations := svc.ObjectMeta.Annotations
	if svc.ObjectMeta.Annotations != nil {
		if value, ok := annotations[l7LoggingAnnotation]; ok {
			return value == "true"
		}
	}

	return false
}
