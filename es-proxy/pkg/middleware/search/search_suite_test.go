// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package search

import (
	"testing"

	"github.com/projectcalico/calico/libcalico-go/lib/testutils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMiddleware(t *testing.T) {
	testutils.HookLogrusForGinkgo()
	RegisterFailHandler(Fail)
	RunSpecs(t, "Search Middleware Suite")
}
