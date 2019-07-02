// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package collector

import (
	"errors"
	"regexp"
	"strings"

	"github.com/google/gopacket/layers"
)

func NewDNSMetaSpecFromGoPacket(dns *layers.DNS) (DNSMeta, DNSSpec, error) {
	if len(dns.Questions) == 0 {
		return DNSMeta{}, DNSSpec{}, errors.New("No questions in DNS packet")
	}

	spec := newDNSSpecFromGoPacket(dns)
	meta := newDNSMetaFromSpecAndGoPacket(dns, spec)

	return meta, spec, nil
}

func newDNSSpecFromGoPacket(dns *layers.DNS) DNSSpec {
	spec := DNSSpec{
		RRSets:       make(DNSRRSets),
		Servers:      make(map[EndpointMetadata]DNSLabels),
		ClientLabels: nil,
		DNSStats: DNSStats{
			Count: 1,
		},
	}
	for _, rr := range append(append(dns.Answers, dns.Additionals...), dns.Authorities...) {
		spec.RRSets.Add(newDNSNameRDataFromGoPacketRR(rr))
	}
	// TODO server metadata
	// TODO client labels
	return spec
}

func newDNSMetaFromSpecAndGoPacket(dns *layers.DNS, spec DNSSpec) DNSMeta {
	return DNSMeta{
		// TODO ClientMeta
		Question: DNSName{
			Name:  canonicalizeDNSName(dns.Questions[0].Name),
			Class: DNSClass(dns.Questions[0].Class),
			Type:  DNSType(dns.Questions[0].Type),
		},
		ResponseCode: dns.ResponseCode,
		RRSetsString: spec.RRSets.String(),
	}
}

func newDNSNameRDataFromGoPacketRR(rr layers.DNSResourceRecord) (DNSName, DNSRData) {
	name := DNSName{
		Name:  canonicalizeDNSName(rr.Name),
		Class: DNSClass(rr.Class),
		Type:  DNSType(rr.Type),
	}

	rdata := DNSRData{
		Raw:     rr.Data,
		Decoded: getRRDecoded(rr),
	}

	return name, rdata
}

func getRRDecoded(rr layers.DNSResourceRecord) interface{} {
	switch rr.Type {
	case layers.DNSTypeA, layers.DNSTypeAAAA:
		return rr.IP
	case layers.DNSTypeNS:
		return string(rr.NS)
	case layers.DNSTypeCNAME:
		return string(rr.CNAME)
	case layers.DNSTypePTR:
		return string(rr.PTR)
	case layers.DNSTypeTXT:
		return rr.TXTs
	case layers.DNSTypeSOA:
		return rr.SOA
	case layers.DNSTypeSRV:
		return rr.SRV
	case layers.DNSTypeMX:
		return rr.MX
	default:
		return rr.Data
	}
}

func canonicalizeDNSName(name []byte) string {
	return regexp.MustCompile(`\.\.+`).ReplaceAllString(strings.ToLower(strings.Trim(string(name), ".")), ".")
}
