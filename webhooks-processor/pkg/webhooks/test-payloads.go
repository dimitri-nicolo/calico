// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package webhooks

import (
	lsApi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

var (
	webhookTestPayloads = map[string]lsApi.Event{
		"ga": {
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
		"dpi": {
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
		"gtf": {
			ID:           "test-event-id",
			Description:  "[TEST] A pod made a DNS lookup for a domain name that appears to be algorithm-generated. This may indicate malware connecting out to a control server for exfiltration or further instructions.",
			Time:         lsApi.TimestampOrDate{},
			Origin:       "Domain Generation Algorithm",
			AttackVector: "Network",
			Severity:     80,
			MitreIDs:     &[]string{"T1568", "T1568.002"},
			MitreTactic:  "Command and Control",
			Mitigations: &[]string{
				"This Global Threat Feeds event is generated for the purpose of webhook testing, no action is required.",
				"Payload of this event is consistent with actual expected payload when a similar event happens in your cluster.",
			},
			Type: "gtf_suspicious_dns_query",
			Record: map[string]any{
				"client_ip": "null",
				"client_labels": map[string]string{
					"projectcalico.org/namespace":      "default",
					"projectcalico.org/orchestrator":   "k8s",
					"projectcalico.org/serviceaccount": "default",
					"run":                              "test-evil-sim-pod",
				},
				"client_name":      "-",
				"client_name_aggr": "test-evil-sim-pod",
				"client_namespace": "default",
				"count":            1,
				"end_time":         "2024-10-10T12:00:00.000000000Z",
				"generated_time":   "2024-10-10T12:00:00.000000000Z",
				"host":             "node-name",
				"id":               "oXOtmJABiNH1R3Pmgr0x",
				"latency": map[string]int{
					"count": 1,
					"max":   16987000,
					"mean":  16987000,
				},
				"latency_count": 1,
				"latency_max":   16987000,
				"latency_mean":  16987000,
				"qclass":        "IN",
				"qname":         "bowjjxxnhkyvygk.biz",
				"qtype":         "A",
				"rcode":         "NXDomain",
				"rrsets": []map[string]any{
					{
						"class": "IN",
						"name":  "biz",
						"rdata": []string{
							"a.gtld.biz admin.tldns.godaddy 1720547603 1800 300 604800 1800",
						},
						"type": "SOA",
					},
				},
				"servers": []map[string]any{
					{
						"ip": "192.168.95.74",
						"labels": map[string]string{
							"k8s-app":                          "kube-dns",
							"pod-template-hash":                "76f75df574",
							"projectcalico.org/namespace":      "kube-system",
							"projectcalico.org/orchestrator":   "k8s",
							"projectcalico.org/serviceaccount": "coredns",
						},
						"name":      "coredns-76f75df574-4vd2p",
						"name_aggr": "coredns-76f75df574-*",
						"namespace": "kube-system",
					},
				},
				"start_time": "2024-10-10T12:00:00.000000000Z",
				"type":       "log",
			},
		},
	}
)
