// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package auth

import (
	"fmt"
	"strings"
	"time"

	authzv1 "k8s.io/api/authorization/v1"
	"k8s.io/apiserver/pkg/authentication/user"

	"github.com/projectcalico/calico/lma/pkg/cache"
)

const (
	TokenReviewCacheMaxTTL = 20 * time.Second
)

type cachingAuthorizer struct {
	delegate RBACAuthorizer
	cache    cache.Cache[string, bool]
}

func NewCachingAuthorizer(cache cache.Cache[string, bool], delegate RBACAuthorizer) RBACAuthorizer {
	return newCachingAuthorizer(cache, delegate)
}

func newCachingAuthorizer(cache cache.Cache[string, bool], delegate RBACAuthorizer) *cachingAuthorizer {

	return &cachingAuthorizer{
		delegate: delegate,
		cache:    cache,
	}
}

// Authorize caches the results of calls to the delegate RBACAuthorizer.Authorize in the case where `resources!=nil && nonResources==nil`.
//
// Concurrent requests for the same uncached key will all be forwarded to the delegate and the cache updated for each result. Ideally
// a single request would be forwarded and the result shared amongst the callers but this increases the complexity for a probable small
// gain, so we will avoid that complexity until production metrics tell us otherwise.
func (a *cachingAuthorizer) Authorize(usr user.Info, resources *authzv1.ResourceAttributes, nonResources *authzv1.NonResourceAttributes) (bool, error) {
	if resources == nil || nonResources != nil {
		return a.delegate.Authorize(usr, resources, nonResources)
	}

	key := toAuthorizeCacheKey(usr, resources)

	if cachedResult, ok := a.cache.Get(key); ok {
		return cachedResult, nil
	}

	delegateResult, err := a.delegate.Authorize(usr, resources, nonResources)
	if err != nil {
		return false, err
	}

	a.cache.Set(key, delegateResult)

	return delegateResult, nil
}

func toAuthorizeCacheKey(uer user.Info, resources *authzv1.ResourceAttributes) string {
	type key struct {
		userName   string
		userUID    string
		userGroups []string
		userExtra  map[string][]string
		attrs      authzv1.ResourceAttributes
	}

	return fmt.Sprintf("%+v", key{
		userName:   uer.GetName(),
		userUID:    uer.GetUID(),
		userGroups: uer.GetGroups(),
		userExtra:  uer.GetExtra(),
		attrs:      *resources,
	})
}

func toAuthorizeCacheKeyManual(uer user.Info, resources *authzv1.ResourceAttributes) string {

	sb := strings.Builder{}
	sb.WriteString("{")
	sb.WriteString("userName:")
	sb.WriteString(uer.GetName())
	sb.WriteString(" userUID:")
	sb.WriteString(uer.GetUID())
	sb.WriteString(" userGroups:[")
	for i, g := range uer.GetGroups() {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(g)
	}
	sb.WriteString("] userExtra:{")
	for k, v := range uer.GetExtra() {
		sb.WriteString(k)
		sb.WriteString(":[")
		for i, s := range v {
			if i > 0 {
				sb.WriteString(" ")
			}
			sb.WriteString(s)
		}
		sb.WriteString("]")
	}
	sb.WriteString("} attrs:{")
	sb.WriteString("Namespace:")
	sb.WriteString(resources.Namespace)
	sb.WriteString(" Verb:")
	sb.WriteString(resources.Verb)
	sb.WriteString(" Group:")
	sb.WriteString(resources.Group)
	sb.WriteString(" Version:")
	sb.WriteString(resources.Version)
	sb.WriteString(" Resource:")
	sb.WriteString(resources.Resource)
	sb.WriteString(" Subresource:")
	sb.WriteString(resources.Subresource)
	sb.WriteString(" Name:")
	sb.WriteString(resources.Name)
	sb.WriteString("}}")

	return sb.String()
}
