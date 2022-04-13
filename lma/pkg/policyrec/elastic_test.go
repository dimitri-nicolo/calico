// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.
package policyrec_test

import (
	"fmt"

	elastic "github.com/olivere/elastic/v7"

	"github.com/projectcalico/calico/lma/pkg/policyrec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	testParamsUnprotected = &policyrec.PolicyRecommendationParams{
		StartTime:     "now-3h",
		EndTime:       "now-0h",
		EndpointName:  "test-app-pod",
		Namespace:     "test-namespace",
		Unprotected:   true,
		DocumentIndex: "test-flow-log-index",
	}

	testParams = &policyrec.PolicyRecommendationParams{
		StartTime:     "now-3h",
		EndTime:       "now-0h",
		EndpointName:  "test-app-pod",
		Namespace:     "test-namespace",
		Unprotected:   false,
		DocumentIndex: "test-flow-log-index",
	}
)

var _ = Describe("Policy Recommendation Unit Tests for functions interfacing with elasticsearch", func() {
	It("Should create a valid elasticsearch query with valid PolicyRecommendationParams", func() {
		By("Validating an unprotected query")
		query := policyrec.BuildElasticQuery(testParamsUnprotected)
		boolQuery, ok := query.(*elastic.BoolQuery)
		Expect(ok).To(BeTrue())
		matchBoolTopLevelQuery(boolQuery, testParamsUnprotected)

		By("Validating a query for all traffic in the default tier")
		query = policyrec.BuildElasticQuery(testParams)
		boolQuery, ok = query.(*elastic.BoolQuery)
		Expect(ok).To(BeTrue())
		matchBoolTopLevelQuery(boolQuery, testParams)
	})
})

// Matches a flow log filtering query of the form
// {
//   "bool": {
//     "must": [
//       {"range": {"start_time": { "gte": "now-3h"}}},
//       {"range": {"end_time": { "lte": "now-0h"}}},
//       {"terms":{"source_type":["net","ns","wep","hep"]}},
//       {"terms":{"dest_type":["net","ns","wep","hep"]}},
//       {"nested": {
//         "path": "policies",
//         "query": {
//           "wildcard": {
//             "policies.all_policies": {
//               "value": "*|default|*|*"
//             }
//           }
//         }
//       }},
//       {"bool": {
//         "should": [
//           {"bool": {
//             "must": [
//               {"term": {"source_name_aggr": "test-app-pod"}},
//               {"term": {"source_namespace": "test-namespace"}}
//             ]
//           }},
//           {"bool": {
//             "must": [
//               {"term": {"dest_name_aggr": "test-app-pod"}},
//               {"term": {"dest_namespace": "test-namespace"}}
//             ]
//           }}
//         ]
//       }}
//     ]
//   }
// }
func matchBoolTopLevelQuery(boolQuery *elastic.BoolQuery, params *policyrec.PolicyRecommendationParams) {
	source, err := boolQuery.Source()
	Expect(err).To(BeNil())
	boolQuerySource, ok := source.(map[string]interface{})
	Expect(ok).To(BeTrue())

	boolQueryClauses, ok := boolQuerySource["bool"].(map[string]interface{})
	Expect(ok).To(BeTrue())
	Expect(len(boolQueryClauses)).To(Equal(1))

	mustQuery, ok := boolQueryClauses["must"].([]interface{})
	Expect(ok).To(BeTrue())

	for _, mustPart := range mustQuery {
		part, ok := mustPart.(map[string]interface{})
		Expect(ok).To(BeTrue())
		for k, v := range part {
			switch k {
			case "range":
				rangeQuery, ok := v.(map[string]interface{})
				Expect(ok).To(BeTrue())
				matchRange(rangeQuery, params)
			case "terms":
				termsQuery, ok := v.(map[string]interface{})
				Expect(ok).To(BeTrue())
				matchTermsType(termsQuery)
			case "nested":
				nestedQuery, ok := v.(map[string]interface{})
				Expect(ok).To(BeTrue())
				var wildcardedPolicyQuery string
				if params.Unprotected {
					wildcardedPolicyQuery = buildWildcardQuery("", params.Namespace)
					matchNestedQuery(nestedQuery, []string{wildcardedPolicyQuery})
				} else {
					defaultTierQuery := buildWildcardQuery("default", "")
					unprotectedQuery := buildWildcardQuery("", params.Namespace)
					matchNestedQuery(nestedQuery, []string{defaultTierQuery, unprotectedQuery})
				}
			case "bool":
				innerBoolQuery, ok := v.(map[string]interface{})
				Expect(ok).To(BeTrue())
				matchInnerBoolShouldQuery(innerBoolQuery, params)
			}
		}

	}
}

// Matches the range query  for "start_time" and "end_time"
// {"range": {"start_time": { "gte": "now-3h"}}},
func matchRange(rangeQuery map[string]interface{}, params *policyrec.PolicyRecommendationParams) {
	for rangeField, rangeParams := range rangeQuery {
		var timeField, timeParam string
		switch rangeField {
		case "start_time":
			timeField = "from"
			timeParam = params.StartTime
		case "end_time":
			timeField = "to"
			timeParam = params.EndTime
		}
		rangeParamsMap, ok := rangeParams.(map[string]interface{})
		Expect(ok).To(BeTrue())
		Expect(rangeParamsMap[timeField]).To(Equal(timeParam))
	}
}

// matches the terms query for endpoint types.
// {"terms":{"dest_type":["net","ns","wep","hep"]}},
func matchTermsType(termsQuery map[string]interface{}) {
	for _, termsParams := range termsQuery {
		termsParamsMap, ok := termsParams.([]interface{})
		Expect(ok).To(BeTrue())
		Expect(termsParamsMap).Should(ConsistOf([]string{"wep", "hep", "ns", "net"}))
	}
}

