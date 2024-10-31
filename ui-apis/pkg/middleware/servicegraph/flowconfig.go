// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/calico/lma/pkg/k8s"
)

type FlowConfig struct {
	L3FlowFlushInterval time.Duration
	L7FlowFlushInterval time.Duration
	DNSLogFlushInterval time.Duration
}

func GetFlowConfig(ctx context.Context, cs k8s.ClientSet) (*FlowConfig, error) {
	// Assume the defaults unless explicitly overridden.
	fc := &FlowConfig{
		L3FlowFlushInterval: time.Minute * 5,
		L7FlowFlushInterval: time.Minute * 5,
		DNSLogFlushInterval: time.Minute * 5,
	}

	felixConfig, err := cs.ProjectcalicoV3().FelixConfigurations().Get(ctx, "default", v1.GetOptions{})
	if err != nil {
		if errors.IsForbidden(err) {
			// If forbidden just use the defaults. We shouldn't prevent graph access in this case.
			return fc, nil
		}
		return nil, err
	}

	if felixConfig.Spec.FlowLogsFlushInterval != nil {
		fc.L7FlowFlushInterval = felixConfig.Spec.FlowLogsFlushInterval.Duration
	}
	if felixConfig.Spec.L7LogsFlushInterval != nil {
		fc.L7FlowFlushInterval = felixConfig.Spec.L7LogsFlushInterval.Duration
	}
	if felixConfig.Spec.DNSLogsFlushInterval != nil {
		fc.DNSLogFlushInterval = felixConfig.Spec.DNSLogsFlushInterval.Duration
	}

	return fc, nil
}
