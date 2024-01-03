// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package fv_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/libcalico-go/lib/testutils"
)

func TestFv(t *testing.T) {
	RegisterFailHandler(Fail)
	testutils.HookLogrusForGinkgo()
	RunSpecs(t, "[FV] Voltron-Guardian e2e Suite")
}
