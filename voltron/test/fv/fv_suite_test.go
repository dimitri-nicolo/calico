// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package fv_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestFv(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "[FV] Voltron-Guardian e2e Suite")
}
