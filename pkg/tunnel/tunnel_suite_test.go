package tunnel_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestTunnel(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tunnel Suite")
}
