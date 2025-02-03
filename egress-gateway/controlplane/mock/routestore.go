package mock

import (
	"github.com/projectcalico/calico/egress-gateway/data"
	"github.com/projectcalico/calico/felix/proto"
)

type Store struct {
	WorkloadsByDst map[string]*proto.RouteUpdate
	TunnelsByDst   map[string]*proto.RouteUpdate
	GatewayUpdate  *proto.RouteUpdate
}

func (s Store) Routes() (
	thisWorkload *proto.RouteUpdate,
	workloadsByNodeName map[string][]*proto.RouteUpdate,
	tunnelsByNodeName map[string][]*proto.RouteUpdate,
) {
	thisWorkload = s.GatewayUpdate
	workloadsByNodeName = make(map[string][]*proto.RouteUpdate)
	tunnelsByNodeName = make(map[string][]*proto.RouteUpdate)

	for _, wl := range s.WorkloadsByDst {
		if _, ok := workloadsByNodeName[wl.DstNodeName]; !ok {
			workloadsByNodeName[wl.DstNodeName] = make([]*proto.RouteUpdate, 0)
		}
		workloadsByNodeName[wl.DstNodeName] = append(workloadsByNodeName[wl.DstNodeName], wl)
	}

	for _, tn := range s.TunnelsByDst {
		if _, ok := tunnelsByNodeName[tn.DstNodeName]; !ok {
			tunnelsByNodeName[tn.DstNodeName] = make([]*proto.RouteUpdate, 0)
		}
		tunnelsByNodeName[tn.DstNodeName] = append(tunnelsByNodeName[tn.DstNodeName], tn)
	}

	return thisWorkload, workloadsByNodeName, tunnelsByNodeName
}

func (s Store) Subscribe(o data.RouteObserver) {
	// chillin
}
