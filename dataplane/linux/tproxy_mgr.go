// +build !windows

// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package intdataplane

import (
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/ip"
	"github.com/projectcalico/felix/ipsets"
	"github.com/projectcalico/felix/logutils"
	"github.com/projectcalico/felix/netlinkshim"
	"github.com/projectcalico/felix/routerule"
	"github.com/projectcalico/felix/routetable"
	"github.com/projectcalico/libcalico-go/lib/set"
)

type tproxyManager struct {
	mark     uint32
	ipSetsV4 *ipsets.IPSets
	ipSetsV6 *ipsets.IPSets
	rt4, rt6 *routetable.RouteTable
	rr4, rr6 *routerule.RouteRules
}

func newTproxyManager(
	mark uint32,
	maxsize int,
	idx4, idx6 int,
	ipSetsV4, ipSetsV6 *ipsets.IPSets,
	dpConfig Config,
	opRecorder logutils.OpRecorder,
) *tproxyManager {
	tpm := &tproxyManager{
		mark: mark,
	}
	if ipSetsV4 != nil {
		ipSetsV4.AddOrReplaceIPSet(
			ipsets.IPSetMetadata{SetID: "tproxy-services", Type: ipsets.IPSetTypeHashIPPort, MaxSize: maxsize},
			[]string{},
		)

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

		anyV4, _ := ip.CIDRFromString("0.0.0.0/0")
		rt.RouteUpdate("lo", routetable.Target{
			Type: routetable.TargetTypeLocal,
			CIDR: anyV4,
		})

		rr.SetRule(routerule.NewRule(4, 1).
			GoToTable(idx4).
			MatchFWMarkWithMask(uint32(mark), uint32(mark)),
		)

		tpm.rr4 = rr
		tpm.rt4 = rt
	}

	if ipSetsV6 != nil {
		ipSetsV6.AddOrReplaceIPSet(
			ipsets.IPSetMetadata{SetID: "tproxy-services", Type: ipsets.IPSetTypeHashIPPort, MaxSize: maxsize},
			[]string{},
		)

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
			4,
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

		anyV6, _ := ip.CIDRFromString("::/0")
		rt.RouteUpdate("lo", routetable.Target{
			Type: routetable.TargetTypeLocal,
			CIDR: anyV6,
		})

		rr.SetRule(routerule.NewRule(6, 1).
			GoToTable(idx6).
			MatchFWMarkWithMask(uint32(mark), uint32(mark)),
		)

		tpm.rr6 = rr
		tpm.rt6 = rt
	}

	return tpm
}

func (m *tproxyManager) OnUpdate(msg interface{}) {
}

func (m *tproxyManager) CompleteDeferredWork() error {
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
