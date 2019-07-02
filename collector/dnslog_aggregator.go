// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package collector

import (
	"sync"
	"time"

	"github.com/google/gopacket/layers"

	"github.com/projectcalico/felix/rules"
)

// dnsLogAggregator builds and implements the DNSLogAggregator and
// DNSLogGetter interfaces.
// The dnsLogAggregator is responsible for creating, aggregating, and storing
// aggregated dns logs until the dns logs are exported.
type dnsLogAggregator struct {
	kind                 AggregationKind
	dnsStore             map[DNSMeta]DNSSpec
	flMutex              sync.RWMutex
	includeLabels        bool
	aggregationStartTime time.Time
	handledAction        rules.RuleAction
}

// NewDNSLogAggregator constructs a DNSLogAggregator
func NewDNSLogAggregator() DNSLogAggregator {
	return &dnsLogAggregator{
		kind:                 Default,
		dnsStore:             make(map[DNSMeta]DNSSpec),
		flMutex:              sync.RWMutex{},
		aggregationStartTime: time.Now(),
	}
}

func (d *dnsLogAggregator) IncludeLabels(b bool) DNSLogAggregator {
	d.includeLabels = b
	return d
}

func (d *dnsLogAggregator) AggregateOver(k AggregationKind) DNSLogAggregator {
	d.kind = k
	return d
}

func (d *dnsLogAggregator) FeedUpdate(dns *layers.DNS) error {
	meta, spec, err := NewDNSMetaSpecFromGoPacket(dns)

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
