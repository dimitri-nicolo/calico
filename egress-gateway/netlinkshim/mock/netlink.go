// package mock mimics the actual netlink dataplane for testing purposes.
// borrows heavily from Felix/netlinkshim/mocknetlink
package mock

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netlink/nl"
	"golang.org/x/sys/unix"

	"github.com/tigera/egress-gateway/netlinkshim"
	netlinkutil "github.com/tigera/egress-gateway/util/netlink"
)

// MockHandle mimics a netlink handle without any actual kernel programming
type MockHandle struct {
	Failures        OperationFlag
	PersistFailures bool
	NetlinkOpen     bool
	mutex           sync.Mutex

	NumLinkAddCalls int

	ImmediateLinkUp bool
	LinksByName     map[string]netlink.Link
	LinksByIndex    map[string]netlink.Link

	RoutesByKey map[string]netlink.Route
	// counters on the numbers of deletions/updates made to each route
	DeletedRoutesByKey map[string]int
	UpdatedRoutesByKey map[string]int // updates include deletions

	NeighsByKey map[string]netlink.Neigh
	// counters on the numbers of deletions/updates made to each neigh
	DeletedNeighsByKey map[string]int
	UpdatedNeighsByKey map[string]int //updates include deletions
}

func New() *MockHandle {
	h := &MockHandle{
		mutex:              sync.Mutex{},
		LinksByName:        make(map[string]netlink.Link),
		LinksByIndex:       make(map[string]netlink.Link),
		RoutesByKey:        make(map[string]netlink.Route),
		DeletedRoutesByKey: make(map[string]int),
		UpdatedRoutesByKey: make(map[string]int),
		NeighsByKey:        make(map[string]netlink.Neigh),
		DeletedNeighsByKey: make(map[string]int),
		UpdatedNeighsByKey: make(map[string]int),
		NetlinkOpen:        true,
	}

	return h
}

func (h *MockHandle) ResetDeltas() {
	h.DeletedRoutesByKey = make(map[string]int)
	h.UpdatedRoutesByKey = make(map[string]int)
	h.DeletedNeighsByKey = make(map[string]int)
	h.UpdatedNeighsByKey = make(map[string]int)
	h.NumLinkAddCalls = 0
}

// validate the mock netlink adheres to the nelink interface
var _ netlinkshim.Handle = (*MockHandle)(nil)

func (h *MockHandle) LinkByName(name string) (netlink.Link, error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if !h.NetlinkOpen {
		return nil, ErrorNetlinkClosed
	}

	if h.shouldFailFor(OpLinkByName) {
		return nil, ErrorGeneric
	}

	link, ok := h.LinksByName[name]
	if h.shouldFailFor(OpMissingLinkByName) || !ok {
		return nil, ErrorNotFound
	}
	return link, nil
}

func (h *MockHandle) LinkByIndex(index int) (netlink.Link, error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if !h.NetlinkOpen {
		return nil, ErrorNetlinkClosed
	}

	if h.shouldFailFor(OpLinkByName) {
		return nil, ErrorGeneric
	}

	link, ok := h.LinksByIndex[strconv.Itoa(index)]
	if h.shouldFailFor(OpMissingLinkByName) || !ok {
		return nil, ErrorNotFound
	}
	return link, nil
}

func (h *MockHandle) LinkAdd(link netlink.Link) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if !h.NetlinkOpen {
		return ErrorNetlinkClosed
	}
	if h.shouldFailFor(OpLinkAdd) {
		return ErrorGeneric
	}
	if h.shouldFailFor(OpUnsupportedLinkAdd) {
		return ErrorNotSupported
	}

	attrs := link.Attrs()
	if _, ok := h.LinksByName[attrs.Name]; ok {
		return ErrorAlreadyExists
	}

	(*attrs).Index = 100 + h.NumLinkAddCalls
	h.LinksByName[attrs.Name] = &MockLink{
		LinkAttrs: *attrs,
		LinkType:  link.Type(),
	}
	// different mapping for the same link - convenient when getting link by Index
	h.LinksByIndex[strconv.Itoa(attrs.Index)] = h.LinksByName[attrs.Name]

	h.NumLinkAddCalls++
	return nil
}

func (h *MockHandle) LinkDel(link netlink.Link) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if !h.NetlinkOpen {
		return ErrorNetlinkClosed
	}

	if h.shouldFailFor(OpLinkDel) {
		return ErrorGeneric
	}

	if _, ok := h.LinksByName[link.Attrs().Name]; !ok {
		return ErrorNotFound
	}

	delete(h.LinksByName, link.Attrs().Name)
	return nil
}

func (h *MockHandle) LinkSetUp(link netlink.Link) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if !h.NetlinkOpen {
		return ErrorNetlinkClosed
	}

	if h.shouldFailFor(OpLinkSetUp) {
		return ErrorGeneric
	}

	name := link.Attrs().Name
	if l, ok := h.LinksByName[name]; ok {
		attrs := *(l.Attrs())

		if h.ImmediateLinkUp {
			attrs.Flags |= net.FlagUp
		}
		attrs.RawFlags |= syscall.IFF_RUNNING
		return nil
	}
	return ErrorNotFound
}

