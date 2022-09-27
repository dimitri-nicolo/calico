// Copyright 2022 Tigera Inc. All rights reserved.
package sync

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestClientUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Client suite")
}
