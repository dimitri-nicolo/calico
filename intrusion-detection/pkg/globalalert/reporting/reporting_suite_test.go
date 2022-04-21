package reporting_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestReporting(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Reporting Suite")
}
