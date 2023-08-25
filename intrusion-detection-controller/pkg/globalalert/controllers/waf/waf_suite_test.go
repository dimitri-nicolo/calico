package waf_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestWafAlertGeneration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "WAF controllers Suite")
}
