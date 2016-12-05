// Copyright (c) 2016 Tigera, Inc. All rights reserved.
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

package collector

import (
	"net"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/projectcalico/felix/go/felix/proto"
)

type lookupManager struct {
	endpoints map[string]*proto.WorkloadEndpointID
	mutex     sync.Mutex
}

func newLookupManager() *lookupManager {
	return &lookupManager{
		endpoints: map[string]*proto.WorkloadEndpointID{},
		mutex:     sync.Mutex{},
	}
}

func (m *lookupManager) OnUpdate(protoBufMsg interface{}) {
	switch msg := protoBufMsg.(type) {
	case *proto.WorkloadEndpointUpdate:
		m.mutex.Lock()
		// TODO (Matt): IPv6; also IP.String() does funny things to IPv6 mapped IPv4 addresses.
		for _, ipv4 := range msg.Endpoint.Ipv4Nets {
			addr, _, err := net.ParseCIDR(ipv4)
			if err != nil {
				log.Warn("Error parsing CIDR ", ipv4)
			}
			m.endpoints[addr.String()] = msg.Id
		}
		m.mutex.Unlock()
	case *proto.WorkloadEndpointRemove:
		m.mutex.Lock()
		// TODO (Matt): IPv6
		// TODO (Matt): Wow this is slow; need to maintain a reverse map...
		// Actually let's fix it later; the remove message doesn't include the IPs.
		//delete(m.endpoints, addr.String())
		m.mutex.Unlock()
	case *proto.HostEndpointUpdate:
		// TODO(smc) Host endpoint updates
		log.WithField("msg", msg).Warn("Message not implemented")
	case *proto.HostEndpointRemove:
		// TODO(smc) Host endpoint updates
		log.WithField("msg", msg).Warn("Message not implemented")
	}
}

func (m *lookupManager) CompleteDeferredWork() error {
	return nil
}

// TODO (Matt): Review return types.
func (m *lookupManager) GetEndpointID(addr net.IP) *proto.WorkloadEndpointID {
	m.mutex.Lock()
	epID := m.endpoints[addr.String()]
	m.mutex.Unlock()
	return epID
}
