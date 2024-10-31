// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package fortimanager_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/libcalico-go/lib/testutils"
)

func TestHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	testutils.HookLogrusForGinkgo()
	RunSpecs(t, "FortiGate Test Suite")
}
