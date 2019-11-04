// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package policyrec_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/libcalico-go/lib/numorstring"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/tigera/compliance/pkg/policyrec"
	"github.com/tigera/lma/pkg/api"
	pelastic "github.com/tigera/lma/pkg/elastic"
	lpolicyrec "github.com/tigera/lma/pkg/policyrec"
)

// Given a source reported flow from deployment app1 to endpoint nginx on port 80,
// the engine should return a policy selecting app1 to nginx, to port 80.
var (
	destPort       = uint16(80)
	destPortInRule = numorstring.SinglePort(destPort)

	proto       = uint8(6)
	protoInRule = numorstring.ProtocolFromInt(proto)

	app1Dep = &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind: "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app1",
			Namespace: "namespace1",
			Labels: map[string]string{
				"app": "app1",
			},
		},
	}
	app1Rs = &appsv1.ReplicaSet{
		TypeMeta: metav1.TypeMeta{
			Kind: "ReplicaSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app1-abcdef",
			Namespace: "namespace1",
			Labels: map[string]string{
				"app": "app1",
			},
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					Kind: "Deployment",
					Name: "app1",
				},
			},
		},
	}

	nginxDep = &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind: "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx",
			Namespace: "namespace1",
			Labels: map[string]string{
				"app": "nginx",
			},
		},
	}
	nginxRs = &appsv1.ReplicaSet{
		TypeMeta: metav1.TypeMeta{
			Kind: "ReplicaSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-12345",
			Namespace: "namespace1",
			Labels: map[string]string{
				"app": "app1",
			},
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					Kind: "Deployment",
					Name: "nginx",
				},
			},
		},
	}

	app1Query = &lpolicyrec.PolicyRecommendationParams{
		StartTime:    "now-1h",
		EndTime:      "now",
		EndpointName: "app1-abcdef-*",
		Namespace:    "namespace1",
	}

	nginxQuery = &lpolicyrec.PolicyRecommendationParams{
		StartTime:    "now-1h",
		EndTime:      "now",
		EndpointName: "nginx-12345-*",
		Namespace:    "namespace1",
	}

	app1Policy = &v3.StagedNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       v3.KindStagedNetworkPolicy,
			APIVersion: v3.GroupVersionCurrent,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default.app1",
			Namespace: "namespace1",
		},
		Spec: v3.StagedNetworkPolicySpec{
			StagedAction: v3.StagedActionSet,
			Tier:         "default",
			Types:        []v3.PolicyType{v3.PolicyTypeEgress},
			Selector:     "app == 'app1'",
			Egress: []v3.Rule{
				v3.Rule{
					Action:   v3.Allow,
					Protocol: &protoInRule,
					Destination: v3.EntityRule{
						Selector: "app == 'nginx'",
						Ports:    []numorstring.Port{destPortInRule},
					},
				},
			},
			// Need to add this to assert equality of recommended policy.
			Ingress: []v3.Rule{},
		},
	}

	nginxPolicy = &v3.StagedNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       v3.KindStagedNetworkPolicy,
			APIVersion: v3.GroupVersionCurrent,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default.nginx",
			Namespace: "namespace1",
		},
		Spec: v3.StagedNetworkPolicySpec{
			StagedAction: v3.StagedActionSet,
			Tier:         "default",
			Types:        []v3.PolicyType{v3.PolicyTypeIngress},
			Selector:     "app == 'nginx'",
			// Need to add this to assert equality of recommended policy.
			Egress: []v3.Rule{},
			Ingress: []v3.Rule{
				v3.Rule{
					Action:   v3.Allow,
					Protocol: &protoInRule,
					Source: v3.EntityRule{
						Selector: "app == 'app1'",
					},
					Destination: v3.EntityRule{
						Ports: []numorstring.Port{destPortInRule},
					},
				},
			},
		},
	}

	app1ToNginxFlows = []*pelastic.CompositeAggregationBucket{
		&pelastic.CompositeAggregationBucket{
			CompositeAggregationKey: []pelastic.CompositeAggregationSourceValue{
				pelastic.CompositeAggregationSourceValue{Name: "source_type", Value: api.FlowLogEndpointTypeWEP},
				pelastic.CompositeAggregationSourceValue{Name: "source_namespace", Value: "namespace1"},
				pelastic.CompositeAggregationSourceValue{Name: "source_name_aggr", Value: "app1-abcdef-*"},
				pelastic.CompositeAggregationSourceValue{Name: "dest_type", Value: api.FlowLogEndpointTypeWEP},
				pelastic.CompositeAggregationSourceValue{Name: "dest_namespace", Value: "namespace1"},
				pelastic.CompositeAggregationSourceValue{Name: "dest_name_aggr", Value: "nginx-12345-*"},
				pelastic.CompositeAggregationSourceValue{Name: "proto", Value: "6"},
				pelastic.CompositeAggregationSourceValue{Name: "dest_ip", Value: ""},
				pelastic.CompositeAggregationSourceValue{Name: "source_ip", Value: ""},
				pelastic.CompositeAggregationSourceValue{Name: "source_port", Value: ""},
				pelastic.CompositeAggregationSourceValue{Name: "dest_port", Value: 80.0},
				pelastic.CompositeAggregationSourceValue{Name: "reporter", Value: "src"},
				pelastic.CompositeAggregationSourceValue{Name: "action", Value: "allow"},
			},
			AggregatedTerms: map[string]*pelastic.AggregatedTerm{
				"source_labels": &pelastic.AggregatedTerm{
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"app=app1": 1,
					},
				},
				"dest_labels": &pelastic.AggregatedTerm{
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"app=nginx": 1,
					},
				},
				"policies": &pelastic.AggregatedTerm{
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"0|__PROFILE__|__PROFILE__.kns.namespace1|allow": 1,
					},
				},
			},
		},
		&pelastic.CompositeAggregationBucket{
			CompositeAggregationKey: []pelastic.CompositeAggregationSourceValue{
				pelastic.CompositeAggregationSourceValue{Name: "source_type", Value: api.FlowLogEndpointTypeWEP},
				pelastic.CompositeAggregationSourceValue{Name: "source_namespace", Value: "namespace1"},
				pelastic.CompositeAggregationSourceValue{Name: "source_name_aggr", Value: "app1-abcdef-*"},
				pelastic.CompositeAggregationSourceValue{Name: "dest_type", Value: api.FlowLogEndpointTypeWEP},
				pelastic.CompositeAggregationSourceValue{Name: "dest_namespace", Value: "namespace1"},
				pelastic.CompositeAggregationSourceValue{Name: "dest_name_aggr", Value: "nginx-12345-*"},
				pelastic.CompositeAggregationSourceValue{Name: "proto", Value: "6"},
				pelastic.CompositeAggregationSourceValue{Name: "dest_ip", Value: ""},
				pelastic.CompositeAggregationSourceValue{Name: "source_ip", Value: ""},
				pelastic.CompositeAggregationSourceValue{Name: "source_port", Value: ""},
				pelastic.CompositeAggregationSourceValue{Name: "dest_port", Value: 80.0},
				pelastic.CompositeAggregationSourceValue{Name: "reporter", Value: "dst"},
				pelastic.CompositeAggregationSourceValue{Name: "action", Value: "allow"},
			},
			AggregatedTerms: map[string]*pelastic.AggregatedTerm{
				"source_labels": &pelastic.AggregatedTerm{
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"app=app1": 1,
					},
				},
				"dest_labels": &pelastic.AggregatedTerm{
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"app=nginx": 1,
					},
				},
				"policies": &pelastic.AggregatedTerm{
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"0|__PROFILE__|__PROFILE__.kns.namespace1|allow": 1,
					},
				},
			},
		},
	}
)

