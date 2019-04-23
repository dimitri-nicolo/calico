// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package snapshot

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/projectcalico/libcalico-go/lib/errors"
	"github.com/projectcalico/libcalico-go/lib/health"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tigera/compliance/pkg/list"
	"github.com/tigera/compliance/pkg/resources"
)

const (
	// Snapshot frequency is hard coded to be daily.
	defaultSnapshotHour = 22
	snapshotHourEnv     = "TIGERA_COMPLIANCE_SNAPSHOT_HOUR"
	maxRetryTime        = 1 * time.Hour
	oneDay              = 24 * time.Hour

	// Health aggregator values.
	HealthName = "snapshotter"
)

var (
	keepAliveInterval = 10 * time.Second

	allResources = resources.GetAllResourceHelpers()
)

// Run is the entrypoint to start running the snapshotter.
func Run(ctx context.Context, listSrc list.Source, listDest list.Destination, healthAgg *health.HealthAggregator) error {
	return (&snapshotter{
		ctx:      ctx,
		health:   healthAgg,
		listSrc:  listSrc,
		listDest: listDest,
	}).run()
}

type snapshotter struct {
	ctx      context.Context
	health   *health.HealthAggregator
	listSrc  list.Source
	listDest list.Destination
}

func healthy(isAlive bool) *health.HealthReport {
	return &health.HealthReport{Live: isAlive, Ready: true}
}

// Run aligns the current state with the last time a snapshot was made with the expected time of the next snapshot and
// then continuously snapshots with 'freq' periodicity.
func (s *snapshotter) run() error {
	log.Info("Executing snapshot continuously once every day at required time")

	// Initialize keep alive ticker.
	keepAliveTicker := time.NewTicker(keepAliveInterval)

	// Initialize resourceSnapshotters
	snapshotters := map[metav1.TypeMeta]*resourceSnapshotter{}
	for _, rh := range allResources {
		tm := rh.TypeMeta()
		snapshotters[tm] = &resourceSnapshotter{
			ctx:      s.ctx,
			kind:     tm,
			clog:     log.WithField("kind", fmt.Sprintf("%s.%s", tm.Kind, tm.APIVersion)),
			listSrc:  s.listSrc,
			listDest: s.listDest,
		}
	}

	// Run snapshot infinitely.
	for {
		// Determine if time for snapshot.
		prev, next := timeOfNextSnapshot()

		// Iterate over resources and store snapshot for each.
		errChan := make(chan error, len(allResources))
		wg := sync.WaitGroup{}
		for _, rh := range allResources {
			wg.Add(1)
			go func(rh resources.ResourceHelper) {
				defer wg.Done()
				tm := rh.TypeMeta()

				// Take the snapshot.
				errChan <- snapshotters[tm].maybeTakeSnapshot(prev, next)
			}(rh)
		}
		wg.Wait()
		close(errChan)

		// Iterate over all the responses coming through the channel and flag unhealthy.
		for err := range errChan {
			if err != nil {
				log.WithError(err).Error("Snapshot failed")
				s.health.Report(HealthName, healthy(false))
				break
			}
		}

		select {
		case <-s.ctx.Done():
			// Context cancelled.
			log.Info("Process terminating")
			keepAliveTicker.Stop()
			return nil

		case <-keepAliveTicker.C:
			// Keep alive timer fired; notify health aggregator.
			log.Info("Waking up from keep-alive timer")
			s.health.Report(HealthName, healthy(true))
		}
	}
}

type resourceResponse struct {
	tm  metav1.TypeMeta
	err error
}

// timeOfNextSnapshot determines the fire time of the previous and next day.
func timeOfNextSnapshot() (time.Time, time.Time) {
	now := time.Now()
	year, month, day := now.Date()
	hour := getSnapshotHour()
	fireTime := time.Date(year, month, day, hour, 0, 0, 0, now.Location())
	if fireTime.Before(now) {
		return fireTime, fireTime.Add(oneDay)
	}
	return fireTime.Add(-oneDay), fireTime
}

// getSnapshotHour returns the configured hour to take snapshots.
func getSnapshotHour() int {
	if she := os.Getenv(snapshotHourEnv); she != "" {
		if sh, err := strconv.ParseUint(she, 10, 8); err == nil && sh < 24 {
			log.WithField("SnapshotHour", sh).Debug("Parsed snapshot hour")
			return int(sh)
		}
	}
	log.WithField("SnapshotHour", defaultSnapshotHour).Debug("Using default snapshot hour")
	return defaultSnapshotHour
}

type resourceSnapshotter struct {
	ctx                context.Context
	kind               metav1.TypeMeta
	clog               *log.Entry
	listSrc            list.Source
	listDest           list.Destination
	timeOfLastSnapshot *time.Time
}

func (r *resourceSnapshotter) maybeTakeSnapshot(prev, next time.Time) error {
	// If timeOfLastSnapshot is not known then populate from an elastic search query.
	if r.timeOfLastSnapshot == nil {
		dayAgo := time.Now().Add(-24 * time.Hour)
		trlist, err := r.listDest.RetrieveList(r.kind, &dayAgo, nil, false)
		if err != nil {
			if _, ok := err.(errors.ErrorResourceDoesNotExist); ok {
				r.clog.Debug("No snapshot exists")
				r.timeOfLastSnapshot = &time.Time{}
			} else {
				r.clog.WithError(err).Error("failed to retrieve last list query")
				return err
			}
		} else if trlist != nil {
			r.clog.WithField("lastSnapshotTime", trlist.RequestCompletedTimestamp).Debug("Found last snapshot")
			r.timeOfLastSnapshot = &trlist.RequestCompletedTimestamp.Time
		}
	}

	// If timeOfLastSnapshot is < prev then we haven't taken a snapshot in this interval. Take a snapshot.
	if r.timeOfLastSnapshot.Before(prev) {
		r.clog.Debug("Querying list")
		trlist, err := r.listSrc.RetrieveList(r.kind)
		if err != nil {
			r.clog.WithError(err).Error("Failed to query list")
			return err
		}

		r.clog.Debug("Writing list")
		if err = r.listDest.StoreList(r.kind, trlist); err != nil {
			r.clog.WithError(err).Error("Failed to write list")
			return err
		}

		r.timeOfLastSnapshot = &trlist.RequestCompletedTimestamp.Time
		r.clog.Info("Successfully snapshotted the list source!")
	} else {
		r.clog.WithField("nextSnapshot", next.Sub(*r.timeOfLastSnapshot)).Info("Time to next snapshot.")
	}
	return nil
}
