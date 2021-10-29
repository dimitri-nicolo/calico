package proto

import "github.com/tigera/egress-gateway/proto"

func IsWireguardTunnel(ru *proto.RouteUpdate) bool {
	return (ru.Type == proto.RouteType_REMOTE_TUNNEL &&
		ru.TunnelType != nil &&
		ru.TunnelType.Wireguard)
}
