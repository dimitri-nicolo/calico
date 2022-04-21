package podtemplate_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestPodtemplate(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Podtemplate Suite")
}
