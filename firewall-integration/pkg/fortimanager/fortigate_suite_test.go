// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package fortimanager_test

import (
	"testing"

	"github.com/projectcalico/calico/libcalico-go/lib/testutils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	testutils.HookLogrusForGinkgo()
	RunSpecs(t, "FortiGate Test Suite")
}
