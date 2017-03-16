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
)

var (
	gaugeDeniedPackets = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "felix_denied_packets",
		Help: "Packets denied.",
	},
		[]string{"srcIP", "policy"},
	)
	gaugeDeniedBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "felix_denied_bytes",
		Help: "Bytes denied.",
	},
		[]string{"srcIP", "policy"},
	)
)

func init() {
	prometheus.MustRegister(gaugeDeniedPackets)
	prometheus.MustRegister(gaugeDeniedBytes)
}

type Fields map[string]string

type MetricsReporter interface {
	Update(data Data) error
	Delete(data Data) error
	Flush() error
	Report(fields Fields, bytes int, packets int) error
	Clear(fields Fields) error
}

func convertFieldsToPrometheusLabels(fields Fields) prometheus.Labels {
	l := make(prometheus.Labels)
	for k, v := range fields {
		l[k] = v
	}
	return l
}

func convertFieldsToLogrusFields(fields Fields) log.Fields {
	f := make(log.Fields)
	for k, v := range fields {
		f[k] = v
	}
	return f
}

// TODO(doublek): When we want different ways of aggregating, this will
// need to be dynamic and a KeyType.
// TODO(doublek): Converting from a AggregateKey to a Field.
type AggregateKey struct {
	policy string
	srcIP  string
}

type AggregateValue struct {
	Counter
	refs int
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
	if data.Action() != DenyAction {
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
		value.bytes += ctr.bytes
		value.packets += ctr.packets
		value.refs++
	} else {
		value = AggregateValue{
			Counter: ctr,
			refs:    1,
		}
	}
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
	value.refs--
	pr.aggStats[key] = value
	if value.refs == 0 {
		f := Fields{
			"policy": ruleTrace,
			"srcIP":  srcIP,
		}
		pr.Clear(f)
		delete(pr.aggStats, key)
	}
	return nil
}

func (pr *PrometheusReporter) Flush() error {
	for key, value := range pr.aggStats {
		f := Fields{
			"policy": key.policy,
			"srcIP":  key.srcIP,
		}
		pr.Report(f, value.bytes, value.packets)
		value.Counter.Reset()
		pr.aggStats[key] = value
	}
	return nil
}

func (pr *PrometheusReporter) Report(fields Fields, bytes int, packets int) error {
	l := convertFieldsToPrometheusLabels(fields)
	log.WithFields(log.Fields{
		"labels":  l,
		"bytes":   bytes,
		"packets": packets,
	}).Debug("Setting prometheus metrics.")
	gaugeDeniedPackets.With(l).Set(float64(packets))
	gaugeDeniedBytes.With(l).Set(float64(bytes))
	return nil
}

func (pr *PrometheusReporter) Clear(fields Fields) error {
	l := convertFieldsToPrometheusLabels(fields)
	log.WithFields(log.Fields{
		"labels": l,
	}).Debug("Deleting prometheus metrics.")
	gaugeDeniedPackets.Delete(l)
	gaugeDeniedBytes.Delete(l)
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
	if data.Action() != DenyAction {
		return nil
	}
	f := Fields{
		"proto":   strconv.Itoa(data.Tuple.proto),
		"srcIp":   data.Tuple.src,
		"srcPort": strconv.Itoa(data.Tuple.l4Src),
		"dstIp":   data.Tuple.dst,
		"dstPort": strconv.Itoa(data.Tuple.l4Dst),
		"policy":  data.RuleTrace.ToString(),
	}
	var bytes, packets int
	if data.ctrIn.packets != 0 {
		bytes = data.ctrIn.bytes
		packets = data.ctrIn.packets
	} else {
		bytes = data.ctrOut.bytes
		packets = data.ctrOut.packets
	}
	return sr.Report(f, bytes, packets)
}

func (sr *SyslogReporter) Delete(data Data) error {
	return nil
}

func (sr *SyslogReporter) Flush() error {
	return nil
}

func (sr *SyslogReporter) Report(fields Fields, bytes int, packets int) error {
	f := convertFieldsToLogrusFields(fields)
	f["action"] = DenyAction
	f["packets"] = packets
	f["bytes"] = bytes
	sr.slog.WithFields(f).Info("")
	return nil
}

func (sr *SyslogReporter) Clear(fields Fields) error {
	// We don't maintain any state, so nothing to do here.
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
