// +build !windows

// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package intdataplane

import (
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/ip"
	"github.com/projectcalico/felix/ipsets"
	"github.com/projectcalico/felix/logutils"
	"github.com/projectcalico/felix/netlinkshim"
	"github.com/projectcalico/felix/proto"
	"github.com/projectcalico/felix/routerule"
	"github.com/projectcalico/felix/routetable"
	"github.com/projectcalico/libcalico-go/lib/set"
)

type tproxyManager struct {
	enabled, enabled6 bool
	mark              uint32
	rt4, rt6          *routetable.RouteTable
	rr4, rr6          *routerule.RouteRules

	iptablesEqualIPsChecker *iptablesEqualIPsChecker
}

const (
	tproxyPodToSelfIPSet = "tproxy-pod-self"
)

type tproxyIPSets interface {
	AddOrReplaceIPSet(meta ipsets.IPSetMetadata, members []string)
	AddMembers(setID string, newMembers []string)
	RemoveMembers(setID string, removedMembers []string)
}

type TProxyOption func(*tproxyManager)

func newTProxyManager(
	dpConfig Config,
	idx4, idx6 int,
	opRecorder logutils.OpRecorder,
	opts ...TProxyOption,
) *tproxyManager {

	ipv6Enabled := dpConfig.IPv6Enabled
	mark := dpConfig.RulesConfig.IptablesMarkProxy

	enabled := dpConfig.RulesConfig.TPROXYModeEnabled()

	tpm := &tproxyManager{
		enabled:  enabled,
		enabled6: ipv6Enabled,
		mark:     mark,
	}

	for _, opt := range opts {
		opt(tpm)
	}

	if idx4 == 0 {
		log.Fatal("RouteTable index for IPv4 is the default table")
	}

	rt := routetable.New(
		nil,
		4,
		false, // vxlan
		dpConfig.NetlinkTimeout,
		nil, // deviceRouteSourceAddress
		dpConfig.DeviceRouteProtocol,
		true, // removeExternalRoutes
		idx4,
		opRecorder,
	)

	rr, err := routerule.New(
		4,
		1, // routing priority
		set.From(idx4),
		routerule.RulesMatchSrcFWMarkTable,
		routerule.RulesMatchSrcFWMarkTable,
		dpConfig.NetlinkTimeout,
		func() (routerule.HandleIface, error) {
			return netlinkshim.NewRealNetlink()
		},
		opRecorder,
	)
	if err != nil {
		log.WithError(err).Panic("Unexpected error creating rule manager")
	}

	if enabled {
		anyV4, _ := ip.CIDRFromString("0.0.0.0/0")
		rt.RouteUpdate("lo", routetable.Target{
			Type: routetable.TargetTypeLocal,
			CIDR: anyV4,
		})

		rr.SetRule(routerule.NewRule(4, 1).
			GoToTable(idx4).
			MatchFWMarkWithMask(uint32(mark), uint32(mark)),
		)
	}

	tpm.rr4 = rr
	tpm.rt4 = rt

	if ipv6Enabled {
		if idx6 == 0 {
			log.Fatal("RouteTable index for IPv6 is the default table")
		}

		rt := routetable.New(
			nil,
			6,
			false, // vxlan
			dpConfig.NetlinkTimeout,
			nil, // deviceRouteSourceAddress
			dpConfig.DeviceRouteProtocol,
			true, // removeExternalRoutes
			idx6,
			opRecorder,
		)

		rr, err := routerule.New(
			6,
			1, // routing priority
			set.From(idx6),
			routerule.RulesMatchSrcFWMarkTable,
			routerule.RulesMatchSrcFWMarkTable,
			dpConfig.NetlinkTimeout,
			func() (routerule.HandleIface, error) {
				return netlinkshim.NewRealNetlink()
			},
			opRecorder,
		)
		if err != nil {
			log.WithError(err).Panic("Unexpected error creating rule manager")
		}

		if enabled {
			anyV6, _ := ip.CIDRFromString("::/0")
			rt.RouteUpdate("lo", routetable.Target{
				Type: routetable.TargetTypeLocal,
				CIDR: anyV6,
			})

			rr.SetRule(routerule.NewRule(6, 1).
				GoToTable(idx6).
				MatchFWMarkWithMask(uint32(mark), uint32(mark)),
			)
		}

		tpm.rr6 = rr
		tpm.rt6 = rt
	}

	return tpm
}

