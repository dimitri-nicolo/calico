// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package collector

type IngressCollector interface {
	ReadLogs()
	Report() <-chan IngressLog
}

type IngressLog struct {
	SrcIp    string
	DstIp    string
	SrcPort  int32
	DstPort  int32
	Protocol string

	// Ingress specific data
	UserAgent string
	// Address of the pod traffic was routed to
	UpstreamAddress string
	// Name of the service traffic was routed to
	UpstreamService       string
	IngressControllerName string
	RequestURI            string
	L7SrcIP               string
}
