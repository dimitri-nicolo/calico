// Copyright 2019-2024 Tigera Inc. All rights reserved.

package searcher

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/feeds/cacher"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/feeds/errorcondition"
	geodb "github.com/projectcalico/calico/intrusion-detection-controller/pkg/feeds/geodb"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/feeds/utils"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/runloop"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/storage"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

type Searcher interface {
	Run(context.Context, cacher.GlobalThreatFeedCacher)
	SetFeed(*v3.GlobalThreatFeed)
	Close()
}

type searcher struct {
	feed         *v3.GlobalThreatFeed
	period       time.Duration
	q            storage.SuspiciousSet
	events       storage.Events
	once         sync.Once
	cancel       context.CancelFunc
	geoDB        geodb.GeoDatabase
	cachedEvents *EventCache
}

func (d *searcher) Run(ctx context.Context, feedCacher cacher.GlobalThreatFeedCacher) {
	d.once.Do(func() {
		ctx, d.cancel = context.WithCancel(ctx)
		go func() {
			defer d.cancel()
			_ = runloop.RunLoop(ctx, func() { d.doSearch(ctx, feedCacher) }, d.period)
		}()
	})
}

func (d *searcher) SetFeed(f *v3.GlobalThreatFeed) {
	d.feed = f.DeepCopy()
}

func (d *searcher) Close() {
	d.cancel()
}

func (d *searcher) doSearch(ctx context.Context, feedCacher cacher.GlobalThreatFeedCacher) {
	getCachedFeedResponse := feedCacher.GetGlobalThreatFeed()
	if getCachedFeedResponse.Err != nil {
		log.WithError(getCachedFeedResponse.Err).Error("[Global Threat Feeds] search failed due to failure to retrieve feed")
		return
	}
	if getCachedFeedResponse.GlobalThreatFeed == nil {
		log.Error("[Global Threat Feeds] can't perform search because the feed doesn't exist")
		return
	}

	d.cachedEvents.Purge()

	// Ensure Global Threat Feed is Enabled before querying Linseed and sending event.
	mode := getCachedFeedResponse.GlobalThreatFeed.Spec.Mode
	if mode != nil && *mode == v3.ThreatFeedModeEnabled {
		log.Debug("[Global Threat Feeds] Check if any flow logs have been generated with a suspicious IP")
		results, lastSuccessfulSearch, setHash, err := d.q.QuerySet(ctx, d.geoDB, getCachedFeedResponse.GlobalThreatFeed)
		if err != nil {
			log.WithError(err).Errorf("[Global Threat Feeds] query failed for %v", getCachedFeedResponse.GlobalThreatFeed.Name)
			utils.AddErrorToFeedStatus(feedCacher, cacher.SearchFailed, err)
			return
		}

		newEvents := []v1.Event{}
		for _, event := range results {
			if !d.cachedEvents.Contains(&event) {
				newEvents = append(newEvents, event)
				d.cachedEvents.Add(&event)
			}
		}

		err = d.events.PutSecurityEventWithID(ctx, newEvents)
		if err != nil {
			log.WithError(err).Errorf("[Global Threat Feeds] failed to store events for %v", getCachedFeedResponse.GlobalThreatFeed.Name)
			utils.AddErrorToFeedStatus(feedCacher, cacher.SearchFailed, err)
			return
		}

		log.Debugf("[Global Threat Feeds] Update feed statusfor %v", getCachedFeedResponse.GlobalThreatFeed.Name)
		updateFeedStatusAfterSuccessfulSearch(feedCacher, lastSuccessfulSearch)
		log.Debugf("[Global Threat Feeds] Update feed after searchfor %v", getCachedFeedResponse.GlobalThreatFeed.Name)
		updateFeedAfterSuccessfulSearch(feedCacher, setHash)

	} else {
		log.WithFields(log.Fields{"feedName": getCachedFeedResponse.GlobalThreatFeed.Name}).Debug("[Global Threat Feeds] Feed is currently not enabled.")
	}
}

