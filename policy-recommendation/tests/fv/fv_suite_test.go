// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package fv

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/testutils"
)

func init() {
	testutils.HookLogrusForGinkgo()
	logrus.SetLevel(logrus.InfoLevel)
}

func TestFv(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Fv Suite")
}
