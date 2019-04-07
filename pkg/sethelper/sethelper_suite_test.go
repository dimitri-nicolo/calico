// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package sethelper

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSetHelper(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Set helper Suite")
}
