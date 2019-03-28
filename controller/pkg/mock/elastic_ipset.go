// Copyright 2019 Tigera Inc. All rights reserved.

package mock

import (
	"context"
	"sync"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/statser"
	"github.com/tigera/intrusion-detection/controller/pkg/sync/elasticipsets"
)

var _ elasticipsets.Controller = NewElasticIPSetController()

type ElasticIPSetController struct {
	m         sync.Mutex
	sets      map[string]db.IPSetSpec
	failFuncs map[string]func()
	statsers  map[string]statser.Statser
	noGC      map[string]struct{}
}

func NewElasticIPSetController() *ElasticIPSetController {
	return &ElasticIPSetController{
		sets:      make(map[string]db.IPSetSpec),
		failFuncs: make(map[string]func()),
		statsers:  make(map[string]statser.Statser),
		noGC:      make(map[string]struct{}),
	}
}

func (c *ElasticIPSetController) Add(name string, set db.IPSetSpec, f func(), stat statser.Statser) {
	c.m.Lock()
	defer c.m.Unlock()
	c.sets[name] = set
	c.failFuncs[name] = f
	c.statsers[name] = stat
}

func (c *ElasticIPSetController) Delete(name string) {
	c.m.Lock()
	defer c.m.Unlock()
	delete(c.sets, name)
	delete(c.failFuncs, name)
	delete(c.statsers, name)
	delete(c.noGC, name)
}

func (c *ElasticIPSetController) NoGC(name string) {
	c.m.Lock()
	defer c.m.Unlock()
	c.noGC[name] = struct{}{}
}

func (c *ElasticIPSetController) StartGC() {
	return
}

func (c *ElasticIPSetController) Run(ctx context.Context) {
	<-ctx.Done()
}
