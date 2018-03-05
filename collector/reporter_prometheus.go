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
)

const checkInterval = 5 * time.Second

type Aggregator interface {
	// Register Metrics that should be reported with a prometheus registry
	RegisterMetrics(registry *prometheus.Registry)
	// OnUpdate is called everytime a new MetricUpdate is received by the
	// PrometheusReporter.
	OnUpdate(mu *MetricUpdate)
	// CheckRetainedMetrics is called everytime the aggregator should check if a retained
	// metric has expired.
	CheckRetainedMetrics(now time.Duration)
}

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
	aggregators     []Aggregator

	// Allow the time function to be mocked for test purposes.
	timeNowFn func() time.Duration
}

func NewPrometheusReporter(port int, retentionTime time.Duration, certFile, keyFile, caFile string) *PrometheusReporter {
	// Set the ticker interval appropriately, we should be checking at least half of the rention time,
	// or the hard-coded check interval (whichever is smaller).
	tickerInterval := retentionTime / 2
	if checkInterval < tickerInterval {
		tickerInterval = checkInterval
	}

	registry := prometheus.NewRegistry()
	return &PrometheusReporter{
		port:            port,
		certFile:        certFile,
		keyFile:         keyFile,
		caFile:          caFile,
		registry:        registry,
		reportChan:      make(chan *MetricUpdate),
		retentionTicker: jitter.NewTicker(tickerInterval, tickerInterval/10),
		retentionTime:   retentionTime,
		timeNowFn:       monotime.Now,
	}
}

func (pr *PrometheusReporter) AddAggregator(agg Aggregator) {
	agg.RegisterMetrics(pr.registry)
	pr.aggregators = append(pr.aggregators, agg)
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
			for _, agg := range pr.aggregators {
				agg.OnUpdate(mu)
			}
		case <-pr.retentionTicker.C:
			//TODO: RLB: Maybe improve this processing using a linked-list (ordered by time)
			now := pr.timeNowFn()
			for _, agg := range pr.aggregators {
				agg.CheckRetainedMetrics(now)
			}
		}
	}
}
