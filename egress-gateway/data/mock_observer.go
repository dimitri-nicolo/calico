package data

import (
	"github.com/projectcalico/calico/felix/proto"
)

type MockObserver struct {
	ThisWorkloadSnapShot *proto.RouteUpdate
	WorkloadsSnapshot    map[string][]*proto.RouteUpdate
	TunnelsSnapshot      map[string][]*proto.RouteUpdate
	NumNotifications     int
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
	o.ThisWorkloadSnapShot, o.WorkloadsSnapshot, o.TunnelsSnapshot = s.Routes()
}
