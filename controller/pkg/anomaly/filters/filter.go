// Copyright 2019 Tigera Inc. All rights reserved.

package filters

import "github.com/tigera/intrusion-detection/controller/pkg/elastic"

type Filter interface {
	Filter([]elastic.RecordSpec) ([]elastic.RecordSpec, error)
}

type Filters []Filter

func (filters Filters) Filter(in []elastic.RecordSpec) (out []elastic.RecordSpec, err error) {
	out = append(in)
	for _, f := range filters {
		out, err = f.Filter(out)
		if err != nil {
			return
		}
	}
	return
}

type NilFilter struct{}

func (NilFilter) Filter(rs []elastic.RecordSpec) ([]elastic.RecordSpec, error) {
	return rs, nil
}
