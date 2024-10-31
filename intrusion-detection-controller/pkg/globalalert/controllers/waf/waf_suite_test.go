package waf_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestWafAlertGeneration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "WAF controllers Suite")
}
