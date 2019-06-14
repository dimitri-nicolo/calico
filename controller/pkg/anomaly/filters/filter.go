// Copyright 2019 Tigera Inc. All rights reserved.

package filters

import (
	"context"

	"github.com/tigera/intrusion-detection/controller/pkg/elastic"
)

type Filter interface {
	Filter(context.Context, []elastic.RecordSpec) ([]elastic.RecordSpec, error)
}

type Filters []Filter

func (filters Filters) Filter(ctx context.Context, in []elastic.RecordSpec) (out []elastic.RecordSpec, err error) {
	out = append(in)
	for _, f := range filters {
		out, err = f.Filter(ctx, out)
		if err != nil {
			return
		}
	}
	return
}

type NilFilter struct{}

func (NilFilter) Filter(ctx context.Context, rs []elastic.RecordSpec) ([]elastic.RecordSpec, error) {
	return rs, nil
}
