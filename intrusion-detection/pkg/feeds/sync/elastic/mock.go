// Copyright 2019 Tigera Inc. All rights reserved.

package elastic

import (
	"context"
	"sync"

	"github.com/tigera/intrusion-detection/controller/pkg/feeds/cacher"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
)

type MockElasticIPSetController struct {
	m           sync.Mutex
	sets        map[string]db.IPSetSpec
	failFuncs   map[string]func(error)
	feedCachers map[string]cacher.GlobalThreatFeedCacher
	noGC        map[string]struct{}
}

func NewMockElasticIPSetController() *MockElasticIPSetController {
	return &MockElasticIPSetController{
		sets:        make(map[string]db.IPSetSpec),
		failFuncs:   make(map[string]func(error)),
		feedCachers: make(map[string]cacher.GlobalThreatFeedCacher),
		noGC:        make(map[string]struct{}),
	}
}

func (c *MockElasticIPSetController) Add(ctx context.Context, name string, set interface{}, f func(error), feedCacher cacher.GlobalThreatFeedCacher) {
	c.m.Lock()
	defer c.m.Unlock()
	c.sets[name] = set.(db.IPSetSpec)
	c.failFuncs[name] = f
	c.feedCachers[name] = feedCacher
}

func (c *MockElasticIPSetController) Delete(ctx context.Context, name string) {
	c.m.Lock()
	defer c.m.Unlock()
	delete(c.sets, name)
	delete(c.failFuncs, name)
	delete(c.feedCachers, name)
	delete(c.noGC, name)
}

func (c *MockElasticIPSetController) NoGC(ctx context.Context, name string) {
	c.m.Lock()
	defer c.m.Unlock()
	c.noGC[name] = struct{}{}
}

func (c *MockElasticIPSetController) StartReconciliation(ctx context.Context) {
	return
}

func (c *MockElasticIPSetController) Run(ctx context.Context) {
	return
}

func (c *MockElasticIPSetController) NotGCable() map[string]struct{} {
	out := make(map[string]struct{})
	c.m.Lock()
	defer c.m.Unlock()
	for k, s := range c.noGC {
		out[k] = s
	}
	return out
}

func (c *MockElasticIPSetController) Sets() map[string]db.IPSetSpec {
	out := make(map[string]db.IPSetSpec)
	c.m.Lock()
	defer c.m.Unlock()
	for k, s := range c.sets {
		out[k] = s
	}
	return out
}

type MockDomainNameSetsController struct {
	m           sync.Mutex
	sets        map[string]db.DomainNameSetSpec
	failFuncs   map[string]func(error)
	feedCachers map[string]cacher.GlobalThreatFeedCacher
	noGC        map[string]struct{}
}

func NewMockDomainNameSetsController() *MockDomainNameSetsController {
	return &MockDomainNameSetsController{
		sets:        make(map[string]db.DomainNameSetSpec),
		failFuncs:   make(map[string]func(error)),
		feedCachers: make(map[string]cacher.GlobalThreatFeedCacher),
		noGC:        make(map[string]struct{}),
	}
}

func (c *MockDomainNameSetsController) Add(ctx context.Context, name string, set interface{}, f func(error), feedCacher cacher.GlobalThreatFeedCacher) {
	c.m.Lock()
	defer c.m.Unlock()
	c.sets[name] = set.(db.DomainNameSetSpec)
	c.failFuncs[name] = f
	c.feedCachers[name] = feedCacher
}

func (c *MockDomainNameSetsController) Delete(ctx context.Context, name string) {
	c.m.Lock()
	defer c.m.Unlock()
	delete(c.sets, name)
	delete(c.failFuncs, name)
	delete(c.feedCachers, name)
	delete(c.noGC, name)
}

func (c *MockDomainNameSetsController) NoGC(ctx context.Context, name string) {
	c.m.Lock()
	defer c.m.Unlock()
	c.noGC[name] = struct{}{}
}

func (c *MockDomainNameSetsController) StartReconciliation(ctx context.Context) {
	return
}

func (c *MockDomainNameSetsController) Run(ctx context.Context) {
	return
}

func (c *MockDomainNameSetsController) NotGCable() map[string]struct{} {
	out := make(map[string]struct{})
	c.m.Lock()
	defer c.m.Unlock()
	for k, s := range c.noGC {
		out[k] = s
	}
	return out
}

func (c *MockDomainNameSetsController) Sets() map[string]db.DomainNameSetSpec {
	out := make(map[string]db.DomainNameSetSpec)
	c.m.Lock()
	defer c.m.Unlock()
	for k, s := range c.sets {
		out[k] = s
	}
	return out
}
