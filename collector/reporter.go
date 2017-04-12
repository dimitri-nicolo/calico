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

// TODO(doublek): Finalize felix_ metric names.
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
	Report(data Data) error
	Expire(data Data) error
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

type ReporterManager struct {
	ReportChan chan Data
	ExpireChan chan Data
	reporters  []MetricsReporter
}

func NewReporterManager() *ReporterManager {
	return &ReporterManager{
		ReportChan: make(chan Data),
		ExpireChan: make(chan Data),
	}
}

func (r *ReporterManager) RegisterMetricsReporter(mr MetricsReporter) {
	r.reporters = append(r.reporters, mr)
}

func (r *ReporterManager) Start() {
	for _, reporter := range r.reporters {
		reporter.Start()
	}
	go r.startManaging()
}

func (r *ReporterManager) startManaging() {
	log.Info("Staring ReporterManager")
	for {
		// TODO(doublek): Channel for stopping the reporter.
		select {
		case data := <-r.ReportChan:
			for _, reporter := range r.reporters {
				reporter.Report(data)
			}
		case data := <-r.ExpireChan:
			for _, reporter := range r.reporters {
				reporter.Expire(data)
			}
		}
	}
}

func filterAndHandleData(handler func(*RuleTrace, Data), data Data) {
	if data.EgressAction() == DenyAction {
		handler(data.EgressRuleTrace, data)
	}
	if data.IngressAction() == DenyAction {
		handler(data.IngressRuleTrace, data)
	}
}

// PrometheusReporter records denied packets and bytes statistics in prometheus metrics.
type PrometheusReporter struct {
	aggStats         map[AggregateKey]AggregateValue
	deleteCandidates set.Set
	deleteChan       chan AggregateKey
	reportChan       chan Data
	expireChan       chan Data
	retentionTimers  map[AggregateKey]*time.Timer
	retentionTime    time.Duration
}

func NewPrometheusReporter(rTime time.Duration) *PrometheusReporter {
	return &PrometheusReporter{
		aggStats:         make(map[AggregateKey]AggregateValue),
		deleteCandidates: set.New(),
		deleteChan:       make(chan AggregateKey),
		reportChan:       make(chan Data),
		expireChan:       make(chan Data),
		retentionTimers:  make(map[AggregateKey]*time.Timer),
		retentionTime:    rTime,
	}
}

func (pr *PrometheusReporter) Start() {
	go pr.startReporter()
}

func (pr *PrometheusReporter) startReporter() {
	log.Info("Staring PrometheusReporter")
	for {
		select {
		case key := <-pr.deleteChan:
			// If a timer was stopped by us, then the key will not exist. Delete only
			// when a timer was fired rather than stopped.
			if _, exists := pr.retentionTimers[key]; exists {
				pr.deleteMetric(key)
			}
		case data := <-pr.reportChan:
			if !data.IsDirty() {
				continue
			}
			filterAndHandleData(pr.reportMetric, data)
		case data := <-pr.expireChan:
			filterAndHandleData(pr.expireMetric, data)
		}
	}
}

func (pr *PrometheusReporter) Report(data Data) error {
	pr.reportChan <- data
	return nil
}

func (pr *PrometheusReporter) reportMetric(ruleTrace *RuleTrace, data Data) {
	key := AggregateKey{ruleTrace.ToString(), data.Tuple.src}
	value, ok := pr.aggStats[key]
	if ok {
		if pr.deleteCandidates.Contains(key) {
			pr.deleteCandidates.Discard(key)
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
	dp, db := data.ctr.DeltaValues()
	value.packets.Add(float64(dp))
	value.bytes.Add(float64(db))
	pr.aggStats[key] = value
	log.Debugf("Metric is %+v", value.packets)
	return
}

func (pr *PrometheusReporter) Expire(data Data) error {
	pr.expireChan <- data
	return nil
}

func (pr *PrometheusReporter) expireMetric(ruleTrace *RuleTrace, data Data) {
	key := AggregateKey{ruleTrace.ToString(), data.Tuple.src}
	value, ok := pr.aggStats[key]
	if !ok || !value.refs.Contains(data.Tuple) {
		return
	}
	// If the data had updated counters this is the time to update our counters.
	// We retain deleted data for a little bit so that prometheus can get a chance
	// to scrape the data.
	if data.IsDirty() {
		dp, db := data.ctr.DeltaValues()
		value.packets.Add(float64(dp))
		value.bytes.Add(float64(db))
		pr.aggStats[key] = value
	}
	value.refs.Discard(data.Tuple)
	pr.aggStats[key] = value
	if value.refs.Len() == 0 {
		pr.markForDeletion(key)
	}
	return
}

func (pr *PrometheusReporter) markForDeletion(key AggregateKey) {
	log.WithField("key", key).Debug("Marking metric for deletion.")
	pr.deleteCandidates.Add(key)
	timer := time.NewTimer(pr.retentionTime)
	pr.retentionTimers[key] = timer
	go func() {
		log.Debugf("Starting retention timer for key %+v", key)
		<-timer.C
		pr.deleteChan <- key
	}()
}

func (pr *PrometheusReporter) deleteMetric(key AggregateKey) {
	log.WithField("key", key).Debug("Cleaning up candidate marked to be deleted.")
	value, ok := pr.aggStats[key]
	if ok {
		gaugeDeniedPackets.Delete(value.labels)
		gaugeDeniedBytes.Delete(value.labels)
		delete(pr.aggStats, key)
	}
	pr.deleteCandidates.Discard(key)
	pr.cleanupRetentionTimer(key)
}

func (pr *PrometheusReporter) cleanupRetentionTimer(key AggregateKey) {
	log.Debugf("Cleaning up retention timer for key %+v", key)
	timer, exists := pr.retentionTimers[key]
	if exists {
		delete(pr.retentionTimers, key)
		timer.Stop()
	}
}

type SyslogReporter struct {
	slog *log.Logger
}

func NewSyslogReporter() *SyslogReporter {
	slog := log.New()
	hook, err := logrus_syslog.NewSyslogHook("", "", syslog.LOG_INFO, "")
	if err != nil {
		log.Errorf("Syslog Reporting is disabled - Syslog Hook could not be configured %v", err)
		return nil
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

func (sr *SyslogReporter) Report(data Data) error {
	if !data.IsDirty() {
		return nil
	}
	filterAndHandleData(sr.log, data)
	return nil
}

func (sr *SyslogReporter) log(ruleTrace *RuleTrace, data Data) {
	packets, bytes := data.ctr.Values()
	f := log.Fields{
		"proto":   strconv.Itoa(data.Tuple.proto),
		"srcIP":   data.Tuple.src,
		"srcPort": strconv.Itoa(data.Tuple.l4Src),
		"dstIP":   data.Tuple.dst,
		"dstPort": strconv.Itoa(data.Tuple.l4Dst),
		"policy":  ruleTrace.ToString(),
		"action":  DenyAction,
		"packets": packets,
		"bytes":   bytes,
	}
	sr.slog.WithFields(f).Info("")
}

func (sr *SyslogReporter) Expire(data Data) error {
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
