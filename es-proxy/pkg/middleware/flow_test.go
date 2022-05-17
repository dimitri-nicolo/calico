package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/google/go-cmp/cmp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/stretchr/testify/mock"

	authzv1 "k8s.io/api/authorization/v1"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/endpoints/request"

	"github.com/projectcalico/calico/compliance/pkg/datastore"
	calicojson "github.com/projectcalico/calico/es-proxy/test/json"
	"github.com/projectcalico/calico/es-proxy/test/thirdpartymock"

	"github.com/projectcalico/calico/lma/pkg/api"
	lmaauth "github.com/projectcalico/calico/lma/pkg/auth"
	celastic "github.com/projectcalico/calico/lma/pkg/elastic"

	"github.com/olivere/elastic/v7"
)

var _ = Describe("FlowLog", func() {
	var (
		mockDoer             *thirdpartymock.MockDoer
		mockRBACAuthoriser   *lmaauth.MockRBACAuthorizer
		mockK8sClientFactory *datastore.MockClusterCtxK8sClientFactory
		flowLogHandler       http.Handler

		defaultUser user.Info
	)

	BeforeEach(func() {
		defaultUser = &user.DefaultInfo{Name: "defaultUser"}

		mockDoer = new(thirdpartymock.MockDoer)
		mockRBACAuthoriser = new(lmaauth.MockRBACAuthorizer)
		mockK8sClientFactory = new(datastore.MockClusterCtxK8sClientFactory)

		client, err := elastic.NewClient(elastic.SetHttpClient(mockDoer), elastic.SetSniff(false), elastic.SetHealthcheck(false))
		Expect(err).ShouldNot(HaveOccurred())

		flowLogHandler = NewFlowHandler(celastic.NewWithClient(client), mockK8sClientFactory)
	})

	AfterEach(func() {
		mockRBACAuthoriser.AssertExpectations(GinkgoT())
		mockK8sClientFactory.AssertExpectations(GinkgoT())
		mockDoer.AssertExpectations(GinkgoT())
	})

	Context("ServeHTTP", func() {
		Context("request parameter validation", func() {
			DescribeTable("it fails if the required parameters are not set",
				func(req *http.Request, expectedCode int, expectedBody string) {
					respRecorder := httptest.NewRecorder()
					flowLogHandler.ServeHTTP(respRecorder, req)

					Expect(respRecorder.Code).Should(Equal(expectedCode))
					Expect(strings.TrimSpace(respRecorder.Body.String())).Should(Equal(expectedBody))
				},
				Entry("when the srcType parameter is missing", createFlowLogRequest(map[string][]string{
					"cluster": {"cluster"}, "srcNamespace": {"default"}, "srcName": {"source"},
					"dstType": {api.FlowLogEndpointTypeWEP}, "dstNamespace": {"default"}, "dstName": {"destination"},
				}), 400, "missing required parameter 'srcType'"),
				Entry("when the srcNamespace parameter is missing", createFlowLogRequest(map[string][]string{
					"cluster": {"cluster"}, "srcType": {api.FlowLogEndpointTypeWEP}, "srcName": {"source"},
					"dstType": {api.FlowLogEndpointTypeWEP}, "dstNamespace": {"default"}, "dstName": {"destination"},
				}), 400, "missing required parameter 'srcNamespace'"),
				Entry("when the srcName parameter is missing", createFlowLogRequest(map[string][]string{
					"cluster": {"cluster"}, "srcType": {api.FlowLogEndpointTypeWEP}, "srcNamespace": {"default"},
					"dstType": {api.FlowLogEndpointTypeWEP}, "dstNamespace": {"default"}, "dstName": {"destination"},
				}), 400, "missing required parameter 'srcName'"),
				Entry("when the dstType parameter is missing and the dstType is wep", createFlowLogRequest(map[string][]string{
					"cluster": {"cluster"}, "srcType": {api.FlowLogEndpointTypeWEP}, "srcNamespace": {"default"},
					"srcName": {"source"}, "dstNamespace": {"default"}, "dstName": {"destination"},
				}), 400, "missing required parameter 'dstType'"),
				Entry("when the dstNamespace parameter is missing", createFlowLogRequest(map[string][]string{
					"cluster": {"cluster"}, "srcType": {api.FlowLogEndpointTypeWEP}, "srcNamespace": {"default"},
					"srcName": {"source"}, "dstType": {api.FlowLogEndpointTypeWEP}, "dstName": {"destination"},
				}), 400, "missing required parameter 'dstNamespace'"),
				Entry("when the dstName parameter is missing", createFlowLogRequest(map[string][]string{
					"cluster": {"cluster"}, "srcType": {api.FlowLogEndpointTypeWEP}, "srcNamespace": {"default"},
					"srcName": {"source"}, "dstType": {api.FlowLogEndpointTypeWEP}, "dstNamespace": {"default"},
				}), 400, "missing required parameter 'dstName'"),
				Entry("when startDateTime is set but not in the RFC3339 format", createFlowLogRequest(map[string][]string{
					"cluster": {"cluster"}, "srcType": {api.FlowLogEndpointTypeWEP}, "srcNamespace": {"default"},
					"srcName": {"source"}, "dstType": {api.FlowLogEndpointTypeWEP}, "dstNamespace": {"default"}, "dstName": {"destination"},
					"startDateTime": {"invalid-start-date-time"},
				}), 400, "failed to parse 'startDateTime' value 'invalid-start-date-time' as RFC3339 datetime or relative time"),
				Entry("when endDateTime is set but not in the RFC3339 format", createFlowLogRequest(map[string][]string{
					"cluster": {"cluster"}, "srcType": {api.FlowLogEndpointTypeWEP}, "srcNamespace": {"default"},
					"srcName": {"source"}, "dstType": {api.FlowLogEndpointTypeWEP}, "dstNamespace": {"default"}, "dstName": {"destination"},
					"endDateTime": {"invalid-end-date-time"},
				}), 400, "failed to parse 'endDateTime' value 'invalid-end-date-time' as RFC3339 datetime or relative time"),
			)

			DescribeTable("it passed parameter validation",
				func(req *http.Request) {
					respRecorder := httptest.NewRecorder()

					req = req.WithContext(request.WithUser(req.Context(), defaultUser))
					mockRBACAuthoriser.On("Authorize", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
					mockK8sClientFactory.On("RBACAuthorizerForCluster", mock.Anything).Return(mockRBACAuthoriser, nil)

					mockDoer.On("Do", mock.Anything).Return(&http.Response{
						StatusCode: http.StatusOK,
						Body: esSearchResultToResponseBody(elastic.SearchResult{
							Hits: &elastic.SearchHits{TotalHits: &elastic.TotalHits{Value: 1}},
						}),
					}, nil)

					flowLogHandler.ServeHTTP(respRecorder, req)

					Expect(respRecorder.Code).Should(Equal(200))
				},
				Entry("when all parameters are properly set", createFlowLogRequest(map[string][]string{
					"cluster": {"cluster"}, "srcType": {api.FlowLogEndpointTypeWEP}, "srcNamespace": {"default"},
					"srcName": {"source"}, "dstType": {api.FlowLogEndpointTypeWEP}, "dstNamespace": {"default"},
					"dstName": {"destination"},
				})),
			)
		})
	})

	When("no results are returned from elasticsearch", func() {
		It("returns a 404", func() {
			req := createFlowLogRequest(map[string][]string{
				"action": {"deny"}, "cluster": {"cluster"}, "srcType": {api.FlowLogEndpointTypeWEP}, "srcNamespace": {"default"},
				"srcName": {"source"}, "dstType": {api.FlowLogEndpointTypeHEP}, "dstName": {"destination"},
				"dstNamespace": {api.GlobalEndpointType},
			})

			req = req.WithContext(request.WithUser(req.Context(), defaultUser))

			mockRBACAuthoriser.On("Authorize", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
			mockK8sClientFactory.On("RBACAuthorizerForCluster", mock.Anything).Return(mockRBACAuthoriser, nil)

			mockDoer.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
				StatusCode: http.StatusOK,
				Body:       esSearchResultToResponseBody(elastic.SearchResult{}),
			}, nil)

			respRecorder := httptest.NewRecorder()
			flowLogHandler.ServeHTTP(respRecorder, req)

			Expect(respRecorder.Code).Should(Equal(404))
		})
	})

	DescribeTable("Elasticsearch query verification", func(req *http.Request, expectedEsQuery calicojson.Map) {
		req = req.WithContext(request.WithUser(req.Context(), defaultUser))

		mockRBACAuthoriser.On("Authorize", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
		mockK8sClientFactory.On("RBACAuthorizerForCluster", mock.Anything).Return(mockRBACAuthoriser, nil)

		expectedJsonObj := calicojson.Map{
			"aggregations": calicojson.Map{
				"dest_labels": calicojson.Map{
					"aggregations": calicojson.Map{"by_kvpair": calicojson.Map{"terms": calicojson.Map{"field": "dest_labels.labels"}}},
					"nested":       calicojson.Map{"path": "dest_labels"},
				},
				"src_policy_report": calicojson.Map{
					"filter": calicojson.Map{"term": calicojson.Map{"reporter": "src"}},
					"aggregations": calicojson.Map{
						"allowed_flow_policies": calicojson.Map{
							"filter": calicojson.Map{"term": calicojson.Map{"action": "allow"}},
							"aggregations": calicojson.Map{
								"policies": calicojson.Map{
									"aggregations": calicojson.Map{"by_tiered_policy": calicojson.Map{"terms": calicojson.Map{"field": "policies.all_policies"}}},
									"nested":       calicojson.Map{"path": "policies"},
								},
							},
						},
						"denied_flow_policies": calicojson.Map{
							"filter": calicojson.Map{"term": calicojson.Map{"action": "deny"}},
							"aggregations": calicojson.Map{
								"policies": calicojson.Map{
									"aggregations": calicojson.Map{"by_tiered_policy": calicojson.Map{"terms": calicojson.Map{"field": "policies.all_policies"}}},
									"nested":       calicojson.Map{"path": "policies"},
								},
							},
						},
					},
				},
				"dest_policy_report": calicojson.Map{
					"filter": calicojson.Map{"term": calicojson.Map{"reporter": "dst"}},
					"aggregations": calicojson.Map{
						"allowed_flow_policies": calicojson.Map{
							"filter": calicojson.Map{"term": calicojson.Map{"action": "allow"}},
							"aggregations": calicojson.Map{
								"policies": calicojson.Map{
									"aggregations": calicojson.Map{"by_tiered_policy": calicojson.Map{"terms": calicojson.Map{"field": "policies.all_policies"}}},
									"nested":       calicojson.Map{"path": "policies"},
								},
							},
						},
						"denied_flow_policies": calicojson.Map{
							"filter": calicojson.Map{"term": calicojson.Map{"action": "deny"}},
							"aggregations": calicojson.Map{
								"policies": calicojson.Map{
									"aggregations": calicojson.Map{"by_tiered_policy": calicojson.Map{"terms": calicojson.Map{"field": "policies.all_policies"}}},
									"nested":       calicojson.Map{"path": "policies"},
								},
							},
						},
					},
				},
				"source_labels": calicojson.Map{
					"aggregations": calicojson.Map{"by_kvpair": calicojson.Map{"terms": calicojson.Map{"field": "source_labels.labels"}}},
					"nested":       calicojson.Map{"path": "source_labels"},
				},
			},
			"query": expectedEsQuery,
			"size":  0,
		}

		mockDoer.On("Do", mock.AnythingOfType("*http.Request")).Run(func(args mock.Arguments) {
			defer GinkgoRecover()
			req := args.Get(0).(*http.Request)

			body, err := ioutil.ReadAll(req.Body)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(req.Body.Close()).ShouldNot(HaveOccurred())

			req.Body = ioutil.NopCloser(bytes.NewBuffer(body))

			requestJson := map[string]interface{}{}
			Expect(json.Unmarshal(body, &requestJson)).ShouldNot(HaveOccurred())
			Expect(calicojson.MustUnmarshalToStandObject(body)).Should(Equal(calicojson.MustUnmarshalToStandObject(expectedJsonObj)),
				cmp.Diff(calicojson.MustUnmarshalToStandObject(body), calicojson.MustUnmarshalToStandObject(expectedJsonObj)))
		}).Return(&http.Response{
			StatusCode: http.StatusOK,
			Body: esSearchResultToResponseBody(elastic.SearchResult{
				Hits: &elastic.SearchHits{TotalHits: &elastic.TotalHits{Value: 1}},
			}),
		}, nil)

		respRecorder := httptest.NewRecorder()
		flowLogHandler.ServeHTTP(respRecorder, req)

		Expect(respRecorder.Code).Should(Equal(200))
	},
		Entry("when startDateTime and endDateTime are not specified",
			createFlowLogRequest(map[string][]string{
				"cluster": {"cluster"}, "srcType": {api.FlowLogEndpointTypeHEP}, "srcName": {"source"},
				"srcNamespace": {api.GlobalEndpointType}, "dstType": {api.FlowLogEndpointTypeWEP}, "dstNamespace": {"default"},
				"dstName": {"destination"},
			}),

			calicojson.Map{"bool": calicojson.Map{
				"filter": []calicojson.Map{
					{"term": calicojson.Map{"source_type": "hep"}},
					{"term": calicojson.Map{"source_name_aggr": "source"}},
					{"term": calicojson.Map{"source_namespace": api.GlobalEndpointType}},
					{"term": calicojson.Map{"dest_type": "wep"}},
					{"term": calicojson.Map{"dest_name_aggr": "destination"}},
					{"term": calicojson.Map{"dest_namespace": "default"}},
				}},
			},
		),
		Entry("when startDateTime and endDateTime are specified",
			createFlowLogRequest(map[string][]string{
				"cluster": {"cluster"}, "srcType": {api.FlowLogEndpointTypeWEP}, "srcNamespace": {"default"},
				"srcName": {"source"}, "dstType": {api.FlowLogEndpointTypeWEP}, "dstNamespace": {"default"}, "dstName": {"destination"},
				"startDateTime": {"2006-01-02T13:04:05Z"}, "endDateTime": {"2006-01-02T15:04:05Z"},
			}),

			calicojson.Map{"bool": calicojson.Map{
				"filter": []calicojson.Map{
					{"term": calicojson.Map{"source_type": "wep"}},
					{"term": calicojson.Map{"source_name_aggr": "source"}},
					{"term": calicojson.Map{"source_namespace": "default"}},
					{"term": calicojson.Map{"dest_type": "wep"}},
					{"term": calicojson.Map{"dest_name_aggr": "destination"}},
					{"term": calicojson.Map{"dest_namespace": "default"}},
					{"range": calicojson.Map{
						"end_time": calicojson.Map{
							"from":          mustParseTime("2006-01-02T13:04:05Z", time.RFC3339).Unix(),
							"include_lower": true,
							"include_upper": false,
							"to":            mustParseTime("2006-01-02T15:04:05Z", time.RFC3339).Unix(),
						},
					}},
				}},
			},
		),
		Entry("when source and destination labels are specified",
			createFlowLogRequest(map[string][]string{
				"cluster": {"cluster"}, "srcType": {api.FlowLogEndpointTypeWEP}, "srcNamespace": {"default"},
				"srcName": {"source"}, "dstType": {api.FlowLogEndpointTypeWEP}, "dstNamespace": {"default"}, "dstName": {"destination"},
				"srcLabels": {createLabelJson("srcname", "=", []string{"srcfoo"}), createLabelJson("srcotherlabel", "!=", []string{"srcbar"})},
				"dstLabels": {createLabelJson("dstname", "=", []string{"srcfoo"}), createLabelJson("dstotherlabel", "!=", []string{"dstbar"})},
			}),
			calicojson.Map{"bool": calicojson.Map{
				"filter": []calicojson.Map{
					{"term": calicojson.Map{"source_type": "wep"}},
					{"term": calicojson.Map{"source_name_aggr": "source"}},
					{"term": calicojson.Map{"source_namespace": "default"}},
					{"term": calicojson.Map{"dest_type": "wep"}},
					{"term": calicojson.Map{"dest_name_aggr": "destination"}},
					{"term": calicojson.Map{"dest_namespace": "default"}},
					{"nested": calicojson.Map{
						"path": "source_labels",
						"query": calicojson.Map{
							"bool": calicojson.Map{
								"filter": []calicojson.Map{
									{"term": calicojson.Map{"source_labels.labels": "srcname=srcfoo"}},
									// IMPORTANT: this is NOT a correct Elasticsearch label query, no label log will EVER
									// have != in it. When conversion of a label selector to an elasticsearch query is fixed
									// this will be updated, but for now this is possible and expected output.
									{"term": calicojson.Map{"source_labels.labels": "srcotherlabel!=srcbar"}},
								},
							},
						},
					}},
					{"nested": calicojson.Map{
						"path": "dest_labels",
						"query": calicojson.Map{
							"bool": calicojson.Map{
								"filter": []calicojson.Map{
									{"term": calicojson.Map{"dest_labels.labels": "dstname=srcfoo"}},
									// IMPORTANT: this is NOT a correct Elasticsearch label query, no label log will EVER
									// have != in it. When conversion of a label selector to an elasticsearch query is fixed
									// this will be updated, but for now this is possible and expected output.
									{"term": calicojson.Map{"dest_labels.labels": "dstotherlabel!=dstbar"}},
								},
							},
						},
					}},
				}},
			},
		),
	)

	Context("RBAC permission validation", func() {
		It("fails the request if the is no user in the request context", func() {
			respRecorder := httptest.NewRecorder()
			req := createFlowLogRequest(map[string][]string{
				"action": {"deny"}, "cluster": {"cluster"}, "srcType": {api.FlowLogEndpointTypeWEP}, "srcNamespace": {"default"},
				"srcName": {"source"}, "dstType": {api.FlowLogEndpointTypeWEP}, "dstNamespace": {"default"}, "dstName": {"destination"},
			})

			flowLogHandler.ServeHTTP(respRecorder, req)

			Expect(respRecorder.Code).Should(Equal(401))
			Expect(strings.TrimSpace(respRecorder.Body.String())).Should(Equal(HttpErrUnauthorizedFlowAccess))
		})

		DescribeTable("fails the request if the user is not authorized to view the requested flow",
			func(srcType, srcNamespace, dstType, dstNamespace string, unAuthResources []*authzv1.ResourceAttributes) {
				for _, res := range unAuthResources {
					mockRBACAuthoriser.
						On("Authorize", defaultUser, res, (*authzv1.NonResourceAttributes)(nil)).
						Return(false, nil).Once()
				}

				mockK8sClientFactory.On("RBACAuthorizerForCluster", "cluster").Return(mockRBACAuthoriser, nil)

				respRecorder := httptest.NewRecorder()
				req := createFlowLogRequest(map[string][]string{
					"cluster": {"cluster"}, "srcType": {srcType}, "srcNamespace": {srcNamespace},
					"srcName": {"source"}, "dstType": {dstType}, "dstNamespace": {dstNamespace}, "dstName": {"destination"},
				})

				req = req.WithContext(request.WithUser(req.Context(), defaultUser))

				flowLogHandler.ServeHTTP(respRecorder, req)

				Expect(respRecorder.Code).Should(Equal(401))
				Expect(strings.TrimSpace(respRecorder.Body.String())).Should(Equal(HttpErrUnauthorizedFlowAccess))
			},
			Entry("when the srcType is hep and the user cannot access hep endpoints",
				api.FlowLogEndpointTypeHEP, api.GlobalEndpointType, api.FlowLogEndpointTypeHEP, api.GlobalEndpointType,
				[]*authzv1.ResourceAttributes{{Verb: "list", Group: "projectcalico.org", Resource: "hostendpoints"}},
			),
			Entry("when the srcType is ns, there is no source namespace, and the user cannot list global network sets",
				api.FlowLogEndpointTypeNetworkSet, api.GlobalEndpointType, api.FlowLogEndpointTypeNetworkSet, api.GlobalEndpointType,
				[]*authzv1.ResourceAttributes{{Verb: "list", Group: "projectcalico.org", Resource: "globalnetworksets"}},
			),
			Entry("when the srcType is ns and the user cannot list network sets in the source namespace",
				api.FlowLogEndpointTypeNetworkSet, "default", api.FlowLogEndpointTypeNetworkSet, "default",
				[]*authzv1.ResourceAttributes{
					{Namespace: "default", Verb: "list", Group: "projectcalico.org", Resource: "networksets"},
				},
			),
			Entry("when the srcType is wep and the user cannot list pods in the source namespace",
				api.FlowLogEndpointTypeWEP, "default", api.FlowLogEndpointTypeWEP, "default",
				[]*authzv1.ResourceAttributes{{Namespace: "default", Verb: "list", Resource: "pods"}},
			),
			Entry("when the dstType is hep and the user cannot access hep endpoints",
				api.FlowLogEndpointTypeNetworkSet, api.GlobalEndpointType, api.FlowLogEndpointTypeHEP, api.GlobalEndpointType,
				[]*authzv1.ResourceAttributes{
					{Verb: "list", Group: "projectcalico.org", Resource: "globalnetworksets"},
					{Verb: "list", Group: "projectcalico.org", Resource: "hostendpoints"},
				},
			),
			Entry("when the dstType is ns, there is no destination namespace, and the user cannot list global network sets",
				api.FlowLogEndpointTypeHEP, api.GlobalEndpointType, api.FlowLogEndpointTypeNetworkSet, api.GlobalEndpointType,
				[]*authzv1.ResourceAttributes{
					{Verb: "list", Group: "projectcalico.org", Resource: "hostendpoints"},
					{Verb: "list", Group: "projectcalico.org", Resource: "globalnetworksets"},
				},
			),
			Entry("when the dstType is ns and the user cannot list network sets in the destination namespace",
				api.FlowLogEndpointTypeHEP, api.GlobalEndpointType, api.FlowLogEndpointTypeNetworkSet, "default",
				[]*authzv1.ResourceAttributes{
					{Verb: "list", Group: "projectcalico.org", Resource: "hostendpoints"},
					{Verb: "list", Namespace: "default", Group: "projectcalico.org", Resource: "networksets"},
				},
			),
			Entry("when the dstType is wep and the user cannot list pods in the destination namespace",
				api.FlowLogEndpointTypeHEP, api.GlobalEndpointType, api.FlowLogEndpointTypeWEP, "default",
				[]*authzv1.ResourceAttributes{
					{Verb: "list", Group: "projectcalico.org", Resource: "hostendpoints"},
					{Namespace: "default", Verb: "list", Resource: "pods"},
				},
			),
		)

		DescribeTable("succeeds when the user is authorized to to access the flow",
			func(srcType, srcNamespace, dstType, dstNamespace string, authResources, unAuthResources []*authzv1.ResourceAttributes) {
				for _, res := range unAuthResources {
					mockRBACAuthoriser.
						On("Authorize", defaultUser, res, (*authzv1.NonResourceAttributes)(nil)).
						Return(false, nil).Once()
				}

				for _, res := range authResources {
					mockRBACAuthoriser.
						On("Authorize", defaultUser, res, (*authzv1.NonResourceAttributes)(nil)).
						Return(true, nil).Once()
				}

				mockK8sClientFactory.On("RBACAuthorizerForCluster", "cluster").Return(mockRBACAuthoriser, nil)

				mockDoer.On("Do", mock.Anything).Return(&http.Response{
					StatusCode: http.StatusOK,
					Body: esSearchResultToResponseBody(elastic.SearchResult{
						Hits: &elastic.SearchHits{TotalHits: &elastic.TotalHits{Value: 1}},
					}),
				}, nil)

				respRecorder := httptest.NewRecorder()
				req := createFlowLogRequest(map[string][]string{
					"action": {"deny"}, "cluster": {"cluster"}, "srcType": {srcType}, "srcNamespace": {srcNamespace}, "srcName": {"source"},
					"dstType": {dstType}, "dstNamespace": {dstNamespace}, "dstName": {"destination"},
				})

				req = req.WithContext(request.WithUser(req.Context(), defaultUser))

				flowLogHandler.ServeHTTP(respRecorder, req)

				Expect(respRecorder.Code).Should(Equal(200))
			},
			Entry("when the user is authorized to list source endpoint hep type but not the destination ns endpoint type",
				api.FlowLogEndpointTypeHEP, api.GlobalEndpointType, api.FlowLogEndpointTypeNetworkSet, api.GlobalEndpointType,
				[]*authzv1.ResourceAttributes{{Verb: "list", Group: "projectcalico.org", Resource: "hostendpoints"}},
				[]*authzv1.ResourceAttributes{{Verb: "list", Group: "projectcalico.org", Resource: "globalnetworksets"}},
			),
			Entry(
				"when the user is authorized to list source non namespaced ns type but not the destination hep type",
				api.FlowLogEndpointTypeNetworkSet, api.GlobalEndpointType, api.FlowLogEndpointTypeHEP, api.GlobalEndpointType,
				[]*authzv1.ResourceAttributes{{Verb: "list", Group: "projectcalico.org", Resource: "globalnetworksets"}},
				[]*authzv1.ResourceAttributes{{Verb: "list", Group: "projectcalico.org", Resource: "hostendpoints"}},
			),
			Entry(
				"when the user is authorized to list source namespaced ns type but not the destination hep type",
				api.FlowLogEndpointTypeNetworkSet, "default", api.FlowLogEndpointTypeHEP, api.GlobalEndpointType,
				[]*authzv1.ResourceAttributes{{Namespace: "default", Verb: "list", Group: "projectcalico.org", Resource: "networksets"}},
				[]*authzv1.ResourceAttributes{{Verb: "list", Group: "projectcalico.org", Resource: "hostendpoints"}},
			),
			Entry(
				"when the user is authorized to list source wep type but not the destination hep type",
				api.FlowLogEndpointTypeWEP, "default", api.FlowLogEndpointTypeHEP, api.GlobalEndpointType,
				[]*authzv1.ResourceAttributes{{Namespace: "default", Verb: "list", Resource: "pods"}},
				[]*authzv1.ResourceAttributes{{Verb: "list", Group: "projectcalico.org", Resource: "hostendpoints"}},
			),

			Entry("when the user is authorized to list destination endpoint hep type but not the source ns endpoint type",
				api.FlowLogEndpointTypeNetworkSet, api.GlobalEndpointType, api.FlowLogEndpointTypeHEP, api.GlobalEndpointType,
				[]*authzv1.ResourceAttributes{{Verb: "list", Group: "projectcalico.org", Resource: "hostendpoints"}},
				[]*authzv1.ResourceAttributes{{Verb: "list", Group: "projectcalico.org", Resource: "globalnetworksets"}},
			),
			Entry(
				"when the user is authorized to list destination non namespaced ns type but not the source hep type",
				api.FlowLogEndpointTypeHEP, api.GlobalEndpointType, api.FlowLogEndpointTypeNetworkSet, api.GlobalEndpointType,
				[]*authzv1.ResourceAttributes{{Verb: "list", Group: "projectcalico.org", Resource: "globalnetworksets"}},
				[]*authzv1.ResourceAttributes{{Verb: "list", Group: "projectcalico.org", Resource: "hostendpoints"}},
			),
			Entry(
				"when the user is authorized to list destination namespaced ns type but not the source hep type",
				api.FlowLogEndpointTypeHEP, api.GlobalEndpointType, api.FlowLogEndpointTypeNetworkSet, "default",
				[]*authzv1.ResourceAttributes{{Namespace: "default", Verb: "list", Group: "projectcalico.org", Resource: "networksets"}},
				[]*authzv1.ResourceAttributes{{Verb: "list", Group: "projectcalico.org", Resource: "hostendpoints"}},
			),
			Entry(
				"when the user is authorized to list destination wep type but not the source hep type",
				api.FlowLogEndpointTypeHEP, api.GlobalEndpointType, api.FlowLogEndpointTypeWEP, "default",
				[]*authzv1.ResourceAttributes{{Namespace: "default", Verb: "list", Resource: "pods"}},
				[]*authzv1.ResourceAttributes{{Verb: "list", Group: "projectcalico.org", Resource: "hostendpoints"}},
			),
		)
	})

	Context("elasticsearch response is properly parsed", func() {
		var (
			req          *http.Request
			respRecorder *httptest.ResponseRecorder
		)

		BeforeEach(func() {
			mockK8sClientFactory.On("RBACAuthorizerForCluster", "cluster").Return(mockRBACAuthoriser, nil)

			req = createFlowLogRequest(map[string][]string{
				"action": {"deny"}, "cluster": {"cluster"}, "srcType": {"wep"}, "srcNamespace": {"source-ns"}, "srcName": {"source"},
				"dstType": {"wep"}, "dstNamespace": {"destination-ns"}, "dstName": {"destination"},
			})
			req = req.WithContext(request.WithUser(req.Context(), defaultUser))

			respRecorder = httptest.NewRecorder()
		})

		Context("for labels", func() {
			BeforeEach(func() {
				mockRBACAuthoriser.On("Authorize", defaultUser, mock.Anything, (*authzv1.NonResourceAttributes)(nil)).Return(true, nil)
			})

			// These table entry tests are run against setting both source and destination labels
			var labelTestCases = []TableEntry{
				Entry("parses and returns a single label",
					[]map[string]interface{}{
						{"doc_count": 1, "key": "labelname=labelvalue"},
					},
					FlowResponseLabels{
						"labelname": {{Count: 1, Value: "labelvalue"}},
					},
				),
				Entry("parses and returns a multiple different labels",
					[]map[string]interface{}{
						{"doc_count": 1, "key": "labelname1=labelvalue1"}, {"doc_count": 1, "key": "labelname2=labelvalue2"},
						{"doc_count": 1, "key": "labelname3=labelvalue3"},
					},
					FlowResponseLabels{
						"labelname1": {{Count: 1, Value: "labelvalue1"}},
						"labelname2": {{Count: 1, Value: "labelvalue2"}},
						"labelname3": {{Count: 1, Value: "labelvalue3"}},
					},
				),
				Entry("parses and returns labels with multiple values",
					[]map[string]interface{}{
						{"doc_count": 1, "key": "labelname=labelvalue1"}, {"doc_count": 1, "key": "labelname=labelvalue2"},
						{"doc_count": 1, "key": "labelname=labelvalue3"},
					},
					FlowResponseLabels{
						"labelname": {{Count: 1, Value: "labelvalue1"}, {Count: 1, Value: "labelvalue2"}, {Count: 1, Value: "labelvalue3"}},
					},
				),
				Entry("skips bucket entries with non string keys and parses the other valid labels",
					[]map[string]interface{}{
						{"doc_count": 1, "key": "labelname=labelvalue1"}, {"doc_count": 1, "key": 2},
						{"doc_count": 1, "key": "labelname=labelvalue3"},
					},
					FlowResponseLabels{
						"labelname": {{Count: 1, Value: "labelvalue1"}, {Count: 1, Value: "labelvalue3"}},
					},
				),
				Entry("skips bucket entries with invalid keys and parses the other valid lables",
					[]map[string]interface{}{
						{"doc_count": 1, "key": "labelname=labelvalue1"}, {"doc_count": 1, "key": "badkey"},
						{"doc_count": 1, "key": "labelname=labelvalue3"},
					},
					FlowResponseLabels{
						"labelname": {{Count: 1, Value: "labelvalue1"}, {Count: 1, Value: "labelvalue3"}},
					},
				),
			}

			DescribeTable("parses the source labels", func(buckets []map[string]interface{}, expectedSrcLabels FlowResponseLabels) {
				mockDoer.On("Do", mock.Anything).Return(&http.Response{
					StatusCode: http.StatusOK,
					Body: esSearchResultToResponseBody(elastic.SearchResult{
						Hits: &elastic.SearchHits{TotalHits: &elastic.TotalHits{Value: 1}},
						Aggregations: elastic.Aggregations{
							"source_labels": calicojson.MustMarshal(map[string]interface{}{
								"by_kvpair": map[string]interface{}{
									"buckets": buckets,
								},
							}),
						},
					}),
				}, nil)

				flowLogHandler.ServeHTTP(respRecorder, req)
				Expect(respRecorder.Code).Should(Equal(200))

				respBody, err := ioutil.ReadAll(respRecorder.Body)
				Expect(err).ShouldNot(HaveOccurred())

				var flResponse FlowResponse
				Expect(json.Unmarshal(respBody, &flResponse))

				Expect(flResponse).Should(Equal(FlowResponse{
					Count:     1,
					SrcLabels: expectedSrcLabels,
				}))
			}, labelTestCases...)

			DescribeTable("parses the destination labels", func(buckets []map[string]interface{}, expectedDstLabels FlowResponseLabels) {
				mockDoer.On("Do", mock.Anything).Return(&http.Response{
					StatusCode: http.StatusOK,
					Body: esSearchResultToResponseBody(elastic.SearchResult{
						Hits: &elastic.SearchHits{TotalHits: &elastic.TotalHits{Value: 1}},
						Aggregations: elastic.Aggregations{
							"dest_labels": calicojson.MustMarshal(map[string]interface{}{
								"by_kvpair": map[string]interface{}{
									"buckets": buckets,
								},
							}),
						},
					}),
				}, nil)

				flowLogHandler.ServeHTTP(respRecorder, req)
				Expect(respRecorder.Code).Should(Equal(200))

				respBody, err := ioutil.ReadAll(respRecorder.Body)
				Expect(err).ShouldNot(HaveOccurred())

				var flResponse FlowResponse
				Expect(json.Unmarshal(respBody, &flResponse))

				Expect(flResponse).Should(Equal(FlowResponse{
					Count:     1,
					DstLabels: expectedDstLabels,
				}))
			}, labelTestCases...)
		})

		Context("for policies", func() {
			DescribeTable("parsing policy hits when completely authorized",
				func(
					srcAllowHits, srcDenyHits, dstAllowHits, dstDenyHits []map[string]interface{},
					expectedSrcPolicyReport, expectedDstPolicyReport *PolicyReport) {
					mockRBACAuthoriser.On("Authorize", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)

					agg := elastic.Aggregations{}
					if srcAllowHits != nil || srcDenyHits != nil {
						srcPolicyHits := calicojson.Map{}
						if srcAllowHits != nil {
							srcPolicyHits["allowed_flow_policies"] = calicojson.Map{
								"policies": calicojson.Map{
									"by_tiered_policy": calicojson.Map{
										"buckets": srcAllowHits,
									},
								},
							}
						}
						if srcDenyHits != nil {
							srcPolicyHits["denied_flow_policies"] = calicojson.Map{
								"policies": calicojson.Map{
									"by_tiered_policy": calicojson.Map{
										"buckets": srcDenyHits,
									},
								},
							}
						}
						agg["src_policy_report"] = calicojson.MustMarshal(srcPolicyHits)
					}

					if dstAllowHits != nil || dstDenyHits != nil {
						dstPolicyHits := calicojson.Map{}
						if dstAllowHits != nil {
							dstPolicyHits["allowed_flow_policies"] = calicojson.Map{
								"policies": calicojson.Map{
									"by_tiered_policy": calicojson.Map{
										"buckets": dstAllowHits,
									},
								},
							}
						}
						if dstDenyHits != nil {
							dstPolicyHits["denied_flow_policies"] = calicojson.Map{
								"policies": calicojson.Map{
									"by_tiered_policy": calicojson.Map{
										"buckets": dstDenyHits,
									},
								},
							}
						}
						agg["dest_policy_report"] = calicojson.MustMarshal(dstPolicyHits)
					}

					mockDoer.On("Do", mock.Anything).Return(&http.Response{
						StatusCode: http.StatusOK,
						Body: esSearchResultToResponseBody(elastic.SearchResult{
							Hits:         &elastic.SearchHits{TotalHits: &elastic.TotalHits{Value: 1}},
							Aggregations: agg,
						}),
					}, nil)

					flowLogHandler.ServeHTTP(respRecorder, req)
					Expect(respRecorder.Code).Should(Equal(200))

					respBody, err := ioutil.ReadAll(respRecorder.Body)
					Expect(err).ShouldNot(HaveOccurred())

					var flResponse FlowResponse
					Expect(json.Unmarshal(respBody, &flResponse))

					Expect(flResponse).Should(Equal(FlowResponse{
						Count:           1,
						SrcPolicyReport: expectedSrcPolicyReport,
						DstPolicyReport: expectedDstPolicyReport,
					}))
				},
				Entry("single policy hit allowed at src and dst",
					[]map[string]interface{}{
						{"key": "0|tier1|namespace1/policy1|allow|0", "doc_count": 1},
					}, nil,
					[]map[string]interface{}{
						{"key": "0|tier2|namespace2/policy2|allow|0", "doc_count": 1},
					}, nil,
					&PolicyReport{
						AllowedFlowPolicies: []*FlowResponsePolicy{
							{Index: 0, Namespace: "namespace1", Tier: "tier1", Name: "policy1", Action: "allow", Count: 1},
						},
					},
					&PolicyReport{
						AllowedFlowPolicies: []*FlowResponsePolicy{
							{Index: 0, Namespace: "namespace2", Tier: "tier2", Name: "policy2", Action: "allow", Count: 1},
						},
					},
				),
				Entry("single policy hit allowed at src and denied at dst",
					[]map[string]interface{}{
						{"key": "0|tier1|namespace1/policy1|allow|0", "doc_count": 1},
					}, nil,
					nil,
					[]map[string]interface{}{
						{"key": "0|tier2|namespace2/policy2|deny", "doc_count": 1},
					},
					&PolicyReport{
						AllowedFlowPolicies: []*FlowResponsePolicy{
							{Index: 0, Namespace: "namespace1", Tier: "tier1", Name: "policy1", Action: "allow", Count: 1},
						},
					},
					&PolicyReport{
						DeniedFlowPolicies: []*FlowResponsePolicy{
							{Index: 0, Namespace: "namespace2", Tier: "tier2", Name: "policy2", Action: "deny", Count: 1},
						},
					},
				),
				Entry("single policy hit denied at src and denied on dst",
					nil,
					[]map[string]interface{}{
						{"key": "0|tier1|namespace1/policy1|deny", "doc_count": 1},
					},
					nil, nil,
					&PolicyReport{
						DeniedFlowPolicies: []*FlowResponsePolicy{
							{Index: 0, Namespace: "namespace1", Tier: "tier1", Name: "policy1", Action: "deny", Count: 1},
						},
					},
					nil,
				),
				Entry("single policy hit allowed and denied at src and dst",
					[]map[string]interface{}{
						{"key": "0|tier1|namespace1/policy1|allow|0", "doc_count": 1},
					},
					[]map[string]interface{}{
						{"key": "0|tier4|namespace4/policy4|deny|-1", "doc_count": 1},
					},
					[]map[string]interface{}{
						{"key": "0|tier3|namespace3/policy3|allow|0", "doc_count": 1},
					},
					[]map[string]interface{}{
						{"key": "0|tier2|namespace2/policy2|deny|-1", "doc_count": 1},
					},
					&PolicyReport{
						AllowedFlowPolicies: []*FlowResponsePolicy{
							{Index: 0, Namespace: "namespace1", Tier: "tier1", Name: "policy1", Action: "allow", Count: 1},
						},
						DeniedFlowPolicies: []*FlowResponsePolicy{
							{Index: 0, Namespace: "namespace4", Tier: "tier4", Name: "policy4", Action: "deny", Count: 1},
						},
					},
					&PolicyReport{
						AllowedFlowPolicies: []*FlowResponsePolicy{
							{Index: 0, Namespace: "namespace3", Tier: "tier3", Name: "policy3", Action: "allow", Count: 1},
						},
						DeniedFlowPolicies: []*FlowResponsePolicy{
							{Index: 0, Namespace: "namespace2", Tier: "tier2", Name: "policy2", Action: "deny", Count: 1},
						},
					},
				),
				// Note that this test isn't exactly valid since a deny at the source means no reported flow at the
				// destination, but this is just to test that the allow / deny logic handles multiple policies for
				// src and dst.
				Entry("multiple policy hits for allowed and denied at src and dst",
					[]map[string]interface{}{
						{"key": "0|tier11|namespace11/tier11.policy11|pass|0", "doc_count": 1},
						{"key": "1|tier12|namespace12/tier12.staged:policy12|deny|-1", "doc_count": 1},
						{"key": "2|tier12|namespace12/tier12.policy13|pass", "doc_count": 1},
						{"key": "3|tier13|namespace13/tier13.policy14|pass|3", "doc_count": 1},
						{"key": "4|tier14|namespace14/tier14.policy15|allow", "doc_count": 1},
					},
					[]map[string]interface{}{
						{"key": "0|tier21|namespace21/tier21.policy21|pass|0", "doc_count": 1},
						{"key": "1|tier22|namespace22/tier22.staged:policy22|deny", "doc_count": 1},
						{"key": "2|tier22|namespace22/tier22.policy23|pass|0", "doc_count": 1},
						{"key": "3|tier23|namespace23/tier23.policy24|pass", "doc_count": 1},
						{"key": "4|tier24|namespace24/tier24.policy25|deny|0", "doc_count": 1},
					},
					[]map[string]interface{}{
						{"key": "0|tier31|namespace31/tier31.policy31|pass|0", "doc_count": 1},
						{"key": "1|tier32|namespace32/tier32.staged:policy32|deny|0", "doc_count": 1},
						{"key": "2|tier32|namespace32/tier32.policy33|pass", "doc_count": 1},
						{"key": "3|tier33|namespace33/tier33.policy34|pass|0", "doc_count": 1},
						{"key": "4|tier34|namespace34/tier34.policy35|allow", "doc_count": 1},
					},
					[]map[string]interface{}{
						{"key": "0|tier41|namespace41/tier41.policy41|pass|-", "doc_count": 1},
						{"key": "1|tier42|namespace42/tier42.staged:policy42|deny", "doc_count": 1},
						{"key": "2|tier42|namespace42/tier42.policy43|pass", "doc_count": 1},
						{"key": "3|tier43|namespace43/tier43.policy44|pass|-", "doc_count": 1},
						{"key": "4|tier44|namespace44/tier44.policy45|deny|-", "doc_count": 1},
					},
					&PolicyReport{
						AllowedFlowPolicies: []*FlowResponsePolicy{
							{Index: 0, Namespace: "namespace11", Tier: "tier11", Name: "policy11", Action: "pass", Count: 1},
							{Index: 1, IsStaged: true, Namespace: "namespace12", Tier: "tier12", Name: "policy12", Action: "deny", Count: 1},
							{Index: 2, Namespace: "namespace12", Tier: "tier12", Name: "policy13", Action: "pass", Count: 1},
							{Index: 3, Namespace: "namespace13", Tier: "tier13", Name: "policy14", Action: "pass", Count: 1},
							{Index: 4, Namespace: "namespace14", Tier: "tier14", Name: "policy15", Action: "allow", Count: 1},
						},
						DeniedFlowPolicies: []*FlowResponsePolicy{
							{Index: 0, Namespace: "namespace21", Tier: "tier21", Name: "policy21", Action: "pass", Count: 1},
							{Index: 1, IsStaged: true, Namespace: "namespace22", Tier: "tier22", Name: "policy22", Action: "deny", Count: 1},
							{Index: 2, Namespace: "namespace22", Tier: "tier22", Name: "policy23", Action: "pass", Count: 1},
							{Index: 3, Namespace: "namespace23", Tier: "tier23", Name: "policy24", Action: "pass", Count: 1},
							{Index: 4, Namespace: "namespace24", Tier: "tier24", Name: "policy25", Action: "deny", Count: 1},
						},
					},
					&PolicyReport{
						AllowedFlowPolicies: []*FlowResponsePolicy{
							{Index: 0, Namespace: "namespace31", Tier: "tier31", Name: "policy31", Action: "pass", Count: 1},
							{Index: 1, IsStaged: true, Namespace: "namespace32", Tier: "tier32", Name: "policy32", Action: "deny", Count: 1},
							{Index: 2, Namespace: "namespace32", Tier: "tier32", Name: "policy33", Action: "pass", Count: 1},
							{Index: 3, Namespace: "namespace33", Tier: "tier33", Name: "policy34", Action: "pass", Count: 1},
							{Index: 4, Namespace: "namespace34", Tier: "tier34", Name: "policy35", Action: "allow", Count: 1},
						},
						DeniedFlowPolicies: []*FlowResponsePolicy{
							{Index: 0, Namespace: "namespace41", Tier: "tier41", Name: "policy41", Action: "pass", Count: 1},
							{Index: 1, IsStaged: true, Namespace: "namespace42", Tier: "tier42", Name: "policy42", Action: "deny", Count: 1},
							{Index: 2, Namespace: "namespace42", Tier: "tier42", Name: "policy43", Action: "pass", Count: 1},
							{Index: 3, Namespace: "namespace43", Tier: "tier43", Name: "policy44", Action: "pass", Count: 1},
							{Index: 4, Namespace: "namespace44", Tier: "tier44", Name: "policy45", Action: "deny", Count: 1},
						},
					},
				),
				Entry("Parses Kubernetes policy",
					nil,
					[]map[string]interface{}{
						{"key": "4|default|namespace/knp.default.policy|deny", "doc_count": 1},
					},
					nil, nil,
					&PolicyReport{
						DeniedFlowPolicies: []*FlowResponsePolicy{
							{Index: 0, IsKubernetes: true, Namespace: "namespace", Tier: "default", Name: "policy", Action: "deny", Count: 1},
						},
					},
					nil,
				),
				Entry("Parses Profile policy",
					nil,
					[]map[string]interface{}{
						{"key": "0|__PROFILE__|__PROFILE__.kns.namespace|deny", "doc_count": 1},
					},
					nil, nil,
					&PolicyReport{
						DeniedFlowPolicies: []*FlowResponsePolicy{
							{Index: 0, IsProfile: true, Namespace: "", Tier: "__PROFILE__", Name: "namespace", Action: "deny", Count: 1},
						},
					},
					nil,
				),
			)

			DescribeTable("obfuscating policies",
				func(srcAllowHits, srcDenyHits, dstAllowHits, dstDenyHits []map[string]interface{},
					expectedSrcPolicyReport, expectedDstPolicyReport *PolicyReport,
					authResources []*authzv1.ResourceAttributes) {
					for _, resource := range authResources {
						mockRBACAuthoriser.On("Authorize", mock.Anything, resource, mock.Anything).Return(true, nil)
					}

					mockRBACAuthoriser.On("Authorize", mock.Anything, mock.Anything, mock.Anything).Return(false, nil).Maybe()

					agg := elastic.Aggregations{}
					if srcAllowHits != nil || srcDenyHits != nil {
						srcPolicyHits := calicojson.Map{}
						if srcAllowHits != nil {
							srcPolicyHits["allowed_flow_policies"] = calicojson.Map{
								"policies": calicojson.Map{
									"by_tiered_policy": calicojson.Map{
										"buckets": srcAllowHits,
									},
								},
							}
						}
						if srcDenyHits != nil {
							srcPolicyHits["denied_flow_policies"] = calicojson.Map{
								"policies": calicojson.Map{
									"by_tiered_policy": calicojson.Map{
										"buckets": srcDenyHits,
									},
								},
							}
						}
						agg["src_policy_report"] = calicojson.MustMarshal(srcPolicyHits)
					}

					if dstAllowHits != nil || dstDenyHits != nil {
						dstPolicyHits := calicojson.Map{}
						if dstAllowHits != nil {
							dstPolicyHits["allowed_flow_policies"] = calicojson.Map{
								"policies": calicojson.Map{
									"by_tiered_policy": calicojson.Map{
										"buckets": dstAllowHits,
									},
								},
							}
						}
						if dstDenyHits != nil {
							dstPolicyHits["denied_flow_policies"] = calicojson.Map{
								"policies": calicojson.Map{
									"by_tiered_policy": calicojson.Map{
										"buckets": dstDenyHits,
									},
								},
							}
						}
						agg["dest_policy_report"] = calicojson.MustMarshal(dstPolicyHits)
					}

					mockDoer.On("Do", mock.Anything).Return(&http.Response{
						StatusCode: http.StatusOK,
						Body: esSearchResultToResponseBody(elastic.SearchResult{
							Hits:         &elastic.SearchHits{TotalHits: &elastic.TotalHits{Value: 1}},
							Aggregations: agg,
						}),
					}, nil)

					flowLogHandler.ServeHTTP(respRecorder, req)
					Expect(respRecorder.Code).Should(Equal(200))

					respBody, err := ioutil.ReadAll(respRecorder.Body)
					Expect(err).ShouldNot(HaveOccurred())

					var flResponse FlowResponse
					Expect(json.Unmarshal(respBody, &flResponse))

					Expect(flResponse).Should(Equal(FlowResponse{
						Count:           1,
						SrcPolicyReport: expectedSrcPolicyReport,
						DstPolicyReport: expectedDstPolicyReport,
					}))
				},
				Entry("single obfuscated policy hit allowed at src and dst",
					[]map[string]interface{}{
						{"key": "0|tier1|namespace/policy|allow|0", "doc_count": 1},
					}, nil,
					[]map[string]interface{}{
						{"key": "0|tier2|namespace/policy2|allow|0", "doc_count": 1},
					}, nil,
					&PolicyReport{
						AllowedFlowPolicies: []*FlowResponsePolicy{
							{Index: 0, Namespace: "*", Tier: "*", Name: "*", Action: "allow", Count: 1},
						},
					},
					&PolicyReport{
						AllowedFlowPolicies: []*FlowResponsePolicy{
							{Index: 0, Namespace: "*", Tier: "*", Name: "*", Action: "allow", Count: 1},
						},
					},
					[]*authzv1.ResourceAttributes{
						{Namespace: "source-ns", Verb: "list", Resource: "pods"},
						{Namespace: "destination-ns", Verb: "list", Resource: "pods"},
					},
				),
				// Note that this test isn't exactly valid since a deny at the source means no reported flow at the
				// destination, but this is just to test that the allow / deny logic handles multiple policies for
				// src and dst.
				Entry("multiple obfuscated passes before non obfuscated deny at src and dst",
					nil,
					[]map[string]interface{}{
						{"key": "0|tier11|namespace1/tier11.policy11|pass|0", "doc_count": 1},
						{"key": "1|tier12|namespace1/tier12.policy12|pass|1", "doc_count": 1},
						{"key": "2|tier13|namespace1/tier13.policy13|deny|2", "doc_count": 1},
					},
					nil,
					[]map[string]interface{}{
						{"key": "0|tier21|namespace2/tier21.policy21|pass|0", "doc_count": 1},
						{"key": "1|tier22|namespace2/tier22.policy22|pass|1", "doc_count": 1},
						{"key": "2|tier23|namespace2/tier23.policy23|deny|2", "doc_count": 1},
					},
					&PolicyReport{
						DeniedFlowPolicies: []*FlowResponsePolicy{
							{Index: 0, Namespace: "*", Tier: "*", Name: "*", Action: "pass", Count: 2},
							{Index: 1, Namespace: "namespace1", Tier: "tier13", Name: "policy13", Action: "deny", Count: 1},
						},
					},
					&PolicyReport{
						DeniedFlowPolicies: []*FlowResponsePolicy{
							{Index: 0, Namespace: "*", Tier: "*", Name: "*", Action: "pass", Count: 2},
							{Index: 1, Namespace: "namespace2", Tier: "tier23", Name: "policy23", Action: "deny", Count: 1},
						},
					},
					[]*authzv1.ResourceAttributes{
						{Namespace: "source-ns", Verb: "list", Resource: "pods"},
						{Namespace: "namespace1", Verb: "list", Group: "projectcalico.org", Resource: "tier.networkpolicies"},
						{Namespace: "namespace2", Verb: "list", Group: "projectcalico.org", Resource: "tier.networkpolicies"},
						{Verb: "get", Group: "projectcalico.org", Resource: "tiers", Name: "tier13"},
						{Verb: "get", Group: "projectcalico.org", Resource: "tiers", Name: "tier23"},
					},
				),
				Entry("multiple obfuscated passes before obfuscated deny",
					nil,
					[]map[string]interface{}{
						{"key": "0|tier11|namespace1/tier11.policy11|pass|0", "doc_count": 1},
						{"key": "1|tier12|namespace1/tier12.policy12|pass|1", "doc_count": 1},
						{"key": "2|tier13|namespace1/tier13.policy13|deny|2", "doc_count": 1},
					},
					nil,
					[]map[string]interface{}{
						{"key": "0|tier21|namespace2/tier21.policy21|pass|0", "doc_count": 1},
						{"key": "1|tier22|namespace2/tier22.policy22|pass|1", "doc_count": 1},
						{"key": "2|tier23|namespace2/tier23.policy23|deny|2", "doc_count": 1},
					},
					&PolicyReport{
						DeniedFlowPolicies: []*FlowResponsePolicy{
							{Index: 0, Namespace: "*", Tier: "*", Name: "*", Action: "deny", Count: 3},
						},
					},
					&PolicyReport{
						DeniedFlowPolicies: []*FlowResponsePolicy{
							{Index: 0, Namespace: "*", Tier: "*", Name: "*", Action: "deny", Count: 3},
						},
					},
					[]*authzv1.ResourceAttributes{
						{Namespace: "source-ns", Verb: "list", Resource: "pods"},
						{Namespace: "destination-ns", Verb: "list", Resource: "pods"},
					},
				),
				Entry("multiple obfuscated passes before non obfuscated staged deny before obfuscated deny",
					nil,
					[]map[string]interface{}{
						{"key": "0|tier11|namespace1/tier11.policy11|pass|0", "doc_count": 1},
						{"key": "1|tier12|namespace1/tier12.policy12|pass", "doc_count": 1},
						{"key": "2|tier13|namespace1/tier13.staged:policy13|deny|0", "doc_count": 1},
						{"key": "3|tier13|namespace11/tier13.policy14|pass|1", "doc_count": 1},
						{"key": "4|tier14|namespace1/tier14.policy15|deny", "doc_count": 1},
					},
					nil,
					[]map[string]interface{}{
						{"key": "0|tier21|namespace2/tier21.policy21|pass|0", "doc_count": 1},
						{"key": "1|tier22|namespace2/tier22.policy22|pass", "doc_count": 1},
						{"key": "2|tier23|namespace2/tier23.staged:policy23|deny|0", "doc_count": 1},
						{"key": "3|tier23|namespace21/tier23.policy24|pass|1", "doc_count": 1},
						{"key": "4|tier24|namespace2/tier24.policy25|deny", "doc_count": 1},
					},
					&PolicyReport{
						DeniedFlowPolicies: []*FlowResponsePolicy{
							{Index: 0, Namespace: "*", Tier: "*", Name: "*", Action: "pass", Count: 2},
							{Index: 1, IsStaged: true, Namespace: "namespace1", Tier: "tier13", Name: "policy13", Action: "deny", Count: 1},
							{Index: 2, Namespace: "*", Tier: "*", Name: "*", Action: "deny", Count: 2},
						},
					},
					&PolicyReport{
						DeniedFlowPolicies: []*FlowResponsePolicy{
							{Index: 0, Namespace: "*", Tier: "*", Name: "*", Action: "pass", Count: 2},
							{Index: 1, IsStaged: true, Namespace: "namespace2", Tier: "tier23", Name: "policy23", Action: "deny", Count: 1},
							{Index: 2, Namespace: "*", Tier: "*", Name: "*", Action: "deny", Count: 2},
						},
					},
					[]*authzv1.ResourceAttributes{
						{Namespace: "source-ns", Verb: "list", Resource: "pods"},
						{Namespace: "destination-ns", Verb: "list", Resource: "pods"},
						{Verb: "get", Group: "projectcalico.org", Resource: "tiers", Name: "tier13"},
						{Verb: "get", Group: "projectcalico.org", Resource: "tiers", Name: "tier23"},
						{Namespace: "namespace1", Verb: "list", Group: "projectcalico.org", Resource: "tier.stagednetworkpolicies"},
						{Namespace: "namespace2", Verb: "list", Group: "projectcalico.org", Resource: "tier.stagednetworkpolicies"},
					},
				),
				Entry("omit obfuscated staged deny combine obfuscated pass and deny",
					nil,
					[]map[string]interface{}{
						{"key": "0|tier11|namespace1/tier11.staged:policy11|deny|0", "doc_count": 1},
						{"key": "1|tier11|namespace1/tier11.policy12|pass", "doc_count": 1},
						{"key": "2|tier12|namespace1/tier12.policy13|deny|2", "doc_count": 1},
					},
					nil,
					[]map[string]interface{}{
						{"key": "0|tier21|namespace2/tier21.staged:policy21|deny|0", "doc_count": 1},
						{"key": "1|tier21|namespace2/tier21.policy22|pass|1", "doc_count": 1},
						{"key": "2|tier22|namespace2/tier22.policy23|deny", "doc_count": 1},
					},
					&PolicyReport{
						DeniedFlowPolicies: []*FlowResponsePolicy{
							{Index: 0, Namespace: "*", Tier: "*", Name: "*", Action: "deny", Count: 2},
						},
					},
					&PolicyReport{
						DeniedFlowPolicies: []*FlowResponsePolicy{
							{Index: 0, Namespace: "*", Tier: "*", Name: "*", Action: "deny", Count: 2},
						},
					},
					[]*authzv1.ResourceAttributes{
						{Namespace: "source-ns", Verb: "list", Resource: "pods"},
						{Namespace: "destination-ns", Verb: "list", Resource: "pods"},
					},
				),
			)
		})
	})
})

func createFlowLogRequest(parameters map[string][]string) *http.Request {
	req, err := http.NewRequest("GET", "", nil)
	Expect(err).ShouldNot(HaveOccurred())

	query := req.URL.Query()
	for k, vs := range parameters {
		for _, v := range vs {
			query.Add(k, v)
		}
	}

	req.URL.RawQuery = query.Encode()
	return req
}

func esSearchResultToResponseBody(searchResult elastic.SearchResult) io.ReadCloser {
	byts, err := json.Marshal(searchResult)
	if err != nil {
		panic(err)
	}

	return ioutil.NopCloser(bytes.NewBuffer(byts))
}

func mustParseTime(timeStr, format string) time.Time {
	t, err := time.Parse(format, timeStr)
	if err != nil {
		panic(err)
	}

	return t
}

func createLabelJson(key, operator string, values []string) string {
	return string(calicojson.MustMarshal(map[string]interface{}{
		"key": key, "operator": operator, "values": values,
	}))
}
