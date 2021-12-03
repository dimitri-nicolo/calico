// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package snort

import (
	"sync"
	"time"
)

// Store will store alerts sent to ES and the last timestamp of a snort alert
type Store struct {
	sync.RWMutex
	// Store sent alerts using date, source and destination
	sentAlerts map[dateSrcDst]bool
	// Store the last timestamp of a snort alert
	lastAlertTimestamp time.Time
}

// NewStore creates a new Store from a given point in time
func NewStore(lastAlertTimestamp time.Time) *Store {
	return &Store{
		sentAlerts:         make(map[dateSrcDst]bool),
		lastAlertTimestamp: lastAlertTimestamp,
	}
}

// Filters can be applied against the stored alerts and new snort alerts
type Filters func(store *Store, alert Alert) bool

// Uniques is a Filter that will filter out previously sent alerts
func Uniques(store *Store, alert Alert) bool {
	store.RLock()
	defer store.RUnlock()

	_, ok := store.sentAlerts[alert.DateSrcDst]
	return ok
}

// Newest is a Filter that will filter out alert before the last know alert timestamp
func Newest(store *Store, alert Alert) bool {
	store.RLock()
	defer store.RUnlock()
	timestamp, err := parseTime(alert.DateSrcDst)
	if err != nil {
		return false
	}

	return store.lastAlertTimestamp.Before(timestamp)
}

// Apply is used to apply multiple filters for new discovered alerts
func (s *Store) Apply(alerts []Alert, filters ...Filters) []Alert {
	if len(filters) == 0 {
		return alerts
	}

	var filteredAlerts []Alert
	for _, alert := range alerts {
		for _, filter := range filters {
			if filter(s, alert) {
				filteredAlerts = append(filteredAlerts, alert)
			}
		}
	}

	return filteredAlerts
}

// Update will update the store with the newest alerts sent to ES
func (s *Store) Update(alerts []Alert) {
	s.Lock()
	defer s.Unlock()

	for _, alert := range alerts {
		s.sentAlerts[alert.DateSrcDst] = true
		t, _ := parseTime(alert.DateSrcDst)
		if s.lastAlertTimestamp.Before(t) {
			s.lastAlertTimestamp = t
		}
	}
}
