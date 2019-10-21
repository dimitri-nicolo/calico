// Copyright (c) 2019 Tigera Inc. All rights reserved.

package elastic

import (
	"context"
	"sync"

	"github.com/tigera/intrusion-detection/controller/pkg/controller"
	"github.com/tigera/intrusion-detection/controller/pkg/elastic"
)

type MockAlertsController struct {
	m         sync.Mutex
	alerts    map[string]*elastic.PutWatchBody
	failFuncs map[string]func(error)
	statsers  map[string]controller.Statser
	noGC      map[string]struct{}
}

func NewMockAlertsController() *MockAlertsController {
	return &MockAlertsController{
		alerts:    make(map[string]*elastic.PutWatchBody),
		failFuncs: make(map[string]func(error)),
		statsers:  make(map[string]controller.Statser),
		noGC:      make(map[string]struct{}),
	}
}

func (c *MockAlertsController) Add(ctx context.Context, name string, alert interface{}, f func(error), stat controller.Statser) {
	c.m.Lock()
	defer c.m.Unlock()
	c.alerts[name] = alert.(*elastic.PutWatchBody)
	c.failFuncs[name] = f
	c.statsers[name] = stat
}

func (c *MockAlertsController) Delete(ctx context.Context, name string) {
	c.m.Lock()
	defer c.m.Unlock()
	delete(c.alerts, name)
	delete(c.failFuncs, name)
	delete(c.statsers, name)
	delete(c.noGC, name)
}

func (c *MockAlertsController) NoGC(ctx context.Context, name string) {
	c.m.Lock()
	defer c.m.Unlock()
	c.noGC[name] = struct{}{}
}

func (c *MockAlertsController) StartReconciliation(ctx context.Context) {
	return
}

func (c *MockAlertsController) Run(ctx context.Context) {
	return
}

func (c *MockAlertsController) NotGCable() map[string]struct{} {
	out := make(map[string]struct{})
	c.m.Lock()
	defer c.m.Unlock()
	for k, s := range c.noGC {
		out[k] = s
	}
	return out
}

func (c *MockAlertsController) Bodies() map[string]*elastic.PutWatchBody {
	out := make(map[string]*elastic.PutWatchBody)
	c.m.Lock()
	defer c.m.Unlock()
	for k, s := range c.alerts {
		out[k] = s
	}
	return out
}
