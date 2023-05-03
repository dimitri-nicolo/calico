// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package auth

import (
	"context"
	"fmt"

	authnv1 "k8s.io/api/authentication/v1"

	"github.com/projectcalico/calico/lma/pkg/cache"
)

type cachingTokenReviewer struct {
	delegate tokenReviewer
	cache    cache.Cache[string, authnv1.TokenReviewStatus]
}

func newCachingTokenReviewer(cache cache.Cache[string, authnv1.TokenReviewStatus], delegate tokenReviewer) *cachingTokenReviewer {
	return &cachingTokenReviewer{
		delegate: delegate,
		cache:    cache,
	}
}

// Review caches the results of calls to the delegate tokenReviewer.Review.
//
// Concurrent requests for the same uncached key will all be forwarded to the delegate and the cache updated for each result. Ideally
// a single request would be forwarded and the result shared amongst the callers but this increases the complexity for a probable small
// gain, so we will avoid that complexity until production metrics tell us otherwise.
func (r *cachingTokenReviewer) Review(ctx context.Context, spec authnv1.TokenReviewSpec) (authnv1.TokenReviewStatus, error) {
	key := toTokenReviewerCacheKey(spec)

	if result, ok := r.cache.Get(key); ok {
		return result, nil
	}

	result, err := r.delegate.Review(ctx, spec)
	if err != nil {
		return authnv1.TokenReviewStatus{}, err
	}

	r.cache.Set(key, result)

	return result, nil
}

func toTokenReviewerCacheKey(spec authnv1.TokenReviewSpec) string {
	return fmt.Sprintf("%+v", spec)
}
