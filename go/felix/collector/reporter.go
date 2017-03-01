// Copyright (c) 2017 Tigera, Inc. All rights reserved.

package collector

import (
	log "github.com/Sirupsen/logrus"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	gaugeDeniedPackets = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "felix_denied_packets",
		Help: "Packets denied.",
	},
		[]string{"sourceIp", "policy"},
	)
	gaugeDeniedBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "felix_denied_bytes",
		Help: "Bytes denied.",
	},
		[]string{"sourceIp", "policy"},
	)
)

func init() {
	prometheus.MustRegister(gaugeDeniedPackets)
	prometheus.MustRegister(gaugeDeniedBytes)
}

type Metrics struct {
	Bytes   int
	Packets int
}

func UpdateMetrics(policy string, sipStats map[string]Metrics) {
	for srcIP, metric := range sipStats {
		log.WithFields(log.Fields{
			"sourceIp": srcIP,
			"policy":   policy,
			"bytes":    metric.Bytes,
			"packets":  metric.Packets,
		}).Info("Setting Metrics.")
		gaugeDeniedPackets.WithLabelValues(srcIP, policy).Set(float64(metric.Packets))
		gaugeDeniedBytes.WithLabelValues(srcIP, policy).Set(float64(metric.Bytes))
	}
}
