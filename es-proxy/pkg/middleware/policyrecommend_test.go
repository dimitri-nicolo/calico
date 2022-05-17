// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/stretchr/testify/mock"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/endpoints/request"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/projectcalico/calico/compliance/pkg/datastore"
	"github.com/projectcalico/calico/lma/pkg/api"
	lmaerror "github.com/projectcalico/calico/lma/pkg/api"
	lmaauth "github.com/projectcalico/calico/lma/pkg/auth"
	"github.com/projectcalico/calico/lma/pkg/elastic"
	"github.com/projectcalico/calico/lma/pkg/policyrec"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/api/pkg/lib/numorstring"
)

const recommendURLPath = "/recommend"

// Given a source reported flow from deployment app1 to endpoint nginx on port 80,
// the engine should return a policy selecting app1 to nginx, to port 80.
var (
	destPort       = uint16(80)
	destPortInRule = numorstring.SinglePort(destPort)

	protoInRule = numorstring.ProtocolFromString("TCP")

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

	app1Query = &policyrec.PolicyRecommendationParams{
		StartTime:    "now-1h",
		EndTime:      "now",
		EndpointName: "app1-abcdef-*",
		Namespace:    "namespace1",
	}

	nginxQuery = &policyrec.PolicyRecommendationParams{
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

	app1ToNginxFlows = []*elastic.CompositeAggregationBucket{
		&elastic.CompositeAggregationBucket{
			CompositeAggregationKey: []elastic.CompositeAggregationSourceValue{
				elastic.CompositeAggregationSourceValue{Name: "source_type", Value: api.FlowLogEndpointTypeWEP},
				elastic.CompositeAggregationSourceValue{Name: "source_namespace", Value: "namespace1"},
				elastic.CompositeAggregationSourceValue{Name: "source_name_aggr", Value: "app1-abcdef-*"},
				elastic.CompositeAggregationSourceValue{Name: "dest_type", Value: api.FlowLogEndpointTypeWEP},
				elastic.CompositeAggregationSourceValue{Name: "dest_namespace", Value: "namespace1"},
				elastic.CompositeAggregationSourceValue{Name: "dest_name_aggr", Value: "nginx-12345-*"},
				elastic.CompositeAggregationSourceValue{Name: "proto", Value: "6"},
				elastic.CompositeAggregationSourceValue{Name: "dest_ip", Value: ""},
				elastic.CompositeAggregationSourceValue{Name: "source_ip", Value: ""},
				elastic.CompositeAggregationSourceValue{Name: "source_port", Value: ""},
				elastic.CompositeAggregationSourceValue{Name: "dest_port", Value: 80.0},
				elastic.CompositeAggregationSourceValue{Name: "reporter", Value: "src"},
				elastic.CompositeAggregationSourceValue{Name: "action", Value: "allow"},
			},
			AggregatedTerms: map[string]*elastic.AggregatedTerm{
				"source_labels": &elastic.AggregatedTerm{
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"app=app1": 1,
					},
				},
				"dest_labels": &elastic.AggregatedTerm{
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"app=nginx": 1,
					},
				},
				"policies": &elastic.AggregatedTerm{
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"0|__PROFILE__|__PROFILE__.kns.namespace1|allow|0": 1,
					},
				},
			},
		},
		&elastic.CompositeAggregationBucket{
			CompositeAggregationKey: []elastic.CompositeAggregationSourceValue{
				elastic.CompositeAggregationSourceValue{Name: "source_type", Value: api.FlowLogEndpointTypeWEP},
				elastic.CompositeAggregationSourceValue{Name: "source_namespace", Value: "namespace1"},
				elastic.CompositeAggregationSourceValue{Name: "source_name_aggr", Value: "app1-abcdef-*"},
				elastic.CompositeAggregationSourceValue{Name: "dest_type", Value: api.FlowLogEndpointTypeWEP},
				elastic.CompositeAggregationSourceValue{Name: "dest_namespace", Value: "namespace1"},
				elastic.CompositeAggregationSourceValue{Name: "dest_name_aggr", Value: "nginx-12345-*"},
				elastic.CompositeAggregationSourceValue{Name: "proto", Value: "6"},
				elastic.CompositeAggregationSourceValue{Name: "dest_ip", Value: ""},
				elastic.CompositeAggregationSourceValue{Name: "source_ip", Value: ""},
				elastic.CompositeAggregationSourceValue{Name: "source_port", Value: ""},
				elastic.CompositeAggregationSourceValue{Name: "dest_port", Value: 80.0},
				elastic.CompositeAggregationSourceValue{Name: "reporter", Value: "dst"},
				elastic.CompositeAggregationSourceValue{Name: "action", Value: "allow"},
			},
			AggregatedTerms: map[string]*elastic.AggregatedTerm{
				"source_labels": &elastic.AggregatedTerm{
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"app=app1": 1,
					},
				},
				"dest_labels": &elastic.AggregatedTerm{
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"app=nginx": 1,
					},
				},
				"policies": &elastic.AggregatedTerm{
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"0|__PROFILE__|__PROFILE__.kns.namespace1|allow|0": 1,
					},
				},
			},
		},
	}

	app1ToNginxEgressFlows = []*elastic.CompositeAggregationBucket{
		&elastic.CompositeAggregationBucket{
			CompositeAggregationKey: []elastic.CompositeAggregationSourceValue{
				elastic.CompositeAggregationSourceValue{Name: "source_type", Value: api.FlowLogEndpointTypeWEP},
				elastic.CompositeAggregationSourceValue{Name: "source_namespace", Value: "namespace1"},
				elastic.CompositeAggregationSourceValue{Name: "source_name_aggr", Value: "app1-abcdef-*"},
				elastic.CompositeAggregationSourceValue{Name: "dest_type", Value: api.FlowLogEndpointTypeWEP},
				elastic.CompositeAggregationSourceValue{Name: "dest_namespace", Value: "namespace1"},
				elastic.CompositeAggregationSourceValue{Name: "dest_name_aggr", Value: "nginx-12345-*"},
				elastic.CompositeAggregationSourceValue{Name: "proto", Value: "6"},
				elastic.CompositeAggregationSourceValue{Name: "dest_ip", Value: ""},
				elastic.CompositeAggregationSourceValue{Name: "source_ip", Value: ""},
				elastic.CompositeAggregationSourceValue{Name: "source_port", Value: ""},
				elastic.CompositeAggregationSourceValue{Name: "dest_port", Value: 80.0},
				elastic.CompositeAggregationSourceValue{Name: "reporter", Value: "src"},
				elastic.CompositeAggregationSourceValue{Name: "action", Value: "allow"},
			},
			AggregatedTerms: map[string]*elastic.AggregatedTerm{
				"source_labels": &elastic.AggregatedTerm{
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"app=app1": 1,
					},
				},
				"dest_labels": &elastic.AggregatedTerm{
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"app=nginx": 1,
					},
				},
				"policies": &elastic.AggregatedTerm{
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"0|__PROFILE__|__PROFILE__.kns.namespace1|allow|0": 1,
					},
				},
			},
		},
	}
)

