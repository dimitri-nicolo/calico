// Copyright (c) 2018-2019 Tigera, Inc. All rights reserved.

package collector

import (
	"net"

	"github.com/google/gopacket/layers"

	"github.com/projectcalico/felix/proto"
)

type Collector interface {
	Start()
	ReportingChannel() chan<- *proto.DataplaneStats
	SubscribeToNflog()
	SetDNSLogReporter(reporter DNSLogReporterInterface)
	LogDNS(src, dst net.IP, dns *layers.DNS)
}
