// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package mutator

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/tigera/es-proxy/pkg/pip"
	"github.com/tigera/es-proxy/pkg/pip/policycalc"
	v1 "k8s.io/api/networking/v1"
)

var _ = Describe("Test pip response hook modify response", func() {

	It("modify response should replace flows with calculated flows", func() {
		p := newMockPip()
		rh := NewPIPResponseHook(p)

		bodyIn := body(replaceActions("", ""))
		resp := mockHttpResponse(bodyIn)

		err := rh.ModifyResponse(resp)
		Expect(err).ToNot(HaveOccurred())

		out := responseBodyToString(*resp)

		expectedBody := body(replaceActions(",\"preview_action\":\"deny\"", ",\"preview_action\":\"unknown\""))
		Expect(out).To(MatchJSON(expectedBody))
	})
})

var _ = Describe("Pip return empty responses if no flows", func() {

	DescribeTable("body should be empty",
		func(bodyIn, expectedBody string) {
			p := newMockPip()
			rh := NewPIPResponseHook(p)

			resp := mockHttpResponse(bodyIn)

			err := rh.ModifyResponse(resp)
			Expect(err).ToNot(HaveOccurred())

			out := responseBodyToString(*resp)

			Expect(out).To(BeEmpty())
		},

		Entry("Empty es response", emptyBody, ""),
	)
})

// mockpip ---------------------------------------------------
func newMockPip() pip.PIP {
	return &mockPip{}
}

type mockPip struct {
}

type mockCalc struct {
}

// GetPolicyCalculator satisfies the PIP interface
// this test mock returns the mock calculator
func (p mockPip) GetPolicyCalculator(ctx context.Context, r []pip.ResourceChange) (policycalc.PolicyCalculator, error) {
	return mockCalc{}, nil
}

// Action satisfies the PolicyCalculator interface
// It calculates the action to return based simply off the original action in the flow.
func (c mockCalc) Action(flow *policycalc.Flow) (processed bool, before, after policycalc.Action) {
	switch flow.Action {
	case policycalc.ActionAllow:
		return true, policycalc.ActionAllow, policycalc.ActionDeny
	case policycalc.ActionDeny:
		return true, policycalc.ActionDeny, policycalc.ActionIndeterminate
	}
	return true, flow.Action, flow.Action
}

// mockHttpResponse the response object returned here simulates the response
// coming back from the round trip request to elasticsearch
// the provided string will become the body of the response
// a dummy context containing a single empty policy change is inserted and
// ensures that policy impact mutator will make the call to the
// CalculateFlowImpact function
func mockHttpResponse(s string) *http.Response {
	r := &http.Response{}
	r.Request = &http.Request{}
	b := bytes.NewBufferString(s)
	body := ioutil.NopCloser(b)
	r.Body = body

	changes := make([]pip.ResourceChange, 1)
	changes[0] = pip.ResourceChange{
		Action:   "update",
		Resource: &v1.NetworkPolicy{},
	}

	//add the context
	ctx := context.WithValue(context.Background(), pip.PolicyImpactContextKey, changes)
	r.Request = r.Request.WithContext(ctx)
	return r
}

// helper function to extract the response body
func responseBodyToString(resp http.Response) string {
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	Expect(err).NotTo(HaveOccurred())
	return string(bodyBytes)
}

// es test response body json templates and generation -----------------------------

// inserts the bucket into the body template
func body(s string) string {
	ret := strings.Replace(bodyTemplate, "@@TEST_BUCKETS@@ ", s, 1)
	return ret
}

// this es response should result in an empty body
var emptyBody = `
{
	"took": 51,
	"timed_out": false,
	"_shards": { "total":5, "successful":5, "skipped":0, "failed":0 },
	"hits": { "total":0, "max_score":null, "hits":[] }
}
`

//body template that we stick the test buckets into
var bodyTemplate = `
{   "took": 27, "timed_out": false,
    "_shards": { "total": 5, "successful": 5, "skipped": 0, "failed": 0 },
    "hits": { "total": 125, "max_score": 0.0, "hits": [] },
    "aggregations": {
        "flog_buckets": {
            "after_key": {
                "source_type": "wep",
                "source_namespace": "stars",
                "source_name": "frontend-*",
                "dest_type": "wep",
                "dest_namespace": "stars",
                "dest_name": "frontend-*",
                "reporter": "src",
                "action": "allow"
            },
@@TEST_BUCKETS@@ 
}}}
`

var bucketsTemplate = `"buckets":[
        { "key": {
                "reporter":"src", "source_type": "wep", "dest_type": "wep", "dest_port": "9200", "action":"allow", 
                "source_namespace": "thing1" , "source_name": "thing1" , 
                "dest_name": "thing1" , "dest_namespace": "thing1", "proto":"thing1"
                @@ACT1@@
                },
                "policies": {"by_tiered_policy": {"buckets": [
                            {"key": "0|default|stars/knp.default.default-deny|deny"},
                            {"key": "0|default|stars/knp.default.frontend-policy|deny"}]}},                
                "source_labels": { "by_kvpair": {"buckets": [{ "key": "role=k8s-apiserver-endpoints" }]}},
                "dest_labels": { "by_kvpair": {"buckets": [{ "key": "pod-template-hash=5b8c565cdd" }]}}                    
        },
        { "key": {
                "reporter":"src", "source_type": "wep", "dest_type": "wep", "dest_port": "9200", "action":"deny", 
                "source_namespace": "thing2" , "source_name": "thing2" , 
                "dest_name": "thing2" , "dest_namespace": "thing2", "proto":"thing2" 
                @@ACT2@@
                },
                "policies": { "by_tiered_policy":{"buckets":[{"key": "2|__PROFILE__|__PROFILE__.kns.calico-monitoring|allow"}]}},
                "source_labels": { "by_kvpair": { "buckets": [] } },
                "dest_labels": { "by_kvpair": { "buckets": [{ "key": "k8s-app=cnx-manager" }]}}
        }
    ]`

// replaceAction replaces the action field
func replaceActions(act1, act2 string) string {
	s := bucketsTemplate
	s = strings.Replace(s, "@@ACT1@@", act1, -1)
	s = strings.Replace(s, "@@ACT2@@", act2, -1)
	return s
}
