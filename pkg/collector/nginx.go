// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package collector

// TODO: Flesh this out
type nginxCollector struct {
	collectedLogs chan IngressLog
}

func NewNginxCollector() IngressCollector {
	return &nginxCollector{
		collectedLogs: make(chan IngressLog),
	}
}

func (nc *nginxCollector) ReadLogs() {
	// TODO: Have this return real data
	// Returns mocked data for testing now.
	nc.collectedLogs <- IngressLog{
		SrcIp:    "10.100.10.1",
		DstIp:    "10.100.100.1",
		SrcPort:  int32(40),
		DstPort:  int32(50),
		Protocol: "tcp",
	}

	nc.collectedLogs <- IngressLog{
		SrcIp:    "10.100.10.1",
		DstIp:    "10.100.100.1",
		SrcPort:  int32(60),
		DstPort:  int32(70),
		Protocol: "tcp",
	}
}

func (nc *nginxCollector) Report() <-chan IngressLog {
	return nc.collectedLogs
}
