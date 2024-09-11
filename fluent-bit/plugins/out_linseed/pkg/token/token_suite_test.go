// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package token

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestLinseedOutPluginToken(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Linseed output plugin token test suite")
}
