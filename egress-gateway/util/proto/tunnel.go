package proto

import "github.com/projectcalico/calico/felix/proto"

// IsWireguardTunnel returns true if a route update represents a remote host's WireGuard tunnel
func IsWireguardTunnel(ru *proto.RouteUpdate) bool {
	return (ru.Type == proto.RouteType_REMOTE_TUNNEL ||
		ru.Type == proto.RouteType_LOCAL_TUNNEL) &&
		ru.TunnelType != nil &&
		ru.TunnelType.Wireguard
}

// IsVXLANTunnel returns true if a route update represents a remote host's VXLAN tunnel
func IsVXLANTunnel(ru *proto.RouteUpdate) bool {
	return (ru.Type == proto.RouteType_REMOTE_TUNNEL ||
		ru.Type == proto.RouteType_LOCAL_TUNNEL) &&
		ru.TunnelType != nil &&
		ru.TunnelType.Vxlan
}

// IsIPIPTunnel returns true if a route update represents a remote host's IPIP tunnel
func IsIPIPTunnel(ru *proto.RouteUpdate) bool {
	return (ru.Type == proto.RouteType_REMOTE_TUNNEL ||
		ru.Type == proto.RouteType_LOCAL_TUNNEL) &&
		ru.TunnelType != nil &&
		ru.TunnelType.Ipip
}

// IsHostTunnel determines if a route update represents a tunnel device in a remote host
func IsHostTunnel(ru *proto.RouteUpdate) bool {
	return (ru.Type == proto.RouteType_REMOTE_TUNNEL ||
		ru.Type == proto.RouteType_LOCAL_TUNNEL) &&
		ru.TunnelType != nil
}
