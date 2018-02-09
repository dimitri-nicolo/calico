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
	"github.com/projectcalico/libcalico-go/lib/set"
)

const checkInterval = 5 * time.Second

type updateType bool

// Calico Metrics
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

	updateTypeReport updateType = true
	updateTypeExpire updateType = false
)

// PrometheusReporter records denied packets and bytes statistics in prometheus metrics.
type PrometheusReporter struct {
	port            int
	certFile        string
	keyFile         string
	caFile          string
	registry        *prometheus.Registry
	reportChan      chan *MetricUpdate
	expireChan      chan *MetricUpdate
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
		expireChan:             make(chan *MetricUpdate),
		retentionTime:          retentionTime,
		retentionTicker:        jitter.NewTicker(tickerInterval, tickerInterval/10),
		ruleAggStats:           make(map[RuleAggregateKey]*RuleAggregateValue),
		retainedRuleAggMetrics: make(map[RuleAggregateKey]time.Duration),
		timeNowFn:              monotime.Now,
	}
}

type RuleAggregateKey struct {
	ruleIDs    RuleIDs
	trafficDir TrafficDirection
}

type RuleAggregateValue struct {
	packets        prometheus.Counter
	bytes          prometheus.Counter
	numConnections prometheus.Gauge
	connections    set.Set
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

func (pr *PrometheusReporter) Expire(mu *MetricUpdate) error {
	pr.expireChan <- mu
	return nil
}

// getRuleAggregateKey returns a hashable key identifying a rule aggregation key.
func (pr *PrometheusReporter) getRuleAggregateKey(mu *MetricUpdate) RuleAggregateKey {
	return RuleAggregateKey{
		ruleIDs:    *mu.ruleIDs,
		trafficDir: mu.trafficDir,
	}
}

// getRulePacketByteLabels returns the Prometheus packet/byte counter labels associated
// with a specific rule and traffic direction.
func (pr *PrometheusReporter) getRulePacketByteLabels(key RuleAggregateKey) prometheus.Labels {
	return prometheus.Labels{
		LABEL_ACTION:      string(key.ruleIDs.Action),
		LABEL_TIER:        key.ruleIDs.Tier,
		LABEL_POLICY:      key.ruleIDs.Policy,
		LABEL_RULE_DIR:    string(key.ruleIDs.Direction),
		LABEL_RULE_IDX:    key.ruleIDs.Index,
		LABEL_TRAFFIC_DIR: string(key.trafficDir),
	}
}

// getRuleConnectionLabels returns the Prometheus connection gauge labels associated
// with a specific rule and traffic direction.
func (pr *PrometheusReporter) getRuleConnectionLabels(key RuleAggregateKey) prometheus.Labels {
	return prometheus.Labels{
		LABEL_TIER:        key.ruleIDs.Tier,
		LABEL_POLICY:      key.ruleIDs.Policy,
		LABEL_RULE_DIR:    string(key.ruleIDs.Direction),
		LABEL_RULE_IDX:    key.ruleIDs.Index,
		LABEL_TRAFFIC_DIR: string(key.trafficDir),
	}
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
		case mu := <-pr.expireChan:
			pr.expireMetric(mu)
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
func (pr *PrometheusReporter) reportMetric(mu *MetricUpdate) {
	pr.handleRuleMetric(mu, updateTypeReport)
}

// expireMetrics is actually similar to reportMetric, it increments our counters from the
// metric update, removes any connection associated with the metric and ensures the metric
// will expire if there are no associated connections and no activity within the retention
// period. Unlike reportMetric, if there is no cached entry for this metric one is not
// created and therefore the metric will not be reported.
func (pr *PrometheusReporter) expireMetric(mu *MetricUpdate) {
	pr.handleRuleMetric(mu, updateTypeExpire)
}

// handleRuleMetric handles reporting and expiration of Rule-aggregated metrics.
func (pr *PrometheusReporter) handleRuleMetric(mu *MetricUpdate, ut updateType) {
	key := pr.getRuleAggregateKey(mu)
	value, ok := pr.ruleAggStats[key]
	if !ok {
		// No entry exists.  If this is a report then create a blank entry and add
		// to the map.  Otherwise, this is an expiration, so just return.
		if ut == updateTypeExpire {
			return
		}

		pbLabels := pr.getRulePacketByteLabels(key)
		cLabels := pr.getRuleConnectionLabels(key)
		value = &RuleAggregateValue{
			packets:        counterRulePackets.With(pbLabels),
			bytes:          counterRuleBytes.With(pbLabels),
			numConnections: gaugeRuleConns.With(cLabels),
			connections:    set.New(),
		}
		pr.ruleAggStats[key] = value
	}

	// Increment the packet counters if non-zero.
	if mu.deltaPackets != 0 && mu.deltaBytes != 0 {
		value.packets.Add(float64(mu.deltaPackets))
		value.bytes.Add(float64(mu.deltaBytes))
	}

	// If this is an active connection (and we aren't expiring the stats), add to our
	// active connections tuple, otherwise make sure it is removed.
	oldConns := value.connections.Len()
	if mu.isConnection && ut == updateTypeReport {
		value.connections.Add(mu.tuple)
	} else {
		value.connections.Discard(mu.tuple)
	}
	newConns := value.connections.Len()

	// If the number of connections has changed then update our connections gauge.
	if oldConns != newConns {
		value.numConnections.Set(float64(newConns))
	}

	// If there are some connections for this rule then keep it active, otherwise (re)set the timeout
	// for this metric to ensure we tidy up after a period of inactivity.
	if newConns > 0 {
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
	_, ok := pr.ruleAggStats[key]
	if ok {
		pbLabels := pr.getRulePacketByteLabels(key)
		cLabels := pr.getRuleConnectionLabels(key)
		counterRulePackets.Delete(pbLabels)
		counterRuleBytes.Delete(pbLabels)
		gaugeRuleConns.Delete(cLabels)
		delete(pr.ruleAggStats, key)
	}
}
