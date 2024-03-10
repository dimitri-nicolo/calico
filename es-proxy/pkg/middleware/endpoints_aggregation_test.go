package middleware

import (
	"bytes"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/calico/libcalico-go/lib/json"
	lapi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	lmaapi "github.com/projectcalico/calico/lma/pkg/apis/v1"
	"github.com/projectcalico/calico/lma/pkg/httputils"
	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/client"
)

var _ = Describe("", func() {
	var req *http.Request

	It("test buildQueryServerEndpointKeyString result", func() {
		result := buildQueryServerEndpointKeyString("host", "ns", "name", "nameaggr")
		Expect(result).To(Equal(".*ns/host-k8s-name"))

		result = buildQueryServerEndpointKeyString("host", "ns", "-", "nameaggr")
		Expect(result).To(Equal(".*ns/host-k8s-nameaggr"))
	})

	Context("test validateEndpointsAggregationRequest", func() {
		DescribeTable("validate ClusterName",
			func(clusterName, clusterIdHeader, expectedCluster string) {
				endpointReq := &EndpointsAggregationRequest{}

				if len(clusterName) > 0 {
					endpointReq.ClusterName = clusterName
				}

				reqBodyBytes, err := json.Marshal(endpointReq)
				Expect(err).ShouldNot(HaveOccurred())

				req, err = http.NewRequest("POST", "https://test", bytes.NewBuffer(reqBodyBytes))
				Expect(err).ShouldNot(HaveOccurred())

				if len(clusterIdHeader) > 0 {
					req.Header.Add("x-cluster-id", clusterIdHeader)
				}

				err = validateEndpointsAggregationRequest(req, endpointReq)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(endpointReq.ClusterName).To(Equal(expectedCluster))

			},
			Entry("should not change ClusterName if it is set in the request body", "cluster-a", "cluster-b", "cluster-a"),
			Entry("should set ClusterName from request header if it is not provided in the request body", "", "cluster-b", "cluster-b"),
			Entry("should set ClusterName to default if it neither provided in the request body nor header", "", "", "cluster"),
		)

		var (
			allow     = lapi.FlowActionAllow
			deny      = lapi.FlowActionDeny
			name      = "policyA"
			namespace = "nsA"
			tier      = "tierA"
		)

		DescribeTable("validate PolicyMatch",
			func(pm *lapi.PolicyMatch, endpointList []string, expectErr bool, errMsg string) {

				epReq := EndpointsAggregationRequest{
					ClusterName:       "",
					QueryEndpointsReq: client.QueryEndpointsReqBody{},
					PolicyMatch:       nil,
					TimeRange:         nil,
					Timeout:           nil,
				}

				epReq.PolicyMatch = pm

				if len(endpointList) > 0 {
					epReq.QueryEndpointsReq = client.QueryEndpointsReqBody{
						EndpointsList: endpointList,
					}
				}

				reqBodyBytes, err := json.Marshal(epReq)
				Expect(err).ShouldNot(HaveOccurred())

				req, err = http.NewRequest("POST", "https://test", bytes.NewBuffer(reqBodyBytes))
				Expect(err).ShouldNot(HaveOccurred())

				err = validateEndpointsAggregationRequest(req, &epReq)

				if expectErr {
					Expect(err).Should(HaveOccurred())
					Expect(err.(*httputils.HttpStatusError).Msg).To(Equal(errMsg))
				} else {
					Expect(err).ShouldNot(HaveOccurred())
				}

			},
			Entry("pass validation when both policy_match and endpointlist is nil / empty",
				nil, []string{}, false, nil),
			Entry("pass validation when only endpointlist is provided ",
				nil, []string{"endpoint1"}, false, nil),
			Entry("fail validation when both policy_match and endpointlist is provided",
				&lapi.PolicyMatch{Action: &allow}, []string{"endpoint1"}, true, "both policyMatch and endpointList can not be provided in the same request"),
			Entry("fail validation when policy_match action is not \"deny\"",
				&lapi.PolicyMatch{Action: &allow}, []string{}, true, "policy_match action can only be set to \"deny\""),
			Entry("pass validation when policy_match action is \"deny\"",
				&lapi.PolicyMatch{Action: &deny}, []string{}, false, nil),
			Entry("fail validation when polict_match.Name is set",
				&lapi.PolicyMatch{Name: &name, Action: &deny}, []string{}, true, "policy_match values provided are not supported for this api"),
			Entry("fail validation when policy_match.NameSpace is set",
				&lapi.PolicyMatch{Namespace: &namespace, Action: &deny}, []string{}, true, "policy_match values provided are not supported for this api"),
			Entry("fail validation when policy_match.Tier is set",
				&lapi.PolicyMatch{Tier: tier, Action: &deny}, []string{}, true, "policy_match values provided are not supported for this api"),
		)

		DescribeTable("validate TimeOut",
			func(timeout, expectedTimeout *v1.Duration) {
				endpointReq := &EndpointsAggregationRequest{}

				if timeout != nil {
					endpointReq.Timeout = timeout
				}

				reqBodyBytes, err := json.Marshal(endpointReq)
				Expect(err).ShouldNot(HaveOccurred())

				req, err = http.NewRequest("POST", "https://test", bytes.NewBuffer(reqBodyBytes))
				Expect(err).ShouldNot(HaveOccurred())

				err = validateEndpointsAggregationRequest(req, endpointReq)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(endpointReq.Timeout).To(Equal(expectedTimeout))
			},
			Entry("should not change timeout if provided",
				&v1.Duration{Duration: 10 * time.Second}, &v1.Duration{Duration: 10 * time.Second}),
			Entry("should set default timeout if not provided",
				nil, &v1.Duration{Duration: DefaultRequestTimeout}),
		)

		testTime := time.Now().UTC()
		DescribeTable("validate TimeRange",
			func(timerange *lmaapi.TimeRange, expectedFrom *time.Time, expectErr bool, errMsg string) {
				endpointReq := &EndpointsAggregationRequest{}

				if timerange != nil {
					endpointReq.TimeRange = timerange
				}

				reqBodyBytes, err := json.Marshal(endpointReq)
				Expect(err).ShouldNot(HaveOccurred())

				req, err = http.NewRequest("POST", "https://test", bytes.NewBuffer(reqBodyBytes))
				Expect(err).ShouldNot(HaveOccurred())

				err = validateEndpointsAggregationRequest(req, endpointReq)

				if expectErr {
					Expect(err).Should(HaveOccurred())
					Expect(err.(*httputils.HttpStatusError).Msg).To(Equal("time_range \"to\" should not be provided"))
				} else {
					Expect(err).ShouldNot(HaveOccurred())
					Expect(endpointReq.TimeRange.To).ToNot(BeNil())

					if expectedFrom != nil {
						Expect(endpointReq.TimeRange.From).To(Equal(*expectedFrom))
					}
				}

			},
			Entry("should fail if timeRange.To is set",
				&lmaapi.TimeRange{To: time.Now().UTC()}, nil, true, "time_range \"to\" should not be provided"),
			Entry("should set timeRange.To to Now if timeRange.From is set",
				&lmaapi.TimeRange{From: testTime}, &testTime, false, ""),
			Entry("should not set timeRange.To if timeRange.From is not set",
				&lmaapi.TimeRange{}, nil, false, ""),
		)
	})

	It("test extractEndpointsFromFlowLogs", func() {
		fl := lapi.FlowLog{
			SourceName:      "src12345",
			SourceNameAggr:  "src*",
			SourceNamespace: "namespace_from",
			DestName:        "dst12345",
			DestNameAggr:    "dst*",
			DestNamespace:   "namespace_to",
		}
		src, dst := extractEndpointsFromFlowLogs(fl)

		Expect(src).To(ContainSubstring(fl.SourceName))
		Expect(dst).To(ContainSubstring(fl.DestName))

	})
})
