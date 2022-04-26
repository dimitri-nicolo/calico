// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package list

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestResourceListing(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "List Suite")
}
