// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package endpoint

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestLinseedOutPluginEndpoint(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Linseed output plugin endpoint test suite")
}
