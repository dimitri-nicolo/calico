// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package docindex

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestLabelSelector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Document index Suite")
}
