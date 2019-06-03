// Copyright 2019 Tigera Inc. All rights reserved.

package watcher

type Job struct {
	DatafeedID  string
	Description string
	Detectors   map[int]string
}

// This needs to be kept in sync with the jobs in /install/data
var Jobs = map[string]Job{
	"inbound_connection_spike": {
		"datafeed-inbound_connection_spike",
		"Inbound connection spike",
		map[int]string{
			0: "Inbound connection spike for a pod within a replica set",
			1: "Inbound connection spike over a whole replica set",
		},
	},
	"ip_sweep_external": {
		"datafeed-ip_sweep_external",
		"IP Sweep - External",
		map[int]string{
			0: "Count of dest. IPs compared with all external IPs",
		},
	},
	"ip_sweep_pods": {
		"datafeed-ip_sweep_pods",
		"IP Sweep - Pods",
		map[int]string{
			0: "Count of dest. IPs compared with all pods",
			1: "Count of dest. IPs compared with replica set",
		},
	},
	"pod_outlier_ip_activity": {
		"datafeed-pod_outlier_ip_activity",
		"Outlier IP Activity - Pods",
		map[int]string{
			0: "Unexpected IP connection for a pod within a replica set",
		},
	},
	"port_scan_external": {
		"datafeed-port_scan_external",
		"Port Scan - External",
		map[int]string{
			0: "Count of destination ports compared with all external IPs",
		},
	},
	"port_scan_pods": {
		"datafeed-port_scan_pods",
		"Port Scan - Pods",
		map[int]string{
			0: "Count of destination ports compared with all pods",
			1: "Count of destination ports compared with replica set",
		},
	},
	"service_bytes_anomaly": {
		"datafeed-service_bytes_anomaly",
		"Service bytes anomaly",
		map[int]string{
			0: "Input bytes spike for a pod within a replica set",
			1: "Input bytes spike over a whole replica set",
			2: "Output bytes spike for a pod within a replica set",
			3: "Output bytes spike over a whole replica set",
		},
	},
}
