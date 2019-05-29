// Copyright 2019 Tigera Inc. All rights reserved.

package filters

import "github.com/tigera/intrusion-detection/controller/pkg/elastic"

type MockFilter struct {
	RS  []elastic.RecordSpec
	Err error
}

func (f MockFilter) Filter([]elastic.RecordSpec) ([]elastic.RecordSpec, error) {
	return f.RS, f.Err
}
