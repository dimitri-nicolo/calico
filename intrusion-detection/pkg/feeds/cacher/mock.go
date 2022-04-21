// Copyright 2019 Tigera Inc. All rights reserved.

package cacher

import (
	"context"

	apiV3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

type MockGlobalThreatFeedCache struct {
	cachedFeed *apiV3.GlobalThreatFeed
}

func (s *MockGlobalThreatFeedCache) Run(ctx context.Context) {
}

func (s *MockGlobalThreatFeedCache) Close() {
}

func (s *MockGlobalThreatFeedCache) GetGlobalThreatFeed() CacheResponse {
	if s.cachedFeed == nil {
		s.cachedFeed = &apiV3.GlobalThreatFeed{}
	}
	return CacheResponse{GlobalThreatFeed: s.cachedFeed, Err: nil}
}

func (s *MockGlobalThreatFeedCache) UpdateGlobalThreatFeed(globalThreatFeed *apiV3.GlobalThreatFeed) CacheResponse {
	s.cachedFeed = globalThreatFeed
	return CacheResponse{GlobalThreatFeed: s.cachedFeed, Err: nil}
}

func (s *MockGlobalThreatFeedCache) UpdateGlobalThreatFeedStatus(globalThreatFeed *apiV3.GlobalThreatFeed) CacheResponse {
	s.cachedFeed = globalThreatFeed
	return CacheResponse{GlobalThreatFeed: s.cachedFeed, Err: nil}
}
