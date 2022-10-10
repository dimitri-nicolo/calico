// Copyright (c) 2018-2021 Tigera, Inc. All rights reserved.
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

package intdataplane

import (
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/felix/dataplane/common"
	"github.com/projectcalico/calico/felix/ip"
	"github.com/projectcalico/calico/felix/ipsets"
	"github.com/projectcalico/calico/felix/proto"
	"github.com/projectcalico/calico/libcalico-go/lib/set"
)

// hostIPManager monitors updates from ifacemonitor for host ip update events. It then flushes host ips into an ipset.
type hostIPManager struct {
	nonHostIfacesRegexp *regexp.Regexp
	// hostIfaceToAddrs maps host interface name to the set of IPs on that interface (reported from the dataplane).
	hostIfaceToAddrs map[string]set.Set[string]
	routesByDest     set.Set[ip.CIDR]

	hostIPSetID     string
	ipsetsDataplane common.IPSetsDataplane
	maxSize         int
	tunIPSetID      string
	routesDirty     bool
}

func newHostIPManager(wlIfacesPrefixes []string,
	ipSetID string,
	ipsets common.IPSetsDataplane,
	maxIPSetSize int, tunIpSetID string) *hostIPManager {

	return newHostIPManagerWithShims(
		wlIfacesPrefixes,
		ipSetID,
		ipsets,
		maxIPSetSize,
		tunIpSetID,
	)
}

func newHostIPManagerWithShims(wlIfacesPrefixes []string,
	ipSetID string,
	ipsets common.IPSetsDataplane,
	maxIPSetSize int, tunIpSetID string) *hostIPManager {

	wlIfacesPattern := "^(" + strings.Join(wlIfacesPrefixes, "|") + ").*"
	wlIfacesRegexp := regexp.MustCompile(wlIfacesPattern)

	return &hostIPManager{
		nonHostIfacesRegexp: wlIfacesRegexp,
		hostIfaceToAddrs:    map[string]set.Set[string]{},
		hostIPSetID:         ipSetID,
		ipsetsDataplane:     ipsets,
		maxSize:             maxIPSetSize,
		tunIPSetID:          tunIpSetID,
		routesByDest:        set.NewBoxed[ip.CIDR](),
		routesDirty:         true,
	}
}

func (m *hostIPManager) getCurrentMembers() []string {
	members := []string{}
	for _, addrs := range m.hostIfaceToAddrs {
		addrs.Iter(func(ip string) error {
			members = append(members, ip)
			return nil
		})
	}

	return members
}

func (m *hostIPManager) OnUpdate(msg interface{}) {
	switch msg := msg.(type) {
	case *ifaceAddrsUpdate:
		log.WithField("update", msg).Info("Interface addrs changed.")
		if m.nonHostIfacesRegexp.MatchString(msg.Name) {
			log.WithField("update", msg).Debug("Not a real host interface, ignoring.")
			return
		}
		if msg.Addrs != nil {
			m.hostIfaceToAddrs[msg.Name] = msg.Addrs
		} else {
			delete(m.hostIfaceToAddrs, msg.Name)
		}

		// Host ip update is a relative rare event. Flush entire ipsets to make it simple.
		metadata := ipsets.IPSetMetadata{
			Type:    ipsets.IPSetTypeHashIP,
			SetID:   m.hostIPSetID,
			MaxSize: m.maxSize,
		}
		m.ipsetsDataplane.AddOrReplaceIPSet(metadata, m.getCurrentMembers())
	case *proto.RouteUpdate:
		m.onRouteUpdate(msg)
	case *proto.RouteRemove:
		m.onRouteRemove(msg)
	}
}

func (m *hostIPManager) onRouteUpdate(update *proto.RouteUpdate) {
	cidr, err := ip.CIDRFromString(update.Dst)
	if err != nil {
		log.Warn("Unable to parse route update destination. Skipping update.")
		return
	}

	m.deleteRoute(cidr)

	if update.Type == proto.RouteType_REMOTE_TUNNEL {
		m.routesByDest.Add(cidr)
		m.routesDirty = true
	}
}

func (m *hostIPManager) onRouteRemove(msg *proto.RouteRemove) {
	cidr, err := ip.CIDRFromString(msg.Dst)
	if err != nil {
		log.Warn("Unable to parse route remove destination. Skipping update.")
		return
	}
	m.deleteRoute(cidr)
}

func (m *hostIPManager) deleteRoute(cidr ip.CIDR) {
	if m.routesByDest.Contains(cidr) {
		m.routesByDest.Discard(cidr)
		m.routesDirty = true
	}
}

func (m *hostIPManager) getCurrentTunMembers() []string {
	members := []string{}
	m.routesByDest.Iter(func(cidr ip.CIDR) error {
		members = append(members, cidr.Addr().String())
		return nil
	})
	return members
}

func (m *hostIPManager) CompleteDeferredWork() error {
	if m.routesDirty {
		metadata := ipsets.IPSetMetadata{
			Type:    ipsets.IPSetTypeHashIP,
			SetID:   m.tunIPSetID,
			MaxSize: m.maxSize,
		}
		m.ipsetsDataplane.AddOrReplaceIPSet(metadata, m.getCurrentTunMembers())
		m.routesDirty = false
	}
	return nil
}
