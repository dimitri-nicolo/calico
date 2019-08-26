// Copyright 2019 Tigera Inc. All rights reserved.

package elastic

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/olivere/elastic"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/statser"
)

type Controller interface {
	// Delete, and NoGC alter the desired state the controller will attempt to
	// maintain, by syncing with the elastic database.

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

type data interface {
	put(ctx context.Context, name string, value interface{}) error
	list(ctx context.Context) ([]db.Meta, error)
	delete(ctx context.Context, m db.Meta) error
}

type controller struct {
	once    sync.Once
	dirty   map[string]update
	noGC    map[string]struct{}
	updates chan update
	data    data
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
	value   interface{}
	fail    func()
	statser statser.Statser
}

const DefaultUpdateQueueLen = 1000
const DefaultElasticReconcilePeriod = 15 * time.Second

var NewTicker = func() *time.Ticker {
	tkr := time.NewTicker(DefaultElasticReconcilePeriod)
	return tkr
}

func (c *controller) add(ctx context.Context, name string, value interface{}, f func(), stat statser.Statser) {
	select {
	case <-ctx.Done():
		return
	case c.updates <- update{name: name, op: opAdd, value: value, fail: f, statser: stat}:
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
	metas, err := c.data.list(ctx)
	if err != nil {
		log.WithError(err).Error("failed to reconcile elastic object")
		for _, u := range c.dirty {
			u.statser.Error(statser.ElasticSyncFailed, err)
		}
		return
	}

	for _, m := range metas {

		if u, ok := c.dirty[m.Name]; ok {
			// value already exists, but is dirty
			c.updateObject(ctx, u)
		} else if _, ok := c.noGC[m.Name]; !ok {
			// Garbage collect
			c.purgeObject(ctx, m)
		} else {
			log.WithField("name", m.Name).Debug("Retained elastic object")
		}
	}

	for _, u := range c.dirty {
		c.updateObject(ctx, u)
	}
}

func (c *controller) updateObject(ctx context.Context, u update) {
	err := c.data.put(ctx, u.name, u.value)
	if err != nil {
		log.WithError(err).WithField("name", u.name).Error("failed to update elastic object")
		u.fail()
		u.statser.Error(statser.ElasticSyncFailed, err)
		return
	}
	// success!
	u.statser.ClearError(statser.ElasticSyncFailed)
	c.noGC[u.name] = struct{}{}
	delete(c.dirty, u.name)
}

func (c *controller) purgeObject(ctx context.Context, m db.Meta) {
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

	err := c.data.delete(ctx, m)
	if elastic.IsNotFound(err) {
		return
	}
	if err != nil {
		log.WithError(err).WithFields(fields).Error("Failed to purge Elastic Sets")
		return
	}
	log.WithFields(fields).Info("GC'd elastic Sets")
	return
}
