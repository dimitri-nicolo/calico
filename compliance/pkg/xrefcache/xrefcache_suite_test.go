// Copyright (c) 2019 Tigera, Inc. SelectAll rights reserved.
package xrefcache

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/libcalico-go/lib/testutils"
)

func TestXrefCache(t *testing.T) {
	testutils.HookLogrusForGinkgo()
	RegisterFailHandler(Fail)
	RunSpecs(t, "Xref XrefCache Suite")
}
