// Copyright (c) 2017 Tigera, Inc. All rights reserved.

package collector

import (
	"encoding/json"
	"fmt"
	"log/syslog"
	"strconv"

	log "github.com/Sirupsen/logrus"
	logrus_syslog "github.com/Sirupsen/logrus/hooks/syslog"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/projectcalico/felix/set"
)

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

type PrometheusReporter struct {
	aggStats map[AggregateKey]AggregateValue
}

func NewPrometheusReporter() *PrometheusReporter {
	return &PrometheusReporter{
		aggStats: make(map[AggregateKey]AggregateValue),
	}
}

func (pr *PrometheusReporter) Update(data Data) error {
	if data.Action() != DenyAction || !data.IsDirty() {
		return nil
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
	return nil
}

func (pr *PrometheusReporter) Delete(data Data) error {
	ruleTrace := data.RuleTrace.ToString()
	srcIP := data.Tuple.src
	key := AggregateKey{ruleTrace, srcIP}
	value, ok := pr.aggStats[key]
	if !ok {
		return nil
	}
	if !value.refs.Contains(data.Tuple) {
		return nil
	}
	value.refs.Discard(data.Tuple)
	pr.aggStats[key] = value
	// TODO(doublek): We are deleting too early. We should probably hang on to
	// counter value for a little bit before deleting the metric value labels.
	if value.refs.Len() == 0 {
		log.WithFields(log.Fields{
			"labels": value.labels,
		}).Debug("Deleting prometheus metrics.")
		gaugeDeniedPackets.Delete(value.labels)
		gaugeDeniedBytes.Delete(value.labels)
		delete(pr.aggStats, key)
	}
	return nil
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
