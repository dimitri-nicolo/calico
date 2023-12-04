// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package providers

import (
	"context"
	"time"

	lsApi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

type Provider interface {
	Validate(map[string]string) error
	Process(context.Context, map[string]string, map[string]string, *lsApi.Event) error
	Config() Config
}

type Config struct {
	RateLimiterDuration time.Duration `default:"1h"`
	RateLimiterCount    uint          `default:"100"`
	RequestTimeout      time.Duration `default:"5s"`
	RetryDuration       time.Duration `default:"2s"`
	RetryTimes          uint          `default:"5"`
}
