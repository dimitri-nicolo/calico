package networksets

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestNetworkset(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Networkset Suite")
}
