// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package cache

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestCommands(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Querycache Cache Suite")
}
