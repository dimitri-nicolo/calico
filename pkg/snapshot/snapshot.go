// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package snapshot

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/libcalico-go/lib/errors"

	"github.com/tigera/compliance/pkg/list"
)

const (
	// Snapshot frequency is hard coded to be daily.
	defaultSnapshotHour = 22
	snapshotHourEnv     = "TIGERA_COMPLIANCE_SNAPSHOT_HOUR"
	maxRetryTime        = 1 * time.Hour
)

var (
	retrySleepTime = 10 * time.Second
)

// Run is the entrypoint to start running the snapshotter.
func Run(ctx context.Context, kind metav1.TypeMeta, listSrc list.Source, listDest list.Destination) error {
	s := &snapshotter{
		ctx:      ctx,
		kind:     kind,
		clog:     logrus.WithField("kind", fmt.Sprintf("%s.%s", kind.Kind, kind.APIVersion)),
		listSrc:  listSrc,
		listDest: listDest,
	}
	return s.run()
}

type snapshotter struct {
	ctx      context.Context
	kind     metav1.TypeMeta
	clog     *logrus.Entry
	listSrc  list.Source
	listDest list.Destination
}

// Run aligns the current state with the last time a snapshot was made with the expected time of the next snapshot and
// then continuously snapshots with 'freq' periodicity.
func (s *snapshotter) run() error {
	// Check for a snapshot written within the last day.
	takeSnapshot, err := s.takeImmediateSnapshot()
	if err != nil {
		s.clog.WithError(err).Error("Failed to determine last list time, exiting...")
		return nil
	}

	// If there is no prior snapshot ...
	if takeSnapshot {
		s.clog.Info("No snapshot found in the last 24 hours, taking an instant snapshot")
		if err = s.storeSnapshot(); err != nil {
			return err
		}
		s.clog.Info("Successfully snapshotted the list source")
	}
	s.clog.Info("Executing snapshot continuously once every day at required time")

	for {
		timeToNextFire := s.timeOfNextSnapshot().Sub(time.Now())
		s.clog.WithField("timeToNextFire", timeToNextFire).Info("Wait for next fire timer")

		// Compute time to next fire.
		expireTimer := time.NewTimer(timeToNextFire)
		select {
		case <-s.ctx.Done():
			s.clog.Info("Process terminating")
			expireTimer.Stop()
			return nil
		case <-expireTimer.C:
			s.clog.Info("Store snapshot")
			if err := s.storeSnapshot(); err != nil {
				return err
			}
			s.clog.Info("Successfully snapshotted the list source")
		}
	}
}

func (s *snapshotter) takeImmediateSnapshot() (bool, error) {
	s.clog.Debug("Check if immediate snapshot is required")
	_, err := s.retry(s.lastListTimeFn())
	if _, ok := err.(errors.ErrorResourceDoesNotExist); ok {
		s.clog.Debug("Take immediate snapshot")
		return true, nil
	}
	return false, err
}

func (s *snapshotter) storeSnapshot() error {
	s.clog.Debug("Querying list")
	l, err := s.retry(s.listQueryFn())
	if err != nil {
		return err
	}
	trlist := l.(*list.TimestampedResourceList)

	s.clog.Debug("Writing list")
	_, err = s.retry(s.listWriteFn(trlist))
	return err
}

func (s *snapshotter) lastListTimeFn() func() (interface{}, error) {
	dayAgo := time.Now().Add(-24 * time.Hour)
	return func() (interface{}, error) {
		return s.listDest.RetrieveList(s.kind, &dayAgo, nil, false)
	}
}

func (s *snapshotter) listQueryFn() func() (interface{}, error) {
	return func() (interface{}, error) {
		return s.listSrc.RetrieveList(s.kind)
	}
}

func (s *snapshotter) listWriteFn(trlist *list.TimestampedResourceList) func() (interface{}, error) {
	return func() (interface{}, error) {
		return nil, s.listDest.StoreList(s.kind, trlist)
	}
}

// timeOfNextSnapshot determines the fire time of the current day and adds 'freq' if the current time is past that.
func (s *snapshotter) timeOfNextSnapshot() time.Time {
	now := time.Now()
	year, month, day := now.Date()
	hour := s.getSnapshotHour()
	fireTime := time.Date(year, month, day, hour, 0, 0, 0, now.Location())
	if fireTime.Before(now) {
		fireTime = time.Date(year, month, day+1, hour, 0, 0, 0, now.Location())
	}
	s.clog.WithField("fireTime", fireTime).Debug("Calculated time of next snapshot")
	return fireTime
}

// getSnapshotHour returns the configured hour to take snapshots.
func (s *snapshotter) getSnapshotHour() int {
	if she := os.Getenv(snapshotHourEnv); she != "" {
		if sh, err := strconv.ParseUint(she, 10, 8); err == nil && sh < 24 {
			s.clog.WithField("SnapshotHour", sh).Debug("Parsed snapshot hour")
			return int(sh)
		}
	}
	s.clog.WithField("SnapshotHour", defaultSnapshotHour).Debug("Using default snapshot hour")
	return defaultSnapshotHour
}

// retry retries a function until it succeeds. On success it returns the functions return value, otherwise it returns
// the last error.
func (s *snapshotter) retry(f func() (interface{}, error)) (interface{}, error) {
	s.clog.Debug("Running function in retry loop")
	expireTimer := time.NewTimer(maxRetryTime)
	for {
		val, err := f()
		if err == nil {
			s.clog.Debug("Function succeeded")
			return val, nil
		}

		// Immediately return a does not exist error.
		if _, doesNotExist := err.(errors.ErrorResourceDoesNotExist); doesNotExist {
			return nil, err
		}

		retryTimer := time.NewTimer(retrySleepTime)
		select {
		case <-retryTimer.C:
			s.clog.Debug("Retry timer popped")
			continue
		case <-expireTimer.C:
			s.clog.WithError(err).Warning("Expiration timer popped, returning last error")
			retryTimer.Stop()
			return nil, err
		case <-s.ctx.Done():
			s.clog.Info("snapshotter terminating")
			retryTimer.Stop()
			expireTimer.Stop()
			return nil, nil
		}
	}
}
