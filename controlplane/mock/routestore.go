package mock

import (
	"github.com/tigera/egress-gateway/data"
	"github.com/tigera/egress-gateway/proto"
)

type Store struct {
	WorkloadsByDst map[string]*proto.RouteUpdate
	TunnelsByDst   map[string]*proto.RouteUpdate
	GatewayUpdate  *proto.RouteUpdate
}

func (s Store) Routes() (
	thisWorkload *proto.RouteUpdate,
	workloadsByNodeName map[string][]proto.RouteUpdate,
	tunnelsByNodeName map[string][]proto.RouteUpdate,
){
	thisWorkload = s.GatewayUpdate
	workloadsByNodeName = make(map[string][]proto.RouteUpdate)
	tunnelsByNodeName = make(map[string][]proto.RouteUpdate)

	for _, wl := range s.WorkloadsByDst {
		if _, ok := workloadsByNodeName[wl.DstNodeName]; !ok {
			workloadsByNodeName[wl.DstNodeName] = make([]proto.RouteUpdate, 0)
		}
		workloadsByNodeName[wl.DstNodeName] = append(workloadsByNodeName[wl.DstNodeName], *wl)
	}

	for _, tn := range s.TunnelsByDst {
		if _, ok := tunnelsByNodeName[tn.DstNodeName]; !ok {
			tunnelsByNodeName[tn.DstNodeName] = make([]proto.RouteUpdate, 0)
		}
		tunnelsByNodeName[tn.DstNodeName] = append(tunnelsByNodeName[tn.DstNodeName], *tn)
	}

	return thisWorkload, workloadsByNodeName, tunnelsByNodeName
}

func (s Store) GatewayWorkload(readFn func(*proto.RouteUpdate)) {
	readFn(s.GatewayUpdate)
}

func (s Store) WorkloadsByNodeName(readFn func(map[string][]proto.RouteUpdate)) {
	workloadsByNodeName := make(map[string][]proto.RouteUpdate)

	readFn(workloadsByNodeName)
}

func (s Store) Workloads(readFn func(map[string]*proto.RouteUpdate)) {
	readFn(s.WorkloadsByDst)
}

func (s Store) Tunnels(readFn func(map[string]*proto.RouteUpdate)) {
	readFn(s.TunnelsByDst)
}

func (s Store) TunnelsByNodeName(readFn func(map[string][]proto.RouteUpdate)) {
	tunnelsByNodeName := make(map[string][]proto.RouteUpdate)

	readFn(tunnelsByNodeName)
}

func (s Store) Subscribe(o data.RouteObserver) {
	//chillin
}
