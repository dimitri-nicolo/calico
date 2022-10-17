package net

import (
	"fmt"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"

	"github.com/projectcalico/calico/egress-gateway/netlinkshim"
)

// GetDefaultRoute inspects routing table 254 of the underlying controlplane for the most generic
// route programmed. Roughly equivalent to running `ip route show 0.0.0.0/0` in the cmdline
func GetDefaultRoute(nl netlinkshim.Handle) (route *netlink.Route, err error) {
	filter := netlink.Route{
		Table: unix.RT_TABLE_MAIN,
		Dst:   nil,
	}

	routes, err := nl.RouteListFiltered(
		netlink.FAMILY_V4,
		&filter,
		netlink.RT_FILTER_DST|netlink.RT_FILTER_TABLE,
	)
	if err != nil {
		return nil, err
	}
	if len(routes) == 0 {
		return nil, fmt.Errorf("could not find default route")
	}

	route = &(routes[0])
	return route, err
}