var _ = Describe("Policy Recommendation", func() {
	var (
		fakeKube k8s.Interface
		ec       *fakeAggregator
	)
	BeforeEach(func() {
		fakeKube = fake.NewSimpleClientset(app1Dep, app1Rs, nginxDep, nginxRs)
		ec = newFakeAggregator()
	})
	DescribeTable("Recommend policies for matching flows and endpoint",
		func(queryResults []*pelastic.CompositeAggregationBucket, queryError error,
			query *lpolicyrec.PolicyRecommendationParams,
			expectedResponse *policyrec.PolicyRecommendationResponse, expectedError error) {

			By("Initializing the engine") // Tempted to say "Start your engines!"
			pre := policyrec.NewPolicyRecommendationEngine(fakeKube, ec)

			By("setting up next results")
			ec.setNextResults(queryResults)
			ec.setNextError(queryError)

			By("getting recommendations")
			response, err := pre.GetRecommendation(context.Background(), query)

			By("the response and error values")
			if expectedError == nil {
				Expect(err).To(BeNil())
			} else {
				Expect(err).ToNot(BeNil())
				Expect(err).To(Equal(expectedError))
			}
			if expectedResponse == nil {
				Expect(expectedResponse).To(BeNil())
			} else {
				Expect(expectedResponse).ToNot(BeNil())
				Expect(response).To(Equal(expectedResponse))
			}
		},
		Entry("for source endpoint", app1ToNginxFlows, nil,
			app1Query,
			&policyrec.PolicyRecommendationResponse{
				Recommendation: &lpolicyrec.Recommendation{
					NetworkPolicies: []*v3.StagedNetworkPolicy{
						app1Policy,
					},
					GlobalNetworkPolicies: []*v3.StagedGlobalNetworkPolicy{},
				},
				ErrorMessage: "",
			}, nil),
		Entry("for destination endpoint", app1ToNginxFlows, nil,
			nginxQuery,
			&policyrec.PolicyRecommendationResponse{
				Recommendation: &lpolicyrec.Recommendation{
					NetworkPolicies: []*v3.StagedNetworkPolicy{
						nginxPolicy,
					},
					GlobalNetworkPolicies: []*v3.StagedGlobalNetworkPolicy{},
				},
				ErrorMessage: "",
			}, nil),
		Entry("for unknown endpoint", nil, fmt.Errorf("No results"),
			&lpolicyrec.PolicyRecommendationParams{
				StartTime:    "now-1h",
				EndTime:      "now",
				EndpointName: "idontexist-*",
				Namespace:    "default",
			}, nil, fmt.Errorf("No results")),
	)

})

// fakeAggregator is a test utility that implements the CompositeAggregator interface
type fakeAggregator struct {
	nextResults []*pelastic.CompositeAggregationBucket
	nextError   error
}

func newFakeAggregator() *fakeAggregator {
	return &fakeAggregator{}
}

func (fa *fakeAggregator) SearchCompositeAggregations(
	context.Context, *pelastic.CompositeAggregationQuery, pelastic.CompositeAggregationKey,
) (<-chan *pelastic.CompositeAggregationBucket, <-chan error) {
	dataChan := make(chan *pelastic.CompositeAggregationBucket, len(fa.nextResults))
	errorChan := make(chan error, 1)
	go func() {
		defer func() {
			close(dataChan)
			close(errorChan)
		}()
		if fa.nextError != nil {
			errorChan <- fa.nextError
			return
		}
		for _, result := range fa.nextResults {
			dataChan <- result
		}
	}()
	return dataChan, errorChan
}

func (fa *fakeAggregator) setNextResults(cab []*pelastic.CompositeAggregationBucket) {
	fa.nextResults = cab
}

func (fa *fakeAggregator) setNextError(err error) {
	fa.nextError = err
}
