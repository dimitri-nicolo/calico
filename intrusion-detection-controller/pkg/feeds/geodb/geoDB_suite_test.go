package geodb

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestGeoDB(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GeoDB Suite")
}
