// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package keyselector

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestLabelSelector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Key Selector Suite")
}
