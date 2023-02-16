// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package waf

import (
	"testing"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
)

func setupTest(t *testing.T) func() {
	cancel := logutils.RedirectLogrusToTestingT(t)
	return func() {
		cancel()
	}
}
