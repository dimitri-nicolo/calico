package fv_test

import (
	"testing"

	"github.com/projectcalico/calico/libcalico-go/lib/testutils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestESProxyFVs(t *testing.T) {
	testutils.HookLogrusForGinkgo()
	RegisterFailHandler(Fail)
	RunSpecs(t, "FV Test Suite")
}
