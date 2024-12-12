// Copyright (c) 2024 Tigera. All rights reserved.
package client

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAuth(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "QueryServerClient Test Suite")
}
