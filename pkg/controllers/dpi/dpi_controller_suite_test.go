// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package dpi_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/libcalico-go/lib/testutils"
)

func TestConverter(t *testing.T) {
	testutils.HookLogrusForGinkgo()
	RegisterFailHandler(Fail)
	RunSpecs(t, "DPI Controller Suite")
}
