package policycalc

import (
	"strings"

	"github.com/projectcalico/libcalico-go/lib/numorstring"
)

var (
	// These string values are the superset of FlowLog definitions and v3 API definitions.
	protovals = map[string]uint8{
		"icmp":    1,
		"ipip":    4,
		"tcp":     6,
		"udp":     17,
		"esp":     50,
		"icmp6":   58,
		"icmpv6":  58,
		"sctp":    132,
		"udplite": 136,
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
