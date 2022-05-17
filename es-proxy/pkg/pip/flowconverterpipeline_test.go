package pip

import (
	"bytes"
	"context"
	"text/template"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/lma/pkg/api"
	"github.com/projectcalico/calico/lma/pkg/elastic"

	"github.com/projectcalico/calico/es-proxy/pkg/pip/config"
	"github.com/projectcalico/calico/es-proxy/pkg/pip/policycalc"
)

// epData encapsulates endpoint data for these tests. It is not a full representation, merely enough to make the
// aggregation interesting - we have tested the actual aggregation of composite objects specificially in other UTs so
// don't need to worry about full tests here.
type epData struct {
	Type      string
	Namespace string
	NameAggr  string
	Port      int
}

// flowData encapsulates flow data for these tests. It is not a full representation, merely enough to make the
// aggregation interesting - we have tested the actual aggregation of composite objects specificially in other UTs so
// don't need to worry about full tests here.
type flowData struct {
	Reporter    string
	Protocol    string
	Action      string
	Source      epData
	Destination epData
	Policies    []string
}

// wepd creates an epData for a WEP.
func wepd(name, namespace string, port int) epData {
	return epData{
		Type:      "wep",
		Namespace: namespace,
		NameAggr:  name,
		Port:      port,
	}
}

// hepd creates an epData for a HEP.
func hepd(name string, port int) epData {
	return epData{
		Type:      "hep",
		NameAggr:  name,
		Namespace: "-",
		Port:      port,
	}
}

var (
	// Template for an ES response. We only include a single flow in each response, but set the key to ensure we
	// continue enumerating.
	flowTemplate, _ = template.New("flow").Parse(`{
  "took": 791,
  "timed_out": false,
  "_shards": {
    "total": 15,
    "successful": 15,
    "skipped": 0,
    "failed": 0
  },
  "hits": {
    "total": {
      "relation": "eq",
      "value": 26095
    },
    "max_score": 0,
    "hits": []
  },
  "aggregations": {
    "flog_buckets": {
      "after_key": {
        "source_type": "{{ .Source.Type }}",
        "source_namespace": "{{ .Source.Namespace }}",
        "source_name": "{{ .Source.NameAggr }}",
        "dest_type": "{{ .Destination.Type }}",
        "dest_namespace":  "{{ .Destination.Namespace }}",
        "dest_name":  "{{ .Destination.NameAggr }}",
        "reporter": "{{ .Reporter }}",
        "action": "{{ .Action }}",
        "proto": "{{ .Protocol }}",
        "source_ip": "0.0.0.0",
        "source_name_full": "-",
        "source_port": {{ .Source.Port }},
        "dest_ip": "0.0.0.0",
        "dest_name_full": "-",
        "dest_port": {{ .Destination.Port }}
      },
      "buckets": [
        {
          "key": {
            "source_type": "{{ .Source.Type }}",
            "source_namespace": "{{ .Source.Namespace }}",
            "source_name": "{{ .Source.NameAggr }}",
            "dest_type": "{{ .Destination.Type }}",
            "dest_namespace":  "{{ .Destination.Namespace }}",
            "dest_name":  "{{ .Destination.NameAggr }}",
            "reporter": "{{ .Reporter }}",
            "action": "{{ .Action }}",
            "proto": "{{ .Protocol }}",
            "source_ip": "0.0.0.0",
            "source_name_full": "-",
            "source_port": {{ .Source.Port }},
            "dest_ip": "0.0.0.0",
            "dest_name_full": "-",
            "dest_port": {{ .Destination.Port }}
          },
          "doc_count": 1,
          "sum_http_requests_denied_in": {
            "value": 1
          },
          "sum_num_flows_started": {
            "value": 1
          },
          "sum_bytes_in": {
            "value": 1
          },
          "sum_packets_out": {
            "value": 1
          },
          "sum_packets_in": {
            "value": 1
          },
          "policies": {
            "doc_count": 1,
            "by_tiered_policy": {
              "doc_count_error_upper_bound": 0,
              "sum_other_doc_count": 0,
              "buckets": [
{{$first := true}}{{range $key, $value := .Policies }}{{if $first}}{{$first = false}}{{else}}                },
{{end}}                {
                  "key": "{{$key}}|{{$value}}",
                  "doc_count": 1
{{end}}
                }
              ]
            }
          },
          "source_labels": {
            "doc_count": 1,
            "by_kvpair": {
              "buckets": [
                {
                  "key": "cluster=tigera-elasticsearch",
                  "doc_count": 1
                }
              ]
            }
          },
          "sum_bytes_out": {
            "value": 1
          },
          "dest_labels": {
            "doc_count": 1,
            "by_kvpair": {
              "buckets": [
                {
                  "key": "cluster=tigera-elasticsearch",
                  "doc_count": 1
                }
              ]
            }
          },
          "sum_http_requests_allowed_in": {
            "value": 1
          },
          "sum_num_flows_completed": {
            "value": 1
          }
        }
      ]
    }
  }
}`)

	defaultPolicy = "allow-cnx|calico-monitoring/allow-cnx.elasticsearch-access|allow"
)

