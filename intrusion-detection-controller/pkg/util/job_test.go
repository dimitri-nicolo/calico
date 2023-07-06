// Copyright 2022 Tigera Inc. All rights reserved.
package util

import (
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

const (
	maxJobNameLength         = 57
	initialTrainingJobSuffix = "initial-training"
	noTenant                 = ""
	tenant                   = "tenant"
)

// GetRFC1123JobName
var _ = DescribeTable("GetRFC1123JobName",
	func(clusterName string, tenant string, detectorName string, expectedName string) {
		name := GetValidInitialTrainingJobName(clusterName, tenant, detectorName, initialTrainingJobSuffix)

		Expect(len(name)).To(BeNumerically("<=", maxJobNameLength))
		Expect(name).To(Equal(expectedName))
	},
	// Test current detector names.
	Entry("DGA detection", "cluster1", noTenant, "dga", "cluster1-dga-initial-training"),
	Entry("DGA detection", "cluster1", tenant, "dga", "tenant.cluster1-dga-initial-training"),
	Entry("DNS latency", "cluster1", noTenant, "dns_latency", "cluster1-dns-latency-initial-training"),
	Entry("DNS latency", "cluster1", tenant, "dns_latency", "tenant.cluster1-dns-latency-initial-training"),
	Entry("Excessive value anomaly in DNS log", "cluster1", noTenant, "generic_dns", "cluster1-generic-dns-initial-training"),
	Entry("Excessive value anomaly in DNS log", "cluster1", tenant, "generic_dns", "tenant.cluster1-generic-dns-initial-training"),
	Entry("Excessive value anomaly in flows log", "cluster1", noTenant, "generic_flows", "cluster1-generic-flows-initial-training"),
	Entry("Excessive value anomaly in flows log", "cluster1", tenant, "generic_flows", "tenant.cluster1-generic-flows-initial-training"),
	Entry("Time series anomaly in L7 log", "cluster1", noTenant, "generic_l7", "cluster1-generic-l7-initial-training"),
	Entry("Time series anomaly in L7 log", "cluster1", tenant, "generic_l7", "tenant.cluster1-generic-l7-initial-training"),
	Entry("HTTP connection spike anomaly", "cluster1", noTenant, "http_connection_spike", "cluster1-http-connection-spike-initial-training"),
	Entry("HTTP connection spike anomaly", "cluster1", tenant, "http_connection_spike", "tenant.cluster1-http-connection-spike-initial-training"),
	Entry("HTTP Response Code detection", "cluster1", noTenant, "http_response_codes", "cluster1-http-response-codes-initial-training"),
	Entry("HTTP Response Code detection", "cluster1", tenant, "http_response_codes", "tenant.cluster1-http-response-codes-initial-training"),
	Entry("HTTP Response Verbs detection", "cluster1", noTenant, "http_verbs", "cluster1-http-verbs-initial-training"),
	Entry("HTTP Response Verbs detection", "cluster1", tenant, "http_verbs", "tenant.cluster1-http-verbs-initial-training"),
	Entry("IP Sweep detection", "cluster1", noTenant, "ip_sweep", "cluster1-ip-sweep-initial-training"),
	Entry("IP Sweep detection", "cluster1", tenant, "ip_sweep", "tenant.cluster1-ip-sweep-initial-training"),
	Entry("L7 bytes", "cluster1", noTenant, "l7_bytes", "cluster1-l7-bytes-initial-training"),
	Entry("L7 bytes", "cluster1", tenant, "l7_bytes", "tenant.cluster1-l7-bytes-initial-training"),
	Entry("DNS Latency anomaly", "cluster1", noTenant, "l7_latency", "cluster1-l7-latency-initial-training"),
	Entry("DNS Latency anomaly", "cluster1", tenant, "l7_latency", "tenant.cluster1-l7-latency-initial-training"),
	Entry("Port Scan detection", "cluster1", noTenant, "port_scan", "cluster1-port-scan-initial-training"),
	Entry("Port Scan detection", "cluster1", tenant, "port_scan", "tenant.cluster1-port-scan-initial-training"),
	Entry("Port Scan detection", "cluster1", noTenant, "port_scan", "cluster1-port-scan-initial-training"),
	Entry("Port Scan detection", "cluster1", tenant, "port_scan", "tenant.cluster1-port-scan-initial-training"),
	Entry("Bytes in detection", "cluster1", noTenant, "bytes_in", "cluster1-bytes-in-initial-training"),
	Entry("Bytes in detection", "cluster1", tenant, "bytes_in", "tenant.cluster1-bytes-in-initial-training"),
	Entry("Bytes out detection", "cluster1", noTenant, "bytes_out", "cluster1-bytes-out-initial-training"),
	Entry("Bytes out detection", "cluster1", tenant, "bytes_out", "tenant.cluster1-bytes-out-initial-training"),
	Entry("Process bytes detection", "cluster1", noTenant, "process_bytes", "cluster1-process-bytes-initial-training"),
	Entry("Process bytes detection", "cluster1", tenant, "process_bytes", "tenant.cluster1-process-bytes-initial-training"),
	Entry("Process restarts detection", "cluster1", noTenant, "process_restarts", "cluster1-process-restarts-initial-training"),
	Entry("Process restarts detection", "cluster1", tenant, "process_restarts", "tenant.cluster1-process-restarts-initial-training"),

	// Test valid names.
	Entry("Cluster and detector are valid", "c-lu.s-ter", noTenant, "d-g.a", "c-lu.s-ter-d-g.a-initial-training"),
	Entry("Cluster and detector are valid", "c-lu.s-ter", tenant, "d-g.a", "tenant.c-lu.s-ter-d-g.a-initial-training"),
	Entry("Valid names, start and end with numerical chars", "7c-lu.s-ter", noTenant, "d-g.a4", "7c-lu.s-ter-d-g.a4-initial-training"),
	Entry("Valid names, start and end with numerical chars", "7c-lu.s-ter", tenant, "d-g.a4", "tenant.7c-lu.s-ter-d-g.a4-initial-training"),

	// Test name edge cases. Names may contain leading or trailing '-' or '.' that will be removed, even though they are valid characters.
	Entry("Cluster name with invalid characters", "$#%@-c!*l#u(s..^t*--e---)}{|r---@----", noTenant, "dga", "c-l-u-s.-t-e-r-dga-initial-training"),
	Entry("Cluster name with invalid characters", "$#%@-c!*l#u(s..^t*--e---)}{|r---@----", tenant, "dga", "tenant.c-l-u-s.-t-e-r-dga-initial-training"),
	Entry("Detector name with invalid characters", "cluster1", noTenant, ".$..#%h!*t(tp*--re---)}{|--@--s%^p:..o\"n?%s--e--..", "cluster1-h-t-tp-re-s-p-.o-n-s-e-initial-training"),
	Entry("Detector name with invalid characters", "cluster1", tenant, ".$..#%h!*t(tp*--re---)}{|--@--s%^p:..o\"n?%s--e--..", "tenant.cluster1-h-t-tp-re-s-p-.o-n-s-e-initial-training"),
	Entry("Cluster and detector name with invalid characters", "--.$#%@-c!*l#u(s..^t*--e---)}{|r---@----", noTenant, ".$..#%h!*t(tp*--re---)}{|--@--s%^p:..o\"n?%s--e--..", "c-l-u-s.-t-e-r-h-t-tp-re-s-p-.o-n-s-e-initial-training"),
	Entry("Cluster and detector name with invalid characters", "--.$#%@-c!*l#u(s..^t*--e---)}{|r---@----", tenant, ".$..#%h!*t(tp*--re---)}{|--@--s%^p:..o\"n?%s--e--..", "tenant.c-l-u-s.-t-e-r-h-t-tp-re-s-p-.o-n-s-e-initial-trai"),
	Entry("Cluster name all invalid characters", ".-.$--#%@-!*#(^*--", noTenant, "dga", "z-dga-initial-training"),
	Entry("Cluster name all invalid characters", ".-.$--#%@-!*#(^*--", tenant, "dga", "tenant.z-dga-initial-training"),
	Entry("Detector name all invalid characters", "cluster-name", noTenant, ".-*)(_).$--#%@-!*#(^*--", "cluster-name-z-initial-training"),
	Entry("Detector name all invalid characters", "cluster-name", tenant, ".-*)(_).$--#%@-!*#(^*--", "tenant.cluster-name-z-initial-training"),
	Entry("Cluster and detector name all invalid characters", ".-.$--#%@-!*#(^*--", noTenant, ".-*)(_).$--#%@-!*#(^*--", "z-z-initial-training"),
	Entry("Cluster and detector name all invalid characters", ".-.$--#%@-!*#(^*--", tenant, ".-*)(_).$--#%@-!*#(^*--", "tenant.z-z-initial-training"),
	Entry("Cluster and detector contain valid characters but invalid substrings", "c.-lu...-..ster", noTenant, "d--g....a", "c.-lu.-.ster-d-g.a-initial-training"),
	Entry("Cluster and detector contain valid characters but invalid substrings", "c.-lu...-..ster", tenant, "d--g....a", "tenant.c.-lu.-.ster-d-g.a-initial-training"),

	// Test capital letters.
	Entry("Cluster and detector contain capital letters", "ClUsTEr23", noTenant, "Http_RequEst", "cluster23-http-request-initial-training"),
	Entry("Cluster and detector contain capital letters", "ClUsTEr23", tenant, "Http_RequEst", "tenant.cluster23-http-request-initial-training"),

	// Test name length.
	Entry("Length of the resulting name does not surpass the max size", "$#%@-c!*l#u(s..^t*--e---)}{|r---@--n-a-m-e--", noTenant, "d&e@t#e#c(t_o)r_n+ame", "c-l-u-s.-t-e-r-n-a-m-e-d-e-t-e-c-t-o-r-n-ame-initial-trai"),
	Entry("Length of the resulting name does not surpass the max size", "$#%@-c!*l#u(s..^t*--e---)}{|r---@--n-a-m-e--", tenant, "d&e@t#e#c(t_o)r_n+ame", "tenant.c-l-u-s.-t-e-r-n-a-m-e-d-e-t-e-c-t-o-r-n-ame-initi"),
)
