// Copyright (c) 2017-2018 Tigera, Inc. All rights reserved.

package collector

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gavv/monotime"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/jitter"
	"github.com/projectcalico/felix/rules"
	"github.com/projectcalico/libcalico-go/lib/set"
)

const checkInterval = 5 * time.Second

// CNX Metrics
var (
	LABEL_TIER        = "tier"
	LABEL_POLICY      = "policy"
	LABEL_RULE_IDX    = "rule_index"
	LABEL_ACTION      = "action"
	LABEL_TRAFFIC_DIR = "traffic_direction"
	LABEL_RULE_DIR    = "rule_direction"

	counterRulePackets = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cnx_policy_rule_packets",
			Help: "Total number of packets handled by CNX policy rules.",
		},
		[]string{LABEL_ACTION, LABEL_TIER, LABEL_POLICY, LABEL_RULE_DIR, LABEL_RULE_IDX, LABEL_TRAFFIC_DIR},
	)
	counterRuleBytes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cnx_policy_rule_bytes",
			Help: "Total number of bytes handled by CNX policy rules.",
		},
		[]string{LABEL_ACTION, LABEL_TIER, LABEL_POLICY, LABEL_RULE_DIR, LABEL_RULE_IDX, LABEL_TRAFFIC_DIR},
	)
	gaugeRuleConns = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cnx_policy_rule_connections",
			Help: "Total number of connections handled by CNX policy rules.",
		},
		[]string{LABEL_TIER, LABEL_POLICY, LABEL_RULE_DIR, LABEL_RULE_IDX, LABEL_TRAFFIC_DIR},
	)

	ruleDirToTrafficDir = map[rules.RuleDirection]rules.TrafficDirection{
		rules.RuleDirIngress: rules.TrafficDirInbound,
		rules.RuleDirEgress:  rules.TrafficDirOutbound,
	}
)

// PrometheusReporter records denied packets and bytes statistics in prometheus metrics.
type PrometheusReporter struct {
	port            int
	certFile        string
	keyFile         string
	caFile          string
	registry        *prometheus.Registry
	reportChan      chan *MetricUpdate
	retentionTime   time.Duration
	retentionTicker *jitter.Ticker

	// Allow the time function to be mocked for test purposes.
	timeNowFn func() time.Duration

	// Stats are aggregated by rule.
	ruleAggStats           map[RuleAggregateKey]*RuleAggregateValue
	retainedRuleAggMetrics map[RuleAggregateKey]time.Duration
}

func NewPrometheusReporter(port int, retentionTime time.Duration, certFile, keyFile, caFile string) *PrometheusReporter {
	// Set the ticker interval appropriately, we should be checking at least half of the rention time,
	// or the hard-coded check interval (whichever is smaller).
	tickerInterval := retentionTime / 2
	if checkInterval < tickerInterval {
		tickerInterval = checkInterval
	}

	registry := prometheus.NewRegistry()
	registry.MustRegister(counterRuleBytes)
	registry.MustRegister(counterRulePackets)
	registry.MustRegister(gaugeRuleConns)
	return &PrometheusReporter{
		port:                   port,
		certFile:               certFile,
		keyFile:                keyFile,
		caFile:                 caFile,
		registry:               registry,
		reportChan:             make(chan *MetricUpdate),
		retentionTime:          retentionTime,
		retentionTicker:        jitter.NewTicker(tickerInterval, tickerInterval/10),
		ruleAggStats:           make(map[RuleAggregateKey]*RuleAggregateValue),
		retainedRuleAggMetrics: make(map[RuleAggregateKey]time.Duration),
		timeNowFn:              monotime.Now,
	}
}

type RuleAggregateKey struct {
	ruleIDs rules.RuleIDs
}

// getRuleAggregateKey returns a hashable key identifying a rule aggregation key.
func getRuleAggregateKey(mu *MetricUpdate) RuleAggregateKey {
	return RuleAggregateKey{
		ruleIDs: *mu.ruleIDs,
	}
}

// PacketByteLabels returns the Prometheus packet/byte counter labels associated
// with a specific rule and traffic direction.
func (k *RuleAggregateKey) PacketByteLabels(trafficDir rules.TrafficDirection) prometheus.Labels {
	return prometheus.Labels{
		LABEL_ACTION:      string(k.ruleIDs.Action),
		LABEL_TIER:        k.ruleIDs.Tier,
		LABEL_POLICY:      k.ruleIDs.Policy,
		LABEL_RULE_DIR:    string(k.ruleIDs.Direction),
		LABEL_RULE_IDX:    k.ruleIDs.Index,
		LABEL_TRAFFIC_DIR: string(trafficDir),
	}
}

