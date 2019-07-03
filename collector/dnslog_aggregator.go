// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package collector

import (
	"sync"
	"time"

	"github.com/projectcalico/felix/rules"
)

// FlowAggregationKind determines the flow log key
type DNSAggregationKind int

const (
	// DNSDefault is based on purely duration.
	DNSDefault DNSAggregationKind = iota
	// DNSPrefixNameAndIP accumulates tuples with everything same but the prefix name and IP
	DNSPrefixNameAndIP
	// DNSQName accumulates tuples by aggregating query names
	DNSQName
)

// dnsLogAggregator builds and implements the DNSLogAggregator and
// DNSLogGetter interfaces.
// The dnsLogAggregator is responsible for creating, aggregating, and storing
// aggregated dns logs until the dns logs are exported.
type dnsLogAggregator struct {
	kind                 DNSAggregationKind
	dnsStore             map[DNSMeta]DNSSpec
	flMutex              sync.RWMutex
	includeLabels        bool
	aggregationStartTime time.Time
	handledAction        rules.RuleAction
}

// NewDNSLogAggregator constructs a DNSLogAggregator
func NewDNSLogAggregator() DNSLogAggregator {
	return &dnsLogAggregator{
		kind:                 DNSDefault,
		dnsStore:             make(map[DNSMeta]DNSSpec),
		flMutex:              sync.RWMutex{},
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

func (d *dnsLogAggregator) FeedUpdate(update DNSUpdate) error {
	meta, spec, err := NewDNSMetaSpecFromUpdate(update, d.kind)

	if err != nil {
		return err
	}

	if _, ok := d.dnsStore[meta]; ok {
		existing := d.dnsStore[meta]
		existing.Merge(spec)
		d.dnsStore[meta] = existing
	} else {
		d.dnsStore[meta] = spec
	}

	return nil
}

func (d *dnsLogAggregator) Get() []*DNSLog {
	var dnsLogs []*DNSLog
	aggregationEndTime := time.Now()
	for meta, spec := range d.dnsStore {
		dnsData := DNSData{meta, spec}
		dnsLogs = append(dnsLogs, dnsData.ToDNSLog(
			d.aggregationStartTime,
			aggregationEndTime,
			d.includeLabels,
		))
	}
	d.dnsStore = make(map[DNSMeta]DNSSpec)
	d.aggregationStartTime = aggregationEndTime
	return dnsLogs
}