func tproxyWithIptablesEqualIPsChecker(checker *iptablesEqualIPsChecker) TProxyOption {
	return func(m *tproxyManager) {
		m.iptablesEqualIPsChecker = checker
	}
}

func (m *tproxyManager) OnUpdate(protoBufMsg interface{}) {
	if !m.enabled {
		return
	}

	if m.iptablesEqualIPsChecker != nil {
		// We get EP updates only for the endpoints local to the node.
		switch msg := protoBufMsg.(type) {
		case *proto.WorkloadEndpointUpdate:
			m.iptablesEqualIPsChecker.OnWorkloadEndpointUpdate(msg)
		case *proto.WorkloadEndpointRemove:
			m.iptablesEqualIPsChecker.OnWorkloadEndpointRemove(msg)
		}
	}
}

func (m *tproxyManager) CompleteDeferredWork() error {
	if m.enabled && m.iptablesEqualIPsChecker != nil {
		return m.iptablesEqualIPsChecker.CompleteDeferredWork()
	}

	return nil
}

func (m *tproxyManager) GetRouteTableSyncers() []routeTableSyncer {
	var rts []routeTableSyncer

	if m.rt4 != nil {
		rts = append(rts, m.rt4)
	}

	if m.rt6 != nil {
		rts = append(rts, m.rt6)
	}

	return rts
}

func (m *tproxyManager) GetRouteRules() []routeRules {
	var rrs []routeRules

	if m.rr4 != nil {
		rrs = append(rrs, m.rr4)
	}

	if m.rr6 != nil {
		rrs = append(rrs, m.rr6)
	}

	return rrs
}

type iptablesEqualIPsChecker struct {
	enabled6 bool

	ipSetsV4, ipSetsV6 tproxyIPSets

	v4Eps map[proto.WorkloadEndpointID][]string
	v6Eps map[proto.WorkloadEndpointID][]string

	ipv4 map[string]int
	ipv6 map[string]int

	ipv4ToAdd map[string]struct{}
	ipv4ToDel map[string]struct{}
	ipv6ToAdd map[string]struct{}
	ipv6ToDel map[string]struct{}
}

func newIptablesEqualIPsChecker(dpConfig Config, ipSetsV4, ipSetsV6 tproxyIPSets) *iptablesEqualIPsChecker {
	enabled6 := dpConfig.IPv6Enabled
	enabled := dpConfig.RulesConfig.TPROXYModeEnabled()

	if enabled && ipSetsV4 == nil {
		log.Fatal("no IPv4 ipsets")
	}
	if enabled && enabled6 && ipSetsV6 == nil {
		log.Fatal("no IPv6 ipsets when IPv6 enabled")
	}

	if enabled {
		ipSetsV4.AddOrReplaceIPSet(
			ipsets.IPSetMetadata{
				SetID:   tproxyPodToSelfIPSet,
				Type:    ipsets.IPSetTypeHashNetNet,
				MaxSize: dpConfig.MaxIPSetSize,
			},
			[]string{},
		)

		if enabled6 {
			ipSetsV6.AddOrReplaceIPSet(
				ipsets.IPSetMetadata{
					SetID:   tproxyPodToSelfIPSet,
					Type:    ipsets.IPSetTypeHashNetNet,
					MaxSize: dpConfig.MaxIPSetSize,
				},
				[]string{},
			)
		}
	}

	return &iptablesEqualIPsChecker{
		enabled6: enabled6,

		ipSetsV4: ipSetsV4,
		ipSetsV6: ipSetsV6,

		v4Eps: make(map[proto.WorkloadEndpointID][]string),
		v6Eps: make(map[proto.WorkloadEndpointID][]string),

		ipv4: make(map[string]int),
		ipv6: make(map[string]int),

		ipv4ToAdd: make(map[string]struct{}),
		ipv4ToDel: make(map[string]struct{}),
		ipv6ToAdd: make(map[string]struct{}),
		ipv6ToDel: make(map[string]struct{}),
	}
}