// ConnectionLabels returns the Prometheus connection gauge labels associated
// with a specific rule and traffic direction.
func (k *RuleAggregateKey) ConnectionLabels() prometheus.Labels {
	return prometheus.Labels{
		LABEL_TIER:        k.ruleIDs.Tier,
		LABEL_POLICY:      k.ruleIDs.Policy,
		LABEL_RULE_DIR:    string(k.ruleIDs.Direction),
		LABEL_RULE_IDX:    k.ruleIDs.Index,
		LABEL_TRAFFIC_DIR: string(ruleDirToTrafficDir[k.ruleIDs.Direction]),
	}
}

type RuleAggregateValue struct {
	inPackets      prometheus.Counter
	inBytes        prometheus.Counter
	outPackets     prometheus.Counter
	outBytes       prometheus.Counter
	numConnections prometheus.Gauge
	tuples         set.Set
	isConnection   bool
}

func (pr *PrometheusReporter) Start() {
	log.Info("Starting PrometheusReporter")
	go pr.servePrometheusMetrics()
	go pr.startReporter()
}

func (pr *PrometheusReporter) Report(mu *MetricUpdate) error {
	pr.reportChan <- mu
	return nil
}

func getRuleAggregateValue(key RuleAggregateKey, isConnection bool) (*RuleAggregateValue, error) {
	value := &RuleAggregateValue{
		tuples:       set.New(),
		isConnection: isConnection,
	}
	switch key.ruleIDs.Direction {
	case rules.RuleDirIngress:
		pbInLabels := key.PacketByteLabels(rules.TrafficDirInbound)
		value.inPackets = counterRulePackets.With(pbInLabels)
		value.inBytes = counterRuleBytes.With(pbInLabels)
		if isConnection {
			pbOutLabels := key.PacketByteLabels(rules.TrafficDirOutbound)
			value.outPackets = counterRulePackets.With(pbOutLabels)
			value.outBytes = counterRuleBytes.With(pbOutLabels)
		}
	case rules.RuleDirEgress:
		pbOutLabels := key.PacketByteLabels(rules.TrafficDirOutbound)
		value.outPackets = counterRulePackets.With(pbOutLabels)
		value.outBytes = counterRuleBytes.With(pbOutLabels)
		if isConnection {
			pbInLabels := key.PacketByteLabels(rules.TrafficDirInbound)
			value.inPackets = counterRulePackets.With(pbInLabels)
			value.inBytes = counterRuleBytes.With(pbInLabels)
		}
	default:
		return nil, fmt.Errorf("Unknown traffic direction in ruleId %v", key.ruleIDs)
	}
	if isConnection {
		cLabels := key.ConnectionLabels()
		value.numConnections = gaugeRuleConns.With(cLabels)
	}
	return value, nil
}

// servePrometheusMetrics starts a lightweight web server to server prometheus metrics.
func (pr *PrometheusReporter) servePrometheusMetrics() {
	for {
		mux := http.NewServeMux()
		handler := promhttp.HandlerFor(pr.registry, promhttp.HandlerOpts{})
		mux.Handle("/metrics", handler)
		var err error
		if pr.certFile != "" && pr.keyFile != "" && pr.caFile != "" {
			caCert, err := ioutil.ReadFile(pr.caFile)
			if err == nil {
				caCertPool := x509.NewCertPool()
				caCertPool.AppendCertsFromPEM(caCert)
				cfg := &tls.Config{
					ClientAuth: tls.RequireAndVerifyClientCert,
					ClientCAs:  caCertPool,
				}
				srv := &http.Server{
					Addr:      fmt.Sprintf(":%v", pr.port),
					Handler:   handler,
					TLSConfig: cfg,
				}
				err = srv.ListenAndServeTLS(pr.certFile, pr.keyFile)
			}
		} else {
			err = http.ListenAndServe(fmt.Sprintf(":%v", pr.port), handler)
		}
		log.WithError(err).Error(
			"Prometheus reporter metrics endpoint failed, trying to restart it...")
		time.Sleep(1 * time.Second)
	}
}

// startReporter starts listening for and processing reports and expired metrics.
func (pr *PrometheusReporter) startReporter() {
	// Loop continuously processing metric reports and expirations.  A single
	// loop ensures access to the aggregated datastructures is single-threaded.
	for {
		select {
		case mu := <-pr.reportChan:
			pr.reportMetric(mu)
		case <-pr.retentionTicker.C:
			//TODO: RLB: Maybe improve this processing using a linked-list (ordered by time)
			now := pr.timeNowFn()
			for key, expirationTime := range pr.retainedRuleAggMetrics {
				if now >= expirationTime {
					pr.deleteRuleAggregateMetric(key)
					delete(pr.retainedRuleAggMetrics, key)
				}
			}
		}
	}
}