//RouteList is a convenience-wrapper of RouteListFiltered - credit github.com/vishvananda/netlink
func (h *MockHandle) RouteList(link netlink.Link, family int) ([]netlink.Route, error) {
	var routeFilter *netlink.Route
	if link != nil {
		routeFilter = &netlink.Route{
			LinkIndex: link.Attrs().Index,
		}
	}
	return h.RouteListFiltered(family, routeFilter, netlink.RT_FILTER_OIF)
}

func (h *MockHandle) RouteListFiltered(family int, filter *netlink.Route, filterMask uint64) ([]netlink.Route, error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if !h.NetlinkOpen {
		return nil, ErrorNetlinkClosed
	}

	var routes []netlink.Route
	for _, route := range h.RoutesByKey {
		// filter by output interface if required
		if filter != nil && filterMask&netlink.RT_FILTER_OIF != 0 && route.LinkIndex != filter.LinkIndex {
			continue
		}

		if route.Table == 0 {
			// for unspec'd table, infer table index
			route.Table = unix.RT_TABLE_MAIN
		}
		if (filter == nil || filterMask&netlink.RT_FILTER_TABLE == 0) && route.Table != unix.RT_TABLE_MAIN {
			// Not filtering by table and does not match main table.
			continue
		}
		if filter != nil && filterMask&netlink.RT_FILTER_TABLE != 0 && route.Table != filter.Table {
			// Filtering by table and table indices do not match.
			continue
		}
		routes = append(routes, route)
	}

	return routes, nil
}

func (h *MockHandle) RouteReplace(route *netlink.Route) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if !h.NetlinkOpen {
		return ErrorNetlinkClosed
	}

	if h.shouldFailFor(OpRouteReplace) {
		return ErrorGeneric
	}

	key := KeyForRoute(route)
	r := *route
	h.RoutesByKey[key] = r
	h.UpdatedRoutesByKey[key]++
	return nil
}

func (h *MockHandle) RouteAdd(route *netlink.Route) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if !h.NetlinkOpen {
		return ErrorNetlinkClosed
	}

	if h.shouldFailFor(OpRouteAdd) {
		return ErrorGeneric
	}

	key := KeyForRoute(route)
	if _, ok := h.RoutesByKey[key]; ok {
		return ErrorAlreadyExists
	} else {
		r := *route
		h.RoutesByKey[key] = r
		h.UpdatedRoutesByKey[key]++
		return nil
	}
}

func (h *MockHandle) RouteDel(route *netlink.Route) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if !h.NetlinkOpen {
		return ErrorNetlinkClosed
	}

	if h.shouldFailFor(OpRouteDel) {
		return ErrorGeneric
	}

	key := KeyForRoute(route)
	h.DeletedRoutesByKey[key]++
	if _, ok := h.RoutesByKey[key]; ok {
		delete(h.RoutesByKey, key)
		h.UpdatedRoutesByKey[key]++
	}

	return nil
}

func (h *MockHandle) NeighList(linkIndex, family int) ([]netlink.Neigh, error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if !h.NetlinkOpen {
		return nil, ErrorNetlinkClosed
	}

	if h.shouldFailFor(OpNeighList) {
		return nil, ErrorGeneric
	}

	var neighs []netlink.Neigh
	for _, n := range h.NeighsByKey {
		if (n.Family == family || family == netlink.FAMILY_ALL) && (n.LinkIndex == linkIndex) {
			neighs = append(neighs, n)
		}
	}
	return neighs, nil
}

func (h *MockHandle) NeighSet(neigh *netlink.Neigh) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if !h.NetlinkOpen {
		return ErrorNetlinkClosed
	}

	if h.shouldFailFor(OpNeighList) {
		return ErrorGeneric
	}

	if neigh.Family == 0 {
		(*neigh).Family = nl.GetIPFamily(neigh.IP)
	}

	key := netlinkutil.KeyForNeigh(*neigh)
	h.NeighsByKey[key] = *neigh
	log.Debugf("NETLINK setting neigh with key '%s': %+v", key, neigh)
	h.UpdatedNeighsByKey[key]++

	return nil
}

func (h *MockHandle) NeighDel(neigh *netlink.Neigh) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if !h.NetlinkOpen {
		return ErrorNetlinkClosed
	}

	if h.shouldFailFor(OpNeighDel) {
		return ErrorGeneric
	}

	key := netlinkutil.KeyForNeigh(*neigh)
	h.DeletedNeighsByKey[key]++
	if _, ok := h.NeighsByKey[key]; ok {
		delete(h.NeighsByKey, key)
		h.UpdatedNeighsByKey[key]++
	}

	return nil
}

func (h *MockHandle) Delete() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.NetlinkOpen = false
}

// shouldFailFor reads any failures flagged by h.Failures to decide whether some Operation 'op' would/should fail
func (h *MockHandle) shouldFailFor(op OperationFlag) bool {
	flagPresent := h.Failures&op != 0

	// if we choose not to persist failures, clear the one we are targeting
	if !h.PersistFailures {
		h.Failures &^= op
	}

	if flagPresent {
		log.WithField("operation", op).Warn("mocknetlink: triggering failure")
	}
	return flagPresent
}

func KeyForRoute(route *netlink.Route) string {
	table := route.Table
	if table == 0 {
		table = unix.RT_TABLE_MAIN
	}
	key := fmt.Sprintf("%v-%v-%v", table, route.LinkIndex, route.Dst)
	return key
}
