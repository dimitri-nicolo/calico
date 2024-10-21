// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package webhooks

import (
	lsApi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

var (
	webhookTestPayloads = map[string]lsApi.Event{
		"global_alert": {
			ID:          "test-event-id",
			Description: "[TEST] The GlobalAlert description",
			Time:        lsApi.TimestampOrDate{},
			Origin:      "your-global-alert",
			Severity:    100,
			MitreIDs:    &[]string{"n/a"},
			MitreTactic: "n/a",
			Mitigations: &[]string{"n/a"},
			Type:        "global_alert",
			Record: map[string]any{
				"source_name_aggr": "jump-pod",
				"source_namespace": "default",
				"sum":              122,
			},
		},
		"deep_packet_inspection": {
			ID:           "test-event-id",
			Description:  "[TEST] Deep Packet Inspection found a matching snort rule(s) for some packets in your network",
			Time:         lsApi.TimestampOrDate{},
			Origin:       "dpi.default/default-namespace-all-endpoints",
			AttackVector: "Network",
			Severity:     100,
			Type:         "deep_packet_inspection",
			Record: map[string]any{
				"snort_alert":              "24/09/27-08:24:10.080704 [**] [1:408:8] \"PROTOCOL-ICMP Echo Reply\" [**] [Classification: Misc activity] [Priority: 3] {ICMP} 8.8.8.8 -\u003e 192.168.142.9",
				"snort_signature_id":       "408",
				"snort_signature_revision": "8",
			},
		},
		"waf": {
			ID:           "test-event-id",
			Description:  "[TEST] Traffic inside your cluster triggered Web Application Firewall rules.",
			Time:         lsApi.TimestampOrDate{},
			Origin:       "Web Application Firewall",
			AttackVector: "Network",
			Severity:     80,
			MitreIDs:     &[]string{"T1190"},
			MitreTactic:  "Initial Access",
			Mitigations: &[]string{
				"This Web Application Firewall event is generated for the purpose of webhook testing, no action is required.",
				"Payload of this event is consistent with actual expected payload when a similar event happens in your cluster.",
			},
			Type: "waf",
			Record: map[string]any{
				"@timestamp": "2024-10-10T12:00:00.000000000Z",
				"destination": map[string]string{
					"hostname":  "",
					"ip":        "10.244.151.190",
					"name":      "frontend-7d56967868-drpjs",
					"namespace": "online-boutique",
					"port_num":  "8080",
				},
				"host":       "aks-agentpool-22979750-vmss000000",
				"level":      "",
				"method":     "GET",
				"msg":        "WAF detected 2 violations [deny]",
				"path":       "/test/artists.php?artist=0+div+1+union%23foo*%2F*bar%0D%0Aselect%23foo%0D%0A1%2C2%2Ccurrent_user",
				"protocol":   "HTTP/1.1",
				"request_id": "460182972949411176",
				"rules": []map[string]string{
					{
						"disruptive": "true",
						"file":       "/etc/modsecurity-ruleset/@owasp_crs/REQUEST-942-APPLICATION-ATTACK-SQLI.conf",
						"id":         "942100",
						"line":       "5195",
						"message":    "SQL Injection Attack Detected via libinjection",
						"severity":   "critical",
					},
					{
						"disruptive": "true",
						"file":       "/etc/modsecurity-ruleset/@owasp_crs/REQUEST-949-BLOCKING-EVALUATION.conf",
						"id":         "949110",
						"line":       "6946",
						"message":    "Inbound Anomaly Score Exceeded (Total Score: 5)",
						"severity":   "emergency",
					},
				},
				"source": map[string]string{
					"hostname":  "",
					"ip":        "10.244.214.122",
					"name":      "busybox",
					"namespace": "online-boutique",
					"port_num":  "33387",
				},
			},
		},
		"gtf_suspicious_*": {
			ID:           "test-event-id",
			Description:  "[TEST] A request originating from default/test-evil-sim-pod queried the domain name minexmr.com, which is listed in the threat feed alienvault.domainthreatfeeds",
			Time:         lsApi.TimestampOrDate{},
			Origin:       "Suspicious DNS Query",
			AttackVector: "Network",
			Severity:     100,
			MitreIDs:     &[]string{"T1041"},
			MitreTactic:  "Exfiltration",
			Mitigations: &[]string{
				"This Global Threat Feeds event is generated for the purpose of webhook testing, no action is required.",
				"Payload of this event is consistent with actual expected payload when a similar event happens in your cluster.",
			},
			Type: "gtf_suspicious_dns_query",
			Record: map[string]any{
				"dns_log_id": "uJWzmZIB6viRfL6dXt2t",
				"feeds": []string{
					"alienvault.domainthreatfeeds",
				},
				"suspicious_domains": []string{
					"minexmr.com",
				},
			},
		},
		"runtime_security": {
			ID:           "test-event-id",
			Description:  "[TEST]A pod was detected executing a process with a known malicious hash signature.",
			Time:         lsApi.TimestampOrDate{},
			Origin:       "Malware",
			AttackVector: "Process",
			Severity:     100,
			MitreIDs:     &[]string{"T1204.002"},
			MitreTactic:  "Execution",
			Mitigations: &[]string{
				"This Container Threat Detection event is generated for the purpose of webhook testing, no action is required.",
				"Payload of this event is consistent with actual expected payload when a similar event happens in your cluster.",
			},
			Type: "runtime_security",
			Record: map[string]any{
				"config_name": "malware-protection",
				"count":       1,
				"end_time":    "2024-10-10T12:00:00.000000000Z",
				"file": map[string]string{
					"host_path": "-",
					"path":      "/code/ransomware",
				},
				"generated_time": "2024-10-10T12:00:00.000000000Z",
				"host":           "bz-40kc-kadm-node-2",
				"pod": map[string]any{
					"container_name": "-",
					"name":           "test-evil-sim-pod",
					"name_aggr":      "test-evil-sim-pod",
					"namespace":      "default",
					"ready":          true,
					"start_time":     "2024-10-10T00:00:00Z",
				},
				"process_start": map[string]any{
					"hashes": map[string]string{
						"md5":    "",
						"sha1":   "",
						"sha256": "7bc9f3ad33b53e51a044099a2cc8cff83e9193eaf099c4f2412e84da103c4910",
					},
					"invocation": "/code/ransomware",
				},
				"start_time": "2024-10-10T12:00:00.000000000Z",
				"type":       "ProcessStart",
			},
		},
	}
)
