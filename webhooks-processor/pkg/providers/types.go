// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package providers

import (
	"context"
	"time"

	lsApi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

type RateLimiterConfig struct {
	RateLimiterDuration time.Duration
	RateLimiterCount    uint
}

type RetryConfig struct {
	RequestTimeout time.Duration
	RetryDuration  time.Duration
	RetryTimes     uint
}

type ProviderConfiguration struct {
	RateLimiterConfig RateLimiterConfig
	RetryConfig       RetryConfig
}

type Provider interface {
	Validate(map[string]string) error
	Process(context.Context, map[string]string, *lsApi.Event) error
	RetryConfig() RetryConfig
	RateLimiterConfig() RateLimiterConfig
}
