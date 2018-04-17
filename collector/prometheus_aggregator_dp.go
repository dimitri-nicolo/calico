// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package collector

import (
	"net"
	"time"

	"github.com/gavv/monotime"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/rules"
)

// Calico Metrics
var (
	gaugeDeniedPackets = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "calico_denied_packets",
		Help: "Total number of packets denied by calico policies.",
	},
		[]string{"srcIP", "policy", LABEL_INSTANCE},
	)
	gaugeDeniedBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "calico_denied_bytes",
		Help: "Total number of bytes denied by calico policies.",
	},
		[]string{"srcIP", "policy", LABEL_INSTANCE},
	)
)

type DeniedPacketsAggregateKey struct {
	policy string
	srcIP  [16]byte
}

func getDeniedPacketsAggregateKey(mu MetricUpdate) DeniedPacketsAggregateKey {
	return DeniedPacketsAggregateKey{
		policy: mu.ruleID.GetDeniedPacketRuleName(),
		srcIP:  mu.tuple.src,
	}
}

type DeniedPacketsAggregateValue struct {
	labels  prometheus.Labels
	packets prometheus.Gauge
	bytes   prometheus.Gauge
	refs    tupleSet
}

// DeniedPacketsAggregator aggregates denied packets and bytes statistics in prometheus metrics.
type DeniedPacketsAggregator struct {
	retentionTime time.Duration
	timeNowFn     func() time.Duration
	// Stats are aggregated by policy (mangled tiered policy rule) and source IP.
	aggStats        map[DeniedPacketsAggregateKey]DeniedPacketsAggregateValue
	retainedMetrics map[DeniedPacketsAggregateKey]time.Duration
	felixHostname   string
}

func NewDeniedPacketsAggregator(rTime time.Duration, felixHostname string) *DeniedPacketsAggregator {
	return &DeniedPacketsAggregator{
		aggStats:        make(map[DeniedPacketsAggregateKey]DeniedPacketsAggregateValue),
		retainedMetrics: make(map[DeniedPacketsAggregateKey]time.Duration),
		retentionTime:   rTime,
		timeNowFn:       monotime.Now,
		felixHostname:   felixHostname,
	}
}

func (dp *DeniedPacketsAggregator) RegisterMetrics(registry *prometheus.Registry) {
	registry.MustRegister(gaugeDeniedPackets)
	registry.MustRegister(gaugeDeniedBytes)
}

func (dp *DeniedPacketsAggregator) OnUpdate(mu MetricUpdate) {
	if mu.ruleID.Action != rules.RuleActionDeny {
		// We only want denied packets. Skip the rest of them.
		return
	}
	switch mu.updateType {
	case UpdateTypeReport:
		dp.reportMetric(mu)
	case UpdateTypeExpire:
		dp.expireMetric(mu)
	}
}

func (dp *DeniedPacketsAggregator) CheckRetainedMetrics(now time.Duration) {
	for key, expirationTime := range dp.retainedMetrics {
		if now >= expirationTime {
			dp.deleteMetric(key)
			delete(dp.retainedMetrics, key)
		}
	}
}

func (dp *DeniedPacketsAggregator) reportMetric(mu MetricUpdate) {
	key := getDeniedPacketsAggregateKey(mu)
	value, ok := dp.aggStats[key]
	if ok {
		_, exists := dp.retainedMetrics[key]
		if exists {
			delete(dp.retainedMetrics, key)
		}
		value.refs.Add(mu.tuple)
	} else {
		l := prometheus.Labels{
			"srcIP":        net.IP(key.srcIP[:16]).String(),
			"policy":       key.policy,
			LABEL_INSTANCE: dp.felixHostname,
		}
		value = DeniedPacketsAggregateValue{
			labels:  l,
			packets: gaugeDeniedPackets.With(l),
			bytes:   gaugeDeniedBytes.With(l),
			refs:    NewTupleSet(),
		}
		value.refs.Add(mu.tuple)
	}
	switch mu.ruleID.Direction {
	case rules.RuleDirIngress:
		value.packets.Add(float64(mu.inMetric.deltaPackets))
		value.bytes.Add(float64(mu.inMetric.deltaBytes))
	case rules.RuleDirEgress:
		value.packets.Add(float64(mu.outMetric.deltaPackets))
		value.bytes.Add(float64(mu.outMetric.deltaBytes))
	default:
		return
	}
	dp.aggStats[key] = value
	return
}

func (dp *DeniedPacketsAggregator) expireMetric(mu MetricUpdate) {
	key := getDeniedPacketsAggregateKey(mu)
	value, ok := dp.aggStats[key]
	if !ok || !value.refs.Contains(mu.tuple) {
		return
	}
	// If the metric update has updated counters this is the time to update our counters.
	// We retain deleted metric for a little bit so that prometheus can get a chance
	// to scrape the metric.
	var deltaPackets, deltaBytes int
	switch mu.ruleID.Direction {
	case rules.RuleDirIngress:
		deltaPackets = mu.inMetric.deltaPackets
		deltaBytes = mu.inMetric.deltaBytes
	case rules.RuleDirEgress:
		deltaPackets = mu.outMetric.deltaPackets
		deltaBytes = mu.outMetric.deltaBytes
	default:
		return
	}
	if deltaPackets != 0 && deltaBytes != 0 {
		value.packets.Add(float64(deltaPackets))
		value.bytes.Add(float64(deltaBytes))
		dp.aggStats[key] = value
	}
	value.refs.Discard(mu.tuple)
	dp.aggStats[key] = value
	if value.refs.Len() == 0 {
		dp.markForDeletion(key)
	}
	return
}

func (dp *DeniedPacketsAggregator) markForDeletion(key DeniedPacketsAggregateKey) {
	log.WithField("key", key).Debug("Marking metric for deletion.")
	dp.retainedMetrics[key] = dp.timeNowFn() + dp.retentionTime
}

func (dp *DeniedPacketsAggregator) deleteMetric(key DeniedPacketsAggregateKey) {
	log.WithField("key", key).Debug("Cleaning up candidate marked to be deleted.")
	value, ok := dp.aggStats[key]
	if ok {
		gaugeDeniedPackets.Delete(value.labels)
		gaugeDeniedBytes.Delete(value.labels)
		delete(dp.aggStats, key)
	}
}
