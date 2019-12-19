package pip

import (
	"bytes"
	"context"
	"text/template"

	"github.com/tigera/es-proxy/pkg/pip/config"
	"github.com/tigera/es-proxy/pkg/pip/policycalc"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	pelastic "github.com/tigera/lma/pkg/elastic"
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

// netd creates an epData for a network endpoint.
func netd(name string, port int) epData {
	return epData{
		Type:     "net",
		NameAggr: name,
		Port:     port,
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
                {
                  "key": "0|allow-cnx|calico-monitoring/allow-cnx.elasticsearch-access|allow",
                  "doc_count": 1
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
)

// flow generates an ES flow response used by the mock ES client.
func flow(reporter, action, protocol string, source, dest epData) string {
	fd := flowData{
		Reporter:    reporter,
		Protocol:    protocol,
		Action:      action,
		Source:      source,
		Destination: dest,
	}
	var tpl bytes.Buffer
	err := flowTemplate.Execute(&tpl, fd)
	Expect(err).NotTo(HaveOccurred())
	return tpl.String()
}

// alwaysAllowCalculator implements the policy calculator interface with an always allow source and dest response for
// the after buckets.
type alwaysAllowCalculator struct{}

func (_ alwaysAllowCalculator) Calculate(flow *policycalc.Flow) (bool, *policycalc.Response, *policycalc.Response) {
	var beforeSourceAction policycalc.Action
	if flow.Reporter == policycalc.ReporterTypeDestination {
		beforeSourceAction = policycalc.ActionAllow
	} else {
		beforeSourceAction = flow.Action
	}
	before := &policycalc.Response{
		Source: policycalc.EndpointResponse{
			Include:  flow.Reporter == policycalc.ReporterTypeSource,
			Action:   beforeSourceAction,
			Policies: []string{"policy1", "policy2"},
		},
		Destination: policycalc.EndpointResponse{
			Include:  flow.Reporter == policycalc.ReporterTypeDestination,
			Action:   flow.Action,
			Policies: []string{"policy1", "policy2"},
		},
	}
	after := &policycalc.Response{
		Source: policycalc.EndpointResponse{
			Include:  flow.Reporter == policycalc.ReporterTypeSource,
			Action:   policycalc.ActionAllow,
			Policies: []string{"policy1", "policy2"},
		},
		Destination: policycalc.EndpointResponse{
			// Add a destination flow if the original src flow was Deny and now we allow.
			Include:  flow.Reporter == policycalc.ReporterTypeDestination || flow.Action == policycalc.ActionDeny,
			Action:   policycalc.ActionAllow,
			Policies: []string{"policy1", "policy2"},
		},
	}
	return true, before, after
}

// alwaysDenyCalculator implements the policy calculator interface with an always deny source and dest response for the
// after buckets.
type alwaysDenyCalculator struct{}

func (_ alwaysDenyCalculator) Calculate(flow *policycalc.Flow) (bool, *policycalc.Response, *policycalc.Response) {
	var beforeSourceAction policycalc.Action
	if flow.Reporter == policycalc.ReporterTypeDestination {
		beforeSourceAction = policycalc.ActionAllow
	} else {
		beforeSourceAction = flow.Action
	}
	before := &policycalc.Response{
		Source: policycalc.EndpointResponse{
			Include:  flow.Reporter == policycalc.ReporterTypeSource,
			Action:   beforeSourceAction,
			Policies: []string{"policy1", "policy2"},
		},
		Destination: policycalc.EndpointResponse{
			Include:  flow.Reporter == policycalc.ReporterTypeDestination,
			Action:   flow.Action,
			Policies: []string{"policy1", "policy2"},
		},
	}
	after := &policycalc.Response{
		Source: policycalc.EndpointResponse{
			Include:  flow.Reporter == policycalc.ReporterTypeSource,
			Action:   policycalc.ActionDeny,
			Policies: []string{"policy1", "policy2"},
		},
		Destination: policycalc.EndpointResponse{
			// Add a destination flow if the original src flow was Deny and now we allow.
			Include: false,
		},
	}
	return true, before, after
}

var _ = Describe("Test handling of aggregated ES response", func() {
	It("handles simple aggregation of results where action does not change", func() {
		By("Creating an ES client with a mocked out search results with all allow actions")
		client := pelastic.NewMockSearchClient([]interface{}{
			// Dest flows.
			flow("dst", "allow", "tcp", hepd("hep1", 100), hepd("hep2", 200)), // + Aggregate before and after
			flow("dst", "allow", "udp", hepd("hep1", 100), hepd("hep2", 200)), // |
			flow("dst", "allow", "tcp", hepd("hep1", 500), hepd("hep2", 600)), // +
			// Source flows.
			flow("src", "allow", "tcp", hepd("hep1", 100), hepd("hep2", 200)), // + Aggregate before and after
			flow("src", "allow", "udp", hepd("hep1", 100), hepd("hep2", 200)), // |
			flow("src", "allow", "tcp", hepd("hep1", 500), hepd("hep2", 600)), // +
			// WEP
			flow("src", "allow", "tcp", wepd("hep1", "ns1", 100), hepd("hep2", 200)), // Missing dest flow
		})

		By("Creating a composite agg query")
		q := &pelastic.CompositeAggregationQuery{
			Name: FlowlogBuckets,
			AggCompositeSourceInfos: PIPCompositeSources,
			AggNestedTermInfos:      AggregatedTerms,
			AggSumInfos:             UIAggregationSums,
		}

		By("Creating a PIP instance with the mock client, and enumerating all aggregated flows (always allow after)")
		pip := pip{esClient: client, cfg: config.MustLoadConfig()}
		flowsChan, _ := pip.SearchAndProcessFlowLogs(context.Background(), q, nil, alwaysAllowCalculator{}, 1000, false)
		var before []*pelastic.CompositeAggregationBucket
		var after []*pelastic.CompositeAggregationBucket
		for flow := range flowsChan {
			before = append(before, flow.Before...)
			after = append(after, flow.After...)
		}

		Expect(before).To(HaveLen(3))
		Expect(before[0].DocCount).To(BeEquivalentTo(3))
		Expect(before[0].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{"source_type", "hep"},
			{"source_namespace", "-"},
			{"source_name", "hep1"},
			{"dest_type", "hep"},
			{"dest_namespace", "-"},
			{"dest_name", "hep2"},
			{"reporter", "dst"},
			{"action", "allow"},
			{"source_action", "allow"},
			{"flow_impacted", false},
		}))
		Expect(before[1].DocCount).To(BeEquivalentTo(3))
		Expect(before[1].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{"source_type", "hep"},
			{"source_namespace", "-"},
			{"source_name", "hep1"},
			{"dest_type", "hep"},
			{"dest_namespace", "-"},
			{"dest_name", "hep2"},
			{"reporter", "src"},
			{"action", "allow"},
			{"source_action", "allow"},
			{"flow_impacted", false},
		}))
		Expect(before[2].DocCount).To(BeEquivalentTo(1))
		Expect(before[2].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{"source_type", "wep"},
			{"source_namespace", "ns1"},
			{"source_name", "hep1"},
			{"dest_type", "hep"},
			{"dest_namespace", "-"},
			{"dest_name", "hep2"},
			{"reporter", "src"},
			{"action", "allow"},
			{"source_action", "allow"},
			{"flow_impacted", false},
		}))

		Expect(after).To(HaveLen(3))
		Expect(after[0].DocCount).To(BeEquivalentTo(3))
		Expect(after[0].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{"source_type", "hep"},
			{"source_namespace", "-"},
			{"source_name", "hep1"},
			{"dest_type", "hep"},
			{"dest_namespace", "-"},
			{"dest_name", "hep2"},
			{"reporter", "dst"},
			{"action", "allow"},
			{"source_action", "allow"},
			{"flow_impacted", false},
		}))
		Expect(after[1].DocCount).To(BeEquivalentTo(3))
		Expect(after[1].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{"source_type", "hep"},
			{"source_namespace", "-"},
			{"source_name", "hep1"},
			{"dest_type", "hep"},
			{"dest_namespace", "-"},
			{"dest_name", "hep2"},
			{"reporter", "src"},
			{"action", "allow"},
			{"source_action", "allow"},
			{"flow_impacted", false},
		}))
		Expect(after[2].DocCount).To(BeEquivalentTo(1))
		Expect(after[2].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{"source_type", "wep"},
			{"source_namespace", "ns1"},
			{"source_name", "hep1"},
			{"dest_type", "hep"},
			{"dest_namespace", "-"},
			{"dest_name", "hep2"},
			{"reporter", "src"},
			{"action", "allow"},
			{"source_action", "allow"},
			{"flow_impacted", false},
		}))
	})

	It("handles source flows changing from deny to allow", func() {
		By("Creating an ES client with a mocked out ES results being a mixture of allow and deny")
		client := pelastic.NewMockSearchClient([]interface{}{
			// Dest flows.
			// flow("dst", "allow", "tcp", hepd("hep1", 100), hepd("hep2", 200)), <- this flow is now deny at source,
			//                                                                       but will reappear in "after flows"
			flow("dst", "deny", "udp", hepd("hep1", 100), hepd("hep2", 200)),  //                // + Aggregated after
			flow("dst", "allow", "tcp", hepd("hep1", 500), hepd("hep2", 600)), //                // +
			// Source flows.
			flow("src", "deny", "tcp", hepd("hep1", 100), hepd("hep2", 200)),  //                // + Aggregated after
			flow("src", "allow", "udp", hepd("hep1", 100), hepd("hep2", 200)), // + Aggregated   // |
			flow("src", "allow", "tcp", hepd("hep1", 500), hepd("hep2", 600)), // + before       // +
			// WEP
			flow("src", "allow", "tcp", wepd("hep1", "ns1", 100), hepd("hep2", 200)), // Missing dest flow
		})

		By("Creating a composite agg query")
		q := &pelastic.CompositeAggregationQuery{
			Name: FlowlogBuckets,
			AggCompositeSourceInfos: PIPCompositeSources,
			AggNestedTermInfos:      AggregatedTerms,
			AggSumInfos:             UIAggregationSums,
		}

		By("Creating a PIP instance with the mock client, and enumerating all aggregated flows (always allow after)")
		pip := pip{esClient: client, cfg: config.MustLoadConfig()}
		flowsChan, _ := pip.SearchAndProcessFlowLogs(context.Background(), q, nil, alwaysAllowCalculator{}, 1000, false)
		var before []*pelastic.CompositeAggregationBucket
		var after []*pelastic.CompositeAggregationBucket
		for flow := range flowsChan {
			before = append(before, flow.Before...)
			after = append(after, flow.After...)
		}

		Expect(before).To(HaveLen(5))
		Expect(before[0].DocCount).To(BeEquivalentTo(1))
		Expect(before[0].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{"source_type", "hep"},
			{"source_namespace", "-"},
			{"source_name", "hep1"},
			{"dest_type", "hep"},
			{"dest_namespace", "-"},
			{"dest_name", "hep2"},
			{"reporter", "dst"},
			{"action", "allow"},
			{"source_action", "allow"},
			{"flow_impacted", true},
		}))
		Expect(before[1].DocCount).To(BeEquivalentTo(1))
		Expect(before[1].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{"source_type", "hep"},
			{"source_namespace", "-"},
			{"source_name", "hep1"},
			{"dest_type", "hep"},
			{"dest_namespace", "-"},
			{"dest_name", "hep2"},
			{"reporter", "dst"},
			{"action", "deny"},
			{"source_action", "allow"},
			{"flow_impacted", true},
		}))
		Expect(before[2].DocCount).To(BeEquivalentTo(2))
		Expect(before[2].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{"source_type", "hep"},
			{"source_namespace", "-"},
			{"source_name", "hep1"},
			{"dest_type", "hep"},
			{"dest_namespace", "-"},
			{"dest_name", "hep2"},
			{"reporter", "src"},
			{"action", "allow"},
			{"source_action", "allow"},
			{"flow_impacted", true},
		}))
		Expect(before[3].DocCount).To(BeEquivalentTo(1))
		Expect(before[3].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{"source_type", "hep"},
			{"source_namespace", "-"},
			{"source_name", "hep1"},
			{"dest_type", "hep"},
			{"dest_namespace", "-"},
			{"dest_name", "hep2"},
			{"reporter", "src"},
			{"action", "deny"},
			{"source_action", "deny"},
			{"flow_impacted", true},
		}))
		Expect(before[4].DocCount).To(BeEquivalentTo(1))
		Expect(before[4].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{"source_type", "wep"},
			{"source_namespace", "ns1"},
			{"source_name", "hep1"},
			{"dest_type", "hep"},
			{"dest_namespace", "-"},
			{"dest_name", "hep2"},
			{"reporter", "src"},
			{"action", "allow"},
			{"source_action", "allow"},
			{"flow_impacted", false},
		}))

		Expect(after).To(HaveLen(3))
		Expect(after[0].DocCount).To(BeEquivalentTo(3))
		Expect(after[0].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{"source_type", "hep"},
			{"source_namespace", "-"},
			{"source_name", "hep1"},
			{"dest_type", "hep"},
			{"dest_namespace", "-"},
			{"dest_name", "hep2"},
			{"reporter", "dst"},
			{"action", "allow"},
			{"source_action", "allow"},
			{"flow_impacted", true},
		}))
		Expect(after[1].DocCount).To(BeEquivalentTo(3))
		Expect(after[1].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{"source_type", "hep"},
			{"source_namespace", "-"},
			{"source_name", "hep1"},
			{"dest_type", "hep"},
			{"dest_namespace", "-"},
			{"dest_name", "hep2"},
			{"reporter", "src"},
			{"action", "allow"},
			{"source_action", "allow"},
			{"flow_impacted", true},
		}))
		Expect(after[2].DocCount).To(BeEquivalentTo(1))
		Expect(after[2].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{"source_type", "wep"},
			{"source_namespace", "ns1"},
			{"source_name", "hep1"},
			{"dest_type", "hep"},
			{"dest_namespace", "-"},
			{"dest_name", "hep2"},
			{"reporter", "src"},
			{"action", "allow"},
			{"source_action", "allow"},
			{"flow_impacted", false},
		}))
	})

	It("handles source flows changing from allow to deny", func() {
		By("Creating an ES client with a mocked out ES results being a mixture of allow and deny")
		client := pelastic.NewMockSearchClient([]interface{}{
			// Dest flows.  All dest flows will be removed in after flows
			flow("dst", "deny", "udp", hepd("hep1", 100), hepd("hep2", 200)),  //                // + Aggregated after
			flow("dst", "allow", "tcp", hepd("hep1", 500), hepd("hep2", 600)), //                // +
			// Source flows.
			flow("src", "deny", "tcp", hepd("hep1", 100), hepd("hep2", 200)),  //                // + Aggregated after
			flow("src", "allow", "udp", hepd("hep1", 100), hepd("hep2", 200)), // + Aggregated   // |
			flow("src", "allow", "tcp", hepd("hep1", 500), hepd("hep2", 600)), // + before       // +
			// WEP
			flow("src", "allow", "tcp", wepd("hep1", "ns1", 100), hepd("hep2", 200)),
		})

		By("Creating a composite agg query")
		q := &pelastic.CompositeAggregationQuery{
			Name: FlowlogBuckets,
			AggCompositeSourceInfos: PIPCompositeSources,
			AggNestedTermInfos:      AggregatedTerms,
			AggSumInfos:             UIAggregationSums,
		}

		By("Creating a PIP instance with the mock client, and enumerating all aggregated flows (always deny after)")
		pip := pip{esClient: client, cfg: config.MustLoadConfig()}
		flowsChan, _ := pip.SearchAndProcessFlowLogs(context.Background(), q, nil, alwaysDenyCalculator{}, 1000, false)
		var before []*pelastic.CompositeAggregationBucket
		var after []*pelastic.CompositeAggregationBucket
		for flow := range flowsChan {
			before = append(before, flow.Before...)
			after = append(after, flow.After...)
		}

		Expect(before).To(HaveLen(5))
		Expect(before[0].DocCount).To(BeEquivalentTo(1))
		Expect(before[0].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{"source_type", "hep"},
			{"source_namespace", "-"},
			{"source_name", "hep1"},
			{"dest_type", "hep"},
			{"dest_namespace", "-"},
			{"dest_name", "hep2"},
			{"reporter", "dst"},
			{"action", "allow"},
			{"source_action", "allow"},
			{"flow_impacted", true},
		}))
		Expect(before[1].DocCount).To(BeEquivalentTo(1))
		Expect(before[1].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{"source_type", "hep"},
			{"source_namespace", "-"},
			{"source_name", "hep1"},
			{"dest_type", "hep"},
			{"dest_namespace", "-"},
			{"dest_name", "hep2"},
			{"reporter", "dst"},
			{"action", "deny"},
			{"source_action", "allow"},
			{"flow_impacted", true},
		}))
		Expect(before[2].DocCount).To(BeEquivalentTo(2))
		Expect(before[2].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{"source_type", "hep"},
			{"source_namespace", "-"},
			{"source_name", "hep1"},
			{"dest_type", "hep"},
			{"dest_namespace", "-"},
			{"dest_name", "hep2"},
			{"reporter", "src"},
			{"action", "allow"},
			{"source_action", "allow"},
			{"flow_impacted", true},
		}))
		Expect(before[3].DocCount).To(BeEquivalentTo(1))
		Expect(before[3].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{"source_type", "hep"},
			{"source_namespace", "-"},
			{"source_name", "hep1"},
			{"dest_type", "hep"},
			{"dest_namespace", "-"},
			{"dest_name", "hep2"},
			{"reporter", "src"},
			{"action", "deny"},
			{"source_action", "deny"},
			{"flow_impacted", true},
		}))
		Expect(before[4].DocCount).To(BeEquivalentTo(1))
		Expect(before[4].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{"source_type", "wep"},
			{"source_namespace", "ns1"},
			{"source_name", "hep1"},
			{"dest_type", "hep"},
			{"dest_namespace", "-"},
			{"dest_name", "hep2"},
			{"reporter", "src"},
			{"action", "allow"},
			{"source_action", "allow"},
			{"flow_impacted", true},
		}))

		Expect(after).To(HaveLen(2))
		Expect(after[0].DocCount).To(BeEquivalentTo(3))
		Expect(after[0].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{"source_type", "hep"},
			{"source_namespace", "-"},
			{"source_name", "hep1"},
			{"dest_type", "hep"},
			{"dest_namespace", "-"},
			{"dest_name", "hep2"},
			{"reporter", "src"},
			{"action", "deny"},
			{"source_action", "deny"},
			{"flow_impacted", true},
		}))
		Expect(after[1].DocCount).To(BeEquivalentTo(1))
		Expect(after[1].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{"source_type", "wep"},
			{"source_namespace", "ns1"},
			{"source_name", "hep1"},
			{"dest_type", "hep"},
			{"dest_namespace", "-"},
			{"dest_name", "hep2"},
			{"reporter", "src"},
			{"action", "deny"},
			{"source_action", "deny"},
			{"flow_impacted", true},
		}))
	})

	It("Should return only impacted flows when impactedOnly parameter is set to true", func() {
		By("Creating an ES client with a mocked out ES results being a mixture of allow and deny")
		client := pelastic.NewMockSearchClient([]interface{}{
			// Dest flows.
			// flow("dst", "allow", "tcp", hepd("hep1", 100), hepd("hep2", 200)), <- this flow is now deny at source,
			//                                                                       but will reappear in "after flows"
			flow("dst", "deny", "udp", hepd("hep1", 100), hepd("hep2", 200)),  //                // + Aggregated after
			flow("dst", "allow", "tcp", hepd("hep1", 500), hepd("hep2", 600)), //                // +
			// Source flows.
			flow("src", "deny", "tcp", hepd("hep1", 100), hepd("hep2", 200)),  //                // + Aggregated after
			flow("src", "allow", "udp", hepd("hep1", 100), hepd("hep2", 200)), // + Aggregated   // |
			flow("src", "allow", "tcp", hepd("hep1", 500), hepd("hep2", 600)), // + before       // +
			// WEP
			flow("src", "allow", "tcp", wepd("hep1", "ns1", 100), hepd("hep2", 200)), // Missing dest flow
		})

		By("Creating a composite agg query")
		q := &pelastic.CompositeAggregationQuery{
			Name: FlowlogBuckets,
			AggCompositeSourceInfos: PIPCompositeSources,
			AggNestedTermInfos:      AggregatedTerms,
			AggSumInfos:             UIAggregationSums,
		}

		By("Creating a PIP instance with the mock client, and enumerating all aggregated flows (always allow after)")
		pip := pip{esClient: client, cfg: config.MustLoadConfig()}
		flowsChan, _ := pip.SearchAndProcessFlowLogs(context.Background(), q, nil, alwaysAllowCalculator{}, 1000, true)
		var before []*pelastic.CompositeAggregationBucket
		var after []*pelastic.CompositeAggregationBucket
		for flow := range flowsChan {
			before = append(before, flow.Before...)
			after = append(after, flow.After...)
		}

		By("Checking the length of the response, if impactedOnly was set to false before would contain 5 results")
		Expect(before).To(HaveLen(4))
		Expect(before[0].DocCount).To(BeEquivalentTo(1))
		Expect(before[0].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{"source_type", "hep"},
			{"source_namespace", "-"},
			{"source_name", "hep1"},
			{"dest_type", "hep"},
			{"dest_namespace", "-"},
			{"dest_name", "hep2"},
			{"reporter", "dst"},
			{"action", "allow"},
			{"source_action", "allow"},
			{"flow_impacted", true},
		}))
		Expect(before[1].DocCount).To(BeEquivalentTo(1))
		Expect(before[1].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{"source_type", "hep"},
			{"source_namespace", "-"},
			{"source_name", "hep1"},
			{"dest_type", "hep"},
			{"dest_namespace", "-"},
			{"dest_name", "hep2"},
			{"reporter", "dst"},
			{"action", "deny"},
			{"source_action", "allow"},
			{"flow_impacted", true},
		}))
		Expect(before[2].DocCount).To(BeEquivalentTo(2))
		Expect(before[2].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{"source_type", "hep"},
			{"source_namespace", "-"},
			{"source_name", "hep1"},
			{"dest_type", "hep"},
			{"dest_namespace", "-"},
			{"dest_name", "hep2"},
			{"reporter", "src"},
			{"action", "allow"},
			{"source_action", "allow"},
			{"flow_impacted", true},
		}))
		Expect(before[3].DocCount).To(BeEquivalentTo(1))
		Expect(before[3].CompositeAggregationKey).To(Equal(pelastic.CompositeAggregationKey{
			{"source_type", "hep"},
			{"source_namespace", "-"},
			{"source_name", "hep1"},
			{"dest_type", "hep"},
			{"dest_namespace", "-"},
			{"dest_name", "hep2"},
			{"reporter", "src"},
			{"action", "deny"},
			{"source_action", "deny"},
			{"flow_impacted", true},
		}))
	})
})
