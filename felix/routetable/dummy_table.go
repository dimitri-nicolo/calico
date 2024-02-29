package routetable

import (
	"github.com/projectcalico/calico/felix/ifacemonitor"
	"github.com/projectcalico/calico/felix/ip"
)

type DummyTable struct {
}

func (_ *DummyTable) OnIfaceStateChanged(_ string, _ ifacemonitor.State) {
}

func (_ *DummyTable) QueueResync() {
}

func (_ *DummyTable) QueueResyncIface(string) {
}

func (_ *DummyTable) Apply() error {
	return nil
}

func (_ *DummyTable) SetRoutes(_ string, _ []Target) {
}

func (_ *DummyTable) RouteRemove(_ string, _ ip.CIDR) {
}

func (_ *DummyTable) RouteUpdate(_ string, _ Target) {
}

func (_ *DummyTable) Index() int {
	return 0
}

func (_ *DummyTable) ReadRoutesFromKernel(ifaceName string) ([]Target, error) {
	return nil, nil
}

func (_ *DummyTable) SetRemoveExternalRoutes(b bool) {
}
