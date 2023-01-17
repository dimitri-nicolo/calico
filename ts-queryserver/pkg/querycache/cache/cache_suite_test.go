// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package cache

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCommands(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Querycache Cache Suite")
}
