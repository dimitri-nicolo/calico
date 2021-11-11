package data

import (
	"github.com/tigera/egress-gateway/proto"
)

type MockObserver struct {
	RoutesSnapshot   map[string]*proto.RouteUpdate
	NumNotifications int
}

// NewMockObserver creates a new observer and registers it with the store
func NewMockObserver(s RouteStore) *MockObserver {
	o := MockObserver{}
	o.NumNotifications = 0
	s.Subscribe(&o)
	return &o
}

func (o *MockObserver) NotifyResync(s RouteStore) {
	o.NumNotifications++
	s.Workloads(func(routes map[string]*proto.RouteUpdate) {
		o.RoutesSnapshot = routes
	})
}
