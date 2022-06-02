// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package user

import (
	"testing"

	"github.com/projectcalico/calico/libcalico-go/lib/testutils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSuite(t *testing.T) {
	testutils.HookLogrusForGinkgo()
	RegisterFailHandler(Fail)
	RunSpecs(t, "User Test Suite")
}
