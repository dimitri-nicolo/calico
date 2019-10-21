// Copyright 2019 Tigera Inc. All rights reserved.

package globalnetworksets

import (
	"context"
	"sync"

	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/statser"
)

type MockGlobalNetworkSetController struct {
	m         sync.Mutex
	local     map[string]*v3.GlobalNetworkSet
	noGC      map[string]struct{}
	failFuncs map[string]func(error)
	statsers  map[string]statser.Statser
}

func NewMockGlobalNetworkSetController() *MockGlobalNetworkSetController {
	return &MockGlobalNetworkSetController{
		local:     make(map[string]*v3.GlobalNetworkSet),
		noGC:      make(map[string]struct{}),
		failFuncs: make(map[string]func(error)),
		statsers:  make(map[string]statser.Statser),
	}
}

func (c *MockGlobalNetworkSetController) Add(s *v3.GlobalNetworkSet, f func(error), stat statser.Statser) {
	c.m.Lock()
	defer c.m.Unlock()
	c.local[s.Name] = s
	c.failFuncs[s.Name] = f
	c.statsers[s.Name] = stat
}

func (c *MockGlobalNetworkSetController) Delete(s *v3.GlobalNetworkSet) {
	c.m.Lock()
	defer c.m.Unlock()
	delete(c.local, s.Name)
	delete(c.noGC, s.Name)
	delete(c.failFuncs, s.Name)
	delete(c.statsers, s.Name)
}

func (c *MockGlobalNetworkSetController) NoGC(s *v3.GlobalNetworkSet) {
	c.m.Lock()
	defer c.m.Unlock()
	c.noGC[s.Name] = struct{}{}
}

func (c *MockGlobalNetworkSetController) Run(ctx context.Context) {
	return
}

func (c *MockGlobalNetworkSetController) Local() map[string]*v3.GlobalNetworkSet {
	out := make(map[string]*v3.GlobalNetworkSet)
	c.m.Lock()
	defer c.m.Unlock()
	for k, s := range c.local {
		out[k] = s
	}
	return out
}

func (c *MockGlobalNetworkSetController) NotGCable() map[string]struct{} {
	out := make(map[string]struct{})
	c.m.Lock()
	defer c.m.Unlock()
	for k, s := range c.noGC {
		out[k] = s
	}
	return out
}

func (c *MockGlobalNetworkSetController) FailFuncs() map[string]func(error) {
	out := make(map[string]func(error))
	c.m.Lock()
	defer c.m.Unlock()
	for k, s := range c.failFuncs {
		out[k] = s
	}
	return out
}
