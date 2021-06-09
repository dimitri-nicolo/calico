// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	"context"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tigera/lma/pkg/k8s"
)

type FlowConfig struct {
	L3FlowFlushInterval time.Duration
	L7FlowFlushInterval time.Duration
	DNSLogFlushInterval time.Duration
}

func GetFlowConfig(ctx context.Context, cs k8s.ClientSet) (*FlowConfig, error) {
	felixConfig, err := cs.FelixConfigurations().Get(ctx, "default", v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// Assume the default flush interval unless explicitly overridden.
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