var _ = Describe("Policy Recommendation", func() {
	var (
		fakeKube           k8s.Interface
		ec                 *fakeAggregator
		mockRBACAuthorizer *lmaauth.MockRBACAuthorizer
	)
	BeforeEach(func() {
		fakeKube = fake.NewSimpleClientset(app1Dep, app1Rs, nginxDep, nginxRs)
		ec = newFakeAggregator()
		mockRBACAuthorizer = new(lmaauth.MockRBACAuthorizer)
	})
	DescribeTable("Recommend policies for matching flows and endpoint",
		func(queryResults []*elastic.CompositeAggregationBucket, queryError error,
			query *policyrec.PolicyRecommendationParams,
			expectedResponse *PolicyRecommendationResponse, statusCode int) {

			mockClientSet := datastore.NewClientSet(fakeKube, nil)

			mockK8sClientFactory := new(datastore.MockClusterCtxK8sClientFactory)
			mockK8sClientFactory.On("RBACAuthorizerForCluster", mock.Anything).Return(mockRBACAuthorizer, nil)
			mockK8sClientFactory.On("ClientSetForCluster", mock.Anything).Return(mockClientSet, nil)

			By("Initializing the engine") // Tempted to say "Start your engines!"
			hdlr := PolicyRecommendationHandler(mockK8sClientFactory, mockClientSet, ec)

			jsonQuery, err := json.Marshal(query)
			Expect(err).To(BeNil())

			req, err := http.NewRequest(http.MethodPost, recommendURLPath, bytes.NewBuffer(jsonQuery))
			Expect(err).To(BeNil())

			mockRBACAuthorizer.On("Authorize", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)

			// add a bogus user
			req = req.WithContext(request.WithUser(req.Context(), &user.DefaultInfo{}))

			By("setting up next results")
			ec.setNextResults(queryResults)
			ec.setNextError(queryError)
			// Always allow

			w := httptest.NewRecorder()
			hdlr.ServeHTTP(w, req)
			Expect(err).To(BeNil())

			if statusCode != http.StatusOK {
				Expect(w.Code).To(Equal(http.StatusNotFound))
				recResponse, err := ioutil.ReadAll(w.Body)
				Expect(err).NotTo(HaveOccurred())
				errorBody := &lmaerror.Error{}
				err = json.Unmarshal(recResponse, errorBody)
				Expect(err).To(BeNil())
				Expect(errorBody.Code).To(Equal(statusCode))
				Expect(errorBody.Feature).To(Equal(lmaerror.PolicyRec))
				return
			}

			recResponse, err := ioutil.ReadAll(w.Body)
			Expect(err).NotTo(HaveOccurred())

			actualRec := &PolicyRecommendationResponse{}
			err = json.Unmarshal(recResponse, actualRec)
			Expect(err).To(BeNil())

			if expectedResponse == nil {
				Expect(actualRec).To(BeNil())
			} else {
				Expect(actualRec).ToNot(BeNil())
				Expect(actualRec).To(Equal(expectedResponse))
			}
		},
		Entry("for source endpoint", app1ToNginxFlows, nil,
			app1Query,
			&PolicyRecommendationResponse{
				Recommendation: &policyrec.Recommendation{
					NetworkPolicies: []*v3.StagedNetworkPolicy{
						app1Policy,
					},
					GlobalNetworkPolicies: []*v3.StagedGlobalNetworkPolicy{},
				},
			}, http.StatusOK),
		Entry("for destination endpoint", app1ToNginxFlows, nil,
			nginxQuery,
			&PolicyRecommendationResponse{
				Recommendation: &policyrec.Recommendation{
					NetworkPolicies: []*v3.StagedNetworkPolicy{
						nginxPolicy,
					},
					GlobalNetworkPolicies: []*v3.StagedGlobalNetworkPolicy{},
				},
			}, http.StatusOK),
		Entry("for destination endpoint with egress only flows - no rules will be computed", app1ToNginxEgressFlows, nil,
			nginxQuery, nil, http.StatusInternalServerError),
		Entry("for unknown endpoint", []*elastic.CompositeAggregationBucket{}, nil,
			&policyrec.PolicyRecommendationParams{
				StartTime:    "now-1h",
				EndTime:      "now",
				EndpointName: "idontexist-*",
				Namespace:    "default",
			}, nil, http.StatusNotFound),
		Entry("for query that errors out - invalid time parameters", nil, fmt.Errorf("Elasticsearch error"),
			&policyrec.PolicyRecommendationParams{
				StartTime:    "now",
				EndTime:      "now-1h",
				EndpointName: "someendpoint-*",
				Namespace:    "default",
			}, nil, http.StatusInternalServerError),
	)
})

// fakeAggregator is a test utility that implements the CompositeAggregator interface
type fakeAggregator struct {
	nextResults []*elastic.CompositeAggregationBucket
	nextError   error
}

func newFakeAggregator() *fakeAggregator {
	return &fakeAggregator{}
}

func (fa *fakeAggregator) SearchCompositeAggregations(
	context.Context, *elastic.CompositeAggregationQuery, elastic.CompositeAggregationKey,
) (<-chan *elastic.CompositeAggregationBucket, <-chan error) {
	dataChan := make(chan *elastic.CompositeAggregationBucket, len(fa.nextResults))
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

func (fa *fakeAggregator) setNextResults(cab []*elastic.CompositeAggregationBucket) {
	fa.nextResults = cab
}

func (fa *fakeAggregator) setNextError(err error) {
	fa.nextError = err
}
