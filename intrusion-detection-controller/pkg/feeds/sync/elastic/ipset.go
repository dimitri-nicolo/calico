// Copyright 2019 Tigera Inc. All rights reserved.

package elastic

import (
	"context"

	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/controller"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/db"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/feeds/cacher"
)

type ipSetData struct {
	ipSet db.IPSet
}

func NewIPSetController(ipSet db.IPSet) controller.Controller {
	return controller.NewController(ipSetData{ipSet}, cacher.ElasticSyncFailed)
}

func (d ipSetData) Put(ctx context.Context, name string, value interface{}) error {
	return d.ipSet.PutIPSet(ctx, name, value.(db.IPSetSpec))
}

func (d ipSetData) List(ctx context.Context) ([]db.Meta, error) {
	return d.ipSet.ListIPSets(ctx)
}

func (d ipSetData) Delete(ctx context.Context, m db.Meta) error {
	return d.ipSet.DeleteIPSet(ctx, m)
}