// Matches the nested query for policies
//      {"nested": {
//        "path": "policies",
//        "query": {
//          "wildcard": {
//            "policies.all_policies": {
//              "value": "*|default|*|*"
//            }
//          }
//        }
//      }},
//
//
// Will also match a nested multiple policy query. Note that the ordering
// of policies to be matched is important.
//     {"nested": {
//        "path": "policies",
//        "query": {
//          "bool": {
//            "should": [
//              {
//                "wildcard": {
//                  "policies.all_policies": {
//                  "value": "*|allow-tigera|*|*"
//                  }
//                }
//              },
//              {
//                "wildcard": {
//                  "policies.all_policies": {
//                  "value": "*|__PROFILE__|__PROFILE__.kns.tigera-intrusion-detection|allow"
//                  }
//                }
//              }
//            ]
//          }
//        }
func matchNestedQuery(nestedQuery map[string]interface{}, wildcardedPolicyQueries []string) {
	for nestedQueryKey, nestedQueryParams := range nestedQuery {
		switch nestedQueryKey {
		case "path":
			pathName, ok := nestedQueryParams.(string)
			Expect(ok).To(BeTrue())
			Expect(pathName).To(Equal("policies"))
		case "query":
			wildcardQuery, ok := nestedQueryParams.(map[string]interface{})
			Expect(ok).To(BeTrue())
			if len(wildcardedPolicyQueries) == 1 {
				matchWildcardQuery(wildcardQuery, wildcardedPolicyQueries[0])
			} else {
				nestedBoolQuery, ok := nestedQueryParams.(map[string]interface{})
				Expect(ok).To(BeTrue())
				boolQuery, ok := nestedBoolQuery["bool"].(map[string]interface{})
				Expect(ok).To(BeTrue())
				shouldQuery, ok := boolQuery["should"].([]interface{})
				Expect(ok).To(BeTrue())
				Expect(len(shouldQuery)).To(Equal(len(wildcardedPolicyQueries)))

				for i, shouldPart := range shouldQuery {
					part, ok := shouldPart.(map[string]interface{})
					Expect(ok).To(BeTrue())
					matchWildcardQuery(part, wildcardedPolicyQueries[i])
				}
			}
		}
	}
}

func matchWildcardQuery(wildcardQuery map[string]interface{}, wildcardedPolicyQuery string) {
	wildcardPolicies, ok := wildcardQuery["wildcard"].(map[string]interface{})
	Expect(ok).To(BeTrue())
	allPolicies, ok := wildcardPolicies["policies.all_policies"].(map[string]interface{})
	Expect(ok).To(BeTrue())
	actualQuery, ok := allPolicies["wildcard"].(string)
	Expect(ok).To(BeTrue())
	Expect(actualQuery).To(Equal(wildcardedPolicyQuery))
}

// Matches the inner "should" query
//      {"bool": {
//        "should": [
//          {"bool": {
//            "must": [
//              {"term": {"source_name_aggr": "test-app-pod"}},
//              {"term": {"source_namespace": "test-namespace"}}
//            ]
//          }},
//          {"bool": {
//            "must": [
//              {"term": {"dest_name_aggr": "test-app-pod"}},
//              {"term": {"dest_namespace": "test-namespace"}}
//            ]
//          }}
//        ]
//      }}
func matchInnerBoolShouldQuery(boolQuery map[string]interface{}, params *policyrec.PolicyRecommendationParams) {
	shouldQuery, ok := boolQuery["should"].([]interface{})
	Expect(ok).To(BeTrue())

	for _, shouldPart := range shouldQuery {
		part, ok := shouldPart.(map[string]interface{})
		Expect(ok).To(BeTrue())
		for k, v := range part {
			Expect(k).To(Equal("bool"))
			innerBoolQuery, ok := v.(map[string]interface{})
			Expect(ok).To(BeTrue())
			mustQuery, ok := innerBoolQuery["must"].([]interface{})
			Expect(ok).To(BeTrue())
			matchInnerMustQuery(mustQuery, params)
		}
	}
}

// Matches inner must queries such as (and corresponding source variant):
//            "must": [
//              {"term": {"dest_name_aggr": "test-app-pod"}},
//              {"term": {"dest_namespace": "test-namespace"}}
//            ]
func matchInnerMustQuery(mustQuery []interface{}, params *policyrec.PolicyRecommendationParams) {
	for _, mustPart := range mustQuery {
		part, ok := mustPart.(map[string]interface{})
		Expect(ok).To(BeTrue())
		for k, v := range part {
			Expect(k).To(Equal("term"))
			termQuery, ok := v.(map[string]interface{})
			Expect(ok).To(BeTrue())
			for tk, tv := range termQuery {
				switch tk {
				case "source_name_aggr", "dest_name_aggr":
					Expect(tv).To(Equal(params.EndpointName))
				case "source_namespace", "dest_namespace":
					Expect(tv).To(Equal(params.Namespace))

				}
			}
		}
	}
}

// Builds a query parameter for matching policies.
// it's an error to specify both tierName and namespace.
// if namespace is specified "*|__PROFILE__|__PROFILE__.kns.test-namespace|allow".
// this corresponds to the "unprotected" endpoints query.
// if tier is specified then returns QQ"*|default|*|*".
// this corresponds to all traffic in the default tier.
func buildWildcardQuery(tierName string, namespace string) string {
	if tierName != "" && namespace != "" {
		panic("Both tiername and namespace cannot be specified")
	} else if tierName != "" {
		return fmt.Sprintf("*|%s|*|*", tierName)
	} else {
		return fmt.Sprintf("*|__PROFILE__|__PROFILE__.kns.%s|allow*", namespace)
	}
}
