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
		name := MakeADJobName("it", tenant, clusterName, detectorName)

		Expect(len(name)).To(BeNumerically("<=", maxJobNameLength))
		Expect(name).To(Equal(expectedName))
	},
	// Test current detector names.
	Entry("DGA detection", "cluster1", noTenant, "dga", "it-cluster1-dga-314af"),
	Entry("DGA detection", "cluster1", tenant, "dga", "it-tenant.cluster1-dga-6ba19"),
	Entry("DNS latency", "cluster1", noTenant, "dns_latency", "it-cluster1-dns-latency-93bc8"),
	Entry("DNS latency", "cluster1", tenant, "dns_latency", "it-tenant.cluster1-dns-latency-9d2e3"),
	Entry("Excessive value anomaly in DNS log", "cluster1", noTenant, "generic_dns", "it-cluster1-generic-dns-2774f"),
	Entry("Excessive value anomaly in DNS log", "cluster1", tenant, "generic_dns", "it-tenant.cluster1-generic-dns-6e8bf"),
	Entry("Excessive value anomaly in flows log", "cluster1", noTenant, "generic_flows", "it-cluster1-generic-flows-c372d"),
	Entry("Excessive value anomaly in flows log", "cluster1", tenant, "generic_flows", "it-tenant.cluster1-generic-flows-14124"),

	// Test valid names.
	Entry("Cluster and detector are valid", "c-lu.s-ter", noTenant, "d-g.a", "it-c-lu.s-ter-d-g.a-ab5ab"),
	Entry("Cluster and detector are valid", "c-lu.s-ter", tenant, "d-g.a", "it-tenant.c-lu.s-ter-d-g.a-943e6"),
	Entry("Valid names, start and end with numerical chars", "7c-lu.s-ter", noTenant, "d-g.a4", "it-7c-lu.s-ter-d-g.a4-ba42f"),
	Entry("Valid names, start and end with numerical chars", "7c-lu.s-ter", tenant, "d-g.a4", "it-tenant.7c-lu.s-ter-d-g.a4-6111f"),

	// Test name edge cases. Names may contain leading or trailing '-' or '.' that will be removed, even though they are valid characters.
	Entry("Cluster name with invalid characters", "$#%@-c!*l#u(s..^t*--e---)}{|r---@----", noTenant, "dga", "it-c-l-u-s.-t-e-r-dga-86252"),
	Entry("Cluster name with invalid characters", "$#%@-c!*l#u(s..^t*--e---)}{|r---@----", tenant, "dga", "it-tenant.-c-l-u-s.-t-e-r-ec39a"),
	Entry("Detector name with invalid characters", "cluster1", noTenant, ".$..#%h!*t(tp*--re---)}{|--@--s%^p:..o\"n?%s--e--..", "it-cluster1-.-.-h-t-tp-re-s-p-63f64"),
	Entry("Detector name with invalid characters", "cluster1", tenant, ".$..#%h!*t(tp*--re---)}{|--@--s%^p:..o\"n?%s--e--..", "it-tenant.cluster1-.-.-h-t-tp-re-d78da"),
	Entry("Cluster and detector name with invalid characters", "--.$#%@-c!*l#u(s..^t*--e---)}{|r---@----", noTenant, ".$..#%h!*t(tp*--re---)}{|--@--s%^p:..o\"n?%s--e--..", "it-.-c-l-u-s.-t-e-r-8e912"),
	Entry("Cluster and detector name with invalid characters", "--.$#%@-c!*l#u(s..^t*--e---)}{|r---@----", tenant, ".$..#%h!*t(tp*--re---)}{|--@--s%^p:..o\"n?%s--e--..", "it-tenant.-.-c-l-u-s.-t-e-r-c6f47"),
	Entry("Cluster name all invalid characters", ".-.$--#%@-!*#(^*--", noTenant, "dga", "it-.-.-dga-3fd2b"),
	Entry("Cluster name all invalid characters", ".-.$--#%@-!*#(^*--", tenant, "dga", "it-tenant.-.-dga-7929c"),
	Entry("Detector name all invalid characters", "cluster-name", noTenant, ".-*)(_).$--#%@-!*#(^*--", "it-cluster-name-eb43d"),
	Entry("Detector name all invalid characters", "cluster-name", tenant, ".-*)(_).$--#%@-!*#(^*--", "it-tenant.cluster-name-029ee"),
	Entry("Cluster and detector name all invalid characters", ".-.$--#%@-!*#(^*--", noTenant, ".-*)(_).$--#%@-!*#(^*--", "it-5f5e2"),
	Entry("Cluster and detector name all invalid characters", ".-.$--#%@-!*#(^*--", tenant, ".-*)(_).$--#%@-!*#(^*--", "it-tenant-46b00"),
	Entry("Cluster and detector contain valid characters but invalid substrings", "c.-lu...-..ster", noTenant, "d--g....a", "it-c.-lu.-.ster-d-g.a-e8c98"),
	Entry("Cluster and detector contain valid characters but invalid substrings", "c.-lu...-..ster", tenant, "d--g....a", "it-tenant.c.-lu.-.ster-d-g.a-1f0b8"),

	// Test capital letters.
	Entry("Cluster and detector contain capital letters", "ClUsTEr23", noTenant, "Http_RequEst", "it-cluster23-http-request-01860"),
	Entry("Cluster and detector contain capital letters", "ClUsTEr23", tenant, "Http_RequEst", "it-tenant.cluster23-http-request-36d94"),

	// Test name length.
	Entry("Length of the resulting name does not surpass the max size", "$#%@-c!*l#u(s..^t*--e---)}{|r---@--n-a-m-e--", noTenant, "d&e@t#e#c(t_o)r_n+ame", "it-c-l-u-s.-t-e-r-n-a-m-e-7b71e"),
	Entry("Length of the resulting name does not surpass the max size", "$#%@-c!*l#u(s..^t*--e---)}{|r---@--n-a-m-e--", tenant, "d&e@t#e#c(t_o)r_n+ame", "it-tenant.-c-l-u-s.-t-e-r-n-4241a"),
)
