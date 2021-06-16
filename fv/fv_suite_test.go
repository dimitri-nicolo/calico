// Copyright (c) 2021 Tigera. All rights reserved.
package fv_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestFv(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Fv Suite")
}
