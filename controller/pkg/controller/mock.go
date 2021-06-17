// Copyright (c) 2019 Tigera Inc. All rights reserved.

package controller

import (
	"context"
	"github.com/tigera/intrusion-detection/controller/pkg/db"
)

type mockSetsData struct {
	ipSet db.IPSet
}

func (d *mockSetsData) Put(ctx context.Context, name string, value interface{}) error {
	return d.ipSet.PutIPSet(ctx, name, value.(db.IPSetSpec))
}

func (d *mockSetsData) List(ctx context.Context) ([]db.Meta, error) {
	return d.ipSet.ListIPSets(ctx)
}

func (d *mockSetsData) Delete(ctx context.Context, m db.Meta) error {
	return d.ipSet.DeleteIPSet(ctx, m)
}
