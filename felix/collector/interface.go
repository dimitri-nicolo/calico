// Copyright (c) 2018-2021 Tigera, Inc. All rights reserved.

package collector

import (
	"net"
	"time"

	"github.com/google/gopacket/layers"

	"github.com/projectcalico/calico/felix/proto"
)

type Collector interface {
	Start() error
	ReportingChannel() chan<- *proto.DataplaneStats
	SetDNSLogReporter(reporter DNSLogReporterInterface)
	LogDNS(server, client net.IP, dns *layers.DNS, latencyIfKnown *time.Duration)
	SetL7LogReporter(reporter L7LogReporterInterface)
	LogL7(hd *proto.HTTPData, data *Data, tuple Tuple, httpDataCount int)
	SetPacketInfoReader(pir PacketInfoReader)
	SetConntrackInfoReader(cir ConntrackInfoReader)
	SetProcessInfoCache(pir ProcessInfoCache)
	SetDomainLookup(dlu EgressDomainCache)
}