// flow generates an ES flow response used by the mock ES client.
func flow(reporter, action, protocol string, source, dest epData, policies ...string) string {
	if len(policies) == 0 {
		policies = []string{defaultPolicy}
	}
	fd := flowData{
		Reporter:    reporter,
		Protocol:    protocol,
		Action:      action,
		Source:      source,
		Destination: dest,
		Policies:    policies,
	}
	var tpl bytes.Buffer
	err := flowTemplate.Execute(&tpl, fd)
	Expect(err).NotTo(HaveOccurred())
	return tpl.String()
}

// alwaysAllowCalculator implements the policy calculator interface with an always allow source and dest response for
// the after buckets.
type alwaysAllowCalculator struct{}

func (c alwaysAllowCalculator) CalculateSource(flow *api.Flow) (bool, policycalc.EndpointResponse, policycalc.EndpointResponse) {
	before := policycalc.EndpointResponse{
		Include: true,
		Action:  flow.ActionFlag,
		Policies: []api.PolicyHit{
			mustCreatePolicyHit("0|tier1|tier1.policy1|pass", 1),
			mustCreatePolicyHit("1|default|default.policy1|allow", 1),
		},
	}
	after := policycalc.EndpointResponse{
		Include: true,
		Action:  api.ActionFlagAllow,
		Policies: []api.PolicyHit{
			mustCreatePolicyHit("0|tier1|tier1.policy1|pass", 1),
			mustCreatePolicyHit("1|default|default.policy1|allow", 1),
		},
	}
	return flow.ActionFlag != api.ActionFlagAllow, before, after
}

func (_ alwaysAllowCalculator) CalculateDest(
	flow *api.Flow, beforeSourceAction, afterSourceAction api.ActionFlag,
) (modified bool, before, after policycalc.EndpointResponse) {
	if beforeSourceAction != api.ActionFlagDeny {
		before = policycalc.EndpointResponse{
			Include: true,
			Action:  flow.ActionFlag,
			Policies: []api.PolicyHit{
				mustCreatePolicyHit("0|tier1|tier1.policy1|pass", 1),
				mustCreatePolicyHit("1|default|default.policy1|allow", 1),
			},
		}
	}
	if afterSourceAction != api.ActionFlagDeny {
		after = policycalc.EndpointResponse{
			// Add a destination flow if the original src flow was Deny and now we allow.
			Include: true,
			Action:  api.ActionFlagAllow,
			Policies: []api.PolicyHit{
				mustCreatePolicyHit("0|tier1|tier1.policy1|pass", 1),
				mustCreatePolicyHit("1|default|default.policy1|allow", 1),
			},
		}
	}
	return flow.ActionFlag != api.ActionFlagAllow, before, after
}

// alwaysDenyCalculator implements the policy calculator interface with an always deny source and dest response for the
// after buckets.
type alwaysDenyCalculator struct{}

func (c alwaysDenyCalculator) CalculateSource(flow *api.Flow) (bool, policycalc.EndpointResponse, policycalc.EndpointResponse) {
	before := policycalc.EndpointResponse{
		Include: true,
		Action:  flow.ActionFlag,
		Policies: []api.PolicyHit{
			mustCreatePolicyHit("0|tier1|tier1.policy1|pass", 1),
			mustCreatePolicyHit("1|default|default.policy1|allow", 1),
		},
	}
	after := policycalc.EndpointResponse{
		Include: true,
		Action:  api.ActionFlagDeny,
		Policies: []api.PolicyHit{
			mustCreatePolicyHit("0|tier1|tier1.policy1|pass", 1),
			mustCreatePolicyHit("1|default|default.policy1|allow", 1),
		},
	}
	return flow.ActionFlag != api.ActionFlagDeny, before, after
}

