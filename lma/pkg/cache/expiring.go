// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package cache

import (
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
)

// expiring a generic expiring cache that records metrics.
//
// # Keys are limited to string types for now due to the underlying go-cache implementation
//
// expiring expiration time is set in NewExpiring() and cannot be changed, we do not support setting expiration time on Set()
type expiring[Key ~string, Value any] struct {
	name  string
	ttl   time.Duration
	cache *cache.Cache
}

func NewExpiring[Key ~string, Value any](cfg ExpiringConfig) (Cache[Key, Value], error) {
	return newExpiring[Key, Value](cfg)
}

func newExpiring[Key ~string, Value any](cfg ExpiringConfig) (*expiring[Key, Value], error) {
	err := cfg.validate()
	if err != nil {
		return nil, err
	}

	c := &expiring[Key, Value]{
		name:  cfg.Name,
		ttl:   cfg.TTL,
		cache: cache.New(cfg.TTL, cfg.ExpiredElementsCleanupInterval),
	}

	c.startMetrics(cfg.Context, cfg.MetricsCollectionInterval)

	return c, nil
}

func (c *expiring[Key, Value]) Get(key Key) (Value, bool) {
	if v, ok := c.cache.Get(string(key)); ok {
		if value, ok := v.(Value); ok {
			c.registerCacheHit()
			return value, true
		} else {
			var expected Value
			panic(fmt.Sprintf("value of wrong type found in cache - expected: %T, actual: %T: %v", expected, v, v))
		}
	}
	c.registerCacheMiss()
	var empty Value
	return empty, false
}

func (c *expiring[Key, Value]) Set(k Key, v Value) {
	c.cache.Set(string(k), v, c.ttl)
}
