package pip

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/numorstring"
	"github.com/projectcalico/libcalico-go/lib/resources"

	pelastic "github.com/tigera/lma/pkg/elastic"

	"github.com/tigera/es-proxy/pkg/pip/config"
	"github.com/tigera/es-proxy/pkg/pip/policycalc"
)

var _ = Describe("Test handling of flow splitting", func() {
	It("handles spliting of flow into the maximum number of possible splits", func() {
		// All flows from A -> B  (both pods)
		// Before conditions:
		//      A(allow)   -> B (allow)     [Current policy allows all flows]
		// After conditions:
		//      A(allow)   -> B (allow)     [Updated policy affects all flows]
		//      A(allow)   -> B (unknown)
		//      A(allow)   -> B (deny)
		//      A(unknown) -> B (allow)
		//      A(unknown) -> B (unknown)
		//      A(unknown) -> B (deny)
		//      A(deny)    -> B (X)
		//
		// Policy to handle the split:
		// Before: no policy before
		// After: Egress  - allow src port 1
		//                - allow src port 2 + service account x   [causes an unknown]
		//                - deny  src port 3
		//        Ingress - allow dst port 1
		//                - allow dst port 2 + service account x   [causes an unknown]
		//                - deny  dst port 3
		//
		// Create a client which has all of the flows that:
		// - allows all both ends using the *before* policy
		// - breaks out into 1 of each of the required after conditions using the *after* policy
		// - has a mixture of allow/deny flows recorded in ES - the policy calculator will recalculate the *before*
		//   flow so will readjust the actual flow data.
		By("Creating an ES client with a mocked out search results with all allow actions")
		client := pelastic.NewMockSearchClient([]interface{}{
			// before: deny/na       after: allow/allow
			//flow("dst", "allow", "tcp", wepd("wepsrc", "ns1", 1), wepd("wepdst", "ns1", 1)), <- denied at source
			flow("src", "deny", "tcp", wepd("wepsrc", "ns1", 1), wepd("wepdst", "ns1", 1)),

			// before: allow/allow   after: allow/unknown
			flow("dst", "allow", "tcp", wepd("wepsrc", "ns1", 1), wepd("wepdst", "ns1", 2)),
			flow("src", "allow", "tcp", wepd("wepsrc", "ns1", 1), wepd("wepdst", "ns1", 2)),

			// before: allow/deny    after: allow/deny
			flow("dst", "deny", "tcp", wepd("wepsrc", "ns1", 1), wepd("wepdst", "ns1", 3)),
			flow("src", "allow", "tcp", wepd("wepsrc", "ns1", 1), wepd("wepdst", "ns1", 3)),

			// before: allow/deny    after: unknown/allow
			flow("dst", "deny", "tcp", wepd("wepsrc", "ns1", 2), wepd("wepdst", "ns1", 1)),
			flow("src", "allow", "tcp", wepd("wepsrc", "ns1", 2), wepd("wepdst", "ns1", 1)),

			// before: allow/allow   after: unknown/unknown
			flow("dst", "allow", "tcp", wepd("wepsrc", "ns1", 2), wepd("wepdst", "ns1", 2)),
			flow("src", "allow", "tcp", wepd("wepsrc", "ns1", 2), wepd("wepdst", "ns1", 2)),

			// before: deny/na       after: deny/na
			//flow("dst", "allow", "tcp", wepd("wepsrc", "ns1", 3), wepd("wepdst", "ns1", 1)), <- denied at source
			flow("src", "deny", "tcp", wepd("wepsrc", "ns1", 3), wepd("wepdst", "ns1", 1)),

			// before: allow/allow   after: unknown/deny
			flow("dst", "allow", "tcp", wepd("wepsrc", "ns1", 2), wepd("wepdst", "ns1", 3)),
			flow("src", "allow", "tcp", wepd("wepsrc", "ns1", 2), wepd("wepdst", "ns1", 3)),
		})

		By("Creating a policy calculator with the required policy updates")
		np := &v3.NetworkPolicy{
			TypeMeta: resources.TypeCalicoNetworkPolicies,
			ObjectMeta: metav1.ObjectMeta{
				Name:      "policy",
				Namespace: "ns1",
			},
			Spec: v3.NetworkPolicySpec{
				Types: []v3.PolicyType{v3.PolicyTypeIngress, v3.PolicyTypeEgress},
				Egress: []v3.Rule{
					{
						Action: v3.Allow,
						Source: v3.EntityRule{
							Ports: []numorstring.Port{
								numorstring.SinglePort(1),
							},
						},
					},
					{
						Action: v3.Allow,
						Source: v3.EntityRule{
							Ports: []numorstring.Port{
								numorstring.SinglePort(2),
							},
							ServiceAccounts: &v3.ServiceAccountMatch{
								Names: []string{"service-account"},
							},
						},
					},
					{
						Action: v3.Deny,
						Source: v3.EntityRule{
							Ports: []numorstring.Port{
								numorstring.SinglePort(3),
							},
						},
					},
				},
				Ingress: []v3.Rule{
					{
						Action: v3.Allow,
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								numorstring.SinglePort(1),
							},
						},
					},
					{
						Action: v3.Allow,
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								numorstring.SinglePort(2),
							},
							ServiceAccounts: &v3.ServiceAccountMatch{
								Names: []string{"service-account"},
							},
						},
					},
					{
						Action: v3.Deny,
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								numorstring.SinglePort(3),
							},
						},
					},
				},
			},
		}
		impacted := make(policycalc.ImpactedResources)
		impacted.Add(resources.GetResourceID(np), policycalc.Impact{Modified: true})
		pc := policycalc.NewPolicyCalculator(
			&config.Config{
				CalculateOriginalAction: true, // <- we want to recalculate the original action
			},
			policycalc.NewEndpointCache(),
			&policycalc.ResourceData{},
			&policycalc.ResourceData{
				Tiers: policycalc.Tiers{
					{policycalc.Policy{CalicoV3Policy: np, ResourceID: resources.GetResourceID(np)}},
				},
			},
			impacted,
		)

		By("Creating a composite agg query")
		q := &pelastic.CompositeAggregationQuery{
			Name:                    pelastic.FlowlogBuckets,
			AggCompositeSourceInfos: PIPCompositeSources,
			AggNestedTermInfos:      pelastic.FlowAggregatedTerms,
			AggSumInfos:             pelastic.FlowAggregationSums,
			MaxBucketsPerQuery:      1, // Set this to ensure we iterate after only a single response.
		}

		By("Creating a PIP instance with the mock client, and enumerating all aggregated flows")
		pip := pip{esClient: client, cfg: config.MustLoadConfig()}
		flowsChan, _ := pip.SearchAndProcessFlowLogs(context.Background(), q, nil, pc, 1000, false, pelastic.NewFlowFilterIncludeAll())
		var before []*pelastic.CompositeAggregationBucket
		var after []*pelastic.CompositeAggregationBucket
		for flow := range flowsChan {
			before = append(before, flow.Before...)
			after = append(after, flow.After...)
		}

		// Before: We expect 1 flow at source, 1 flow at dest.
		// After:  We expect 3 flows at source, 6 flows at dest (there is no corresponding dest flow for source deny)
		Expect(before).To(HaveLen(2))
		Expect(after).To(HaveLen(9))

		// Ordering is by reporter, action, source_action.
		Expect(before[0].DocCount).To(BeEquivalentTo(7))
		Expect(before[0].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{Name: "source_type", Value: "wep"},
			{Name: "source_namespace", Value: "ns1"},
			{Name: "source_name", Value: "wepsrc"},
			{Name: "dest_type", Value: "wep"},
			{Name: "dest_namespace", Value: "ns1"},
			{Name: "dest_name", Value: "wepdst"},
			{Name: "reporter", Value: "dst"},
			{Name: "action", Value: "allow"},
			{Name: "source_action", Value: "allow"},
			{Name: "flow_impacted", Value: true},
		}))

		Expect(before[1].DocCount).To(BeEquivalentTo(7))
		Expect(before[1].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{Name: "source_type", Value: "wep"},
			{Name: "source_namespace", Value: "ns1"},
			{Name: "source_name", Value: "wepsrc"},
			{Name: "dest_type", Value: "wep"},
			{Name: "dest_namespace", Value: "ns1"},
			{Name: "dest_name", Value: "wepdst"},
			{Name: "reporter", Value: "src"},
			{Name: "action", Value: "allow"},
			{Name: "source_action", Value: "allow"},
			{Name: "flow_impacted", Value: true},
		}))

		Expect(after[0].DocCount).To(BeEquivalentTo(1))
		Expect(after[0].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{Name: "source_type", Value: "wep"},
			{Name: "source_namespace", Value: "ns1"},
			{Name: "source_name", Value: "wepsrc"},
			{Name: "dest_type", Value: "wep"},
			{Name: "dest_namespace", Value: "ns1"},
			{Name: "dest_name", Value: "wepdst"},
			{Name: "reporter", Value: "dst"},
			{Name: "action", Value: "allow"},
			{Name: "source_action", Value: "allow"},
			{Name: "flow_impacted", Value: true},
		}))

		Expect(after[1].DocCount).To(BeEquivalentTo(1))
		Expect(after[1].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{Name: "source_type", Value: "wep"},
			{Name: "source_namespace", Value: "ns1"},
			{Name: "source_name", Value: "wepsrc"},
			{Name: "dest_type", Value: "wep"},
			{Name: "dest_namespace", Value: "ns1"},
			{Name: "dest_name", Value: "wepdst"},
			{Name: "reporter", Value: "dst"},
			{Name: "action", Value: "allow"},
			{Name: "source_action", Value: "unknown"},
			{Name: "flow_impacted", Value: true},
		}))

		Expect(after[2].DocCount).To(BeEquivalentTo(1))
		Expect(after[2].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{Name: "source_type", Value: "wep"},
			{Name: "source_namespace", Value: "ns1"},
			{Name: "source_name", Value: "wepsrc"},
			{Name: "dest_type", Value: "wep"},
			{Name: "dest_namespace", Value: "ns1"},
			{Name: "dest_name", Value: "wepdst"},
			{Name: "reporter", Value: "dst"},
			{Name: "action", Value: "deny"},
			{Name: "source_action", Value: "allow"},
			{Name: "flow_impacted", Value: true},
		}))

		Expect(after[3].DocCount).To(BeEquivalentTo(1))
		Expect(after[3].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{Name: "source_type", Value: "wep"},
			{Name: "source_namespace", Value: "ns1"},
			{Name: "source_name", Value: "wepsrc"},
			{Name: "dest_type", Value: "wep"},
			{Name: "dest_namespace", Value: "ns1"},
			{Name: "dest_name", Value: "wepdst"},
			{Name: "reporter", Value: "dst"},
			{Name: "action", Value: "deny"},
			{Name: "source_action", Value: "unknown"},
			{Name: "flow_impacted", Value: true},
		}))

		Expect(after[4].DocCount).To(BeEquivalentTo(1))
		Expect(after[4].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{Name: "source_type", Value: "wep"},
			{Name: "source_namespace", Value: "ns1"},
			{Name: "source_name", Value: "wepsrc"},
			{Name: "dest_type", Value: "wep"},
			{Name: "dest_namespace", Value: "ns1"},
			{Name: "dest_name", Value: "wepdst"},
			{Name: "reporter", Value: "dst"},
			{Name: "action", Value: "unknown"},
			{Name: "source_action", Value: "allow"},
			{Name: "flow_impacted", Value: true},
		}))

		Expect(after[5].DocCount).To(BeEquivalentTo(1))
		Expect(after[5].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{Name: "source_type", Value: "wep"},
			{Name: "source_namespace", Value: "ns1"},
			{Name: "source_name", Value: "wepsrc"},
			{Name: "dest_type", Value: "wep"},
			{Name: "dest_namespace", Value: "ns1"},
			{Name: "dest_name", Value: "wepdst"},
			{Name: "reporter", Value: "dst"},
			{Name: "action", Value: "unknown"},
			{Name: "source_action", Value: "unknown"},
			{Name: "flow_impacted", Value: true},
		}))

		Expect(after[6].DocCount).To(BeEquivalentTo(3))
		Expect(after[6].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{Name: "source_type", Value: "wep"},
			{Name: "source_namespace", Value: "ns1"},
			{Name: "source_name", Value: "wepsrc"},
			{Name: "dest_type", Value: "wep"},
			{Name: "dest_namespace", Value: "ns1"},
			{Name: "dest_name", Value: "wepdst"},
			{Name: "reporter", Value: "src"},
			{Name: "action", Value: "allow"},
			{Name: "source_action", Value: "allow"},
			{Name: "flow_impacted", Value: true},
		}))

		Expect(after[7].DocCount).To(BeEquivalentTo(1))
		Expect(after[7].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{Name: "source_type", Value: "wep"},
			{Name: "source_namespace", Value: "ns1"},
			{Name: "source_name", Value: "wepsrc"},
			{Name: "dest_type", Value: "wep"},
			{Name: "dest_namespace", Value: "ns1"},
			{Name: "dest_name", Value: "wepdst"},
			{Name: "reporter", Value: "src"},
			{Name: "action", Value: "deny"},
			{Name: "source_action", Value: "deny"},
			{Name: "flow_impacted", Value: true},
		}))

		Expect(after[8].DocCount).To(BeEquivalentTo(3))
		Expect(after[8].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{Name: "source_type", Value: "wep"},
			{Name: "source_namespace", Value: "ns1"},
			{Name: "source_name", Value: "wepsrc"},
			{Name: "dest_type", Value: "wep"},
			{Name: "dest_namespace", Value: "ns1"},
			{Name: "dest_name", Value: "wepdst"},
			{Name: "reporter", Value: "src"},
			{Name: "action", Value: "unknown"},
			{Name: "source_action", Value: "unknown"},
			{Name: "flow_impacted", Value: true},
		}))
	})
})
