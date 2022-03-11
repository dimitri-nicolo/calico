// Copyright (c) 2021-2022 Tigera, Inc. All rights reserved.

package query

import (
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = DescribeTable("Events", func(atom Atom, ok bool) {
	actual := atom
	err := IsValidEventsKeysAtom(&actual)
	if ok {
		Expect(err).ShouldNot(HaveOccurred())
	} else {
		Expect(err).Should(HaveOccurred())
	}
},
	Entry("_id", Atom{Key: "_id", Value: "foo"}, true),
	Entry("alert", Atom{Key: "alert", Value: "foo"}, true),
	Entry("dest_ip", Atom{Key: "dest_ip", Value: "1.2.3.4"}, true),
	Entry("dest_ip invalid", Atom{Key: "dest_ip", Value: "invalid-ip"}, false),
	Entry("dest_name", Atom{Key: "dest_name", Value: "foo"}, true),
	Entry("dest_name_aggr", Atom{Key: "dest_name_aggr", Value: "foo"}, true),
	Entry("dest_namespace", Atom{Key: "dest_namespace", Value: "foo"}, true),
	Entry("dest_port", Atom{Key: "dest_port", Value: "1234"}, true),
	Entry("dest_port invalid", Atom{Key: "dest_port", Value: "-1"}, false),
	Entry("host", Atom{Key: "host", Value: "foo"}, true),
	Entry("origin", Atom{Key: "origin", Value: "foo"}, true),
	Entry("source_ip", Atom{Key: "source_ip", Value: "1.2.3.4"}, true),
	Entry("source_ip invalid", Atom{Key: "source_ip", Value: "invalid-ip"}, false),
	Entry("source_name", Atom{Key: "source_name", Value: "foo"}, true),
	Entry("source_name_aggr", Atom{Key: "source_name_aggr", Value: "foo"}, true),
	Entry("source_namespace", Atom{Key: "source_namespace", Value: "foo"}, true),
	Entry("source_port", Atom{Key: "source_port", Value: "1234"}, true),
	Entry("source_port invalid", Atom{Key: "source_port", Value: "-1"}, false),
	Entry("type alert", Atom{Key: "type", Value: "alert"}, true),
	Entry("type anomaly_detection_job", Atom{Key: "type", Value: "anomaly_detection_job"}, true),
	Entry("type deep_packet_inspection", Atom{Key: "type", Value: "deep_packet_inspection"}, true),
	Entry("type global_alert", Atom{Key: "type", Value: "global_alert"}, true),
	Entry("type gtf_suspicious_dns_query", Atom{Key: "type", Value: "gtf_suspicious_dns_query"}, true),
	Entry("type gtf_suspicious_flow", Atom{Key: "type", Value: "gtf_suspicious_flow"}, true),
	Entry("type honeypod", Atom{Key: "type", Value: "honeypod"}, true),
	Entry("type suspicious_dns_query", Atom{Key: "type", Value: "suspicious_dns_query"}, true),
	Entry("type suspicious_flow", Atom{Key: "type", Value: "suspicious_flow"}, true),
	Entry("type invalid", Atom{Key: "type", Value: "invalid"}, false),
)
