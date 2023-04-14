// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package calicoresources

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/libcalico-go/lib/testutils"
)

func TestStagedNetworkPolicy(t *testing.T) {
	testutils.HookLogrusForGinkgo()
	RegisterFailHandler(Fail)
	RunSpecs(t, "StagedNetworkPolicy Suite")
}
