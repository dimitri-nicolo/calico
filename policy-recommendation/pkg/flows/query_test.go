package flows

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	lapi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
)

// MockResults results for most tests. Consists of two returned flows.
var MockResults = []rest.MockResult{
	{
		Body: lapi.List[lapi.L3Flow]{
			TotalHits: 7,
			Items: []lapi.L3Flow{
				{
					// key-1
					Key: lapi.L3FlowKey{
						Action:   lapi.FlowActionAllow,
						Reporter: lapi.FlowReporterSource,
						Source: lapi.Endpoint{
							Namespace:      "namespace1",
							AggregatedName: "netshoot-cb7967547-*",
							Type:           lapi.WEP,
						},
						Destination: lapi.Endpoint{
							Namespace:      "-",
							AggregatedName: "pub",
							Type:           lapi.Network,
							Port:           80,
						},
						Protocol: "tcp",
					},
					LogStats:    &lapi.LogStats{FlowLogCount: 4370},
					DestDomains: []string{"www.google.com"},
				},
				{
					// key-2
					Key: lapi.L3FlowKey{
						Action:   lapi.FlowActionAllow,
						Reporter: lapi.FlowReporterSource,
						Source: lapi.Endpoint{
							Namespace:      "namespace1",
							AggregatedName: "netshoot-cb7967547-*",
							Type:           lapi.WEP,
						},
						Destination: lapi.Endpoint{
							Namespace:      "-",
							AggregatedName: "pub",
							Type:           lapi.Network,
							Port:           81,
						},
						Protocol: "tcp",
					},
					LogStats:    &lapi.LogStats{FlowLogCount: 3698},
					DestDomains: []string{"www.website.com"},
				},
				{
					// key-3
					Key: lapi.L3FlowKey{
						Action:   lapi.FlowActionAllow,
						Reporter: lapi.FlowReporterSource,
						Source: lapi.Endpoint{
							Namespace:      "namespace3",
							AggregatedName: "netshoot3-cb7967547-*",
							Type:           lapi.WEP,
						},
						Destination: lapi.Endpoint{
							Namespace:      "-",
							AggregatedName: "pub",
							Type:           lapi.Network,
							Port:           666,
						},
						Protocol: "tcp",
					},
					LogStats:    &lapi.LogStats{FlowLogCount: 3698},
					DestDomains: []string{"www.tigera.io"},
				},
				{
					// key-4
					Key: lapi.L3FlowKey{
						Action:   lapi.FlowActionAllow,
						Reporter: lapi.FlowReporterSource,
						Source: lapi.Endpoint{
							Namespace:      "namespace1",
							AggregatedName: "netshoot-cb7967547-*",
							Type:           lapi.WEP,
						},
						Destination: lapi.Endpoint{
							Namespace:      "-",
							AggregatedName: "pub",
							Type:           lapi.Network,
							Port:           81,
						},
						Protocol: "tcp",
					},
					LogStats:    &lapi.LogStats{FlowLogCount: 3698},
					DestDomains: []string{"www.google.com"},
				},
				{
					// key-5
					Key: lapi.L3FlowKey{
						Action:   lapi.FlowActionAllow,
						Reporter: lapi.FlowReporterSource,
						Source: lapi.Endpoint{
							Namespace:      "namespace3",
							AggregatedName: "netshoot3-cb7967547-*",
							Type:           lapi.WEP,
						},
						Destination: lapi.Endpoint{
							Namespace:      "namespace1",
							AggregatedName: "netshoot1-cb7967547-*",
							Type:           lapi.WEP,
						},
						Protocol: "tcp",
					},
					LogStats: &lapi.LogStats{FlowLogCount: 3698},
				},
				{
					// key-6
					Key: lapi.L3FlowKey{
						Action:   lapi.FlowActionAllow,
						Reporter: lapi.FlowReporterSource,
						Source: lapi.Endpoint{
							Namespace:      "namespace3",
							AggregatedName: "netshoot3-cb7967547-*",
							Type:           lapi.WEP,
						},
						Destination: lapi.Endpoint{
							Namespace:      "namespace1",
							AggregatedName: "netshoot1-cb7967547-*",
							Type:           lapi.WEP,
						},
						Protocol: "udp",
					},
					LogStats: &lapi.LogStats{FlowLogCount: 3698},
				},
				{
					// key-7
					Key: lapi.L3FlowKey{
						Action:   lapi.FlowActionAllow,
						Reporter: lapi.FlowReporterSource,
						Source: lapi.Endpoint{
							Namespace:      "namespace3",
							AggregatedName: "netshoot3-cb7967547-*",
							Type:           lapi.WEP,
						},
						Destination: lapi.Endpoint{
							Namespace:      "namespace3",
							AggregatedName: "netshoot1-cb7967547-*",
							Type:           lapi.WEP,
						},
						Protocol: "tcp",
					},
					LogStats: &lapi.LogStats{FlowLogCount: 3698},
				},
			},
		},
	},
}

