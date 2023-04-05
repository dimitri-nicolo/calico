// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package accesslog_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Access Log Suite")
}
