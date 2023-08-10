// Copyright (c) 2019-2023 Tigera, Inc. All rights reserved.

package dnslog

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/felix/rules"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

// AggregationKind determines how DNS logs are aggregated
type AggregationKind int

const (
	// DNSDefault means no aggregation, other than for identical logs within the aggregation
	// time period (aka flush interval).
	DNSDefault AggregationKind = iota
	// DNSPrefixNameAndIP aggregates logs with the same DNS information and client name prefix,
	// i.e. masking the client name and IP.
	DNSPrefixNameAndIP
)

// Aggregator is responsible for creating, aggregating, and storing
// aggregated dns logs until the dns logs are exported.
type Aggregator struct {
	kind                 AggregationKind
	dnsStore             map[DNSMeta]DNSSpec
	dnsMutex             sync.Mutex
	includeLabels        bool
	aggregationStartTime time.Time
	perNodeLimit         int
	handledAction        rules.RuleAction
	numUnloggedUpdates   int
}

// NewAggregator constructs an Aggregator
func NewAggregator() *Aggregator {
	return &Aggregator{
		kind:                 DNSDefault,
		dnsStore:             make(map[DNSMeta]DNSSpec),
		aggregationStartTime: time.Now(),
	}
}

func (a *Aggregator) IncludeLabels(b bool) *Aggregator {
	a.includeLabels = b
	return a
}

func (a *Aggregator) AggregateOver(k AggregationKind) *Aggregator {
	a.kind = k
	return a
}

func (a *Aggregator) PerNodeLimit(l int) *Aggregator {
	a.perNodeLimit = l
	return a
}

func (a *Aggregator) FeedUpdate(update Update) error {
	meta, spec, err := newMetaSpecFromUpdate(update, a.kind)
	if err != nil {
		return err
	}

	// Ensure that we can't add or aggregate new logs into the store at the
	// same time as existing logs are being flushed out.
	a.dnsMutex.Lock()
	defer a.dnsMutex.Unlock()

	if _, ok := a.dnsStore[meta]; ok {
		existing := a.dnsStore[meta]
		existing.Merge(spec)
		a.dnsStore[meta] = existing
	} else if (a.perNodeLimit == 0) || (len(a.dnsStore) < a.perNodeLimit) {
		a.dnsStore[meta] = spec
	} else {
		a.numUnloggedUpdates++
	}

	return nil
}

func (a *Aggregator) Get() []*v1.DNSLog {
	var dnsLogs []*v1.DNSLog
	aggregationEndTime := time.Now()

	// Ensure that we can't add or aggregate new logs into the store at the
	// same time as existing logs are being flushed out.
	a.dnsMutex.Lock()
	defer a.dnsMutex.Unlock()

	for meta, spec := range a.dnsStore {
		dnsData := DNSData{meta, spec}
		dnsLogs = append(dnsLogs, dnsData.ToDNSLog(
			a.aggregationStartTime,
			aggregationEndTime,
			a.includeLabels,
		))
	}
	if a.numUnloggedUpdates > 0 {
		log.Warningf(
			"%v DNS responses were not logged, because of DNSLogsFilePerNodeLimit being set to %v",
			a.numUnloggedUpdates,
			a.perNodeLimit,
		)
		// Emit an Elastic log to alert about the unlogged updates.  This log has no content
		// except for the time period and the number of updates that could not be fully
		// logged.
		excessLog := &v1.DNSLog{
			StartTime: a.aggregationStartTime,
			EndTime:   aggregationEndTime,
			Type:      v1.DNSLogTypeUnlogged,
			Count:     uint(a.numUnloggedUpdates),
		}
		dnsLogs = append(dnsLogs, excessLog)
	}
	a.dnsStore = make(map[DNSMeta]DNSSpec)
	a.aggregationStartTime = aggregationEndTime
	a.numUnloggedUpdates = 0
	return dnsLogs
}
