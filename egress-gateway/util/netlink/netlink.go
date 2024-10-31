package netlink

import (
	"fmt"
	"strings"

	nl "github.com/vishvananda/netlink"
)

// NeighsEqual compares two netlink neighbours and decides if they are equivalent.
func NeighsEqual(n1 nl.Neigh, n2 nl.Neigh) bool {
	return (n1.IP.Equal(n2.IP) &&
		strings.Compare(n1.HardwareAddr.String(), n2.HardwareAddr.String()) == 0 &&
		n1.LLIPAddr.Equal(n2.LLIPAddr) &&
		n1.LinkIndex == n2.LinkIndex &&
		n1.Family == n2.Family) &&
		n1.State == n2.State &&
		n1.Flags == n2.Flags
}

// KeyForNeigh generates unique strings for neighs, in such a way that key collisions roughly correlate with kernel collisions
func KeyForNeigh(n nl.Neigh) string {
	linkIdx := n.LinkIndex
	ip := n.IP.String()
	family := n.Family
	flags := n.Flags

	// mac address isnt factored into this, as tools such as `ip neigh` actually allow for duplicate MAC entries with different IP's
	key := fmt.Sprintf("%v-%v-%v-%v", linkIdx, family, ip, flags)
	return key
}
