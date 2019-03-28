// Copyright 2019 Tigera Inc. All rights reserved.

package elasticipsets

import (
	"context"
	"fmt"
	"time"

	"github.com/tigera/intrusion-detection/controller/pkg/statser"

	"github.com/olivere/elastic"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
)

type Controller interface {
	// Add, Delete, and GC alter the desired state the controller will attempt to
	// maintain, by syncing with the Kubernetes API server.

	// Add or update a new Set including the spec
	Add(name string, set db.IPSetSpec, f func(), stat statser.Statser)

	// Delete removes a Set from the desired state.
	Delete(name string)

	// NoGC marks a Set as not eligible for garbage collection
	// until deleted. This is useful when we don't know the contents of a
	// Set, but know it should not be deleted.
	NoGC(name string)

	// StartReconciliation indicates that all Sets we don't want garbage
	// collected have either Add() or NoGC() called on them, and we can start
	// reconciling our desired state with the actual state.
	StartReconciliation()

	// Run starts processing Sets
	Run(context.Context)
}

type controller struct {
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

func NewController(ipSet db.IPSet) Controller {
	return &controller{
		ipSet:   ipSet,
		dirty:   make(map[string]update),
		noGC:    make(map[string]struct{}),
		updates: make(chan update, DefaultUpdateQueueLen),
	}
}

func (c *controller) Add(name string, set db.IPSetSpec, f func(), stat statser.Statser) {
	c.updates <- update{name: name, op: opAdd, set: set, fail: f, statser: stat}
}

func (c *controller) Delete(name string) {
	c.updates <- update{name: name, op: opDelete}
}

func (c *controller) NoGC(name string) {
	c.updates <- update{name: name, op: opNoGC}
}

func (c *controller) StartReconciliation() {
	c.updates <- update{op: opStart}
}

func (c *controller) Run(ctx context.Context) {

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
	tkr := time.NewTicker(DefaultElasticReconcilePeriod)
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
			err := c.ipSet.PutIPSet(ctx, ipSetMeta.Name, u.set)
			if err != nil {
				log.WithError(err).WithField("name", ipSetMeta.Name).Error("failed to update ipset")
				u.fail()
				u.statser.Error(statser.ElasticSyncFailed, err)
				continue
			}
			// success!
			u.statser.ClearError(statser.ElasticSyncFailed)
			c.noGC[ipSetMeta.Name] = struct{}{}
			delete(c.dirty, ipSetMeta.Name)
		} else if _, ok := c.noGC[ipSetMeta.Name]; !ok {
			// Garbage collect
			err := c.purgeElasticIPSet(ctx, ipSetMeta)
			if err != nil {
				log.WithError(err).Error("failed to purge elastic IP set")
				continue
			}
		} else {
			log.WithField("name", ipSetMeta.Name).Debug("Retained Elastic IPSet")
		}
	}

	for name, u := range c.dirty {
		err := c.ipSet.PutIPSet(ctx, name, u.set)
		if err != nil {
			log.WithError(err).WithField("name", name).Error("failed to update ipset")
			u.fail()
			u.statser.Error(statser.ElasticSyncFailed, err)
			continue
		}
		// success!
		u.statser.ClearError(statser.ElasticSyncFailed)
		c.noGC[name] = struct{}{}
		delete(c.dirty, name)
	}
}

func (c *controller) purgeElasticIPSet(ctx context.Context, m db.IPSetMeta) error {
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
		return nil
	}
	if err != nil {
		log.WithError(err).WithFields(fields).Error("Failed to purge Elastic IPSet")
		return err
	}
	log.WithFields(fields).Info("GC'd elastic IPSet")
	return nil
}