// reportMetric increments our counters from the metric update and ensures the metric
// will expire if there are no associated connections and no activity within the
// retention period.
// expireMetrics is actually similar to reportMetric, it increments our counters from the
// metric update, removes any connection associated with the metric and ensures the metric
// will expire if there are no associated connections and no activity within the retention
// period. Unlike reportMetric, if there is no cached entry for this metric one is not
// created and therefore the metric will not be reported.
func (pr *PrometheusReporter) reportMetric(mu *MetricUpdate) {
	pr.handleRuleMetric(mu)
}

// handleRuleMetric handles reporting and expiration of Rule-aggregated metrics.
func (pr *PrometheusReporter) handleRuleMetric(mu *MetricUpdate) {
	var (
		value *RuleAggregateValue
		err   error
	)
	key := getRuleAggregateKey(mu)
	value, ok := pr.ruleAggStats[key]
	if !ok {
		// No entry exists.  If this is a report then create a blank entry and add
		// to the map.  Otherwise, this is an expiration, so just return.
		if mu.updateType == UpdateTypeExpire {
			return
		}

		value, err = getRuleAggregateValue(key, mu.isConnection)
		if err != nil {
			log.WithField("key", key).Debugf("Cannot update metric. Skipping update.")
			return
		}
		pr.ruleAggStats[key] = value
	}

	// Increment the packet counters if non-zero.
	if mu.inMetric.deltaPackets != 0 && mu.inMetric.deltaBytes != 0 {
		value.inPackets.Add(float64(mu.inMetric.deltaPackets))
		value.inBytes.Add(float64(mu.inMetric.deltaBytes))
	}
	if mu.outMetric.deltaPackets != 0 && mu.outMetric.deltaBytes != 0 {
		value.outPackets.Add(float64(mu.outMetric.deltaPackets))
		value.outBytes.Add(float64(mu.outMetric.deltaBytes))
	}

	// If this is an active connection (and we aren't expiring the stats), add to our
	// active connections tuple, otherwise make sure it is removed.
	oldTuples := value.tuples.Len()
	if mu.isConnection && mu.updateType == UpdateTypeReport {
		value.tuples.Add(mu.tuple)
	} else {
		value.tuples.Discard(mu.tuple)
	}
	newTuples := value.tuples.Len()

	// If the number of connections has changed then update our connections gauge.
	if mu.isConnection && oldTuples != newTuples {
		value.numConnections.Set(float64(newTuples))
	}

	// If there are some connections for this rule then keep it active, otherwise (re)set the timeout
	// for this metric to ensure we tidy up after a period of inactivity.
	if newTuples > 0 {
		pr.unmarkRuleAggregateForDeletion(key)
	} else {
		pr.markRuleAggregateForDeletion(key)
	}
}

// unmarkRuleAggregateForDeletion removes a rule aggregate metric from the expiration
// list.
func (pr *PrometheusReporter) unmarkRuleAggregateForDeletion(key RuleAggregateKey) {
	log.WithField("key", key).Debug("Unmarking rule aggregate metric for deletion.")
	delete(pr.retainedRuleAggMetrics, key)
}

// markRuleAggregateForDeletion marks a rule aggregate metric for expiration.
func (pr *PrometheusReporter) markRuleAggregateForDeletion(key RuleAggregateKey) {
	log.WithField("key", key).Debug("Marking rule aggregate metric for deletion.")
	pr.retainedRuleAggMetrics[key] = pr.timeNowFn() + pr.retentionTime
}

// deleteRuleAggregateMetric deletes the prometheus metrics associated with the
// supplied key.
func (pr *PrometheusReporter) deleteRuleAggregateMetric(key RuleAggregateKey) {
	log.WithField("key", key).Debug("Cleaning up rule aggregate metric previously marked to be deleted.")
	value, ok := pr.ruleAggStats[key]
	if !ok {
		// Nothing to do here.
		return
	}
	pbInLabels := key.PacketByteLabels(rules.TrafficDirInbound)
	pbOutLabels := key.PacketByteLabels(rules.TrafficDirOutbound)
	cLabels := key.ConnectionLabels()
	switch key.ruleIDs.Direction {
	case rules.RuleDirIngress:
		counterRulePackets.Delete(pbInLabels)
		counterRuleBytes.Delete(pbInLabels)
		if value.isConnection {
			counterRulePackets.Delete(pbOutLabels)
			counterRuleBytes.Delete(pbOutLabels)
			gaugeRuleConns.Delete(cLabels)
		}
	case rules.RuleDirEgress:
		counterRulePackets.Delete(pbOutLabels)
		counterRuleBytes.Delete(pbOutLabels)
		if value.isConnection {
			counterRulePackets.Delete(pbInLabels)
			counterRuleBytes.Delete(pbInLabels)
			gaugeRuleConns.Delete(cLabels)
		}
	}
	delete(pr.ruleAggStats, key)
}