func includesNet(n string, set []string) bool {
	for _, s := range set {
		if s == n {
			return true
		}
	}
	return false
}

func diffNets(now, before []string) (add, del []string) {
	for _, n := range now {
		if !includesNet(n, before) {
			add = append(add, n)
		}
	}

	if len(now) == len(add) {
		return
	}

	for _, b := range before {
		if !includesNet(b, now) {
			del = append(del, b)
		}
	}

	return
}

func onWorkloadEndpointUpdate(
	id *proto.WorkloadEndpointID,
	nets []string,
	eps map[proto.WorkloadEndpointID][]string,
	refs map[string]int,
	toAdd, toDel map[string]struct{},
) {
	add, del := diffNets(nets, eps[*id])

	eps[*id] = nets

	for _, ip := range add {
		refs[ip]++
		if refs[ip] == 1 {
			toAdd[ip] = struct{}{}
		}
		delete(toDel, ip)
	}
	for _, ip := range del {
		refs[ip]--
		if refs[ip] <= 0 {
			delete(refs, ip)
			toDel[ip] = struct{}{}
			delete(toAdd, ip)
		}
	}
}

func (c *iptablesEqualIPsChecker) OnWorkloadEndpointUpdate(msg *proto.WorkloadEndpointUpdate) {
	onWorkloadEndpointUpdate(msg.Id, msg.Endpoint.Ipv4Nets, c.v4Eps, c.ipv4, c.ipv4ToAdd, c.ipv4ToDel)

	if c.enabled6 {
		onWorkloadEndpointUpdate(msg.Id, msg.Endpoint.Ipv6Nets, c.v6Eps, c.ipv6, c.ipv6ToAdd, c.ipv6ToDel)
	}
}

func onWorkloadEndpointRemove(
	id *proto.WorkloadEndpointID,
	eps map[proto.WorkloadEndpointID][]string,
	refs map[string]int,
	toAdd, toDel map[string]struct{},
) {
	for _, ip := range eps[*id] {
		refs[ip]--
		if refs[ip] <= 0 {
			delete(refs, ip)
			toDel[ip] = struct{}{}
			delete(toAdd, ip)
		}
	}
	delete(eps, *id)
}

func (c *iptablesEqualIPsChecker) OnWorkloadEndpointRemove(msg *proto.WorkloadEndpointRemove) {
	onWorkloadEndpointRemove(msg.Id, c.v4Eps, c.ipv4, c.ipv4ToAdd, c.ipv4ToDel)

	if c.enabled6 {
		onWorkloadEndpointRemove(msg.Id, c.v6Eps, c.ipv6, c.ipv6ToAdd, c.ipv6ToDel)
	}
}

func (c *iptablesEqualIPsChecker) CompleteDeferredWork() error {
	var add []string
	for ipv4 := range c.ipv4ToAdd {
		add = append(add, ipv4+","+ipv4)
	}
	c.ipSetsV4.AddMembers(tproxyPodToSelfIPSet, add)

	if c.enabled6 {
		var add6 []string
		for ipv6 := range c.ipv6ToAdd {
			add6 = append(add6, ipv6+","+ipv6)
		}
		c.ipSetsV6.AddMembers(tproxyPodToSelfIPSet, add6)
	}

	var del []string
	for ipv4 := range c.ipv4ToDel {
		del = append(del, ipv4+","+ipv4)
	}
	c.ipSetsV4.RemoveMembers(tproxyPodToSelfIPSet, del)

	if c.enabled6 {
		var del6 []string
		for ipv6 := range c.ipv6ToDel {
			del6 = append(del6, ipv6+","+ipv6)
		}
		c.ipSetsV6.RemoveMembers(tproxyPodToSelfIPSet, del6)
	}

	c.ipv4ToAdd = make(map[string]struct{})
	c.ipv4ToDel = make(map[string]struct{})
	c.ipv6ToAdd = make(map[string]struct{})
	c.ipv6ToDel = make(map[string]struct{})

	return nil
}
