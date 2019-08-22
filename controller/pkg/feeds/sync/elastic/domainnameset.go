// Copyright 2019 Tigera Inc. All rights reserved.

package elastic

import (
	"context"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/statser"
)

type DomainNameSetController interface {
	Controller

	// Add alters the desired state the controller will attempt to
	// maintain, by syncing with the elastic database.

	// Add or update a new Set including the spec. f is function the controller should call
	// if we fail to update, and stat is the Statser we should report or clear errors on.
	Add(ctx context.Context, name string, set db.DomainNameSetSpec, f func(), stat statser.Statser)
}

type dnSetController controller

type dnSetData struct {
	dnSet db.DomainNameSet
}

func NewDomainNameSetController(sets db.DomainNameSet) DomainNameSetController {
	return &dnSetController{
		dirty:   make(map[string]update),
		noGC:    make(map[string]struct{}),
		updates: make(chan update, DefaultUpdateQueueLen),
		data:    dnSetData{sets},
	}
}

func (c *dnSetController) Add(ctx context.Context, name string, set db.DomainNameSetSpec, f func(), stat statser.Statser) {
	(*controller)(c).add(ctx, name, set, f, stat)
}

func (c *dnSetController) Delete(ctx context.Context, name string) {
	(*controller)(c).Delete(ctx, name)
}

func (c *dnSetController) NoGC(ctx context.Context, name string) {
	(*controller)(c).NoGC(ctx, name)
}

func (c *dnSetController) StartReconciliation(ctx context.Context) {
	(*controller)(c).StartReconciliation(ctx)
}

func (c *dnSetController) Run(ctx context.Context) {
	(*controller)(c).Run(ctx)
}

func (d dnSetData) put(ctx context.Context, name string, value interface{}) error {
	return d.dnSet.PutDomainNameSet(ctx, name, value.(db.DomainNameSetSpec))
}

func (d dnSetData) list(ctx context.Context) ([]db.Meta, error) {
	return d.dnSet.ListDomainNameSets(ctx)
}

func (d dnSetData) delete(ctx context.Context, m db.Meta) error {
	return d.dnSet.DeleteDomainNameSet(ctx, m)
}
