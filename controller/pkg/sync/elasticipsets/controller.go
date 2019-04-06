// Copyright 2019 Tigera Inc. All rights reserved.

package elasticipsets

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/olivere/elastic"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/statser"
)

type Controller interface {
	// Add, Delete, and NoGC alter the desired state the controller will attempt to
	// maintain, by syncing with the elastic database.

	// Add or update a new Set including the spec. f is function the controller should call
	// if we fail to update, and stat is the Statser we should report or clear errors on.
	Add(ctx context.Context, name string, set db.IPSetSpec, f func(), stat statser.Statser)

	// Delete removes a Set from the desired state.
	Delete(ctx context.Context, name string)

	// NoGC marks a Set as not eligible for garbage collection
	// until deleted. This is useful when we don't know the contents of a
	// Set, but know it should not be deleted.
	NoGC(ctx context.Context, name string)

	// StartReconciliation indicates that all Sets we don't want garbage
	// collected have either Add() or NoGC() called on them, and we can start
	// reconciling our desired state with the actual state.
	StartReconciliation(ctx context.Context)

	// Run starts processing Sets
	Run(context.Context)
}

type controller struct {
	once    sync.Once
	ipSet   db.IPSet
	dirty   map[string]update
	noGC    map[string]struct{}
	updates chan update
}

type op int

const (
	opAdd op = iota
	opDelete
	opNoGC
	opStart
)

type update struct {
	name    string
	op      op
	set     db.IPSetSpec
	fail    func()
	statser statser.Statser
}

const DefaultUpdateQueueLen = 1000
const DefaultElasticReconcilePeriod = 15 * time.Second

var NewTicker = func() *time.Ticker {
	tkr := time.NewTicker(DefaultElasticReconcilePeriod)
	return tkr
}

func NewController(ipSet db.IPSet) Controller {
	return &controller{
		ipSet:   ipSet,
		dirty:   make(map[string]update),
		noGC:    make(map[string]struct{}),
		updates: make(chan update, DefaultUpdateQueueLen),
	}
}

func (c *controller) Add(ctx context.Context, name string, set db.IPSetSpec, f func(), stat statser.Statser) {
	select {
	case <-ctx.Done():
		return
	case c.updates <- update{name: name, op: opAdd, set: set, fail: f, statser: stat}:
		return
	}
}

func (c *controller) Delete(ctx context.Context, name string) {
	select {
	case <-ctx.Done():
		return
	case c.updates <- update{name: name, op: opDelete}:
		return
	}
}

func (c *controller) NoGC(ctx context.Context, name string) {
	select {
	case <-ctx.Done():
		return
	case c.updates <- update{name: name, op: opNoGC}:
		return
	}
}

func (c *controller) StartReconciliation(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	case c.updates <- update{op: opStart}:
		return
	}
}

func (c *controller) Run(ctx context.Context) {
	c.once.Do(func() {
		go c.run(ctx)
	})
}

func (c *controller) run(ctx context.Context) {

	log.Debug("starting elastic controller")

	// Initially, we're just processing state updates, and not triggering any
	// reconcilliation.
UpdateLoop:
	for {
		select {
		case <-ctx.Done():
			return
		case u := <-c.updates:
			if u.op == opStart {
				break UpdateLoop
			}
			c.processUpdate(u)
		}
	}

	log.Debug("elastic controller reconciliation started")

	// After getting the startGC, we can also include state sync processing
	tkr := NewTicker()
	defer tkr.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case u := <-c.updates:
			if u.op == opStart {
				continue
			}
			c.processUpdate(u)
		case <-tkr.C:
			c.reconcile(ctx)
		}
	}
}

func (c *controller) processUpdate(u update) {
	switch u.op {
	case opAdd:
		c.dirty[u.name] = u
	case opDelete:
		delete(c.dirty, u.name)
		delete(c.noGC, u.name)
	case opNoGC:
		c.noGC[u.name] = struct{}{}
	default:
		panic(fmt.Sprintf("unhandled op type %d", u.op))
	}
}

func (c *controller) reconcile(ctx context.Context) {
	ipSets, err := c.ipSet.ListIPSets(ctx)
	if err != nil {
		log.WithError(err).Error("failed to reconcile elastic IP sets")
		for _, u := range c.dirty {
			u.statser.Error(statser.ElasticSyncFailed, err)
		}
		return
	}

	for _, ipSetMeta := range ipSets {

		if u, ok := c.dirty[ipSetMeta.Name]; ok {
			// set already exists, but is dirty
			c.updateElasticIPSet(ctx, u)
		} else if _, ok := c.noGC[ipSetMeta.Name]; !ok {
			// Garbage collect
			c.purgeElasticIPSet(ctx, ipSetMeta)
		} else {
			log.WithField("name", ipSetMeta.Name).Debug("Retained Elastic IPSet")
		}
	}

	for _, u := range c.dirty {
		c.updateElasticIPSet(ctx, u)
	}
}

func (c *controller) updateElasticIPSet(ctx context.Context, u update) {
	err := c.ipSet.PutIPSet(ctx, u.name, u.set)
	if err != nil {
		log.WithError(err).WithField("name", u.name).Error("failed to update ipset")
		u.fail()
		u.statser.Error(statser.ElasticSyncFailed, err)
		return
	}
	// success!
	u.statser.ClearError(statser.ElasticSyncFailed)
	c.noGC[u.name] = struct{}{}
	delete(c.dirty, u.name)
}

func (c *controller) purgeElasticIPSet(ctx context.Context, m db.IPSetMeta) {
	var fields log.Fields
	if m.Version != nil {
		fields = log.Fields{
			"name":    m.Name,
			"version": m.Version,
		}
	} else {
		fields = log.Fields{
			"name":    m.Name,
			"version": "nil",
		}
	}

	err := c.ipSet.DeleteIPSet(ctx, m)
	if elastic.IsNotFound(err) {
		return
	}
	if err != nil {
		log.WithError(err).WithFields(fields).Error("Failed to purge Elastic IPSet")
		return
	}
	log.WithFields(fields).Info("GC'd elastic IPSet")
	return
}
