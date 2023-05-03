// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package cache

import (
	"context"
	"errors"
	"time"
)

type ExpiringConfig struct {
	Context context.Context
	Name    string
	TTL     time.Duration

	ExpiredElementsCleanupInterval time.Duration // defaults to 30 Seconds
	MetricsCollectionInterval      time.Duration // defaults to 10 Seconds
}

func (c *ExpiringConfig) validate() error {
	if c.Context == nil {
		return errors.New("context is required")
	}
	if c.Name == "" {
		return errors.New("name is required")
	}
	if c.TTL <= 0 {
		return errors.New("TTL is required")
	}

	if c.ExpiredElementsCleanupInterval == 0 {
		c.ExpiredElementsCleanupInterval = 30 * time.Second
	}
	if c.MetricsCollectionInterval == 0 {
		c.MetricsCollectionInterval = 10 * time.Second
	}

	return nil
}
