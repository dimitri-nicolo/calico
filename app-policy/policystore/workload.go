// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package policystore

import (
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/app-policy/types"
	"github.com/projectcalico/calico/felix/ip"
	"github.com/projectcalico/calico/felix/proto"
)

type workloadUpdateHandler struct {
	// workloads we actively know about
	workloads map[proto.WorkloadEndpointID]*wepWrap
}

type wepWrap struct {
	*proto.WorkloadEndpoint
	activeIp4nets map[string]struct{}
}

func newWep(w *proto.WorkloadEndpoint) *wepWrap {
	ip4s := make(map[string]struct{})
	for _, ip4 := range w.Ipv4Nets {
		ip4s[ip4] = struct{}{}
	}
	return &wepWrap{w, ip4s}
}

func newWorkloadEndpointUpdateHandler() *workloadUpdateHandler {
	return &workloadUpdateHandler{
		workloads: make(map[proto.WorkloadEndpointID]*wepWrap),
	}
}

func (wuh *workloadUpdateHandler) onResourceUpdate(upd interface{}, cb types.IPToEndpointsIndex) {
	switch v := upd.(type) {
	case *proto.WorkloadEndpointRemove:
		wuh.onWorkloadEndpointRemove(v, cb)
	case *proto.WorkloadEndpointUpdate:
		wuh.onWorkloadEndpointUpdate(v, cb)
	default:
	}
}

func (wuh *workloadUpdateHandler) onWorkloadEndpointUpdate(upd *proto.WorkloadEndpointUpdate, cb types.IPToEndpointsIndex) {
	id, ep := *upd.Id, newWep(upd.Endpoint)

	var removals []string

	incomingIps := map[string]struct{}{}
	for _, ip4 := range ep.Ipv4Nets {
		incomingIps[ip4] = struct{}{}
	}

	// before we track the incoming ep, see if we have data about it already
	if existing, ok := wuh.workloads[id]; ok {
		// we know about this workload, compare incoming set with existing
		// to determine removals
		for ip4 := range existing.activeIp4nets {
			if _, ok := incomingIps[ip4]; !ok {
				removals = append(removals, ip4)
				delete(existing.activeIp4nets, ip4)
			}
		}
		for ip4 := range incomingIps {
			if _, ok := existing.activeIp4nets[ip4]; !ok {
				existing.activeIp4nets[ip4] = struct{}{}
			}
		}
		wuh.workloads[id].WorkloadEndpoint = ep.WorkloadEndpoint
	} else {
		wuh.workloads[id] = ep
	}

	log.Debugf("onWEPUpdate removals %v", removals)

	for _, net4 := range removals {
		ipNet4, err := ip.ParseCIDROrIP(net4)
		if err != nil {
			log.Debug("error parsing cidr or ip", net4)
			continue
		}
		cb.Delete(ipNet4, &proto.WorkloadEndpointRemove{
			Id: upd.Id,
		})
	}

	// process upserts
	if existing, ok := wuh.workloads[id]; ok {
		for net4 := range existing.activeIp4nets {
			ipNet4, err := ip.ParseCIDROrIP(net4)
			if err != nil {
				log.Debug("error parsing cidr or ip", net4)

			}
			log.Debug("upsert occurred for ", ipNet4)
			cb.Update(ipNet4, upd)
		}
	}
}

func (wuh *workloadUpdateHandler) onWorkloadEndpointRemove(upd *proto.WorkloadEndpointRemove, cb types.IPToEndpointsIndex) {
	// find former known workload, delete its last known ips
	if existing, ok := wuh.workloads[*upd.Id]; ok {
		for net4 := range existing.activeIp4nets {
			ipNet4, err := ip.ParseCIDROrIP(net4)
			if err != nil {
				log.Debug("error parsing cidr or ip", net4)
				continue
			}
			cb.Delete(ipNet4, upd)
		}
	}
	// finally, forget about this workload
	delete(wuh.workloads, *upd.Id)
}
