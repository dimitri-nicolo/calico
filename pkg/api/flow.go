// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package api

import (
	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

const (
	FlowLogGlobalNamespace        = "-"
	FlowLogEndpointTypeWEP        = "wep"
	FlowLogEndpointTypeHEP        = "hep"
	FlowLogEndpointTypeNetworkSet = "ns"
	FlowLogEndpointTypeNetwork    = "net"
	FlowLogNetworkPublic          = "pub"
	FlowLogNetworkPrivate         = "pvt"
)

// Container type to hold the EndpointsReportFlow and/or an error.
type FlowLogResult struct {
	*apiv3.EndpointsReportFlow
	Err error
}
