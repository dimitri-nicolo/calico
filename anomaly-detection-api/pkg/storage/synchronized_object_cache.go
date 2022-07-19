package storage

import (
	"sync"
)

// SynchronizedObjectCache is an implementation of ObjectCache with synchronization
// between concurrent Get and Set operations
type SynchronizedObjectCache struct {
	lock  sync.Mutex
	cache map[string]interface{}
}

func NewSynchronizedObjectCache() *SynchronizedObjectCache {
	oc := &SynchronizedObjectCache{
		cache: map[string]interface{}{},
	}

	return oc
}

func (s *SynchronizedObjectCache) Set(key string, value interface{}) interface{} {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.cache[key] = value
	return s.cache[key]
}

func (s *SynchronizedObjectCache) Get(key string) interface{} {
	s.lock.Lock()
	defer s.lock.Unlock()

	object, ok := s.cache[key]

	if !ok {
		return nil
	}

	return object
}
