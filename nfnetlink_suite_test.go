package nfnetlink_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestNfnetlink(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Nfnetlink Suite")
}
