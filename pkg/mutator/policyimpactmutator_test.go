// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package mutator

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	"github.com/tigera/es-proxy/pkg/pip"
	"github.com/tigera/es-proxy/pkg/pip/flow"
)

func init() {
}

var _ = Describe("Test pip response hook modify response", func() {

	DescribeTable("modify response should replace flows with calculated flows",
		func(bodyIn, expectedBody string) {
			p := newMockPip()
			rh := NewPIPResponseHook(p)

			resp := mockHttpResponse(bodyIn)

			err := rh.ModifyResponse(resp)
			Expect(err).ToNot(HaveOccurred())

			out := responseBodyToString(*resp)

			Expect(out).To(MatchJSON(expectedBody))
		},

		// inorder for the test json to look correct before and after the request
		// need to remove the preview action tags from the input test body json
		// and insert the preview action tags into the expected test body json
		Entry("Evolve First", body(clearPreviewActionTags(bucketsA)), body(insertPreviewActionTags(bucketsB))),
		Entry("Evolve Second", body(clearPreviewActionTags(bucketsB)), body(insertPreviewActionTags(bucketsC))),
		Entry("No evolution", body(clearPreviewActionTags(bucketsC)), body(insertPreviewActionTags(bucketsC))),
	)

})

var _ = Describe("Pip return empty respones if no flows", func() {

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


func (p mockPip) Load(ctx context.Context, r []pip.ResourceChange) error {
	return nil
}

// CalculateFlowImpact satisfies the PIP interface
// this test mock version "Evolves" pokemon labels embeded in the
// test json.
// It will also add preview actions fields to select items
func (p mockPip) CalculateFlowImpact(ctx context.Context, flows []flow.Flow) ([]flow.Flow, error) {
	for i, f := range flows {
		flows[i].Source.Namespace = evolve(f.Source.Namespace)
		flows[i].Source.Name = evolve(f.Source.Namespace)
		flows[i].Dest.Namespace = evolve(f.Dest.Namespace)
		flows[i].Dest.Name = evolve(f.Dest.Name)
		flows[i].Proto = evolve(f.Proto)

		flows[i] = addPreviewAction(flows[i])

		log.Info("EVOLVED ", f.Source.Namespace, " TO ", flows[i].Source.Namespace)
	}
	return flows, nil
}

// evolves the pokemon strings in a particular order
func evolve(s string) string {
	e := func(str string, poke string, evld string) string {
		return strings.Replace(str, poke, evld, -1)
	}
	s = e(s, "Wortortle", "Blastoise")
	s = e(s, "Squirtle", "Wortortle")
	s = e(s, "Metapod", "Butterfree")
	s = e(s, "Caterpie", "Metapod")
	s = e(s, "Pidgeotto", "Pidgeot")
	s = e(s, "Pidgey", "Pidgeotto")
	return s
}

// inserts preview actions at particular places in certain requests
func addPreviewAction(flow flow.Flow) flow.Flow {
	if flow.Source.Name == "Metapod" {
		flow.PreviewAction = "allow"
	}
	if flow.Source.Name == "Blastoise" {
		flow.PreviewAction = "deny"
	}
	if flow.Source.Name == "Pidgeotto" {
		flow.PreviewAction = "unknown"
	}
	return flow
}

// mockHttpResponse the response object returned here simulates the response
// comming back from the round trip request to elastic search
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
		Action:  "update",
		Resource: &v3.NetworkPolicy{},
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

//create our various test buckets
var bucketsA = bucketsOfPokemon("Squirtle", "Caterpie", "Pidgey", "", "", "")
var bucketsB = bucketsOfPokemon("Wortortle", "Metapod", "Pidgeotto", "", "@@ACT_ALLOW@@", "@@ACT_UNKOWN@@")
var bucketsC = bucketsOfPokemon("Blastoise", "Butterfree", "Pidgeot", "@@ACT_DENY@@", "", "")

// inserts the bucket into the body template
func body(s string) string {
	ret := strings.Replace(bodyTemplate, "@@TEST_BUCKETS@@ ", s, 1)
	return ret
}

// removes the preview action @@ACT tags from the test json
func clearPreviewActionTags(s string) string {
	s = strings.Replace(s, "@@ACT_ALLOW@@", "", -1)
	s = strings.Replace(s, "@@ACT_DENY@@", "", -1)
	s = strings.Replace(s, "@@ACT_UNKOWN@@", "", -1)
	return s
}

// replaces the preview action @@ACT tags with the correct action json string
func insertPreviewActionTags(s string) string {
	s = strings.Replace(s, "@@ACT_ALLOW@@", ",\"preview_action\":\"allow\"", -1)
	s = strings.Replace(s, "@@ACT_DENY@@", ",\"preview_action\":\"deny\"", -1)
	s = strings.Replace(s, "@@ACT_UNKOWN@@", ",\"preview_action\":\"unknown\"", -1)
	return s
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
                "source_namespace": "@@PK1@@" , "source_name": "@@PK1@@" , 
                "dest_name": "@@PK1@@" , "dest_namespace": "@@PK1@@", "proto":"@@PK1@@"
                @@ACT1@@
                },
                "policies": {"by_tiered_policy": {"buckets": [
                            {"key": "0|default|stars/knp.default.default-deny|deny"},
                            {"key": "0|default|stars/knp.default.frontend-policy|deny"}]}},                
                "source_labels": { "by_kvpair": {"buckets": [{ "key": "role=k8s-apiserver-endpoints" }]}},
                "dest_labels": { "by_kvpair": {"buckets": [{ "key": "pod-template-hash=5b8c565cdd" }]}}                    
        },
        { "key": {
                "reporter":"src", "source_type": "wep", "dest_type": "wep", "dest_port": "9200", "action":"allow", 
                "source_namespace": "@@PK2@@" , "source_name": "@@PK2@@" , 
                "dest_name": "@@PK2@@" , "dest_namespace": "@@PK2@@", "proto":"@@PK2@@" 
                @@ACT2@@
                },
                "policies": { "by_tiered_policy":{"buckets":[{"key": "2|__PROFILE__|__PROFILE__.kns.calico-monitoring|allow"}]}},
                "source_labels": { "by_kvpair": { "buckets": [] } },
                "dest_labels": { "by_kvpair": { "buckets": [{ "key": "k8s-app=cnx-manager" }]}}
        },
        { "key": {
                "reporter":"src", "source_type": "wep", "dest_type": "wep", "dest_port": "9200", "action":"allow", 
                "source_namespace": "@@PK3@@" ,"source_name": "@@PK3@@" , 
                "dest_name": "@@PK3@@" , "dest_namespace": "@@PK3@@" , "proto":"@@PK3@@"
                @@ACT3@@
                },
                "policies": { "by_tiered_policy":{"buckets":[]}},
                "source_labels": { "by_kvpair": { "buckets": [] } },
                "dest_labels": { "by_kvpair": { "buckets": [] } }
        } 
    ]`

//this alters the buckets templates
func bucketsOfPokemon(pk1, pk2, pk3, act1, act2, act3 string) string {
	s := bucketsTemplate
	s = strings.Replace(s, "@@PK1@@", pk1, -1)
	s = strings.Replace(s, "@@PK2@@", pk2, -1)
	s = strings.Replace(s, "@@PK3@@", pk3, -1)
	s = strings.Replace(s, "@@ACT1@@", act1, -1)
	s = strings.Replace(s, "@@ACT2@@", act2, -1)
	s = strings.Replace(s, "@@ACT3@@", act3, -1)
	return s
}
