package worker

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestWorkerAbstractStruct(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Abstract Worker Test Suite")
}