func TestQueryFlows(t *testing.T) {
	RegisterTestingT(t)

	var mockLinseedClient client.MockClient

	ctx := context.Background()

	t.Run("should return error if nil params are passed", func(t *testing.T) {
		mockLinseedClient = client.NewMockClient("")
		q := NewRecommendationFlowLogQuery(ctx, mockLinseedClient, "test-cluster-id")
		flows, err := q.QueryFlows(nil)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("invalid flow query parameters"))
		Expect(flows).To(BeNil())
	})

	t.Run("should fetch flows when valid params are provided", func(t *testing.T) {
		mockLinseedClient = client.NewMockClient("")
		mockLinseedClient.SetResults(MockResults...)

		q := NewRecommendationFlowLogQuery(ctx, mockLinseedClient, "test-cluster-id")
		params := &RecommendationFlowLogQueryParams{
			StartTime:   1 * time.Minute,
			EndTime:     30 * time.Second,
			Namespace:   "default",
			Unprotected: false,
		}

		flows, err := q.QueryFlows(params)
		Expect(err).ToNot(HaveOccurred())
		Expect(flows).To(HaveLen(7))
	})
}

func TestBuildPolicyMatchQueries(t *testing.T) {
	RegisterTestingT(t)

	t.Run("buildPendingPolicyMatchQuery with unprotected=false", func(t *testing.T) {
		params := &RecommendationFlowLogQueryParams{
			StartTime:   1 * time.Minute,
			EndTime:     1 * time.Minute,
			Namespace:   "my-namespace",
			Unprotected: false,
		}
		fp := buildPendingPolicyMatchQuery(params)

		Expect(fp.PolicyMatches).To(BeNil())
		Expect(fp.PendingPolicyMatches).ToNot(BeNil())
		Expect(fp.NamespaceMatches).To(HaveLen(1))
		Expect(fp.NamespaceMatches[0].Namespaces).To(ConsistOf("my-namespace"))
		Expect(fp.PendingPolicyMatches).To(HaveLen(2))
		Expect(fp.PendingPolicyMatches[0].Tier).To(Equal("default"))
		Expect(fp.PendingPolicyMatches[1].Tier).To(Equal("__PROFILE__"))
		Expect(*fp.PendingPolicyMatches[1].Action).To(Equal(lapi.FlowActionAllow))
	})

	t.Run("buildPendingPolicyMatchQuery with unprotected=true", func(t *testing.T) {
		params := &RecommendationFlowLogQueryParams{
			StartTime:   5 * time.Minute,
			EndTime:     2 * time.Minute,
			Namespace:   "",
			Unprotected: true,
		}
		fp := buildPendingPolicyMatchQuery(params)

		Expect(fp.PolicyMatches).To(BeNil())
		Expect(fp.PendingPolicyMatches).ToNot(BeNil())
		Expect(fp.PendingPolicyMatches).To(HaveLen(1))
		Expect(fp.PendingPolicyMatches[0].Tier).To(Equal("__PROFILE__"))
		Expect(*fp.PendingPolicyMatches[0].Action).To(Equal(lapi.FlowActionAllow))

	})

	t.Run("buildAllPolicyMatchQuery with unprotected=false", func(t *testing.T) {
		params := &RecommendationFlowLogQueryParams{
			StartTime:   1 * time.Minute,
			EndTime:     1 * time.Minute,
			Unprotected: false,
		}
		fp := buildAllPolicyMatchQuery(params)

		Expect(fp.PendingPolicyMatches).To(BeNil())
		Expect(fp.PolicyMatches).ToNot(BeNil())
		Expect(fp.PolicyMatches).To(HaveLen(2))
		Expect(fp.PolicyMatches[0].Tier).To(Equal("default"))
		Expect(fp.PolicyMatches[1].Tier).To(Equal("__PROFILE__"))
		Expect(*fp.PolicyMatches[1].Action).To(Equal(lapi.FlowActionAllow))
	})

	t.Run("buildAllPolicyMatchQuery with unprotected=true", func(t *testing.T) {
		params := &RecommendationFlowLogQueryParams{
			StartTime:   3 * time.Minute,
			EndTime:     30 * time.Second,
			Unprotected: true,
		}
		fp := buildAllPolicyMatchQuery(params)

		Expect(fp.PendingPolicyMatches).To(BeNil())
		Expect(fp.PolicyMatches).ToNot(BeNil())
		Expect(fp.PolicyMatches).To(HaveLen(1))
		Expect(fp.PolicyMatches[0].Tier).To(Equal("__PROFILE__"))
		Expect(*fp.PolicyMatches[0].Action).To(Equal(lapi.FlowActionAllow))
	})
}
