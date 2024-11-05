package types

import (
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/felix/ip"
	"github.com/projectcalico/calico/felix/proto"
)

type IPToEndpointsIndex interface {
	Keys(k ip.Addr) []proto.WorkloadEndpointID
	Get(k ip.Addr) []*proto.WorkloadEndpoint
	Update(k ip.Addr, v *proto.WorkloadEndpointUpdate)
	Delete(k ip.Addr, v *proto.WorkloadEndpointRemove)
}

type wlMap map[proto.WorkloadEndpointID]*proto.WorkloadEndpoint

func NewIPToEndpointsIndex() IPToEndpointsIndex {
	return &IPToEndpointsIndexer{
		make(map[ip.Addr]wlMap),
	}
}

type IPToEndpointsIndexer struct {
	store map[ip.Addr]wlMap
}

func (index *IPToEndpointsIndexer) Keys(k ip.Addr) (res []proto.WorkloadEndpointID) {
	for item := range index.store[k] {
		res = append(res, item)
	}
	return
}

func (index *IPToEndpointsIndexer) Get(k ip.Addr) (res []*proto.WorkloadEndpoint) {
	log.Trace("before get: ", index.printKeys())
	for _, item := range index.store[k] {
		res = append(res, item)
	}
	return
}

func (index *IPToEndpointsIndexer) printKeys() []string {
	res := []string{}
	for entry := range index.store {
		res = append(res, entry.String())
	}
	return res
}

func (index *IPToEndpointsIndexer) Update(k ip.Addr, v *proto.WorkloadEndpointUpdate) {
	if log.IsLevelEnabled(log.TraceLevel) {
		log.Trace("before update: ", index.printKeys())
		defer log.Trace("after update: ", index.printKeys())
	}
	if _, ok := index.store[k]; !ok {
		index.store[k] = make(wlMap)
	}

	index.store[k][*v.Id] = v.Endpoint
}

func (index *IPToEndpointsIndexer) Delete(k ip.Addr, v *proto.WorkloadEndpointRemove) {
	if log.IsLevelEnabled(log.TraceLevel) {
		log.Trace("before delete: ", index.printKeys())
		defer log.Trace("after delete: ", index.printKeys())
	}
	if _, ok := index.store[k]; !ok {
		return
	}
	delete(index.store[k], *v.Id)
	if len(index.store[k]) == 0 {
		delete(index.store, k)
	}
}
