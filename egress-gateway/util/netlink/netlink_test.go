package netlink

import (
	"net"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"
)

func TestNeighsEqual(t *testing.T) {
	RegisterTestingT(t)

	mac, _ := net.ParseMAC("FF:FF:FF:FF:FF:FF")
	n1 := netlink.Neigh{
		LinkIndex:    100,
		Family:       netlink.FAMILY_V4,
		State:        netlink.NUD_PERMANENT,
		HardwareAddr: mac,
		IP:           net.ParseIP("10.10.10.1"),
		LLIPAddr:     nil,
	}
	n2 := netlink.Neigh{
		LinkIndex:    100,
		Family:       netlink.FAMILY_V4,
		State:        netlink.NUD_PERMANENT,
		HardwareAddr: mac,
		IP:           net.ParseIP("10.10.10.1"),
		LLIPAddr:     nil,
	}

	Expect(NeighsEqual(n1, n2)).To(BeTrue())

	n1.LinkIndex = 101
	Expect(NeighsEqual(n1, n2)).NotTo(BeTrue())
}
