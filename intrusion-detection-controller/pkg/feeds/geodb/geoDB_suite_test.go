package geodb

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestGeoDB(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GeoDB Suite")
}
