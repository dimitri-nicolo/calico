// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package main

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestLinseedOutPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Linseed output plugin test suite")
}
