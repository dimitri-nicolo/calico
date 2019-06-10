// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package collector

// TODO: Flesh this out
type nginxCollector struct{}

func NewNginxCollector() IngressCollector {
	return &nginxCollector{}
}

func (nc *nginxCollector) ReadLogs() {
	// TODO: Fill this in
}

func (nc *nginxCollector) Report() <-chan IngressLog {
	// TODO: Have this return real data
	// Returns mocked data for testing now.
	logs := make(chan IngressLog)
	logs <- IngressLog{
		SrcIp:    "10.100.10.1",
		DstIp:    "10.100.100.1",
		SrcPort:  int32(40),
		DstPort:  int32(50),
		Protocol: "http",
	}

	logs <- IngressLog{
		SrcIp:    "10.100.10.1",
		DstIp:    "10.100.100.1",
		SrcPort:  int32(60),
		DstPort:  int32(70),
		Protocol: "http",
	}
	return logs
}
