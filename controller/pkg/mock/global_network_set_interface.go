// Copyright 2019 Tigera Inc. All rights reserved.

package mock

import (
	"context"
	"sync"

	"github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/tigera/intrusion-detection/controller/pkg/statser"
)

type GlobalNetworkSetInterface struct {
	GlobalNetworkSet *v3.GlobalNetworkSet
	Error            error
	CreateError      error
	GetError         error
	UpdateError      error
	WatchError       error
	W                *Watch
}

func (m *GlobalNetworkSetInterface) Create(gns *v3.GlobalNetworkSet) (*v3.GlobalNetworkSet, error) {
	if m.CreateError != nil {
		return nil, m.CreateError
	}
	m.GlobalNetworkSet = gns
	return gns, m.Error
}

func (m *GlobalNetworkSetInterface) Update(gns *v3.GlobalNetworkSet) (*v3.GlobalNetworkSet, error) {
	if m.UpdateError != nil {
		return nil, m.UpdateError
	}
	m.GlobalNetworkSet = gns
	return gns, m.Error
}

func (m *GlobalNetworkSetInterface) Delete(name string, options *v1.DeleteOptions) error {
	return m.Error
}

func (m *GlobalNetworkSetInterface) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return m.Error
}

func (m *GlobalNetworkSetInterface) Get(name string, options v1.GetOptions) (*v3.GlobalNetworkSet, error) {
	if m.GetError != nil {
		return nil, m.GetError
	}
	return m.GlobalNetworkSet, m.Error
}

func (m *GlobalNetworkSetInterface) List(opts v1.ListOptions) (*v3.GlobalNetworkSetList, error) {
	return nil, m.Error
}

func (m *GlobalNetworkSetInterface) Watch(opts v1.ListOptions) (watch.Interface, error) {
	if m.WatchError == nil {
		if m.W == nil {
			m.W = &Watch{make(chan watch.Event)}
		}
		return m.W, nil
	} else {
		return nil, m.WatchError
	}
}

func (m *GlobalNetworkSetInterface) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v3.GlobalNetworkSet, err error) {
	return nil, m.Error
}

type GlobalNetworkSetController struct {
	m         sync.Mutex
	local     map[string]*v3.GlobalNetworkSet
	noGC      map[string]struct{}
	failFuncs map[string]func()
	statsers  map[string]statser.Statser
}

func NewGlobalNetworkSetController() *GlobalNetworkSetController {
	return &GlobalNetworkSetController{
		local:     make(map[string]*v3.GlobalNetworkSet),
		noGC:      make(map[string]struct{}),
		failFuncs: make(map[string]func()),
		statsers:  make(map[string]statser.Statser),
	}
}

func (c *GlobalNetworkSetController) Add(s *v3.GlobalNetworkSet, f func(), stat statser.Statser) {
	c.m.Lock()
	defer c.m.Unlock()
	c.local[s.Name] = s
	c.failFuncs[s.Name] = f
	c.statsers[s.Name] = stat
}

func (c *GlobalNetworkSetController) Delete(s *v3.GlobalNetworkSet) {
	c.m.Lock()
	defer c.m.Unlock()
	delete(c.local, s.Name)
	delete(c.noGC, s.Name)
	delete(c.failFuncs, s.Name)
	delete(c.statsers, s.Name)
}

func (c *GlobalNetworkSetController) NoGC(s *v3.GlobalNetworkSet) {
	c.m.Lock()
	defer c.m.Unlock()
	c.noGC[s.Name] = struct{}{}
}

func (c *GlobalNetworkSetController) Run(ctx context.Context) {
	return
}

func (c *GlobalNetworkSetController) Local() map[string]*v3.GlobalNetworkSet {
	out := make(map[string]*v3.GlobalNetworkSet)
	c.m.Lock()
	defer c.m.Unlock()
	for k, s := range c.local {
		out[k] = s
	}
	return out
}

func (c *GlobalNetworkSetController) NotGCable() map[string]struct{} {
	out := make(map[string]struct{})
	c.m.Lock()
	defer c.m.Unlock()
	for k, s := range c.noGC {
		out[k] = s
	}
	return out
}

func (c *GlobalNetworkSetController) FailFuncs() map[string]func() {
	out := make(map[string]func())
	c.m.Lock()
	defer c.m.Unlock()
	for k, s := range c.failFuncs {
		out[k] = s
	}
	return out
}
