// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package collector

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/rules"
)

// DNSAggregationKind determines how DNS logs are aggregated
type DNSAggregationKind int

const (
	// DNSDefault means no aggregation, other than for identical logs within the aggregation
	// time period (aka flush interval).
	DNSDefault DNSAggregationKind = iota
	// DNSPrefixNameAndIP aggregates logs with the same DNS information and client name prefix,
	// i.e. masking the client name and IP.
	DNSPrefixNameAndIP
)

// dnsLogAggregator builds and implements the DNSLogAggregator and
// DNSLogGetter interfaces.
// The dnsLogAggregator is responsible for creating, aggregating, and storing
// aggregated dns logs until the dns logs are exported.
type dnsLogAggregator struct {
	kind                 DNSAggregationKind
	dnsStore             map[DNSMeta]DNSSpec
	dnsMutex             sync.Mutex
	includeLabels        bool
	aggregationStartTime time.Time
	perNodeLimit         int
	handledAction        rules.RuleAction
	numUnloggedUpdates   int
}

// NewDNSLogAggregator constructs a DNSLogAggregator
func NewDNSLogAggregator() DNSLogAggregator {
	return &dnsLogAggregator{
		kind:                 DNSDefault,
		dnsStore:             make(map[DNSMeta]DNSSpec),
		aggregationStartTime: time.Now(),
	}
}

func (d *dnsLogAggregator) IncludeLabels(b bool) DNSLogAggregator {
	d.includeLabels = b
	return d
}

func (d *dnsLogAggregator) AggregateOver(k DNSAggregationKind) DNSLogAggregator {
	d.kind = k
	return d
}

func (d *dnsLogAggregator) PerNodeLimit(l int) DNSLogAggregator {
	d.perNodeLimit = l
	return d
}

func (d *dnsLogAggregator) FeedUpdate(update DNSUpdate) error {
	meta, spec, err := NewDNSMetaSpecFromUpdate(update, d.kind)

	if err != nil {
		return err
	}

	// Ensure that we can't add or aggregate new logs into the store at the
	// same time as existing logs are being flushed out.
	d.dnsMutex.Lock()
	defer d.dnsMutex.Unlock()

	if _, ok := d.dnsStore[meta]; ok {
		existing := d.dnsStore[meta]
		existing.Merge(spec)
		d.dnsStore[meta] = existing
	} else if (d.perNodeLimit == 0) || (len(d.dnsStore) < d.perNodeLimit) {
		d.dnsStore[meta] = spec
	} else {
		d.numUnloggedUpdates++
	}

	return nil
}

func (d *dnsLogAggregator) Get() []*DNSLog {
	var dnsLogs []*DNSLog
	aggregationEndTime := time.Now()

	// Ensure that we can't add or aggregate new logs into the store at the
	// same time as existing logs are being flushed out.
	d.dnsMutex.Lock()
	defer d.dnsMutex.Unlock()

	for meta, spec := range d.dnsStore {
		dnsData := DNSData{meta, spec}
		dnsLogs = append(dnsLogs, dnsData.ToDNSLog(
			d.aggregationStartTime,
			aggregationEndTime,
			d.includeLabels,
		))
	}
	if d.numUnloggedUpdates > 0 {
		log.Warningf(
			"%v DNS responses were not logged, because of DNSLogsFilePerNodeLimit being set to %v",
			d.numUnloggedUpdates,
			d.perNodeLimit,
		)
	}
	d.dnsStore = make(map[DNSMeta]DNSSpec)
	d.aggregationStartTime = aggregationEndTime
	d.numUnloggedUpdates = 0
	return dnsLogs
}
