// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package managed_cluster_controller_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/libcalico-go/lib/testutils"
)

func TestManagedClusterController(t *testing.T) {
	testutils.HookLogrusForGinkgo()
	RegisterFailHandler(Fail)
	RunSpecs(t, "ManagedCluster Controller Suite")
}
