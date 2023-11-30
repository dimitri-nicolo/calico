// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package flows_test

import (
	"testing"

	"github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/assert"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/flows"
)

func TestPolicyMatchQueryBuilder(t *testing.T) {
	type testResult struct {
		error     bool
		errorMsg  string
		boolQuery *elastic.BoolQuery
	}

	testcases := []struct {
		name          string
		policyMatches []v1.PolicyMatch
		testResult    testResult
	}{
		{
			name:          "error when there is an empty PolicyMatch",
			policyMatches: []v1.PolicyMatch{{}},
			testResult: testResult{
				error:     true,
				errorMsg:  "PolicyMatch passed to BuildPolicyMatchQuery cannot be empty",
				boolQuery: nil,
			},
		},
		{
			name:          "should no return error when the PolicyMatch slice is empty",
			policyMatches: []v1.PolicyMatch{},
			testResult: testResult{
				error:     false,
				errorMsg:  "",
				boolQuery: nil,
			},
		},
		{
			name: "return non-nil BoolQuery when valid PolicyMatch is passed",
			policyMatches: []v1.PolicyMatch{{
				Tier:   "default",
				Action: ActionPtr(v1.FlowActionDeny),
			}},
			testResult: testResult{
				error:     false,
				errorMsg:  "",
				boolQuery: elastic.NewBoolQuery(),
			},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			bq, err := flows.BuildPolicyMatchQuery(tt.policyMatches)

			if tt.testResult.error {
				assert.Error(t, err)
				assert.Equal(t, tt.testResult.errorMsg, err.Error())
				assert.Nil(t, bq)
			} else {
				assert.NoError(t, err)
				if tt.testResult.boolQuery == nil {
					assert.Nil(t, bq)
				} else {
					assert.NotNil(t, bq)
				}
			}
		})
	}
}
