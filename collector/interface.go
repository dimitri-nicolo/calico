// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package collector

import "github.com/projectcalico/felix/proto"

type Collector interface {
	Start()
	ReportingChannel() chan<- *proto.DataplaneStats
	SubscribeToNflog()
}