func (_ alwaysDenyCalculator) CalculateDest(
	flow *api.Flow, beforeSourceAction, afterSourceAction api.ActionFlag,
) (modified bool, before, after policycalc.EndpointResponse) {
	if beforeSourceAction != api.ActionFlagDeny {
		before = policycalc.EndpointResponse{
			Include: true,
			Action:  flow.ActionFlag,
			Policies: []api.PolicyHit{
				mustCreatePolicyHit("0|tier1|tier1.policy1|deny", 1),
				mustCreatePolicyHit("1|default|default.policy1|deny", 1),
			},
		}
	}
	if afterSourceAction != api.ActionFlagDeny {
		before = policycalc.EndpointResponse{
			Include: true,
			Action:  api.ActionFlagDeny,
			Policies: []api.PolicyHit{
				mustCreatePolicyHit("0|tier1|tier1.policy1|pass", 1),
				mustCreatePolicyHit("1|default|default.policy1|deny", 1),
			},
		}
	}
	return flow.ActionFlag != api.ActionFlagDeny, before, after
}

var _ = Describe("Test relationship between PIP and API queries", func() {
	It("has the same lower set of indexes", func() {
		Expect(PIPCompositeSourcesRawIdxSourceType).To(Equal(elastic.FlowCompositeSourcesIdxSourceType))
		Expect(PIPCompositeSourcesRawIdxSourceNamespace).To(Equal(elastic.FlowCompositeSourcesIdxSourceNamespace))
		Expect(PIPCompositeSourcesRawIdxSourceNameAggr).To(Equal(elastic.FlowCompositeSourcesIdxSourceNameAggr))
		Expect(PIPCompositeSourcesRawIdxDestType).To(Equal(elastic.FlowCompositeSourcesIdxDestType))
		Expect(PIPCompositeSourcesRawIdxDestNamespace).To(Equal(elastic.FlowCompositeSourcesIdxDestNamespace))
		Expect(PIPCompositeSourcesRawIdxDestNameAggr).To(Equal(elastic.FlowCompositeSourcesIdxDestNameAggr))

	})
})

