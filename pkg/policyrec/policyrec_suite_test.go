package policyrec_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPolicyrec(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Policyrec Suite")
}
