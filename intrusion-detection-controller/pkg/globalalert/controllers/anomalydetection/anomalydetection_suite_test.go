// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package anomalydetection_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"

	"testing"
)

func init() {
	// Disable truncation of output in Gomega failure messages.
	format.MaxLength = 0
}

func TestAnomalydetection(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Anomalydetection controllers Suite")
}
