// Code generated by "stringer -type=RouteClass"; DO NOT EDIT.

package routetable

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[RouteClassLocalWorkload-0]
	_ = x[RouteClassBPFSpecial-1]
	_ = x[RouteClassWireguard-2]
	_ = x[RouteClassVXLANSameSubnet-3]
	_ = x[RouteClassVXLANTunnel-4]
	_ = x[RouteClassIPAMBlockDrop-5]
	_ = x[RouteClassEgress-6]
	_ = x[RouteClassAWSDefault-7]
	_ = x[RouteClassAWSThrow-8]
	_ = x[RouteClassTPROXY-9]
	_ = x[RouteClassMax-10]
}

const _RouteClass_name = "RouteClassLocalWorkloadRouteClassBPFSpecialRouteClassWireguardRouteClassVXLANSameSubnetRouteClassVXLANTunnelRouteClassIPAMBlockDropRouteClassEgressRouteClassAWSDefaultRouteClassAWSThrowRouteClassTPROXYRouteClassMax"

var _RouteClass_index = [...]uint8{0, 23, 43, 62, 87, 108, 131, 147, 167, 185, 201, 214}

func (i RouteClass) String() string {
	if i < 0 || i >= RouteClass(len(_RouteClass_index)-1) {
		return "RouteClass(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _RouteClass_name[_RouteClass_index[i]:_RouteClass_index[i+1]]
}
