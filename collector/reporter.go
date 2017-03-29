// Copyright (c) 2017 Tigera, Inc. All rights reserved.

package collector

import (
	"encoding/json"
	"fmt"
	"log/syslog"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	logrus_syslog "github.com/Sirupsen/logrus/hooks/syslog"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/projectcalico/felix/set"
)

const RetentionTime = time.Duration(10) * time.Second

var (
	gaugeDeniedPackets = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "felix_collector_denied_packets",
		Help: "Packets denied.",
	},
		[]string{"srcIP", "policy"},
	)
	gaugeDeniedBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "felix_collector_denied_bytes",
		Help: "Bytes denied.",
	},
		[]string{"srcIP", "policy"},
	)
)

func init() {
	prometheus.MustRegister(gaugeDeniedPackets)
	prometheus.MustRegister(gaugeDeniedBytes)
}

type MetricsReporter interface {
	Start()
	Update(data Data) error
	Delete(data Data) error
}

// TODO(doublek): When we want different ways of aggregating, this will
// need to be dynamic and a KeyType.
type AggregateKey struct {
	policy string
	srcIP  string
}

type AggregateValue struct {
	labels  prometheus.Labels
	packets prometheus.Gauge
	bytes   prometheus.Gauge
	refs    set.Set
}

// PrometheusReporter records denied packets and bytes statistics in prometheus metrics.
//
type PrometheusReporter struct {
	aggStats        map[AggregateKey]AggregateValue
	dropCandidates  set.Set
	dropChan        chan AggregateKey
	updateChan      chan Data
	deleteChan      chan Data
	retentionTimers map[AggregateKey]*time.Timer
}

func NewPrometheusReporter() *PrometheusReporter {
	return &PrometheusReporter{
		aggStats:        make(map[AggregateKey]AggregateValue),
		dropCandidates:  set.New(),
		dropChan:        make(chan AggregateKey),
		updateChan:      make(chan Data),
		deleteChan:      make(chan Data),
		retentionTimers: make(map[AggregateKey]*time.Timer),
	}
}

func (pr *PrometheusReporter) Start() {
	go pr.startReporter()
}

func (pr *PrometheusReporter) startReporter() {
	log.Info("Staring PrometheusReporter")
	for {
		select {
		case key := <-pr.dropChan:
			pr.dropMetric(key)
		case data := <-pr.updateChan:
			pr.updateMetric(data)
		case data := <-pr.deleteChan:
			pr.deleteMetric(data)
		}
	}
}

func (pr *PrometheusReporter) Update(data Data) error {
	pr.updateChan <- data
	return nil
}

func (pr *PrometheusReporter) updateMetric(data Data) {
	if data.Action() != DenyAction || !data.IsDirty() {
		return
	}
	key := AggregateKey{data.RuleTrace.ToString(), data.Tuple.src}
	var ctr Counter
	// TODO(doublek): This is a temporary workaround until direction awareness
	// of tuples/data via NFLOG makes its way in.
	if !data.ctrIn.IsZero() {
		ctr = data.ctrIn
	} else {
		ctr = data.ctrOut
	}
	value, ok := pr.aggStats[key]
	if ok {
		if pr.dropCandidates.Contains(key) {
			pr.dropCandidates.Discard(key)
			pr.cleanupRetentionTimer(key)
		}
		value.refs.Add(data.Tuple)
	} else {
		l := prometheus.Labels{
			"srcIP":  key.srcIP,
			"policy": key.policy,
		}
		value = AggregateValue{
			labels:  l,
			packets: gaugeDeniedPackets.With(l),
			bytes:   gaugeDeniedBytes.With(l),
			refs:    set.FromArray([]Tuple{data.Tuple}),
		}
	}
	dp, db := ctr.DeltaValues()
	value.packets.Add(float64(dp))
	value.bytes.Add(float64(db))
	pr.aggStats[key] = value
	return
}

func (pr *PrometheusReporter) Delete(data Data) error {
	pr.deleteChan <- data
	return nil
}

func (pr *PrometheusReporter) deleteMetric(data Data) {
	ruleTrace := data.RuleTrace.ToString()
	srcIP := data.Tuple.src
	key := AggregateKey{ruleTrace, srcIP}
	value, ok := pr.aggStats[key]
	if !ok {
		return
	}
	if !value.refs.Contains(data.Tuple) {
		return
	}
	value.refs.Discard(data.Tuple)
	pr.aggStats[key] = value
	// TODO(doublek): We are deleting too early. We should probably hang on to
	// counter value for a little bit before deleting the metric value labels.
	if value.refs.Len() == 0 {
		pr.markForDrop(key)
	}
	return
}

func (pr *PrometheusReporter) markForDrop(key AggregateKey) {
	log.WithField("key", key).Debug("Marking for dropping metric.")
	pr.dropCandidates.Add(key)
	timer := time.NewTimer(RetentionTime)
	pr.retentionTimers[key] = timer
	go func() {
		<-timer.C
		pr.dropChan <- key
	}()
}

func (pr *PrometheusReporter) dropMetric(key AggregateKey) {
	log.WithField("key", key).Debug("Cleaning up candidate marked to be dropped.")
	value, ok := pr.aggStats[key]
	if ok {
		gaugeDeniedPackets.Delete(value.labels)
		gaugeDeniedBytes.Delete(value.labels)
		delete(pr.aggStats, key)
	}
	pr.dropCandidates.Discard(key)
	pr.cleanupRetentionTimer(key)
}

func (pr *PrometheusReporter) cleanupRetentionTimer(key AggregateKey) {
	timer, exists := pr.retentionTimers[key]
	if exists && !timer.Stop() {
		<-timer.C
	}
	delete(pr.retentionTimers, key)
}

type SyslogReporter struct {
	slog *log.Logger
}

func NewSyslogReporter() *SyslogReporter {
	slog := log.New()
	hook, err := logrus_syslog.NewSyslogHook("", "", syslog.LOG_INFO, "")
	if err != nil {
		panic(fmt.Sprintf("Syslog Hook could not be configured %v", err))
	}
	slog.Hooks.Add(hook)
	slog.Formatter = &DataOnlyJSONFormatter{}
	return &SyslogReporter{
		slog: slog,
	}
}

func (sr *SyslogReporter) Start() {
	log.Info("Staring SyslogReporter")
}

func (sr *SyslogReporter) Update(data Data) error {
	if data.Action() != DenyAction || !data.IsDirty() {
		return nil
	}
	var bytes, packets int
	if data.ctrIn.packets != 0 {
		packets, bytes = data.ctrIn.Values()
	} else {
		packets, bytes = data.ctrOut.Values()
	}
	f := log.Fields{
		"proto":   strconv.Itoa(data.Tuple.proto),
		"srcIP":   data.Tuple.src,
		"srcPort": strconv.Itoa(data.Tuple.l4Src),
		"dstIP":   data.Tuple.dst,
		"dstPort": strconv.Itoa(data.Tuple.l4Dst),
		"policy":  data.RuleTrace.ToString(),
		"action":  DenyAction,
		"packets": packets,
		"bytes":   bytes,
	}
	sr.slog.WithFields(f).Info("")
	return nil
}

func (sr *SyslogReporter) Delete(data Data) error {
	return nil
}

// Logrus Formatter that strips the log entry of messages, time and log level and
// outputs *only* entry.Data.
type DataOnlyJSONFormatter struct{}

func (f *DataOnlyJSONFormatter) Format(entry *log.Entry) ([]byte, error) {
	serialized, err := json.Marshal(entry.Data)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal data to JSON %v", err)
	}
	return append(serialized, '\n'), nil
}
