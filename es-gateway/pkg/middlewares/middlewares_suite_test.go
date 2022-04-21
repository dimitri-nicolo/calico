// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package middlewares_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMiddlewares(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Middlewares Suite")
}