var _ = Describe("Test handling of aggregated ES response", func() {
	It("handles simple aggregation of results where action does not change", func() {
		By("Creating an ES client with a mocked out search results with all allow actions")
		client := elastic.NewMockSearchClient([]interface{}{
			// Dest api.
			flow("dst", "allow", "tcp", hepd("hep1", 100), hepd("hep2", 200)), // + Aggregate before and after
			flow("dst", "allow", "udp", hepd("hep1", 100), hepd("hep2", 200)), // |
			flow("dst", "allow", "tcp", hepd("hep1", 500), hepd("hep2", 600)), // +
			// Source api.
			flow("src", "allow", "tcp", hepd("hep1", 100), hepd("hep2", 200)), // + Aggregate before and after
			flow("src", "allow", "udp", hepd("hep1", 100), hepd("hep2", 200)), // |
			flow("src", "allow", "tcp", hepd("hep1", 500), hepd("hep2", 600)), // +
			// WEP
			flow("src", "allow", "tcp", wepd("hep1", "ns1", 100), hepd("hep2", 200)), // Missing dest flow
		})

		By("Creating a composite agg query")
		q := &elastic.CompositeAggregationQuery{
			Name:                    api.FlowlogBuckets,
			AggCompositeSourceInfos: PIPCompositeSources,
			AggNestedTermInfos:      elastic.FlowAggregatedTerms,
			AggSumInfos:             elastic.FlowAggregationSums,
			MaxBucketsPerQuery:      1, // Set this to ensure we iterate after only a single response.
		}

		By("Creating a PIP instance with the mock client, and enumerating all aggregated flows (always allow after)")
		pip := pip{esClient: client, cfg: config.MustLoadConfig()}
		flowsChan, _ := pip.SearchAndProcessFlowLogs(context.Background(), q, nil, alwaysAllowCalculator{}, 1000, false, elastic.NewFlowFilterIncludeAll())
		var before []*elastic.CompositeAggregationBucket
		var after []*elastic.CompositeAggregationBucket
		for flow := range flowsChan {
			before = append(before, flow.Before...)
			after = append(after, flow.After...)
		}

		Expect(before).To(HaveLen(3))
		Expect(before[0].DocCount).To(BeEquivalentTo(3))
		Expect(before[0].CompositeAggregationKey).To(Equal(elastic.CompositeAggregationKey{
			{Name: "source_type", Value: "hep"},
			{Name: "source_namespace", Value: "-"},
			{Name: "source_name", Value: "hep1"},
			{Name: "dest_type", Value: "hep"},
			{Name: "dest_namespace", Value: "-"},
			{Name: "dest_name", Value: "hep2"},
			{Name: "reporter", Value: "dst"},
			{Name: "action", Value: "allow"},
			{Name: "source_action", Value: "allow"},
			{Name: "flow_impacted", Value: false},
		}))
		Expect(before[1].DocCount).To(BeEquivalentTo(3))
		Expect(before[1].CompositeAggregationKey).To(Equal(elastic.CompositeAggregationKey{
			{Name: "source_type", Value: "hep"},
			{Name: "source_namespace", Value: "-"},
			{Name: "source_name", Value: "hep1"},
			{Name: "dest_type", Value: "hep"},
			{Name: "dest_namespace", Value: "-"},
			{Name: "dest_name", Value: "hep2"},
			{Name: "reporter", Value: "src"},
			{Name: "action", Value: "allow"},
			{Name: "source_action", Value: "allow"},
			{Name: "flow_impacted", Value: false},
		}))
		Expect(before[2].DocCount).To(BeEquivalentTo(1))
		Expect(before[2].CompositeAggregationKey).To(Equal(elastic.CompositeAggregationKey{
			{Name: "source_type", Value: "wep"},
			{Name: "source_namespace", Value: "ns1"},
			{Name: "source_name", Value: "hep1"},
			{Name: "dest_type", Value: "hep"},
			{Name: "dest_namespace", Value: "-"},
			{Name: "dest_name", Value: "hep2"},
			{Name: "reporter", Value: "src"},
			{Name: "action", Value: "allow"},
			{Name: "source_action", Value: "allow"},
			{Name: "flow_impacted", Value: false},
		}))

		Expect(after).To(HaveLen(3))
		Expect(after[0].DocCount).To(BeEquivalentTo(3))
		Expect(after[0].CompositeAggregationKey).To(Equal(elastic.CompositeAggregationKey{
			{Name: "source_type", Value: "hep"},
			{Name: "source_namespace", Value: "-"},
			{Name: "source_name", Value: "hep1"},
			{Name: "dest_type", Value: "hep"},
			{Name: "dest_namespace", Value: "-"},
			{Name: "dest_name", Value: "hep2"},
			{Name: "reporter", Value: "dst"},
			{Name: "action", Value: "allow"},
			{Name: "source_action", Value: "allow"},
			{Name: "flow_impacted", Value: false},
		}))
		Expect(after[1].DocCount).To(BeEquivalentTo(3))
		Expect(after[1].CompositeAggregationKey).To(Equal(elastic.CompositeAggregationKey{
			{Name: "source_type", Value: "hep"},
			{Name: "source_namespace", Value: "-"},
			{Name: "source_name", Value: "hep1"},
			{Name: "dest_type", Value: "hep"},
			{Name: "dest_namespace", Value: "-"},
			{Name: "dest_name", Value: "hep2"},
			{Name: "reporter", Value: "src"},
			{Name: "action", Value: "allow"},
			{Name: "source_action", Value: "allow"},
			{Name: "flow_impacted", Value: false},
		}))
		Expect(after[2].DocCount).To(BeEquivalentTo(1))
		Expect(after[2].CompositeAggregationKey).To(Equal(elastic.CompositeAggregationKey{
			{Name: "source_type", Value: "wep"},
			{Name: "source_namespace", Value: "ns1"},
			{Name: "source_name", Value: "hep1"},
			{Name: "dest_type", Value: "hep"},
			{Name: "dest_namespace", Value: "-"},
			{Name: "dest_name", Value: "hep2"},
			{Name: "reporter", Value: "src"},
			{Name: "action", Value: "allow"},
			{Name: "source_action", Value: "allow"},
			{Name: "flow_impacted", Value: false},
		}))
	})

	It("handles source flows changing from deny to allow", func() {
		By("Creating an ES client with a mocked out ES results being a mixture of allow and deny")
		client := elastic.NewMockSearchClient([]interface{}{
			// Dest api.
			// flow("dst", "allow", "tcp", hepd("hep1", 100), hepd("hep2", 200)), <- this flow is now deny at source,
			//                                                                       but will reappear in "after flows"
			flow("dst", "deny", "udp", hepd("hep1", 100), hepd("hep2", 200)),  //                // + AggregatedProtoPorts after
			flow("dst", "allow", "tcp", hepd("hep1", 500), hepd("hep2", 600)), //                // +
			// Source api.
			flow("src", "deny", "tcp", hepd("hep1", 100), hepd("hep2", 200)),  //                // + AggregatedProtoPorts after
			flow("src", "allow", "udp", hepd("hep1", 100), hepd("hep2", 200)), // + AggregatedProtoPorts   // |
			flow("src", "allow", "tcp", hepd("hep1", 500), hepd("hep2", 600)), // + before       // +
			// WEP
			flow("src", "allow", "tcp", wepd("hep1", "ns1", 100), hepd("hep2", 200)), // Missing dest flow
		})

		By("Creating a composite agg query")
		q := &elastic.CompositeAggregationQuery{
			Name:                    api.FlowlogBuckets,
			AggCompositeSourceInfos: PIPCompositeSources,
			AggNestedTermInfos:      elastic.FlowAggregatedTerms,
			AggSumInfos:             elastic.FlowAggregationSums,
			MaxBucketsPerQuery:      1, // Set this to ensure we iterate after only a single response.
		}

		By("Creating a PIP instance with the mock client, and enumerating all aggregated flows (always allow after)")
		pip := pip{esClient: client, cfg: config.MustLoadConfig()}
		flowsChan, _ := pip.SearchAndProcessFlowLogs(context.Background(), q, nil, alwaysAllowCalculator{}, 1000, false, elastic.NewFlowFilterIncludeAll())
		var before []*elastic.CompositeAggregationBucket
		var after []*elastic.CompositeAggregationBucket
		for flow := range flowsChan {
			before = append(before, flow.Before...)
			after = append(after, flow.After...)
		}

		Expect(before).To(HaveLen(5))
		Expect(before[0].DocCount).To(BeEquivalentTo(1))
		Expect(before[0].CompositeAggregationKey).To(Equal(elastic.CompositeAggregationKey{
			{Name: "source_type", Value: "hep"},
			{Name: "source_namespace", Value: "-"},
			{Name: "source_name", Value: "hep1"},
			{Name: "dest_type", Value: "hep"},
			{Name: "dest_namespace", Value: "-"},
			{Name: "dest_name", Value: "hep2"},
			{Name: "reporter", Value: "dst"},
			{Name: "action", Value: "allow"},
			{Name: "source_action", Value: "allow"},
			{Name: "flow_impacted", Value: true},
		}))
		Expect(before[1].DocCount).To(BeEquivalentTo(1))
		Expect(before[1].CompositeAggregationKey).To(Equal(elastic.CompositeAggregationKey{
			{Name: "source_type", Value: "hep"},
			{Name: "source_namespace", Value: "-"},
			{Name: "source_name", Value: "hep1"},
			{Name: "dest_type", Value: "hep"},
			{Name: "dest_namespace", Value: "-"},
			{Name: "dest_name", Value: "hep2"},
			{Name: "reporter", Value: "dst"},
			{Name: "action", Value: "deny"},
			{Name: "source_action", Value: "allow"},
			{Name: "flow_impacted", Value: true},
		}))
		Expect(before[2].DocCount).To(BeEquivalentTo(2))
		Expect(before[2].CompositeAggregationKey).To(Equal(elastic.CompositeAggregationKey{
			{Name: "source_type", Value: "hep"},
			{Name: "source_namespace", Value: "-"},
			{Name: "source_name", Value: "hep1"},
			{Name: "dest_type", Value: "hep"},
			{Name: "dest_namespace", Value: "-"},
			{Name: "dest_name", Value: "hep2"},
			{Name: "reporter", Value: "src"},
			{Name: "action", Value: "allow"},
			{Name: "source_action", Value: "allow"},
			{Name: "flow_impacted", Value: true},
		}))
		Expect(before[3].DocCount).To(BeEquivalentTo(1))
		Expect(before[3].CompositeAggregationKey).To(Equal(elastic.CompositeAggregationKey{
			{Name: "source_type", Value: "hep"},
			{Name: "source_namespace", Value: "-"},
			{Name: "source_name", Value: "hep1"},
			{Name: "dest_type", Value: "hep"},
			{Name: "dest_namespace", Value: "-"},
			{Name: "dest_name", Value: "hep2"},
			{Name: "reporter", Value: "src"},
			{Name: "action", Value: "deny"},
			{Name: "source_action", Value: "deny"},
			{Name: "flow_impacted", Value: true},
		}))
		Expect(before[4].DocCount).To(BeEquivalentTo(1))
		Expect(before[4].CompositeAggregationKey).To(Equal(elastic.CompositeAggregationKey{
			{Name: "source_type", Value: "wep"},
			{Name: "source_namespace", Value: "ns1"},
			{Name: "source_name", Value: "hep1"},
			{Name: "dest_type", Value: "hep"},
			{Name: "dest_namespace", Value: "-"},
			{Name: "dest_name", Value: "hep2"},
			{Name: "reporter", Value: "src"},
			{Name: "action", Value: "allow"},
			{Name: "source_action", Value: "allow"},
			{Name: "flow_impacted", Value: false},
		}))

		Expect(after).To(HaveLen(3))
		Expect(after[0].DocCount).To(BeEquivalentTo(3))
		Expect(after[0].CompositeAggregationKey).To(Equal(elastic.CompositeAggregationKey{
			{Name: "source_type", Value: "hep"},
			{Name: "source_namespace", Value: "-"},
			{Name: "source_name", Value: "hep1"},
			{Name: "dest_type", Value: "hep"},
			{Name: "dest_namespace", Value: "-"},
			{Name: "dest_name", Value: "hep2"},
			{Name: "reporter", Value: "dst"},
			{Name: "action", Value: "allow"},
			{Name: "source_action", Value: "allow"},
			{Name: "flow_impacted", Value: true},
		}))
		Expect(after[1].DocCount).To(BeEquivalentTo(3))
		Expect(after[1].CompositeAggregationKey).To(Equal(elastic.CompositeAggregationKey{
			{Name: "source_type", Value: "hep"},
			{Name: "source_namespace", Value: "-"},
			{Name: "source_name", Value: "hep1"},
			{Name: "dest_type", Value: "hep"},
			{Name: "dest_namespace", Value: "-"},
			{Name: "dest_name", Value: "hep2"},
			{Name: "reporter", Value: "src"},
			{Name: "action", Value: "allow"},
			{Name: "source_action", Value: "allow"},
			{Name: "flow_impacted", Value: true},
		}))
		Expect(after[2].DocCount).To(BeEquivalentTo(1))
		Expect(after[2].CompositeAggregationKey).To(Equal(elastic.CompositeAggregationKey{
			{Name: "source_type", Value: "wep"},
			{Name: "source_namespace", Value: "ns1"},
			{Name: "source_name", Value: "hep1"},
			{Name: "dest_type", Value: "hep"},
			{Name: "dest_namespace", Value: "-"},
			{Name: "dest_name", Value: "hep2"},
			{Name: "reporter", Value: "src"},
			{Name: "action", Value: "allow"},
			{Name: "source_action", Value: "allow"},
			{Name: "flow_impacted", Value: false},
		}))
	})

	It("handles source flows changing from allow to deny", func() {
		By("Creating an ES client with a mocked out ES results being a mixture of allow and deny")
		client := elastic.NewMockSearchClient([]interface{}{
			flow("dst", "deny", "udp", hepd("hep1", 100), hepd("hep2", 200)),
			flow("src", "allow", "udp", hepd("hep1", 100), hepd("hep2", 200)),
			flow("src", "deny", "tcp", hepd("hep1", 100), hepd("hep2", 200)),

			flow("dst", "allow", "tcp", hepd("hep1", 500), hepd("hep2", 600)),
			flow("src", "allow", "tcp", hepd("hep1", 500), hepd("hep2", 600)),

			flow("src", "allow", "tcp", wepd("hep1", "ns1", 100), hepd("hep2", 200)),
		})

		By("Creating a composite agg query")
		q := &elastic.CompositeAggregationQuery{
			Name:                    api.FlowlogBuckets,
			AggCompositeSourceInfos: PIPCompositeSources,
			AggNestedTermInfos:      elastic.FlowAggregatedTerms,
			AggSumInfos:             elastic.FlowAggregationSums,
			MaxBucketsPerQuery:      1, // Set this to ensure we iterate after only a single response.
		}

		By("Creating a PIP instance with the mock client, and enumerating all aggregated flows (always deny after)")
		pip := pip{esClient: client, cfg: config.MustLoadConfig()}
		flowsChan, _ := pip.SearchAndProcessFlowLogs(context.Background(), q, nil, alwaysDenyCalculator{}, 1000, false, elastic.NewFlowFilterIncludeAll())
		var before []*elastic.CompositeAggregationBucket
		var after []*elastic.CompositeAggregationBucket
		for flow := range flowsChan {
			before = append(before, flow.Before...)
			after = append(after, flow.After...)
		}

		Expect(before).To(HaveLen(5))
		Expect(before[0].DocCount).To(BeEquivalentTo(1))
		Expect(before[0].CompositeAggregationKey).To(Equal(elastic.CompositeAggregationKey{
			{Name: "source_type", Value: "hep"},
			{Name: "source_namespace", Value: "-"},
			{Name: "source_name", Value: "hep1"},
			{Name: "dest_type", Value: "hep"},
			{Name: "dest_namespace", Value: "-"},
			{Name: "dest_name", Value: "hep2"},
			{Name: "reporter", Value: "dst"},
			{Name: "action", Value: "allow"},
			{Name: "source_action", Value: "allow"},
			{Name: "flow_impacted", Value: true},
		}))
		Expect(before[1].DocCount).To(BeEquivalentTo(1))
		Expect(before[1].CompositeAggregationKey).To(Equal(elastic.CompositeAggregationKey{
			{Name: "source_type", Value: "hep"},
			{Name: "source_namespace", Value: "-"},
			{Name: "source_name", Value: "hep1"},
			{Name: "dest_type", Value: "hep"},
			{Name: "dest_namespace", Value: "-"},
			{Name: "dest_name", Value: "hep2"},
			{Name: "reporter", Value: "dst"},
			{Name: "action", Value: "deny"},
			{Name: "source_action", Value: "allow"},
			{Name: "flow_impacted", Value: true},
		}))
		Expect(before[2].DocCount).To(BeEquivalentTo(2))
		Expect(before[2].CompositeAggregationKey).To(Equal(elastic.CompositeAggregationKey{
			{Name: "source_type", Value: "hep"},
			{Name: "source_namespace", Value: "-"},
			{Name: "source_name", Value: "hep1"},
			{Name: "dest_type", Value: "hep"},
			{Name: "dest_namespace", Value: "-"},
			{Name: "dest_name", Value: "hep2"},
			{Name: "reporter", Value: "src"},
			{Name: "action", Value: "allow"},
			{Name: "source_action", Value: "allow"},
			{Name: "flow_impacted", Value: true},
		}))
		Expect(before[3].DocCount).To(BeEquivalentTo(1))
		Expect(before[3].CompositeAggregationKey).To(Equal(elastic.CompositeAggregationKey{
			{Name: "source_type", Value: "hep"},
			{Name: "source_namespace", Value: "-"},
			{Name: "source_name", Value: "hep1"},
			{Name: "dest_type", Value: "hep"},
			{Name: "dest_namespace", Value: "-"},
			{Name: "dest_name", Value: "hep2"},
			{Name: "reporter", Value: "src"},
			{Name: "action", Value: "deny"},
			{Name: "source_action", Value: "deny"},
			{Name: "flow_impacted", Value: true},
		}))
		Expect(before[4].DocCount).To(BeEquivalentTo(1))
		Expect(before[4].CompositeAggregationKey).To(Equal(elastic.CompositeAggregationKey{
			{Name: "source_type", Value: "wep"},
			{Name: "source_namespace", Value: "ns1"},
			{Name: "source_name", Value: "hep1"},
			{Name: "dest_type", Value: "hep"},
			{Name: "dest_namespace", Value: "-"},
			{Name: "dest_name", Value: "hep2"},
			{Name: "reporter", Value: "src"},
			{Name: "action", Value: "allow"},
			{Name: "source_action", Value: "allow"},
			{Name: "flow_impacted", Value: true},
		}))

		Expect(after).To(HaveLen(2))
		Expect(after[0].DocCount).To(BeEquivalentTo(3))
		Expect(after[0].CompositeAggregationKey).To(Equal(elastic.CompositeAggregationKey{
			{Name: "source_type", Value: "hep"},
			{Name: "source_namespace", Value: "-"},
			{Name: "source_name", Value: "hep1"},
			{Name: "dest_type", Value: "hep"},
			{Name: "dest_namespace", Value: "-"},
			{Name: "dest_name", Value: "hep2"},
			{Name: "reporter", Value: "src"},
			{Name: "action", Value: "deny"},
			{Name: "source_action", Value: "deny"},
			{Name: "flow_impacted", Value: true},
		}))
		Expect(after[1].DocCount).To(BeEquivalentTo(1))
		Expect(after[1].CompositeAggregationKey).To(Equal(elastic.CompositeAggregationKey{
			{Name: "source_type", Value: "wep"},
			{Name: "source_namespace", Value: "ns1"},
			{Name: "source_name", Value: "hep1"},
			{Name: "dest_type", Value: "hep"},
			{Name: "dest_namespace", Value: "-"},
			{Name: "dest_name", Value: "hep2"},
			{Name: "reporter", Value: "src"},
			{Name: "action", Value: "deny"},
			{Name: "source_action", Value: "deny"},
			{Name: "flow_impacted", Value: true},
		}))
	})

	It("Should return only impacted flows when impactedOnly parameter is set to true", func() {
		By("Creating an ES client with a mocked out ES results being a mixture of allow and deny")
		client := elastic.NewMockSearchClient([]interface{}{
			// Dest api.
			// flow("dst", "allow", "tcp", hepd("hep1", 100), hepd("hep2", 200)), <- this flow is now deny at source,
			//                                                                       but will reappear in "after flows"
			flow("dst", "deny", "udp", hepd("hep1", 100), hepd("hep2", 200)),  //                // + AggregatedProtoPorts after
			flow("dst", "allow", "tcp", hepd("hep1", 500), hepd("hep2", 600)), //                // +
			// Source api.
			flow("src", "deny", "tcp", hepd("hep1", 100), hepd("hep2", 200)),  //                // + AggregatedProtoPorts after
			flow("src", "allow", "udp", hepd("hep1", 100), hepd("hep2", 200)), // + AggregatedProtoPorts   // |
			flow("src", "allow", "tcp", hepd("hep1", 500), hepd("hep2", 600)), // + before       // +
			// WEP
			flow("src", "allow", "tcp", wepd("hep1", "ns1", 100), hepd("hep2", 200)), // Missing dest flow
		})

		By("Creating a composite agg query")
		q := &elastic.CompositeAggregationQuery{
			Name:                    api.FlowlogBuckets,
			AggCompositeSourceInfos: PIPCompositeSources,
			AggNestedTermInfos:      elastic.FlowAggregatedTerms,
			AggSumInfos:             elastic.FlowAggregationSums,
			MaxBucketsPerQuery:      1, // Set this to ensure we iterate after only a single response.
		}

		By("Creating a PIP instance with the mock client, and enumerating all aggregated flows (always allow after)")
		pip := pip{esClient: client, cfg: config.MustLoadConfig()}
		flowsChan, _ := pip.SearchAndProcessFlowLogs(context.Background(), q, nil, alwaysAllowCalculator{}, 1000, true, elastic.NewFlowFilterIncludeAll())
		var before []*elastic.CompositeAggregationBucket
		for flow := range flowsChan {
			before = append(before, flow.Before...)
		}

		By("Checking the length of the response, if impactedOnly was set to false before would contain 5 results")
		Expect(before).To(HaveLen(4))
		Expect(before[0].DocCount).To(BeEquivalentTo(1))
		Expect(before[0].CompositeAggregationKey).To(Equal(elastic.CompositeAggregationKey{
			{Name: "source_type", Value: "hep"},
			{Name: "source_namespace", Value: "-"},
			{Name: "source_name", Value: "hep1"},
			{Name: "dest_type", Value: "hep"},
			{Name: "dest_namespace", Value: "-"},
			{Name: "dest_name", Value: "hep2"},
			{Name: "reporter", Value: "dst"},
			{Name: "action", Value: "allow"},
			{Name: "source_action", Value: "allow"},
			{Name: "flow_impacted", Value: true},
		}))
		Expect(before[1].DocCount).To(BeEquivalentTo(1))
		Expect(before[1].CompositeAggregationKey).To(Equal(elastic.CompositeAggregationKey{
			{Name: "source_type", Value: "hep"},
			{Name: "source_namespace", Value: "-"},
			{Name: "source_name", Value: "hep1"},
			{Name: "dest_type", Value: "hep"},
			{Name: "dest_namespace", Value: "-"},
			{Name: "dest_name", Value: "hep2"},
			{Name: "reporter", Value: "dst"},
			{Name: "action", Value: "deny"},
			{Name: "source_action", Value: "allow"},
			{Name: "flow_impacted", Value: true},
		}))
		Expect(before[2].DocCount).To(BeEquivalentTo(2))
		Expect(before[2].CompositeAggregationKey).To(Equal(elastic.CompositeAggregationKey{
			{Name: "source_type", Value: "hep"},
			{Name: "source_namespace", Value: "-"},
			{Name: "source_name", Value: "hep1"},
			{Name: "dest_type", Value: "hep"},
			{Name: "dest_namespace", Value: "-"},
			{Name: "dest_name", Value: "hep2"},
			{Name: "reporter", Value: "src"},
			{Name: "action", Value: "allow"},
			{Name: "source_action", Value: "allow"},
			{Name: "flow_impacted", Value: true},
		}))
		Expect(before[3].DocCount).To(BeEquivalentTo(1))
		Expect(before[3].CompositeAggregationKey).To(Equal(elastic.CompositeAggregationKey{
			{Name: "source_type", Value: "hep"},
			{Name: "source_namespace", Value: "-"},
			{Name: "source_name", Value: "hep1"},
			{Name: "dest_type", Value: "hep"},
			{Name: "dest_namespace", Value: "-"},
			{Name: "dest_name", Value: "hep2"},
			{Name: "reporter", Value: "src"},
			{Name: "action", Value: "deny"},
			{Name: "source_action", Value: "deny"},
			{Name: "flow_impacted", Value: true},
		}))
	})
})

func mustCreatePolicyHit(policyStr string, count int) api.PolicyHit {
	policyHit, err := api.PolicyHitFromFlowLogPolicyString(policyStr, int64(count))
	Expect(err).ShouldNot(HaveOccurred())

	return policyHit
}
