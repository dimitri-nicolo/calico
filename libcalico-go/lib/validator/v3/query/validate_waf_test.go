// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package query

import (
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = DescribeTable("WAF", func(input Atom, ok bool) {
	actual := input

	err := IsValidWAFAtom(&actual)
	if ok {
		Expect(err).ShouldNot(HaveOccurred())
	} else {
		Expect(err).Should(HaveOccurred())
	}
},
	Entry("timestamp", Atom{Key: "timestamp", Value: "2022-01-17"}, true),
	Entry("unique_id", Atom{Key: "unique_id", Value: "624b739c-eab9-4666-b36c-923841c88d0a"}, true),
	Entry("uri", Atom{Key: "uri", Value: "/test/artists.php"}, true),
	Entry("owasp_host", Atom{Key: "owasp_host", Value: "192.168.101.138"}, true),
	Entry("owasp_file", Atom{Key: "owasp_file", Value: "/etc/waf/REQUEST-942-APPLICATION-ATTACK-SQLI.conf"}, true),
	Entry("owasp_line", Atom{Key: "owasp_line", Value: "1570"}, true),
	Entry("owasp_line", Atom{Key: "owasp_line", Value: "-1"}, false),
	Entry("owasp_id", Atom{Key: "owasp_id", Value: "942432"}, true),
	Entry("owasp_id", Atom{Key: "owasp_id", Value: "-1"}, false),
	Entry("owasp_severity", Atom{Key: "owasp_severity", Value: "4"}, true),
	Entry("owasp_severity", Atom{Key: "owasp_severity", Value: "-1"}, false),
	Entry("node", Atom{Key: "node", Value: "ip-172-16-101-111.us-west-2.compute.internal"}, true),
)
