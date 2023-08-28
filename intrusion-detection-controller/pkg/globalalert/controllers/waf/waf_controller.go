// Copyright 2023 Tigera Inc. All rights reserved.

package waf

import (
	"context"
	"time"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/globalalert/controllers/controller"
)

// wafAlertController is responsible for watching WAF logs in a cluster
// and creating corresponding events.
type wafAlertController struct {
	clusterName string
	cancel      context.CancelFunc
	wafLogs     client.WAFLogsInterface
	events      client.EventsInterface
	logsCache   WafLogsCache
}

// NewWafAlertController returns a wafAlertController for handling waf events
func NewWafAlertController(linseedClient client.Client, clusterName string, tenantID string, namespace string) controller.Controller {
	c := &wafAlertController{
		clusterName: clusterName,
		wafLogs:     linseedClient.WAFLogs(clusterName),
		events:      linseedClient.Events(clusterName),
		logsCache: WafLogsCache{
			lastWafTimestamp: time.Unix(0, 0),
			wafLogs:          []cacheInfo{},
		},
	}
	return c
}

// Run monitors waf logs and create events accordingly.
func (c *wafAlertController) Run(parentCtx context.Context) {
	var ctx context.Context
	ctx, c.cancel = context.WithCancel(parentCtx)
	log.Infof("Starting waf alert controller for cluster %s", c.clusterName)

	err := c.InitLogsCache(ctx)
	if err != nil {
		log.WithError(err).Warn("failed to init logs cache")
	}
	// Then loop forever...
	for {
		c.ManageCache(ctx)

		err := c.ProcessWafLogs(ctx)
		if err != nil {
			log.WithError(err).Error("error while processing waf logs")
		}

		timer := time.NewTimer(30 * time.Second)
		select {
		case <-timer.C:
			timer.Stop()
		case <-ctx.Done():
			log.Debug("Stop handling WAF events due to context cancellation")
			timer.Stop()
			return
		}
	}
}

func (c *wafAlertController) ManageCache(ctx context.Context) {
	timeRange := time.Now().Add(-(30 * time.Minute))
	newCache := []cacheInfo{}
	for _, log := range c.logsCache.wafLogs {
		if !log.timestamp.Before(timeRange) {
			newCache = append(newCache, log)
		}
	}

	c.logsCache.wafLogs = newCache
}

func (c *wafAlertController) InitLogsCache(ctx context.Context) error {
	log.Debug("Building Cache of existing waf Logs")
	now := time.Now()
	aWeekAgo := now.Add(-(time.Minute * 30))
	params := &v1.WAFLogParams{
		QueryParams: v1.QueryParams{
			TimeRange: &lmav1.TimeRange{
				From: aWeekAgo,
				To:   now.Add(MaxTimeSkew),
			},
		},
		QuerySortParams: v1.QuerySortParams{
			Sort: []v1.SearchRequestSortBy{
				{
					Field:      "@timestamp",
					Descending: true,
				},
			},
		},
	}

	logs, err := c.wafLogs.List(ctx, params)
	if err != nil {
		log.WithError(err).WithField("params", params).Error("error reading WAF logs from linseed")
		return err
	}

	for _, wafLog := range logs.Items {
		c.logsCache.Add(wafLog)
	}
	if len(logs.Items) != 0 {
		c.logsCache.lastWafTimestamp = logs.Items[0].Timestamp
	} else {
		c.logsCache.lastWafTimestamp = time.Now()
	}

	return nil
}

// Close cancels the WafAlertForwarder context.
func (c *wafAlertController) Close() {
	c.cancel()
}

func (c *wafAlertController) ProcessWafLogs(ctx context.Context) error {
	log.Debug("Processing WAF logs")
	now := time.Now()
	params := &v1.WAFLogParams{
		QueryParams: v1.QueryParams{
			TimeRange: &lmav1.TimeRange{
				From: c.logsCache.lastWafTimestamp.Add(-MaxTimeSkew),
				To:   now.Add(MaxTimeSkew),
			},
		},
		QuerySortParams: v1.QuerySortParams{
			Sort: []v1.SearchRequestSortBy{
				{
					Field:      "@timestamp",
					Descending: true,
				},
			},
		},
	}

	logs, err := c.wafLogs.List(ctx, params)
	if err != nil {
		log.WithError(err).WithField("params", params).Error("error reading WAF logs from linseed")
		return err
	}

	wafEvents := []v1.Event{}
	for _, wafLog := range logs.Items {
		if !c.logsCache.Contains(wafLog) {
			c.logsCache.Add(wafLog)
			// generate the new alerts/events from the waflogs
			wafEvents = append(wafEvents, NewWafEvent(wafLog))
		}
	}

	if len(wafEvents) > 0 {
		log.Debugf("About to create %d WAF Events", len(wafEvents))
		_, err = c.events.Create(ctx, wafEvents)
		if err != nil {
			return err
		}

	}
	// by ordering the waf logs in descending order the newest logs will be first
	if len(logs.Items) != 0 {
		c.logsCache.lastWafTimestamp = logs.Items[0].Timestamp
	} else {
		c.logsCache.lastWafTimestamp = time.Now()
	}

	return nil
}
