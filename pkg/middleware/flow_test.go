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

	calicojson "github.com/tigera/es-proxy/test/json"

	celastic "github.com/tigera/lma/pkg/elastic"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/stretchr/testify/mock"

	authzv1 "k8s.io/api/authorization/v1"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/endpoints/request"

	"github.com/tigera/compliance/pkg/datastore"
	"github.com/tigera/es-proxy/test/thirdpartymock"
	"github.com/tigera/lma/pkg/api"
	lmaauth "github.com/tigera/lma/pkg/auth"

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
				Entry("when the action parameter is missing", createFlowLogRequest(map[string][]string{
					"cluster": {"cluster"}, "srcType": {api.FlowLogEndpointTypeWEP}, "srcNamespace": {"default"}, "srcName": {"source"},
					"dstType": {api.FlowLogEndpointTypeWEP}, "dstNamespace": {"default"}, "dstName": {"destination"}, "reporter": {"dst"},
				}), 400, "missing required parameter 'action'"),
				Entry("when the cluster parameter is missing", createFlowLogRequest(map[string][]string{
					"action": {"deny"}, "srcType": {api.FlowLogEndpointTypeWEP}, "srcNamespace": {"default"}, "srcName": {"source"},
					"dstType": {api.FlowLogEndpointTypeWEP}, "dstNamespace": {"default"}, "dstName": {"destination"}, "reporter": {"dst"},
				}), 400, "missing required parameter 'cluster'"),
				Entry("when the srcType parameter is missing", createFlowLogRequest(map[string][]string{
					"action": {"deny"}, "cluster": {"cluster"}, "srcNamespace": {"default"}, "srcName": {"source"},
					"dstType": {api.FlowLogEndpointTypeWEP}, "dstNamespace": {"default"}, "dstName": {"destination"}, "reporter": {"dst"},
				}), 400, "missing required parameter 'srcType'"),
				Entry("when the srcNamespace parameter is missing", createFlowLogRequest(map[string][]string{
					"action": {"deny"}, "cluster": {"cluster"}, "srcType": {api.FlowLogEndpointTypeWEP}, "srcName": {"source"},
					"dstType": {api.FlowLogEndpointTypeWEP}, "dstNamespace": {"default"}, "dstName": {"destination"}, "reporter": {"dst"},
				}), 400, "missing required parameter 'srcNamespace'"),
				Entry("when the srcName parameter is missing", createFlowLogRequest(map[string][]string{
					"action": {"deny"}, "cluster": {"cluster"}, "srcType": {api.FlowLogEndpointTypeWEP}, "srcNamespace": {"default"},
					"dstType": {api.FlowLogEndpointTypeWEP}, "dstNamespace": {"default"}, "dstName": {"destination"}, "reporter": {"dst"},
				}), 400, "missing required parameter 'srcName'"),
				Entry("when the dstType parameter is missing and the dstType is wep", createFlowLogRequest(map[string][]string{
					"action": {"deny"}, "cluster": {"cluster"}, "srcType": {api.FlowLogEndpointTypeWEP}, "srcNamespace": {"default"},
					"srcName": {"source"}, "dstNamespace": {"default"}, "dstName": {"destination"}, "reporter": {"dst"},
				}), 400, "missing required parameter 'dstType'"),
				Entry("when the dstNamespace parameter is missing", createFlowLogRequest(map[string][]string{
					"action": {"deny"}, "cluster": {"cluster"}, "srcType": {api.FlowLogEndpointTypeWEP}, "srcNamespace": {"default"},
					"srcName": {"source"}, "dstType": {api.FlowLogEndpointTypeWEP}, "dstName": {"destination"}, "reporter": {"dst"},
				}), 400, "missing required parameter 'dstNamespace'"),
				Entry("when the dstName parameter is missing", createFlowLogRequest(map[string][]string{
					"action": {"deny"}, "cluster": {"cluster"}, "srcType": {api.FlowLogEndpointTypeWEP}, "srcNamespace": {"default"},
					"srcName": {"source"}, "dstType": {api.FlowLogEndpointTypeWEP}, "dstNamespace": {"default"}, "reporter": {"dst"},
				}), 400, "missing required parameter 'dstName'"),
				Entry("when the reporter parameter is missing", createFlowLogRequest(map[string][]string{
					"cluster": {"cluster"}, "srcType": {api.FlowLogEndpointTypeWEP}, "srcNamespace": {"default"}, "srcName": {"source"},
					"dstType": {api.FlowLogEndpointTypeWEP}, "dstNamespace": {"default"}, "dstName": {"destination"},
				}), 400, "missing required parameter 'reporter'"),
				Entry("when startDateTime is set but not in the RFC3339 format", createFlowLogRequest(map[string][]string{
					"action": {"deny"}, "cluster": {"cluster"}, "srcType": {api.FlowLogEndpointTypeWEP}, "srcNamespace": {"default"},
					"srcName": {"source"}, "dstType": {api.FlowLogEndpointTypeWEP}, "dstNamespace": {"default"}, "dstName": {"destination"},
					"reporter": {"dst"}, "startDateTime": {"invalid-start-date-time"},
				}), 400, "failed to parse 'startDateTime' value 'invalid-start-date-time' as RFC3339 datetime"),
				Entry("when endDateTime is set but not in the RFC3339 format", createFlowLogRequest(map[string][]string{
					"action": {"deny"}, "cluster": {"cluster"}, "srcType": {api.FlowLogEndpointTypeWEP}, "srcNamespace": {"default"},
					"srcName": {"source"}, "dstType": {api.FlowLogEndpointTypeWEP}, "dstNamespace": {"default"}, "dstName": {"destination"},
					"reporter": {"dst"}, "endDateTime": {"invalid-end-date-time"},
				}), 400, "failed to parse 'endDateTime' value 'invalid-end-date-time' as RFC3339 datetime"),
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
					"action": {"deny"}, "cluster": {"cluster"}, "srcType": {api.FlowLogEndpointTypeWEP}, "srcNamespace": {"default"},
					"srcName": {"source"}, "dstType": {api.FlowLogEndpointTypeWEP}, "dstNamespace": {"default"}, "dstName": {"destination"},
					"reporter": {"dst"},
				})),
			)
		})
	})

	When("no results are returned from elasticsearch", func() {
		It("returns a 404", func() {
			req := createFlowLogRequest(map[string][]string{
				"action": {"deny"}, "cluster": {"cluster"}, "srcType": {api.FlowLogEndpointTypeWEP}, "srcNamespace": {"default"},
				"srcName": {"source"}, "dstType": {api.FlowLogEndpointTypeHEP}, "dstName": {"destination"},
				"dstNamespace": {api.GlobalEndpointType}, "reporter": {"dst"},
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
				"policies": calicojson.Map{
					"aggregations": calicojson.Map{"by_tiered_policy": calicojson.Map{"terms": calicojson.Map{"field": "policies.all_policies"}}},
					"nested":       calicojson.Map{"path": "policies"},
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
				"action": {"deny"}, "cluster": {"cluster"}, "srcType": {api.FlowLogEndpointTypeHEP}, "srcName": {"source"},
				"srcNamespace": {api.GlobalEndpointType}, "dstType": {api.FlowLogEndpointTypeWEP}, "dstNamespace": {"default"},
				"dstName": {"destination"}, "reporter": {"dst"},
			}),

			calicojson.Map{"bool": calicojson.Map{
				"filter": []calicojson.Map{
					{"term": calicojson.Map{"action": "deny"}},
					{"term": calicojson.Map{"source_type": "hep"}},
					{"term": calicojson.Map{"source_name_aggr": "source"}},
					{"term": calicojson.Map{"source_namespace": api.GlobalEndpointType}},
					{"term": calicojson.Map{"dest_type": "wep"}},
					{"term": calicojson.Map{"dest_name_aggr": "destination"}},
					{"term": calicojson.Map{"dest_namespace": "default"}},
					{"term": calicojson.Map{"reporter": "dst"}},
				}},
			},
		),
		Entry("when startDateTime and endDateTime are specified",
			createFlowLogRequest(map[string][]string{
				"action": {"deny"}, "cluster": {"cluster"}, "srcType": {api.FlowLogEndpointTypeWEP}, "srcNamespace": {"default"},
				"srcName": {"source"}, "dstType": {api.FlowLogEndpointTypeWEP}, "dstNamespace": {"default"}, "dstName": {"destination"},
				"reporter": {"dst"}, "startDateTime": {"2006-01-02T13:04:05Z"}, "endDateTime": {"2006-01-02T15:04:05Z"},
			}),

			calicojson.Map{"bool": calicojson.Map{
				"filter": []calicojson.Map{
					{"term": calicojson.Map{"action": "deny"}},
					{"term": calicojson.Map{"source_type": "wep"}},
					{"term": calicojson.Map{"source_name_aggr": "source"}},
					{"term": calicojson.Map{"source_namespace": "default"}},
					{"term": calicojson.Map{"dest_type": "wep"}},
					{"term": calicojson.Map{"dest_name_aggr": "destination"}},
					{"term": calicojson.Map{"dest_namespace": "default"}},
					{"term": calicojson.Map{"reporter": "dst"}},
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
				"action": {"deny"}, "cluster": {"cluster"}, "srcType": {api.FlowLogEndpointTypeWEP}, "srcNamespace": {"default"},
				"srcName": {"source"}, "dstType": {api.FlowLogEndpointTypeWEP}, "dstNamespace": {"default"}, "dstName": {"destination"},
				"reporter":  {"dst"},
				"srcLabels": {createLabelJson("srcname", "=", []string{"srcfoo"}), createLabelJson("srcotherlabel", "!=", []string{"srcbar"})},
				"dstLabels": {createLabelJson("dstname", "=", []string{"srcfoo"}), createLabelJson("dstotherlabel", "!=", []string{"dstbar"})},
			}),
			calicojson.Map{"bool": calicojson.Map{
				"filter": []calicojson.Map{
					{"term": calicojson.Map{"action": "deny"}},
					{"term": calicojson.Map{"source_type": "wep"}},
					{"term": calicojson.Map{"source_name_aggr": "source"}},
					{"term": calicojson.Map{"source_namespace": "default"}},
					{"term": calicojson.Map{"dest_type": "wep"}},
					{"term": calicojson.Map{"dest_name_aggr": "destination"}},
					{"term": calicojson.Map{"dest_namespace": "default"}},
					{"term": calicojson.Map{"reporter": "dst"}},
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
				"reporter": {"dst"},
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
					"action": {"deny"}, "cluster": {"cluster"}, "srcType": {srcType}, "srcNamespace": {srcNamespace},
					"srcName": {"source"}, "dstType": {dstType}, "dstNamespace": {dstNamespace}, "dstName": {"destination"},
					"reporter": {"dst"},
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
					"dstType": {dstType}, "dstNamespace": {dstNamespace}, "dstName": {"destination"}, "reporter": {"dst"},
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
				"dstType": {"wep"}, "dstNamespace": {"destination-ns"}, "dstName": {"destination"}, "reporter": {"dst"},
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
			DescribeTable("parsing policies hits when completely authorized",
				func(buckets []map[string]interface{}, expectedPolicies []*FlowResponsePolicy) {
					mockRBACAuthoriser.On("Authorize", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)

					mockDoer.On("Do", mock.Anything).Return(&http.Response{
						StatusCode: http.StatusOK,
						Body: esSearchResultToResponseBody(elastic.SearchResult{
							Hits: &elastic.SearchHits{TotalHits: &elastic.TotalHits{Value: 1}},
							Aggregations: elastic.Aggregations{
								"policies": calicojson.MustMarshal(map[string]interface{}{
									"by_tiered_policy": map[string]interface{}{
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
						Count:    1,
						Policies: expectedPolicies,
					}))
				},
				Entry("single policy hit",
					[]map[string]interface{}{
						{"key": "0|tier1|namespace1/policy1|allow", "doc_count": 1},
					},
					[]*FlowResponsePolicy{
						{Index: 0, Namespace: "namespace1", Tier: "tier1", Name: "policy1", Action: "allow", Count: 1},
					},
				),
				Entry("multiple policy hits",
					[]map[string]interface{}{
						{"key": "0|tier1|namespace1/tier1.policy1|pass", "doc_count": 1},
						{"key": "1|tier2|namespace2/tier2.staged:policy2|deny", "doc_count": 1},
						{"key": "2|tier2|namespace2/tier2.policy3|pass", "doc_count": 1},
						{"key": "3|tier3|namespace3/tier3.policy4|pass", "doc_count": 1},
						{"key": "4|tier4|namespace4/tier4.policy5|deny", "doc_count": 1},
					},
					[]*FlowResponsePolicy{
						{Index: 0, Namespace: "namespace1", Tier: "tier1", Name: "policy1", Action: "pass", Count: 1},
						{Index: 1, Staged: true, Namespace: "namespace2", Tier: "tier2", Name: "policy2", Action: "deny", Count: 1},
						{Index: 2, Namespace: "namespace2", Tier: "tier2", Name: "policy3", Action: "pass", Count: 1},
						{Index: 3, Namespace: "namespace3", Tier: "tier3", Name: "policy4", Action: "pass", Count: 1},
						{Index: 4, Namespace: "namespace4", Tier: "tier4", Name: "policy5", Action: "deny", Count: 1},
					},
				),
			)

			DescribeTable("obfuscating policies",
				func(buckets []map[string]interface{}, expectedPolicies []*FlowResponsePolicy, authResources []*authzv1.ResourceAttributes) {
					for _, resource := range authResources {
						mockRBACAuthoriser.On("Authorize", mock.Anything, resource, mock.Anything).Return(true, nil)
					}

					mockRBACAuthoriser.On("Authorize", mock.Anything, mock.Anything, mock.Anything).Return(false, nil).Maybe()
					mockDoer.On("Do", mock.Anything).Return(&http.Response{
						StatusCode: http.StatusOK,
						Body: esSearchResultToResponseBody(elastic.SearchResult{
							Hits: &elastic.SearchHits{TotalHits: &elastic.TotalHits{Value: 1}},
							Aggregations: elastic.Aggregations{
								"policies": calicojson.MustMarshal(map[string]interface{}{
									"by_tiered_policy": map[string]interface{}{
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
						Count:    1,
						Policies: expectedPolicies,
					}))
				},
				Entry("single obfuscated policy hit",
					[]map[string]interface{}{
						{"key": "0|tier1|namespace/policy|allow", "doc_count": 1},
					},
					[]*FlowResponsePolicy{
						{Index: 0, Namespace: "*", Tier: "*", Name: "*", Action: "allow", Count: 1},
					},
					[]*authzv1.ResourceAttributes{
						{Namespace: "source-ns", Verb: "list", Resource: "pods"},
						{Namespace: "destination-ns", Verb: "list", Resource: "pods"},
					},
				),
				Entry("multiple obfuscated passes before non obfuscated deny",
					[]map[string]interface{}{
						{"key": "0|tier1|namespace/tier1.policy1|pass", "doc_count": 1},
						{"key": "1|tier2|namespace/tier2.policy2|pass", "doc_count": 1},
						{"key": "2|tier3|namespace/tier3.policy3|deny", "doc_count": 1},
					},
					[]*FlowResponsePolicy{
						{Index: 0, Namespace: "*", Tier: "*", Name: "*", Action: "pass", Count: 2},
						{Index: 1, Namespace: "namespace", Tier: "tier3", Name: "policy3", Action: "deny", Count: 1},
					},
					[]*authzv1.ResourceAttributes{
						{Namespace: "source-ns", Verb: "list", Resource: "pods"},
						{Namespace: "destination-ns", Verb: "list", Resource: "pods"},
						{Verb: "get", Group: "projectcalico.org", Resource: "tiers", Name: "tier3"},
						{Namespace: "namespace", Verb: "list", Group: "projectcalico.org", Resource: "tier.networkpolicies"},
					},
				),
				Entry("multiple obfuscated passes before obfuscated deny",
					[]map[string]interface{}{
						{"key": "0|tier1|namespace/tier1.policy1|pass", "doc_count": 1},
						{"key": "1|tier2|namespace/tier2.policy2|pass", "doc_count": 1},
						{"key": "2|tier3|namespace/tier3.policy3|deny", "doc_count": 1},
					},
					[]*FlowResponsePolicy{
						{Index: 0, Namespace: "*", Tier: "*", Name: "*", Action: "deny", Count: 3},
					},
					[]*authzv1.ResourceAttributes{
						{Namespace: "source-ns", Verb: "list", Resource: "pods"},
						{Namespace: "destination-ns", Verb: "list", Resource: "pods"},
					},
				),
				Entry("multiple obfuscated passes before non obfuscated staged deny before obfuscated deny",
					[]map[string]interface{}{
						{"key": "0|tier1|namespace/tier1.policy1|pass", "doc_count": 1},
						{"key": "1|tier2|namespace/tier2.policy2|pass", "doc_count": 1},
						{"key": "2|tier3|namespace/tier3.staged:policy3|deny", "doc_count": 1},
						{"key": "3|tier3|namespace1/tier3.policy4|pass", "doc_count": 1},
						{"key": "4|tier4|namespace/tier4.policy5|deny", "doc_count": 1},
					},
					[]*FlowResponsePolicy{
						{Index: 0, Namespace: "*", Tier: "*", Name: "*", Action: "pass", Count: 2},
						{Index: 1, Staged: true, Namespace: "namespace", Tier: "tier3", Name: "policy3", Action: "deny", Count: 1},
						{Index: 2, Namespace: "*", Tier: "*", Name: "*", Action: "deny", Count: 2},
					},
					[]*authzv1.ResourceAttributes{
						{Namespace: "source-ns", Verb: "list", Resource: "pods"},
						{Namespace: "destination-ns", Verb: "list", Resource: "pods"},
						{Verb: "get", Group: "projectcalico.org", Resource: "tiers", Name: "tier3"},
						{Namespace: "namespace", Verb: "list", Group: "projectcalico.org", Resource: "tier.stagednetworkpolicies"},
					},
				),
				Entry("omit obfuscated staged deny combine obfuscated pass and deny",
					[]map[string]interface{}{
						{"key": "0|tier1|namespace/tier1.staged:policy1|deny", "doc_count": 1},
						{"key": "1|tier1|namespace/tier1.policy2|pass", "doc_count": 1},
						{"key": "2|tier2|namespace/tier2.policy3|deny", "doc_count": 1},
					},
					[]*FlowResponsePolicy{
						{Index: 0, Namespace: "*", Tier: "*", Name: "*", Action: "deny", Count: 2},
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
