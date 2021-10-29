package mock

import (
	"context"
	"time"

	"github.com/tigera/egress-gateway/data"
	"github.com/tigera/egress-gateway/proto"
)

type Store struct {
	RoutesByWorkloadCIDR map[string]*proto.RouteUpdate
}

func (s Store) Routes(readFn func(map[string]*proto.RouteUpdate)) {
	readFn(s.RoutesByWorkloadCIDR)
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
