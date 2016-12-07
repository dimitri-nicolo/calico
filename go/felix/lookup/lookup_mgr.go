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

package lookup

import (
	"net"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/projectcalico/felix/go/felix/proto"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

// TODO (Matt): WorkloadEndpoints are only local; so we can't get nice information for the remote ends.
type LookupManager struct {
	endpoints map[string]*proto.WorkloadEndpointID
	mutex     sync.Mutex
}

func NewLookupManager() *LookupManager {
	return &LookupManager{
		endpoints: map[string]*proto.WorkloadEndpointID{},
		mutex:     sync.Mutex{},
	}
}

func (m *LookupManager) OnUpdate(protoBufMsg interface{}) {
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
		// Need reverse map; the remove message doesn't include the IPs.
		//delete(m.endpoints, addr.String())
		m.mutex.Unlock()
	case *proto.HostEndpointUpdate:
		// TODO(Matt) Host endpoint updates
		log.WithField("msg", msg).Warn("Message not implemented")
	case *proto.HostEndpointRemove:
		// TODO(Matt) Host endpoint updates
		log.WithField("msg", msg).Warn("Message not implemented")
	}
}

func (m *LookupManager) CompleteDeferredWork() error {
	return nil
}

// TODO (Matt): Review return types.  Convert the proto.s to model.s when we get them.
func (m *LookupManager) GetEndpointID(addr net.IP) *proto.WorkloadEndpointID {
	m.mutex.Lock()
	epID := m.endpoints[addr.String()]
	m.mutex.Unlock()
	return epID
}

func (m *LookupManager) GetPolicyIndex(epKey *model.WorkloadEndpointKey, policyKey *model.PolicyKey) int {
	// TODO (Matt): Need to handle profiles as well as tiered policy.
	ti := ep.TierInfo
	if profile return fold(sum, len(Policies) in TierInfos in ep.Tiers)
	else walk tiers until policyKey.Tier, summing len(Policies) then + index(policy in tier)
	return 3
}
