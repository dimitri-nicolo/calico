// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package server_test

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/tigera/compliance/pkg/policyrec"
	lpolicyrec "github.com/tigera/lma/pkg/policyrec"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	recommendedPolicy = &policyrec.PolicyRecommendationResponse{
		Recommendation: &lpolicyrec.Recommendation{
			NetworkPolicies: []*v3.StagedNetworkPolicy{
				&v3.StagedNetworkPolicy{
					TypeMeta: metav1.TypeMeta{
						Kind:       v3.KindStagedNetworkPolicy,
						APIVersion: v3.GroupVersionCurrent,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "testNamespace",
					},
				},
			},
		},
	}

	errorWhenRecommending = &policyrec.PolicyRecommendationResponse{
		ErrorMessage: "Flows did not match",
	}

	goodParams = lpolicyrec.PolicyRecommendationParams{
		StartTime:    "now",
		EndTime:      "now-1m",
		EndpointName: "test-*",
		Namespace:    "testNamespace",
	}

	badParams = lpolicyrec.PolicyRecommendationParams{
		StartTime:    "now-1m",
		EndTime:      "now",
		EndpointName: "doesnt-exist-*",
		Namespace:    "unknown-namespace",
	}
)

var _ = Describe("Policy Recommendation Handler", func() {
	DescribeTable("Recommend policies for matching flows and endpoint",
		func(params lpolicyrec.PolicyRecommendationParams, response *policyrec.PolicyRecommendationResponse) {
			By("Starting a test server")
			t := startTester()

			By("Setting recommendation response")
			t.precResponses = response

			t.getRecommendation(http.StatusOK, params, response)
		},
		Entry("should return recommended policy", goodParams, recommendedPolicy),
		Entry("should return error response", badParams, errorWhenRecommending),
	)
})
