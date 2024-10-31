// Copyright 2023 Tigera Inc. All rights reserved.

package waf

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/cache"

	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/controller"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/util"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
)

const (
	// controllerInterval is how often this controller checks for new logs
	controllerInterval = time.Second * 30
	// ttl for cache items to live
	logsCacheTTL = time.Minute * 30

	// Potentally the maximum time skew difference between components generating WAF logs
	// What is time Skew?
	// difference between the clocks of different nodes in the managed cluster and/or
	// differing latencies between when a WAF log is constructed (with @timestamp: now())
	// on a managed cluster node and when it actually hits ES.
	MaxTimeSkew = 5 * time.Minute
)

// wafAlertController is responsible for watching WAF logs in a cluster
// and creating corresponding events.
type wafAlertController struct {
	clusterName        string
	cancel             context.CancelFunc
	wafLogs            client.WAFLogsInterface
	events             client.EventsInterface
	logsCache          *WAFLogsCache
	lastQueryTimestamp time.Time
	ping               chan struct{}
	fifo               *cache.DeltaFIFO
}

// NewWafAlertController returns a wafAlertController for handling waf events
func NewWafAlertController(linseedClient client.Client, clusterName string, tenantID string, namespace string) controller.Controller {
	c := &wafAlertController{
		clusterName:        clusterName,
		wafLogs:            linseedClient.WAFLogs(clusterName),
		events:             linseedClient.Events(clusterName),
		logsCache:          NewWAFLogsCache(logsCacheTTL),
		lastQueryTimestamp: time.Now().UTC(),
	}
	return c
}

// Run monitors waf logs and create events accordingly.
func (c *wafAlertController) Run(parentCtx context.Context) {
	var ctx context.Context
	ctx, c.cancel = context.WithCancel(parentCtx)
	log.Infof("[WAF] Starting waf alert controller for cluster %s", c.clusterName)

	err := c.InitLogsCache(ctx)
	if err != nil {
		log.WithError(err).Warn("[WAF] failed to init logs cache")
	}

	go c.runWafLogsProcessingLoop(ctx)
}

func (c *wafAlertController) runWafLogsProcessingLoop(ctx context.Context) {
	for {
		err := c.ProcessWafLogs(ctx)
		if err != nil {
			log.WithError(err).Error("[WAF] error while processing waf logs")
		}

		c.pong()
		timer := time.NewTimer(controllerInterval)
		defer timer.Stop()
		select {
		case <-timer.C:
			timer.Stop()
		case <-ctx.Done():
			log.Debug("[WAF] Stop handling WAF events due to context cancellation")
			timer.Stop()
			return
		}
	}
}

func (c *wafAlertController) refreshLastQueryTime() {
	c.lastQueryTimestamp = time.Now().UTC()
}

func (c *wafAlertController) timeRangeTo() time.Time {
	return time.Now().Add(MaxTimeSkew).UTC()
}

func (c *wafAlertController) InitLogsCache(ctx context.Context) error {
	log.Debug("[WAF] Building Cache of existing waf Logs")
	// we only want to cache logs that have already been an event/alert
	// fill the cache up to capacity according to max ttl value
	fromPeriod := c.lastQueryTimestamp.Add(-(logsCacheTTL))
	eventParams := &v1.EventParams{
		QueryParams: v1.QueryParams{
			TimeRange: &lmav1.TimeRange{
				From: fromPeriod,
				To:   c.timeRangeTo(),
			},
		},
		LogSelectionParams: v1.LogSelectionParams{
			Selector: "type = waf",
		},
		QuerySortParams: v1.QuerySortParams{
			Sort: []v1.SearchRequestSortBy{
				{
					Field: "time",
				},
			},
		},
	}
	events, err := c.events.List(ctx, eventParams)
	if err != nil {
		log.WithError(err).WithField("params", eventParams).Error("[WAF] error reading events logs from linseed")
		return err
	}

	c.refreshLastQueryTime()
	for _, event := range events.Items {
		var v v1.WAFLog
		if err := event.GetRecord(&v); err != nil {
			log.
				WithField("event", event).
				Error("[WAF] cannot add event to cache")
			continue
		}
		c.logsCache.Add(&v)
	}

	return nil
}

// Close cancels the WafAlertForwarder context.
func (c *wafAlertController) Close() {
	// check if the cancel function has been called by another goroutine
	if c.cancel != nil {
		c.cancel()
	}
}

// Ping is used to ensure the watcher's main loop is running and not blocked.
func (c *wafAlertController) Ping(ctx context.Context) error {
	// Enqueue a ping
	err := c.fifo.Update(util.Ping{})
	if err != nil {
		// Local fifo & cache should never error.
		panic(err)
	}

	// Wait for the ping to be processed, or context to expire.
	select {
	case <-ctx.Done():
		return ctx.Err()

	// Since this channel is unbuffered, this will block if the main loop is not
	// running, or has itself blocked.
	case <-c.ping:
		return nil
	}
}

// pong is called from the main processing loop to reply to a ping.
func (c *wafAlertController) pong() {
	// Nominally, a sync.Cond would work nicely here rather than a channel,
	// which would allow us to wake up all pingers at once. However, sync.Cond
	// doesn't allow timeouts, so we stick with channels and one pong() per ping.
	c.ping <- struct{}{}
}

func (c *wafAlertController) ProcessWafLogs(ctx context.Context) error {
	log.Debug("[WAF] Processing WAF logs")
	// prune cache first
	c.logsCache.Purge()
	// then we're ready for new entries
	params := &v1.WAFLogParams{
		QueryParams: v1.QueryParams{
			TimeRange: &lmav1.TimeRange{
				From: c.lastQueryTimestamp.Add(-MaxTimeSkew),
				To:   c.timeRangeTo(),
			},
		},
		QuerySortParams: v1.QuerySortParams{
			Sort: []v1.SearchRequestSortBy{
				{
					Field:      "@timestamp",
					Descending: false,
				},
			},
		},
	}

	logs, err := c.wafLogs.List(ctx, params)
	if err != nil {
		log.WithError(err).WithField("params", params).Error("[WAF] error reading WAF logs from linseed")
		return err
	}

	// query was successful, update last query time
	c.refreshLastQueryTime()
	batchCache := make(map[cacheKey]bool)

	wafEvents := []v1.Event{}
	for _, wafLog := range logs.Items {
		wafkey := logKey(&wafLog)
		if !c.logsCache.Contains(&wafLog) && !batchCache[wafkey] {
			// generate the new alerts/events from the waflogs
			batchCache[wafkey] = true
			wafEvents = append(wafEvents, NewWafEvent(wafLog))
		}
	}

	if len(wafEvents) > 0 {
		log.Debugf("[WAF] About to create %d WAF Events", len(wafEvents))
		_, err = c.events.Create(ctx, wafEvents)
		if err != nil {
			return err
		}

		// Add waflog to cache after push, in case push fails
		for _, event := range wafEvents {
			wafLog, _ := event.Record.(v1.WAFLog)
			c.logsCache.Add(&wafLog)
		}
	}

	return nil
}
