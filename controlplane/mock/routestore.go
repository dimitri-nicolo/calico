package mock

import (
	"context"
	"time"

	"github.com/tigera/egress-gateway/data"
	"github.com/tigera/egress-gateway/proto"
)

type Store struct {
	WorkloadsByDst  map[string]*proto.RouteUpdate
	TunnelsByNodeIP map[string]*proto.RouteUpdate
	GatewayUpdate   *proto.RouteUpdate
}


func (s Store) GatewayWorkload(readFn func(*proto.RouteUpdate)) {
	readFn(s.GatewayUpdate)
}

func (s Store) WorkloadsByNodeName(readFn func(map[string][]proto.RouteUpdate)) {

}

func (s Store) Workloads(readFn func(map[string]*proto.RouteUpdate)) {
	readFn(s.WorkloadsByDst)
}

func (s Store) Tunnels(readFn func(map[string]*proto.RouteUpdate)) {
	readFn(s.TunnelsByNodeIP)
}

func (s Store) TunnelsByNodeName(readFn func(map[string][]proto.RouteUpdate)) {

}

func (s Store) Subscribe(o data.RouteObserver) {
	//chillin
}

func (s Store) SyncForever(ctx context.Context) {
	for {
		time.Sleep(1 * time.Second)
		// chillllllllin
	}
}
