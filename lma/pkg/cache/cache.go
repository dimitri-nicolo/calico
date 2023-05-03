// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package cache

// Cache has a basic Get & Set functionality and also provides access to metric values
//
// The `Metric` functions return the current prometheus metric values, mainly for tests to confirm they are what's expected.
//
// Applications that want to include these metrics in their /metrics endpoint need to call `cache.RegisterMetricsWith(...)`
type Cache[Key ~string, Value any] interface {

	// Get returns (value, true) if it is found (a cache hit), and (emptyValue, false) if not found or has expired (a cache miss)
	Get(key Key) (Value, bool)

	// Set puts the new value in the cache for the specified key, overwriting any previous value
	Set(k Key, v Value)
}
