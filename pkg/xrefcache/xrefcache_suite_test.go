// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package xrefcache

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestXrefCache(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Xref XrefCache Suite")
}
