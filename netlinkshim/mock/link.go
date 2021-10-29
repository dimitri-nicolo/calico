package mock

import "github.com/vishvananda/netlink"

type MockLink struct {
	Addrs     []netlink.Addr
	LinkAttrs netlink.LinkAttrs
	LinkType  string
}

// Attrs is an implementation on the netlink.Link interface - returns the links attributes
func (l *MockLink) Attrs() *netlink.LinkAttrs {
	return &l.LinkAttrs
}

func (l *MockLink) Type() string {
	return l.LinkType
}
