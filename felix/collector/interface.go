// Copyright (c) 2018-2023 Tigera, Inc. All rights reserved.

package collector

import (
	"net"
	"time"

	"github.com/google/gopacket/layers"

	"github.com/projectcalico/calico/felix/collector/dataplane"
	"github.com/projectcalico/calico/felix/collector/dnslog"
	"github.com/projectcalico/calico/felix/collector/l7log"
	"github.com/projectcalico/calico/felix/collector/types/tuple"
	"github.com/projectcalico/calico/felix/proto"
)

type Collector interface {
	Start() error
	ReportingChannel() chan<- *proto.DataplaneStats
	SetDNSLogReporter(dnslog.ReporterInterface)
	LogDNS(net.IP, net.IP, *layers.DNS, *time.Duration)
	SetL7LogReporter(l7log.ReporterInterface)
	LogL7(*proto.HTTPData, *dataplane.Data, tuple.Tuple, int)
	SetPacketInfoReader(dataplane.PacketInfoReader)
	SetConntrackInfoReader(dataplane.ConntrackInfoReader)
	SetProcessInfoCache(dataplane.ProcessInfoCache)
	SetDomainLookup(dataplane.EgressDomainCache)
}
