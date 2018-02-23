package collector

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"time"

	"github.com/gavv/monotime"
	"github.com/projectcalico/felix/jitter"
	"github.com/projectcalico/felix/rules"
	"github.com/projectcalico/libcalico-go/lib/set"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

// Calico Metrics
var (
	gaugeDeniedPackets = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "calico_denied_packets",
		Help: "Total number of packets denied by calico policies.",
	},
		[]string{"srcIP", "policy"},
	)
	gaugeDeniedBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "calico_denied_bytes",
		Help: "Total number of bytes denied by calico policies.",
	},
		[]string{"srcIP", "policy"},
	)
)

type AggregateKey struct {
	policy string
	srcIP  [16]byte
}

func getAggregateKey(mu *MetricUpdate) AggregateKey {
	policy := fmt.Sprintf("%s|%s|%s|%s", mu.ruleIDs.Tier, mu.ruleIDs.Policy, mu.ruleIDs.Index, mu.ruleIDs.Action)
	srcIP := mu.tuple.src
	return AggregateKey{policy, srcIP}
}

type AggregateValue struct {
	labels  prometheus.Labels
	packets prometheus.Gauge
	bytes   prometheus.Gauge
	refs    set.Set
}

// DPPrometheusReporter records denied packets and bytes statistics in prometheus metrics.
type DPPrometheusReporter struct {
	port            int
	certFile        string
	keyFile         string
	caFile          string
	registry        *prometheus.Registry
	aggStats        map[AggregateKey]AggregateValue
	reportChan      chan *MetricUpdate
	retainedMetrics map[AggregateKey]time.Duration
	retentionTime   time.Duration
	retentionTicker *jitter.Ticker
}

func NewDPPrometheusReporter(port int, rTime time.Duration, certFile, keyFile, caFile string) *DPPrometheusReporter {
	registry := prometheus.NewRegistry()
	registry.MustRegister(gaugeDeniedPackets)
	registry.MustRegister(gaugeDeniedBytes)
	return &DPPrometheusReporter{
		port:            port,
		certFile:        certFile,
		keyFile:         keyFile,
		caFile:          caFile,
		registry:        registry,
		aggStats:        make(map[AggregateKey]AggregateValue),
		reportChan:      make(chan *MetricUpdate),
		retainedMetrics: make(map[AggregateKey]time.Duration),
		retentionTime:   rTime,
		retentionTicker: jitter.NewTicker(CheckInterval, CheckInterval/10),
	}
}

func (pr *DPPrometheusReporter) Start() {
	log.Info("Starting DPPrometheusReporter")
	go pr.servePrometheusMetrics()
	go pr.startReporter()
}

func (pr *DPPrometheusReporter) servePrometheusMetrics() {
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

func (pr *DPPrometheusReporter) startReporter() {

	for {
		select {
		case mu := <-pr.reportChan:
			switch mu.updateType {
			case UpdateTypeReport:
				pr.reportMetric(mu)
			case UpdateTypeExpire:
				pr.expireMetric(mu)
			}
		case <-pr.retentionTicker.C:
			for key, expirationTime := range pr.retainedMetrics {
				if monotime.Since(expirationTime) >= pr.retentionTime {
					pr.deleteMetric(key)
					delete(pr.retainedMetrics, key)
				}
			}
		}
	}
}

func (pr *DPPrometheusReporter) Report(mu *MetricUpdate) error {
	if mu.ruleIDs.Action != rules.ActionDeny {
		// We only want denied packets. Skip the rest of them.
		return nil
	}
	pr.reportChan <- mu
	return nil
}

func (pr *DPPrometheusReporter) reportMetric(mu *MetricUpdate) {
	key := getAggregateKey(mu)
	value, ok := pr.aggStats[key]
	if ok {
		_, exists := pr.retainedMetrics[key]
		if exists {
			delete(pr.retainedMetrics, key)
		}
		value.refs.Add(mu.tuple)
	} else {
		l := prometheus.Labels{
			"srcIP":  net.IP(key.srcIP[:16]).String(),
			"policy": key.policy,
		}
		value = AggregateValue{
			labels:  l,
			packets: gaugeDeniedPackets.With(l),
			bytes:   gaugeDeniedBytes.With(l),
			refs:    set.FromArray([]Tuple{mu.tuple}),
		}
	}
	switch mu.ruleIDs.Direction {
	case rules.RuleDirIngress:
		value.packets.Add(float64(mu.inMetric.deltaPackets))
		value.bytes.Add(float64(mu.inMetric.deltaBytes))
	case rules.RuleDirEgress:
		value.packets.Add(float64(mu.outMetric.deltaPackets))
		value.bytes.Add(float64(mu.outMetric.deltaBytes))
	default:
		return
	}
	pr.aggStats[key] = value
	return
}

func (pr *DPPrometheusReporter) expireMetric(mu *MetricUpdate) {
	key := getAggregateKey(mu)
	value, ok := pr.aggStats[key]
	if !ok || !value.refs.Contains(mu.tuple) {
		return
	}
	// If the metric update has updated counters this is the time to update our counters.
	// We retain deleted metric for a little bit so that prometheus can get a chance
	// to scrape the metric.
	var deltaPackets, deltaBytes int
	switch mu.ruleIDs.Direction {
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
		pr.aggStats[key] = value
	}
	value.refs.Discard(mu.tuple)
	pr.aggStats[key] = value
	if value.refs.Len() == 0 {
		pr.markForDeletion(key)
	}
	return
}

func (pr *DPPrometheusReporter) markForDeletion(key AggregateKey) {
	log.WithField("key", key).Debug("Marking metric for deletion.")
	pr.retainedMetrics[key] = monotime.Now()
}

func (pr *DPPrometheusReporter) deleteMetric(key AggregateKey) {
	log.WithField("key", key).Debug("Cleaning up candidate marked to be deleted.")
	value, ok := pr.aggStats[key]
	if ok {
		gaugeDeniedPackets.Delete(value.labels)
		gaugeDeniedBytes.Delete(value.labels)
		delete(pr.aggStats, key)
	}
}
