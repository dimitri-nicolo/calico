package middlewares
//
//import (
//	"io/ioutil"
//	"net/http"
//	"strings"
//
//	. "github.com/onsi/ginkgo/extensions/table"
//	. "github.com/onsi/gomega"
//)
//
//var _ = DescribeTable("Tenant and clusterID extraction", func(req *http.Request, expectedTenant, expectedCluster string, expectedErr bool) {
//	tenant, cluster, _, err := extractMetricsFromRequest(req)
//	Expect(tenant).To(Equal(expectedTenant))
//	Expect(cluster).To(Equal(expectedCluster))
//	Expect(err != nil).To(Equal(expectedErr))
//},
//
//Entry("Benchmarker EE", createTestRequest(http.MethodGet, "/tigera_secure_ee_benchmark_results.cluster.lma-20210818-000000", ""), "", "cluster", nil),
//Entry("Benchmarker Cloud", createTestRequest(http.MethodGet, "/tigera_secure_ee_benchmark_results.tenant.cluster.lma-20210818-000000", ""), "tenant", "cluster", nil),
//
//Entry("Audit EE", createTestRequest(http.MethodGet, "/tigera_secure_ee_audit_kube.cluster.fluentd-20210819-000006", ""), "", "cluster", nil),
//Entry("Audit Cloud", createTestRequest(http.MethodGet, "/tigera_secure_ee_audit_kube.tenant.cluster.fluentd-20210819-000006", ""), "tenant", "cluster", nil),
//
//Entry("Snapshots EE", createTestRequest(http.MethodGet, "/tigera_secure_ee_snapshots.cluster.lma-20210818-000000", ""), "", "cluster", nil),
//Entry("Snapshot Cloud", createTestRequest(http.MethodGet, "/tigera_secure_ee_snapshots.tenant.cluster.lma-20210818-000000", ""), "tenant", "cluster", nil),
//
//Entry("BGP EE", createTestRequest(http.MethodGet, "/tigera_secure_ee_bgp.cluster.fluentd-20210823-000003", ""), "", "cluster", nil),
//Entry("BGP Cloud", createTestRequest(http.MethodGet, "/tigera_secure_ee_bgp.tenant.cluster.fluentd-20210823-000003", ""), "tenant", "cluster", nil),
//
//Entry("DNS EE", createTestRequest(http.MethodGet, "/tigera_secure_ee_dns.cluster.fluentd-20210818-000001", ""), "", "cluster", nil),
//Entry("DNS Cloud", createTestRequest(http.MethodGet, "/tigera_secure_ee_dns.tenant.cluster.fluentd-20210818-000001", ""), "tenant", "cluster", nil),
//
//Entry("Flows EE", createTestRequest(http.MethodGet, "/tigera_secuere_ee_flows.cluster.fluentd-20210818-000001", ""), "", "cluster", nil),
//Entry("Flows Cloud", createTestRequest(http.MethodGet, "/tigera_secure_ee_flows.tenant.cluster.fluentd-20210818-000001", ""), "tenant", "cluster", nil),
//
//Entry("Events EE", createTestRequest(http.MethodGet, "/tigera_secure_ee_events.cluster", ""), "", "cluster", nil),
//Entry("Events Cloud", createTestRequest(http.MethodGet, "/tigera_secure_ee_events.tenant.cluster", ""), "tenant", "cluster", nil),
//
//Entry("Stats endpoint", createTestRequest(http.MethodGet, "/_all/_stats?level=shards", ""), "", "", nil),
//Entry("Cluster endpoint", createTestRequest(http.MethodGet, "/_cluster/health", ""), "", "", nil),
//Entry("Cluster settings", createTestRequest(http.MethodGet, "/_all/_settings", ""), "", "", nil),
//Entry("Cluster settings", createTestRequest(http.MethodGet, "/_cluster/settings", ""), "", "", nil),
//Entry("Kibana", createTestRequest(http.MethodPost, "/tigera-kibana/api/index_management/indices/reload", ""), "", "", nil),
//
//Entry("Bulk", createTestRequest(http.MethodPost, "/_bulk", ""), "tenant", "cluster", nil),
//
//)
//
//func createTestRequest(method string, uri string, body string) *http.Request {
//	return &http.Request{
//		RequestURI: uri,
//		ContentLength: int64(len(body)),
//		Body: ioutil.NopCloser(strings.NewReader(body)),
//		Method: method,
//	}
//}