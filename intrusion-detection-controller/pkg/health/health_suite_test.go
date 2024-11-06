package health

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestIDCHealthSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "IDC Health Suite Tests")
}
