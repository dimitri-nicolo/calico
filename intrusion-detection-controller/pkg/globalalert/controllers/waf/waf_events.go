// Copyright 2023 Tigera Inc. All rights reserved.

package waf

import (
	"time"

	"github.com/projectcalico/calico/libcalico-go/lib/validator/v3/query"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

type WAFLogsCache struct {
	cache  map[cacheKey]time.Time
	maxTTL time.Duration
}

func NewWAFLogsCache(ttl time.Duration) *WAFLogsCache {
	return &WAFLogsCache{
		cache:  make(map[cacheKey]time.Time),
		maxTTL: ttl,
	}
}

type cacheKey struct {
	requestID string
}

func logKey(v *v1.WAFLog) cacheKey {
	return cacheKey{
		requestID: v.RequestId,
	}
}

// Contains checks if we've seen the waf log before
func (c *WAFLogsCache) Contains(wafLog *v1.WAFLog) bool {
	_, ok := c.cache[logKey(wafLog)]
	return ok
}

// Add adds the uuid requestId of the waf log
func (c *WAFLogsCache) Add(wafLog *v1.WAFLog) {
	c.cache[logKey(wafLog)] = time.Now()
}

// cull expiring entries
func (c *WAFLogsCache) Purge() {
	timeRange := time.Now().Add(-(c.maxTTL))
	for k, ts := range c.cache {
		if ts.Before(timeRange) {
			// evict
			delete(c.cache, k)
		}
	}
}

func NewWafEvent(l v1.WAFLog) v1.Event {

	return v1.Event{
		Type: query.WafEventType,
		// Deviating from original implementation here
		Origin: "Web Application Firewall",

		// GlobalAlert use time.Now() but it makes more sense to use the
		// timestamp from the WAF log...
		Time:        v1.NewEventTimestamp(time.Now().Unix()),
		Name:        "WAF Event",
		Description: "Traffic inside your cluster triggered Web Application Firewall rules",
		// Bad but not too bad :) Open for feedback
		Severity:        80,
		Host:            l.Host,
		Protocol:        l.Protocol,
		SourceIP:        &l.Source.IP,
		SourceName:      l.Source.PodName,
		SourceNamespace: l.Source.PodNameSpace,
		DestIP:          &l.Destination.IP,
		DestName:        l.Destination.PodName,
		DestNamespace:   l.Destination.PodNameSpace,
		MitreIDs:        &[]string{"T1190"},
		Mitigations:     &[]string{"Review the source of this event - an attacker could be inside your cluster attempting to exploit your web application. Calico network policy can be used to block the connection if the activity is not expected"},
		AttackVector:    "Network",
		MitreTactic:     "Initial Access",

		Record: l,
	}
}
