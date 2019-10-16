package policyrec_test

import (
	//"github.com/projectcalico/libcalico-go/lib/testutils"
	elastic "github.com/olivere/elastic/v7"
	"github.com/tigera/lma/pkg/policyrec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	testParams = &policyrec.PolicyRecommendationParams{
		StartTime:     "now-3h",
		EndTime:       "now-0h",
		EndpointName:  "test-app-pod",
		Namespace:     "test-namespace",
		DocumentIndex: "test-flow-log-index",
	}

	testQuery = `{
  "bool": {
    "must": [
      {"range": {"start_time": { "gte": "now-3h"}}},
      {"range": {"end_time": { "lte": "now-0h"}}},
      {"terms":{"source_type":["net","ns","wep","hep"]}},
      {"terms":{"dest_type":["net","ns","wep","hep"]}},
      {"nested": {
        "path": "policies",
        "query": {
          "wildcard": {
            "policies.all_policies": {
              "value": "*|__PROFILE__|__PROFILE__.kns.test-namespace|allow"
            }
          }
        }
      }},
      {"bool": {
        "should": [
          {"bool": {
            "must": [
              {"term": {"source_name_aggr": "test-app-pod"}},
              {"term": {"source_namespace": "test-namespace"}}
            ]
          }},
          {"bool": {
            "must": [
              {"term": {"dest_name_aggr": "test-app-pod"}},
              {"term": {"dest_namespace": "test-namespace"}}
            ]
          }}
        ]
      }}
    ]
  }
}`
)

var _ = Describe("Policy Recommendation Unit Tests for functions interfacing with elasticsearch", func() {
	It("Should create a valid elasticsearch query with valid PolicyRecommendationParams", func() {
		query := policyrec.BuildElasticQuery(testParams)
		Expect(elastic.NewRawStringQuery(testQuery)).To(Equal(query))
	})
})
