package tunnelmgr

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestTunnelManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tunnel Manager Suite")
}
