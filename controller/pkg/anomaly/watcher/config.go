// Copyright 2019 Tigera Inc. All rights reserved.

package watcher

type Job struct {
	DatafeedID string
}

// This needs to be kept in sync with the jobs in /install/data
var Jobs = map[string]Job{
	"inbound_connection_spike": {
		"datafeed-inbound_connection_spike",
	},
	"ip_sweep_external": {
		"datafeed-ip_sweep_external",
	},
	"ip_sweep_pods": {
		"datafeed-ip_sweep_pods",
	},
	"pod_outlier_ip_activity": {
		"datafeed-pod_outlier_ip_activity",
	},
	"port_scan_external": {
		"datafeed-port_scan_external",
	},
	"port_scan_pods": {
		"datafeed-port_scan_pods",
	},
	"service_bytes_anomaly": {
		"datafeed-service_bytes_anomaly",
	},
}
