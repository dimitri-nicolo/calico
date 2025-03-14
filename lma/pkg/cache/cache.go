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

// LoadingCache is a Cache that loads the value when is not found in the cache,
// ensuring that concurrent requests for the same key will wait for the first
// request to load the value.
//
// If the load fails, the same error will be returned to all requests waiting for
// that value to load, and nothing will be written to the cache.
type LoadingCache[Key ~string, Value any] interface {

	// GetOrLoad returns the value for the key, or if the value it not present, it
	// will call the loader function, store the result in the cache and return it.
	//
	//
	// If the loader returns an error, the error is returned to all callers waiting on it, and nothing will be written to the cache.
	GetOrLoad(key Key, loader func() (Value, error)) (Value, error)
}
