package mock

import (
	"strings"
)

type OperationFlag uint32

const (
	OpLinkByName OperationFlag = 1 << iota
	OpMissingLinkByName
	OpLinkAdd
	OpUnsupportedLinkAdd
	OpLinkDel
	OpLinkSetUp
	OpRouteListFiltered
	OpRouteReplace
	OpRouteAdd
	OpRouteDel
	OpNeighList
	OpNeighSet
	OpNeighDel
	OpDelete
)

// String returns a list of operations flagged by the OperationFlag
func (o OperationFlag) String() string {
	ops := []string{}
	if o == 0 {
		return "None"
	}
	if o&OpLinkByName != 0 {
		ops = append(ops, "LinkByName")
	}
	if o&OpMissingLinkByName != 0 {
		ops = append(ops, "MissingLinkByName")
	}
	if o&OpLinkAdd != 0 {
		ops = append(ops, "LinkAdd")
	}
	if o&OpUnsupportedLinkAdd != 0 {
		ops = append(ops, "LinkAdd")
	}
	if o&OpLinkDel != 0 {
		ops = append(ops, "LinkDel")
	}
	if o&OpLinkSetUp != 0 {
		ops = append(ops, "LinkSetUp")
	}
	if o&OpRouteListFiltered != 0 {
		ops = append(ops, "RouteListFiltered")
	}
	if o&OpRouteAdd != 0 {
		ops = append(ops, "RouteAdd")
	}
	if o&OpRouteReplace != 0 {
		ops = append(ops, "RouteReplace")
	}
	if o&OpRouteDel != 0 {
		ops = append(ops, "RouteDel")
	}
	if o&OpNeighList != 0 {
		ops = append(ops, "NeighList")
	}
	if o&OpNeighSet != 0 {
		ops = append(ops, "NeighSet")
	}
	if o&OpNeighDel != 0 {
		ops = append(ops, "NeighDel")
	}
	if o&OpDelete != 0 {
		ops = append(ops, "Delete")
	}

	return strings.Join(ops, ",")
}
