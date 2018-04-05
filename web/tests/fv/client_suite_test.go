// Copyright (c) 2018 Tigera, Inc. All rights reserved.
package fv

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/libcalico-go/lib/testutils"
)

func TestClient(t *testing.T) {
	testutils.HookLogrusForGinkgo()
	RegisterFailHandler(Fail)
	RunSpecs(t, "queryserver FV Suite")
}
