// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	"context"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type FlowConfig struct {
	L3FlowFlushInterval time.Duration
	L7FlowFlushInterval time.Duration
	DNSLogFlushInterval time.Duration
}

func GetFlowConfig(ctx context.Context, rd *RequestData) (*FlowConfig, error) {
	felixConfig, err := rd.appCluster.FelixConfigurations().Get(ctx, "default", v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// Assume the default flush interval unless explicily overridden.
	l3FlowFlushInterval := time.Minute * 5
	if felixConfig.Spec.FlowLogsFlushInterval != nil {
		l3FlowFlushInterval = felixConfig.Spec.FlowLogsFlushInterval.Duration
	}
	l7LogsFlushInterval := time.Minute * 5
	if felixConfig.Spec.L7LogsFlushInterval != nil {
		l7LogsFlushInterval = felixConfig.Spec.L7LogsFlushInterval.Duration
	}
	dnsLogsFlushInterval := time.Minute * 5
	if felixConfig.Spec.DNSLogsFlushInterval != nil {
		dnsLogsFlushInterval = felixConfig.Spec.DNSLogsFlushInterval.Duration
	}

	return &FlowConfig{
		L3FlowFlushInterval: l3FlowFlushInterval,
		L7FlowFlushInterval: l7LogsFlushInterval,
		DNSLogFlushInterval: dnsLogsFlushInterval,
	}, nil
}
