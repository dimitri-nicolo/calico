// Copyright 2019 Tigera Inc. All rights reserved.

package elastic

import (
	"context"

	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/controller"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/db"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/feeds/cacher"
)

type dnSetData struct {
	dnSet db.DomainNameSet
}

func NewDomainNameSetController(sets db.DomainNameSet) controller.Controller {
	return controller.NewController(dnSetData{sets}, cacher.ElasticSyncFailed)
}

func (d dnSetData) Put(ctx context.Context, name string, value interface{}) error {
	return d.dnSet.PutDomainNameSet(ctx, name, value.(db.DomainNameSetSpec))
}

func (d dnSetData) List(ctx context.Context) ([]db.Meta, error) {
	return d.dnSet.ListDomainNameSets(ctx)
}

func (d dnSetData) Delete(ctx context.Context, m db.Meta) error {
	return d.dnSet.DeleteDomainNameSet(ctx, m)
}
