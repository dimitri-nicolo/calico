// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package middleware

import (
	"fmt"
	"net/http"
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

func genPath(q string) string {
	return fmt.Sprintf("/%s/_search", q)
}
func genPathPs(q string, ps string) string {
	return fmt.Sprintf("/%s/_search%s", q, ps)
}

var _ = Describe("Test request to resource name conversion", func() {

	DescribeTable("successful conversion",
		func(req *http.Request, expectedName string) {
			rn, err := getResourceNameFromReq(req)

			Expect(err).NotTo(HaveOccurred())
			Expect(rn).To(Equal(expectedName))
		},

		Entry("Flow conversion",
			&http.Request{URL: &url.URL{Path: genPath("tigera_secure_ee_flows.cluster.*")}},
			"flows"),
		Entry("Flow conversion with cluster-name",
			&http.Request{URL: &url.URL{Path: genPath("tigera_secure_ee_flows.cluster-name.*")}},
			"flows"),
		Entry("Flow conversion with cluster.name",
			&http.Request{URL: &url.URL{Path: genPath("tigera_secure_ee_flows.cluster.name.*")}},
			"flows"),
		Entry("Flow conversion postfix *",
			&http.Request{URL: &url.URL{Path: genPath("tigera_secure_ee_flows*")}},
			"flows"),
		Entry("Flow conversion",
			&http.Request{URL: &url.URL{Path: genPathPs("tigera_secure_ee_flows.cluster.*", "?size=0")}},
			"flows"),
		Entry("All audit conversion (...audit_*)",
			&http.Request{URL: &url.URL{Path: genPath("tigera_secure_ee_audit_*.cluster.*")}},
			"audit*"),
		Entry("All audit alternate conversion (...audit*)",
			&http.Request{URL: &url.URL{Path: genPath("tigera_secure_ee_audit*.cluster.*")}},
			"audit*"),
		Entry("All audit conversion with cluster-name",
			&http.Request{URL: &url.URL{Path: genPath("tigera_secure_ee_audit_*.cluster-name.*")}},
			"audit*"),
		Entry("All audit conversion with cluster.name",
			&http.Request{URL: &url.URL{Path: genPath("tigera_secure_ee_audit_*.cluster.name.*")}},
			"audit*"),
		Entry("Audit ee conversion",
			&http.Request{URL: &url.URL{Path: genPath("tigera_secure_ee_audit_ee*.cluster.*")}},
			"audit_ee"),
		Entry("Audit ee conversion with cluster-name",
			&http.Request{URL: &url.URL{Path: genPath("tigera_secure_ee_audit_ee*.cluster-name.*")}},
			"audit_ee"),
		Entry("Audit ee conversion with cluster.name",
			&http.Request{URL: &url.URL{Path: genPath("tigera_secure_ee_audit_ee*.cluster.name.*")}},
			"audit_ee"),
		Entry("Audit kube conversion with cluster.name",
			&http.Request{URL: &url.URL{Path: genPath("tigera_secure_ee_audit_kube*.cluster.name.*")}},
			"audit_kube"),
		Entry("Events conversion",
			&http.Request{URL: &url.URL{Path: genPath("tigera_secure_ee_events*")}},
			"events"),
		Entry("Flow conversion with extra prefix",
			&http.Request{URL: &url.URL{Path: "this.should/be_tolerated/tigera_secure_ee_flows.cluster.*/_search"}},
			"flows"),
		Entry("Flow conversion from flowLogNames endpoint",
			&http.Request{URL: &url.URL{Path: "/flowLogNames"}},
			"flows"),
		Entry("Flow conversion from flowLogNamespaces endpoint",
			&http.Request{URL: &url.URL{Path: "/flowLogNamespaces"}},
			"flows"),
		Entry("Flow conversion from flowLogs endpoint",
			&http.Request{URL: &url.URL{Path: "/flowLogs"}},
			"flows"),
		Entry("Flow conversion from flowLogNames endpoint with extra prefix",
			&http.Request{URL: &url.URL{Path: "this.should/be_tolerated/flowLogNames"}},
			"flows"),
		Entry("Flow conversion from flowLogNamespaces endpoint with extra prefix",
			&http.Request{URL: &url.URL{Path: "this.should/be_tolerated/flowLogNamespaces"}},
			"flows"),
		Entry("Flow conversion from flowLogs endpoint with extra prefix",
			&http.Request{URL: &url.URL{Path: "this.should/be_tolerated/flowLogs"}},
			"flows"),
	)
	DescribeTable("failed conversion",
		func(req *http.Request) {
			_, err := getResourceNameFromReq(req)

			Expect(err).To(HaveOccurred())
		},

		Entry("missing first slash",
			&http.Request{URL: &url.URL{Path: "tigera_secure_ee_flows.cluster.*/_search"}}),
		Entry("No trailing _search",
			&http.Request{URL: &url.URL{Path: "/tigera_secure_ee_flows.cluster.*/"}}),
		Entry("Wrong index name",
			&http.Request{URL: &url.URL{Path: "/tigera_wrong_ee_flows.cluster.*/_search"}}),
		Entry("missing first slash",
			&http.Request{URL: &url.URL{Path: "flowLogs"}}),
		Entry("lower cased endpoint name",
			&http.Request{URL: &url.URL{Path: "/flowlogs"}}),
	)
})
