// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package metrics

import (
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/node/pkg/bgp"
)

// DefaultPrometheusPort is the default port value for the Prometheus metrics server
const DefaultPrometheusPort = 9900

// For testing purposes we define an exit function that we can override.
var exitFunction = os.Exit

// Run is responsible for starting up the BGP Prometheus metrics reporter.
// An empty channel can be passed into the function to trigger the metrics
// reporter to halt operation.
func Run(stop <-chan struct{}, fipsModeEnabled bool) {
	configureLogging()

	// BGP Prometheus metrics reporter directly relies on BIRD (for BGP stats).
	// Thus it can only run if BIRD is running.
	if os.Getenv("CALICO_NETWORKING_BACKEND") != "bird" {
		log.Errorf("Unable to start BGP Prometheus metrics server, BGP is disabled.")
		terminate()
	}

	log.Info("Starting up BGP Prometheus metrics reporter")

	// Determine the name for this node. This binary is only invoked after
	// the startup binary and the modified environments have been sourced.
	// Therefore, the NODENAME environment will always be set at this point.
	nodename := os.Getenv("NODENAME")
	if nodename == "" {
		log.Errorf("NODENAME environment is not set.")
		terminate()
	}
	log.Infof("BGP Prometheus metrics nodename: %s\n", nodename)

	// Determine value to use for BGP Prometheus metrics port.
	metricsPort := DefaultPrometheusPort
	portStr := os.Getenv("BGP_PROMETHEUSREPORTERPORT")
	if portStr != "" {
		portInt, err := strconv.Atoi(portStr)
		if err != nil {
			log.Warningf("Invalid value provided for BGP Prometheus metrics port: %v", err)
		} else {
			metricsPort = portInt
		}
	}
	log.Infof("BGP Prometheus metrics port: %d\n", metricsPort)

	// Reuse the cert, key, ca files for Prometheus policy reporter (in Felix) for
	// encrypting traffic; no need to validate whether non-empty here (this is
	// taken care of within reporter).
	certFile := os.Getenv("FELIX_PROMETHEUSREPORTERCERTFILE")
	keyFile := os.Getenv("FELIX_PROMETHEUSREPORTERKEYFILE")
	caFile := os.Getenv("FELIX_PROMETHEUSREPORTERCAFILE")
	log.WithFields(log.Fields{
		"port":            metricsPort,
		"fipsModeEnabled": fipsModeEnabled,
		certFile:          certFile,
		keyFile:           keyFile,
		caFile:            caFile,
	}).Info("Starting prometheus BGP reporter")
	pr := newPrometheusBGPReporter(
		metricsPort,
		certFile,
		keyFile,
		caFile,
		fipsModeEnabled,
	)
	log.Infof("Created BGP Prometheus metrics reporter with config: %+v\n", pr)

	// Add various aggregators for different BGP metrics.
	log.Info("Adding aggregators for BGP Prometheus metrics reporter")
	pr.addMetricAggregator(newPeerCountAggregator(nodename, bgp.Peers))
	pr.addMetricAggregator(newRouteCountAggregator(nodename, bgp.Peers))

	// Add various stats getters for retrieving BGP stats (used to compute metrics).
	log.Info("Adding stats getters for BGP Prometheus metrics reporter")
	pr.addStatsGetter(func() (*bgp.Stats, error) { return bgp.GetPeers(bgp.IPv4) })
	pr.addStatsGetter(func() (*bgp.Stats, error) { return bgp.GetPeers(bgp.IPv6) })

	// Block on call to kick off the BGP metrics reporter
	log.Info("BGP Prometheus metrics reporter started ...")
	pr.start(stop)
}

// terminate prints a terminate message and exists with status 1.
func terminate() {
	log.Warn("Terminating")
	exitFunction(1)
}

func configureLogging() {
	// Default to info level logging
	logLevel := log.InfoLevel

	rawLogLevel := os.Getenv("BGP_PROMETHEUSREPORTER_LOGLEVEL")
	if rawLogLevel != "" {
		parsedLevel, err := log.ParseLevel(rawLogLevel)
		if err == nil {
			logLevel = parsedLevel
		} else {
			log.WithError(err).Error("Failed to parse log level, defaulting to info.")
		}
	}

	log.SetLevel(logLevel)
	log.Infof("Early log level set to %v", logLevel)
}
