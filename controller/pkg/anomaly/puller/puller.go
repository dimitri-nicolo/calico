// Copyright 2019 Tigera Inc. All rights reserved.

package puller

import (
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"

	"github.com/tigera/intrusion-detection/controller/pkg/anomaly/filters"
	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/elastic"
	"github.com/tigera/intrusion-detection/controller/pkg/events"
	"github.com/tigera/intrusion-detection/controller/pkg/runloop"
)

const (
	maxConcurrency   = 1
	PullPeriod       = time.Minute
	RetryPeriod      = time.Minute * 2
	MinAnomalyScore  = 80
	LookbackPeriod   = 24 * time.Hour
	AnomalyDetection = "anomaly_detection"
	Severity         = 100
)

var sem *semaphore.Weighted

func init() {
	sem = semaphore.NewWeighted(maxConcurrency)
}

type Puller interface {
	Run(context.Context)
	Close()
}

type puller struct {
	name   string
	xPack  elastic.XPack
	events db.Events
	filter filters.Filter
	cancel context.CancelFunc
	once   sync.Once
}

func NewPuller(name string, xPack elastic.XPack, events db.Events, filter filters.Filter) Puller {
	return &puller{
		name:   name,
		xPack:  xPack,
		events: events,
		filter: filter,
	}
}

func (p *puller) Run(ctx context.Context) {
	p.once.Do(func() {
		ctx, p.cancel = context.WithCancel(ctx)

		log.WithField("name", p.name).Info("Processing anomaly detection job")

		f, reschedule := runloop.RunLoopWithReschedule()
		go func() {
			_ = f(ctx, func() {
				if err := sem.Acquire(ctx, 1); err != nil {
					return
				}
				if err := p.pull(ctx); err != nil {
					_ = reschedule()
				}
				sem.Release(1)
			}, PullPeriod, func() {}, RetryPeriod,
			)
		}()
	})
}

func (p *puller) Close() {
	p.cancel()
}

func (p *puller) pull(ctx context.Context) error {
	fields := log.Fields{
		"name": p.name,
	}

	log.WithFields(fields).Debug("Fetching")
	rs, err := p.fetch(ctx)
	if err != nil {
		log.WithFields(fields).WithError(err).Error("Error fetching records from XPack")
		return err
	}

	log.WithFields(fields).Debug("Filtering")
	rs, err = p.filter.Filter(rs)
	if err != nil {
		log.WithFields(fields).WithError(err).Error("Error filtering records")
		return err
	}

	log.WithFields(fields).Debug("Putting events")
	for _, r := range rs {
		if err := p.events.PutSecurityEvent(ctx, p.generateEvent(r)); err != nil {
			log.WithFields(fields).WithError(err).Error("Error putting security event")
			return err
		}
	}

	return nil
}

func (p *puller) fetch(ctx context.Context) ([]elastic.RecordSpec, error) {
	return p.xPack.GetRecords(ctx, p.name, &elastic.GetRecordsOptions{
		RecordScore:    MinAnomalyScore,
		ExcludeInterim: true,
		Start:          &elastic.Time{time.Now().Add(-LookbackPeriod)},
	})
}

func (p *puller) generateEvent(r elastic.RecordSpec) events.SecurityEvent {
	return events.SecurityEvent{
		Time:          r.Timestamp.Time.Unix(),
		Type:          AnomalyDetection,
		Severity:      Severity,
		Sources:       []string{p.name},
		AnomalyRecord: r,
	}
}
