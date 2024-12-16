// Copyright (c) 2018-2024 Tigera, Inc. All rights reserved.

package collector

import (
	"net"
	"time"

	"github.com/google/gopacket/layers"

	"github.com/projectcalico/calico/felix/collector/types"
	"github.com/projectcalico/calico/felix/collector/types/tuple"
	dpsets "github.com/projectcalico/calico/felix/dataplane/ipsets"
	"github.com/projectcalico/calico/felix/dataplane/windows/ipsets"
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
	AddNewDomainDataplaneToIpSetsManager(ipsets.IPFamily, *dpsets.IPSetsManager)
}
