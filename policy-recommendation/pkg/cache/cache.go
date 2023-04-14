// Copyright (c) 2022 Tigera Inc. All rights reserved.

package cache

import "sync"

// ObjectCache provides an interface for creating a simple cache
type ObjectCache[T any] interface {
	// Set sets the key to the provided value
	Set(key string, value T) T

	// Get gets the value associated with the given key.
	Get(key string) T

	// Delete removes the key value from being stored in the cache
	Delete(key string)

	// GetAll returns all the values stored in the cache as a slice
	GetAll() []T
}

func NewSynchronizedObjectCache[T any]() *SynchronizedObjectCache[T] {
	return &SynchronizedObjectCache[T]{
		cache: map[string]T{},
	}
}

// SynchronizedObjectCache is an implementation of ObjectCache with synchronization
// between concurrent Get and Set operations
type SynchronizedObjectCache[T any] struct {
	lock  sync.Mutex
	cache map[string]T
}

func (s *SynchronizedObjectCache[T]) Set(key string, value T) T {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.cache[key] = value
	return s.cache[key]
}

func (s *SynchronizedObjectCache[T]) Get(key string) T {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.cache[key]
}

func (s *SynchronizedObjectCache[T]) Delete(key string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.cache, key)
}

func (s *SynchronizedObjectCache[T]) GetAll() []T {
	r := make([]T, 0, len(s.cache))
	for _, t := range s.cache {
		r = append(r, t)
	}
	return r
}
