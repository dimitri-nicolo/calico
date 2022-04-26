// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package internet

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestInternetHelpers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Internet Suite")
}
