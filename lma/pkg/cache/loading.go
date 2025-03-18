package cache

import (
	"sync"
)

// loading is a generic cache that automatically loads values on cache misses.
type loading[Key ~string, Value any] struct {
	cache      Cache[Key, Value]
	mutex      sync.RWMutex
	inProgress map[Key]*loaderStatus[Value]
}

// NewLoadingCache wraps a Cache with a function that loads the value when it is not found in the cache.
//
// Concurrent requests for the same key will wait for the first request to load the value, avoiding duplicate work.
func NewLoadingCache[Key ~string, Value any](cache Cache[Key, Value]) LoadingCache[Key, Value] {
	return newLoading(cache)
}

func newLoading[Key ~string, Value any](cache Cache[Key, Value]) *loading[Key, Value] {
	return &loading[Key, Value]{
		cache:      cache,
		inProgress: make(map[Key]*loaderStatus[Value]),
	}
}

func (l *loading[Key, Value]) GetOrLoad(key Key, loader func() (Value, error)) (Value, error) {
	l.mutex.Lock()

	if value, ok := l.cache.Get(key); ok {
		l.mutex.Unlock()
		return value, nil
	}

	// if another request is already loading this key, unlock the cache and wait for the result
	if status, fetching := l.inProgress[key]; fetching {
		l.mutex.Unlock()
		value, err := status.wait()
		return value, err
	}

	// create and store the new status
	status := &loaderStatus[Value]{
		mutex: sync.Mutex{},
	}
	l.inProgress[key] = status

	// lock the status while we call the loader
	status.mutex.Lock()
	defer status.mutex.Unlock()

	// unlock the cache before calling the loader
	l.mutex.Unlock()

	// call the loader
	loadResult, loadErr := loader()

	// re-lock the cache so we can store the result
	l.mutex.Lock()

	// store the result in the cache if the load was successful
	if loadErr == nil {
		l.cache.Set(key, loadResult)
	}

	// remove the status frm the inProgress map while still locked
	delete(l.inProgress, key)

	l.mutex.Unlock()

	// notify others waiting on this result
	status.value = loadResult
	status.err = loadErr

	return loadResult, loadErr
}

type loaderStatus[Value any] struct {
	mutex sync.Mutex
	value Value
	err   error
}

// wait for the loader function to unlock the mutex (by attempting to lock and immediately unlock it), then return the result.
func (s *loaderStatus[Value]) wait() (Value, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.value, s.err
}
