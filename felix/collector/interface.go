// Copyright (c) 2018-2023 Tigera, Inc. All rights reserved.

package collector

import (
	"net"
	"time"

	"github.com/google/gopacket/layers"

	"github.com/projectcalico/calico/felix/collector/types"
	"github.com/projectcalico/calico/felix/collector/types/tuple"
	"github.com/projectcalico/calico/felix/proto"
)

type Collector interface {
	Start() error
	ReportingChannel() chan<- *proto.DataplaneStats
	SetDNSLogReporter(types.Reporter)
	LogDNS(net.IP, net.IP, *layers.DNS, *time.Duration)
	SetL7LogReporter(types.Reporter)
	LogL7(*proto.HTTPData, *Data, tuple.Tuple, int)
	RegisterMetricsReporter(types.Reporter)
	SetDataplaneInfoReader(DataplaneInfoReader)
	SetPacketInfoReader(PacketInfoReader)
	SetConntrackInfoReader(ConntrackInfoReader)
	SetProcessInfoCache(ProcessInfoCache)
	SetDomainLookup(EgressDomainCache)
}