func NewSearcher(feed *v3.GlobalThreatFeed, period time.Duration, suspiciousSet storage.SuspiciousSet, events storage.Events, geoDB geodb.GeoDatabase, maxLinseedTimeSkew time.Duration) Searcher {
	return &searcher{
		feed:         feed.DeepCopy(),
		period:       period,
		q:            suspiciousSet,
		events:       events,
		geoDB:        geoDB,
		cachedEvents: NewEventCache(maxLinseedTimeSkew),
	}
}

// updateFeedAfterSuccessfulSearch is called after a query to IPSet/DomainNameSet succeeds.
// It updates the value of the annotation db.DomainNameSetHashKey/db.IpSetHashKey of the corresponding GlobalThreatFeed CR with a retry mechanism.
// A retry only kicks off when the update failure is caused by a StatusConflict and it will retry at most cacher.MaxUpdateRetry times
func updateFeedAfterSuccessfulSearch(feedCacher cacher.GlobalThreatFeedCacher, setHash string) {
	getCachedFeedResponse := feedCacher.GetGlobalThreatFeed()
	if getCachedFeedResponse.Err != nil {
		log.WithError(getCachedFeedResponse.Err).
			Error("[Global Threat Feeds] abort updating feed because failed to retrieve cached GlobalThreatFeed CR")
		return
	}
	if getCachedFeedResponse.GlobalThreatFeed == nil {
		log.Error("[Global Threat Feeds] abort updating feed because cached GlobalThreatFeed CR cannot be empty")
		return
	}

	toBeUpdated := getCachedFeedResponse.GlobalThreatFeed
	for i := 1; i <= cacher.MaxUpdateRetry; i++ {
		log.Debug(fmt.Sprintf("[Global Threat Feeds] %d/%d attempt to update feed after successful search", i, cacher.MaxUpdateRetry))
		if toBeUpdated.Spec.Content == v3.ThreatFeedContentDomainNameSet {
			updateAnnotation(toBeUpdated, storage.DomainNameSetHashKey, setHash)
		} else {
			updateAnnotation(toBeUpdated, storage.IpSetHashKey, setHash)
		}
		updateResponse := feedCacher.UpdateGlobalThreatFeed(toBeUpdated)
		updateErr := updateResponse.Err
		if updateErr == nil {
			log.Debugf("[Global Threat Feeds] attempt to update feed %v after successful search succeeded, exiting the loop", feedCacher.GetGlobalThreatFeed().GlobalThreatFeed.Name)
			return
		}
		statusErr, ok := updateErr.(*errors.StatusError)
		if !ok || statusErr.Status().Code != http.StatusConflict {
			log.WithError(updateErr).Errorf("[Global Threat Feeds] abort updating feed %v after successful search due to unrecoverable failure", feedCacher.GetGlobalThreatFeed().GlobalThreatFeed.Name)
			return
		}
		log.WithError(updateErr).Errorf("[Global Threat Feeds] failed updating feed %v after successful search", feedCacher.GetGlobalThreatFeed().GlobalThreatFeed.Name)
		toBeUpdated = updateResponse.GlobalThreatFeed
	}
}

