// Copyright 2019 Tigera Inc. All rights reserved.

package cacher

import (
	"context"
	"sync"

	apiV3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

type MockGlobalThreatFeedCache struct {
	mutex      sync.Mutex
	cachedFeed *apiV3.GlobalThreatFeed
}

// NewMockGlobalThreatFeedCache ensures all mock Global Threat Feeds are mode Enabled so tests pass.
func NewMockGlobalThreatFeedCache() *MockGlobalThreatFeedCache {
	mode := new(apiV3.ThreatFeedMode)
	*mode = apiV3.ThreatFeedModeEnabled

	return &MockGlobalThreatFeedCache{
		cachedFeed: &apiV3.GlobalThreatFeed{
			Spec: apiV3.GlobalThreatFeedSpec{
				Mode: mode,
			},
		},
	}
}

func (s *MockGlobalThreatFeedCache) Run(_ context.Context) {
}

func (s *MockGlobalThreatFeedCache) Close() {
}

func (s *MockGlobalThreatFeedCache) GetGlobalThreatFeed() CacheResponse {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.cachedFeed == nil {
		s.cachedFeed = &apiV3.GlobalThreatFeed{}
	}
	return CacheResponse{GlobalThreatFeed: s.cachedFeed.DeepCopy(), Err: nil}
}

func (s *MockGlobalThreatFeedCache) UpdateGlobalThreatFeed(globalThreatFeed *apiV3.GlobalThreatFeed) CacheResponse {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.cachedFeed = globalThreatFeed
	return CacheResponse{GlobalThreatFeed: s.cachedFeed.DeepCopy(), Err: nil}
}

func (s *MockGlobalThreatFeedCache) UpdateGlobalThreatFeedStatus(globalThreatFeed *apiV3.GlobalThreatFeed) CacheResponse {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.cachedFeed = globalThreatFeed
	return CacheResponse{GlobalThreatFeed: s.cachedFeed.DeepCopy(), Err: nil}
}
