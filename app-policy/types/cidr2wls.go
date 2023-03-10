package types

import (
	"github.com/projectcalico/calico/felix/ip"
	"github.com/projectcalico/calico/felix/proto"
)

type IPToEndpointsIndex interface {
	Get(k ip.CIDR) []*proto.WorkloadEndpoint
	Update(k ip.CIDR, v *proto.WorkloadEndpointUpdate)
	Delete(k ip.CIDR, v *proto.WorkloadEndpointRemove)
}

type wlMap map[proto.WorkloadEndpointID]*proto.WorkloadEndpoint

func NewIPToEndpointsIndex() IPToEndpointsIndex {
	return &IPToEndpointsIndexer{
		make(map[ip.CIDR]wlMap),
	}
}

type IPToEndpointsIndexer struct {
	store map[ip.CIDR]wlMap
}

func (index *IPToEndpointsIndexer) Get(k ip.CIDR) (res []*proto.WorkloadEndpoint) {
	for _, item := range index.store[k] {
		res = append(res, item)
	}
	return
}

func (index *IPToEndpointsIndexer) Update(k ip.CIDR, v *proto.WorkloadEndpointUpdate) {
	if _, ok := index.store[k]; !ok {
		index.store[k] = make(wlMap)
		index.Update(k, v)
		return
	}

	index.store[k][*v.Id] = v.Endpoint
}

func (index *IPToEndpointsIndexer) Delete(k ip.CIDR, v *proto.WorkloadEndpointRemove) {
	if _, ok := index.store[k]; !ok {
		return
	}
	delete(index.store[k], *v.Id)
	if len(index.store[k]) == 0 {
		delete(index.store, k)
	}
}
