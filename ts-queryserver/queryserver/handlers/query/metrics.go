// Copyright (c) 2022-2023 Tigera, Inc. All rights reserved.
package query

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// prometheus metrics
	prometheusRegistry = prometheus.NewRegistry()
	prometheusHandler  = promhttp.HandlerFor(prometheusRegistry, promhttp.HandlerOpts{})

	hostEndpointsGauge = promauto.With(prometheusRegistry).NewGaugeVec(prometheus.GaugeOpts{
		Name: "queryserver_host_endpoints_total",
		Help: "Total number of host endpoints in the cluster. Type can be one of unlabeled, unprotected, or empty.",
	}, []string{"namespace", "type"})

	workloadEndpointsGauge = promauto.With(prometheusRegistry).NewGaugeVec(prometheus.GaugeOpts{
		Name: "queryserver_workload_endpoints_total",
		Help: "Total number of workload endpoints in a cluster or a namespace. Type can be one of unlabeled, unprotected, failed, or empty.",
	}, []string{"namespace", "type"})

	networkPolicyGauge = promauto.With(prometheusRegistry).NewGaugeVec(prometheus.GaugeOpts{
		Name: "queryserver_network_policy_total",
		Help: "Total number of network policies in a cluster or a namespace. Type can be one of unmatched or empty.",
	}, []string{"namespace", "type"})

	globalNetworkPolicyGauge = promauto.With(prometheusRegistry).NewGaugeVec(prometheus.GaugeOpts{
		Name: "queryserver_global_network_policy_total",
		Help: "Total number of global network policies in a cluster. Type can be one of unmatched or empty.",
	}, []string{"type"})

	nodeGauge = promauto.With(prometheusRegistry).NewGaugeVec(prometheus.GaugeOpts{
		Name: "queryserver_node_total",
		Help: "Total number of nodes in a cluster. Type can be one of no-endpoints, no-host-endpoints, no-workload-endpoints, or empty",
	}, []string{"type"})
)
