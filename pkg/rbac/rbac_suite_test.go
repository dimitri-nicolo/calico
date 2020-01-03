// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package rbac_test

import (
	"testing"

	"github.com/projectcalico/libcalico-go/lib/testutils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPolicyRec(t *testing.T) {
	testutils.HookLogrusForGinkgo()
	RegisterFailHandler(Fail)
	RunSpecs(t, "Calculator Suite")
}
