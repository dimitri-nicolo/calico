// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package state

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestTunnelManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "State Test Suite")
}
