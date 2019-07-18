package fv_test

import (
	"os"
	"testing"

	"github.com/projectcalico/libcalico-go/lib/testutils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestESProxyFVs(t *testing.T) {
	testutils.HookLogrusForGinkgo()
	RegisterFailHandler(Fail)
	RunSpecs(t, "FV Test Suite")
}

func getEnvOrDefaultString(key string, defaultValue string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	} else {
		return val
	}
}
