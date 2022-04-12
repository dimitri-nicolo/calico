// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package api

import (
	"strings"

	"github.com/tigera/api/pkg/lib/numorstring"
)

var (
	ProtoICMP    uint8 = 1
	ProtoIPIP    uint8 = 4
	ProtoTCP     uint8 = 6
	ProtoUDP     uint8 = 17
	ProtoESP     uint8 = 50
	ProtoICMPv6  uint8 = 58
	ProtoSCTP    uint8 = 132
	ProtoUDPLite uint8 = 136
)

var (
	// These string values are the superset of FlowLog definitions and v3 API definitions.
	protovals = map[string]uint8{
		"icmp":    ProtoICMP,
		"ipip":    ProtoIPIP,
		"tcp":     ProtoTCP,
		"udp":     ProtoUDP,
		"esp":     ProtoESP,
		"icmp6":   ProtoICMPv6,
		"icmpv6":  ProtoICMPv6,
		"sctp":    ProtoSCTP,
		"udplite": ProtoUDPLite,
	}

	// These string values are the superset of FlowLog definitions and v3 API definitions.
	protostrings = map[uint8]string{
		ProtoICMP:    "icmp",
		ProtoIPIP:    "ipip",
		ProtoTCP:     "tcp",
		ProtoUDP:     "udp",
		ProtoESP:     "esp",
		ProtoICMPv6:  "icmpv6",
		ProtoSCTP:    "sctp",
		ProtoUDPLite: "udplite",
	}
)

func GetProtocolNumber(p *numorstring.Protocol) *uint8 {
	if p == nil {
		return nil
	}
	if num, err := p.NumValue(); err == nil {
		return &num
	}
	if num, ok := protovals[strings.ToLower(p.String())]; ok {
		return &num
	}
	return nil
}

// GetProtocol tries to convert a uint8 to a numorstring.Protocol.
// It fills in a string value of a protocol where possible, otherwise
// returns a numerical numorstring.Protocol.
func GetProtocol(num uint8) numorstring.Protocol {
	if str, ok := protostrings[num]; ok {
		return numorstring.ProtocolFromString(str)
	}
	return numorstring.ProtocolFromInt(num)
}
