// Copyright 2019 Tigera Inc. All rights reserved.

package elastic

import (
	"context"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/statser"
)

type IPSetController interface {
	Controller

	// Add alters the desired state the controller will attempt to
	// maintain, by syncing with the elastic database.

	// Add or update a new Set including the spec. f is function the controller should call
	// if we fail to update, and stat is the Statser we should report or clear errors on.
	Add(ctx context.Context, name string, set db.IPSetSpec, f func(), stat statser.Statser)
}

type ipSetController controller

func NewIPSetController(ipSet db.Sets) IPSetController {
	return &ipSetController{
		dirty:   make(map[string]update),
		noGC:    make(map[string]struct{}),
		updates: make(chan update, DefaultUpdateQueueLen),
		kind:    db.KindIPSet,
		db:      ipSet,
	}
}

func (c *ipSetController) Add(ctx context.Context, name string, set db.IPSetSpec, f func(), stat statser.Statser) {
	(*controller)(c).add(ctx, name, set, f, stat)
}

func (c *ipSetController) Delete(ctx context.Context, name string) {
	(*controller)(c).Delete(ctx, name)
}

func (c *ipSetController) NoGC(ctx context.Context, name string) {
	(*controller)(c).NoGC(ctx, name)
}

func (c *ipSetController) StartReconciliation(ctx context.Context) {
	(*controller)(c).StartReconciliation(ctx)
}

func (c *ipSetController) Run(ctx context.Context) {
	(*controller)(c).Run(ctx)
}