// updateFeedStatusAfterSuccessfulSearch is called after a query to IPSet/DomainNameSet succeeds.
// It updates the LastSuccessfulSearch timestamp of the corresponding GlobalThreatFeed CR.
// It also removes all the errors with type cacher.SearchFailed from the error conditions of the corresponding GlobalThreatFeed CR.
// The update is performed with a retry mechanism.
// A retry only kicks off when the update failure is caused by a StatusConflict and it will retry at most cacher.MaxUpdateRetry times
func updateFeedStatusAfterSuccessfulSearch(feedCacher cacher.GlobalThreatFeedCacher, lastSuccessfulSearch time.Time) {
	getCachedFeedResponse := feedCacher.GetGlobalThreatFeed()
	if getCachedFeedResponse.Err != nil {
		log.WithError(getCachedFeedResponse.Err).
			Error("[Global Threat Feeds] abort updating feed status after successful search because failed to retrieve cached GlobalThreatFeed CR")
		return
	}
	if getCachedFeedResponse.GlobalThreatFeed == nil {
		log.Error("[Global Threat Feeds] abort updating feed status after successful search because cached GlobalThreatFeed CR cannot be empty")
		return
	}

	toBeUpdated := getCachedFeedResponse.GlobalThreatFeed
	for i := 1; i <= cacher.MaxUpdateRetry; i++ {
		log.Debug(fmt.Sprintf("[Global Threat Feeds] %d/%d attempt to update feed %v status after successful search", i, cacher.MaxUpdateRetry, getCachedFeedResponse.GlobalThreatFeed.Name))
		if toBeUpdated.Status.LastSuccessfulSearch == nil || lastSuccessfulSearch.After(toBeUpdated.Status.LastSuccessfulSearch.Time) {
			toBeUpdated.Status.LastSuccessfulSearch = &metav1.Time{Time: lastSuccessfulSearch}
		} else {
			log.Errorf("[Global Threat Feeds] abort updating feed %v status after successful search because the current attempt is out of date", getCachedFeedResponse.GlobalThreatFeed.Name)
			return
		}
		errorcondition.ClearError(&toBeUpdated.Status, cacher.SearchFailed)
		updateResponse := feedCacher.UpdateGlobalThreatFeedStatus(toBeUpdated)
		updateErr := updateResponse.Err
		if updateErr == nil {
			log.Debugf("[Global Threat Feeds] attempt to update feed %v status after successful search succeeded, exiting the loop", getCachedFeedResponse.GlobalThreatFeed.Name)
			return
		}
		statusErr, ok := updateErr.(*errors.StatusError)
		if !ok || statusErr.Status().Code != http.StatusConflict {
			log.WithError(updateErr).Errorf("[Global Threat Feeds] abort updating feed %v status after successful search due to unrecoverable failure", getCachedFeedResponse.GlobalThreatFeed.Name)
			return
		}
		log.WithError(updateErr).Errorf("[Global Threat Feeds] failed updating feed %v status after successful search", getCachedFeedResponse.GlobalThreatFeed.Name)
		toBeUpdated = updateResponse.GlobalThreatFeed
	}
}

func updateAnnotation(globalThreatFeed *v3.GlobalThreatFeed, key, val string) {
	annotations := globalThreatFeed.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[key] = val
	globalThreatFeed.SetAnnotations(annotations)
	log.WithField("name", globalThreatFeed.Name).Debug("[Global Threat Feeds] updated global threat feed annotation")
}

type cacheKey struct {
	ID string
}

type EventCache struct {
	cache  map[cacheKey]time.Time
	maxTTL time.Duration
}

func NewEventCache(maxTTL time.Duration) *EventCache {
	return &EventCache{
		cache:  make(map[cacheKey]time.Time),
		maxTTL: maxTTL,
	}
}

// Contains checks if we've seen the event before
func (c *EventCache) Contains(event *v1.Event) bool {
	key := logKey(event)

	_, ok := c.cache[key]
	return ok
}

// Add adds an event's uniquely generated ID to the cache
func (c *EventCache) Add(event *v1.Event) {
	key := logKey(event)
	c.cache[key] = time.Now()
}

// cull expiring entries
func (c *EventCache) Purge() {
	timeCutOff := time.Now().Add(-(c.maxTTL))
	for k, ts := range c.cache {
		if ts.Before(timeCutOff) {
			// evict
			delete(c.cache, k)
		}
	}
}

func logKey(v *v1.Event) cacheKey {
	return cacheKey{
		ID: v.ID,
	}
}
