// Copyright 2019 Tigera Inc. All rights reserved.

package health

import (
	"context"
)

type MockPinger struct {
	err error
}

func (p MockPinger) Ping(context.Context) error {
	return p.err
}

type MockReadier struct {
	ready bool
}

func (r MockReadier) Ready() bool {
	return r.ready
}
