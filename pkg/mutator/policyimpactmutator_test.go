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
	"github.com/tigera/es-proxy/pkg/pip"
	"github.com/tigera/es-proxy/pkg/pip/flow"
)

func init() {
}

var _ = Describe("Test pip response hook modify response", func() {

	DescribeTable("modify response should replace flows with calculated flows",
		func(bodyIn, expectedBody string) {
			p := newMockPip()
			rh := NewResponseHook(p)

			resp := newHttpResponse(bodyIn)

			err := rh.ModifyResponse(resp)
			Expect(err).ToNot(HaveOccurred())

			out := responseBodyToString(*resp)

			Expect(removePreviewActions(out)).To(MatchJSON(expectedBody))

			//these are kind of a hack to keep from having to parse the json
			// but they show that the preview_action is being added
			if strings.Contains(out, "Metapod") {
				Expect(out).To(ContainSubstring("\"preview_action\":\"allow\""))
			}
			if strings.Contains(out, "Blastoise") {
				Expect(out).To(ContainSubstring("\"preview_action\":\"deny\""))
			}
			if strings.Contains(out, "Pidgeotto") {
				Expect(out).To(ContainSubstring("\"preview_action\":\"unknown\""))
			}

		},

		Entry("Evolve First", body(bucketsA), body(bucketsB)),
		Entry("Evovle Second", body(bucketsB), body(bucketsC)),
		Entry("No evolution", body(bucketsC), body(bucketsC)),
	)

})

var _ = Describe("Pip return empty respones if no flows", func() {

	DescribeTable("body should be empty",
		func(bodyIn, expectedBody string) {
			p := newMockPip()
			rh := NewResponseHook(p)

			resp := newHttpResponse(bodyIn)

			err := rh.ModifyResponse(resp)
			Expect(err).ToNot(HaveOccurred())

			out := responseBodyToString(*resp)

			Expect(out).To(BeEmpty())
		},

		Entry("Empty es response", emptyBody, ""),
	)
})

//we need to strip the preview action from the json so the json matcher works
func removePreviewActions(s string) string {
	s = strings.Replace(s, "\"preview_action\":\"allow\",", "", -1)
	s = strings.Replace(s, "\"preview_action\":\"deny\",", "", -1)
	s = strings.Replace(s, "\"preview_action\":\"unknown\",", "", -1)
	return s
}

// mockpip ---------------------------------------------------
func newMockPip() pip.PIP {
	return &mockPip{}
}

type mockPip struct {
}

func (p mockPip) CalculateFlowImpact(ctx context.Context, changes []pip.NetworkPolicyChange, flows []flow.Flow) ([]flow.Flow, error) {
	for i, f := range flows {
		flows[i].Src_NS = evolve(f.Src_NS)
		flows[i].Src_name = evolve(f.Src_NS)
		flows[i].Dest_NS = evolve(f.Dest_NS)
		flows[i].Dest_name = evolve(f.Dest_name)

		flows[i] = addPreviewAction(flows[i])

		log.Info("EVOLVED ", f.Src_NS, " TO ", flows[i].Src_NS)
	}
	return flows, nil
}

//mock http response
func newHttpResponse(s string) *http.Response {
	r := &http.Response{}
	b := bytes.NewBufferString(s)
	body := ioutil.NopCloser(b)
	r.Body = body
	return r
}

func responseBodyToString(resp http.Response) string {
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	Expect(err).NotTo(HaveOccurred())
	return string(bodyBytes)
}

func addPreviewAction(flow flow.Flow) flow.Flow {
	if flow.Src_name == "Metapod" {
		flow.PreviewAction = "allow"
	}
	if flow.Src_name == "Blastoise" {
		flow.PreviewAction = "deny"
	}
	if flow.Src_name == "Pidgeotto" {
		flow.PreviewAction = "unknown"
	}
	return flow
}

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
{
    "took": 27,
    "timed_out": false,
    "_shards": {
        "total": 5,
        "successful": 5,
        "skipped": 0,
        "failed": 0
    },
    "hits": {
        "total": 125,
        "max_score": 0.0,
        "hits": []
    },
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

var bucketsA = `"buckets":[
        { "key": {
                "reporter":"src",
                "source_type": "wep",
                "dest_type": "wep",
                "source_namespace": "Squirtle" ,
                "source_name": "Squirtle" , 
                "dest_name": "Squirtle" ,   
                "dest_namespace": "Squirtle" ,
                "dest_port": "9200",
                "action":"allow"
                },
                "policies": {"by_tiered_policy": {"buckets": [
                            {"key": "0|default|stars/knp.default.default-deny|deny"},
                            {"key": "0|default|stars/knp.default.frontend-policy|deny"}]}},                
                "source_labels": { "by_kvpair": {"buckets": [{ "key": "role=k8s-apiserver-endpoints" }]}},
                "dest_labels": { "by_kvpair": {"buckets": [{ "key": "pod-template-hash=5b8c565cdd" }]}}                    
        },
        { "key": {
                "reporter":"src",
                "source_type": "wep",
                "dest_type": "wep",
                "source_namespace": "Caterpie" ,
                "source_name": "Caterpie" , 
                "dest_name": "Caterpie" ,   
                "dest_namespace": "Caterpie" ,
                "dest_port": "9200",
                "action":"allow"
                },
                "policies": { "by_tiered_policy":{"buckets":[{"key": "2|__PROFILE__|__PROFILE__.kns.calico-monitoring|allow"}]}},
                "source_labels": { "by_kvpair": { "buckets": [] } },
                "dest_labels": { "by_kvpair": { "buckets": [{ "key": "k8s-app=cnx-manager" }]}}
        },
        { "key": {
                "reporter":"src",
                "source_type": "wep",
                "dest_type": "wep",
                "source_namespace": "Pidgey" ,
                "source_name": "Pidgey" , 
                "dest_name": "Pidgey" ,   
                "dest_namespace": "Pidgey" ,
                "dest_port": "9200",
                "action":"allow"
                },
                "policies": { "by_tiered_policy":{"buckets":[]}},
                "source_labels": { "by_kvpair": { "buckets": [] } },
                "dest_labels": { "by_kvpair": { "buckets": [] } }
        } 
    ]`

var bucketsB = `"buckets":[
        { "key": {
                "reporter":"src",
                "source_type": "wep",
                "dest_type": "wep",
                "source_namespace":"Wortortle" ,
                "source_name":"Wortortle" , 
                "dest_name":"Wortortle" ,   
                "dest_namespace":"Wortortle" ,
                "dest_port": "9200",
                "action":"allow"
                },
                "policies": {"by_tiered_policy": {"buckets": [
                            {"key": "0|default|stars/knp.default.default-deny|deny"},
                            {"key": "0|default|stars/knp.default.frontend-policy|deny"}]}},                
                "source_labels": { "by_kvpair": {"buckets": [{ "key": "role=k8s-apiserver-endpoints" }]}},
                "dest_labels": { "by_kvpair": {"buckets": [{ "key": "pod-template-hash=5b8c565cdd" }]}}                    
        },
        { "key": {
                "reporter":"src",
                "source_type": "wep",
                "dest_type": "wep",
                "source_namespace":"Metapod" ,
                "source_name":"Metapod" , 
                "dest_name":"Metapod" ,   
                "dest_namespace":"Metapod" ,
                "dest_port": "9200",
                "action":"allow"
                },
                "policies": { "by_tiered_policy":{"buckets":[{"key": "2|__PROFILE__|__PROFILE__.kns.calico-monitoring|allow"}]}},
                "source_labels": { "by_kvpair": { "buckets": [] } },
                "dest_labels": { "by_kvpair": { "buckets": [{ "key": "k8s-app=cnx-manager" }]}}
        },
        { "key": {
                "reporter":"src",
                "source_type": "wep",
                "dest_type": "wep",
                "source_namespace":"Pidgeotto" ,
                "source_name":"Pidgeotto" , 
                "dest_name":"Pidgeotto" ,   
                "dest_namespace":"Pidgeotto" ,
                "dest_port": "9200",
                "action":"allow"
                },
                "policies": { "by_tiered_policy":{"buckets":[]}},
                "source_labels": { "by_kvpair": { "buckets": [] } },
                "dest_labels": { "by_kvpair": { "buckets": [] } }

        } 
    ]`

var bucketsC = `"buckets": [
        { "key": {
                "reporter":"src",
                "source_type": "wep",
                "dest_type": "wep",
                "source_namespace":"Blastoise" ,
                "source_name":"Blastoise" , 
                "dest_name":"Blastoise" ,   
                "dest_namespace":"Blastoise" ,
                "dest_port": "9200",
                "action":"allow"
                },
                "policies": {"by_tiered_policy": {"buckets": [
                            {"key": "0|default|stars/knp.default.default-deny|deny"},
                            {"key": "0|default|stars/knp.default.frontend-policy|deny"}]}},                
                "source_labels": { "by_kvpair": {"buckets": [{ "key": "role=k8s-apiserver-endpoints" }]}},
                "dest_labels": { "by_kvpair": {"buckets": [{ "key": "pod-template-hash=5b8c565cdd" }]}}                    
        },
        { "key": {
                "reporter":"src",
                "source_type": "wep",
                "dest_type": "wep",
                "source_namespace":"Butterfree" ,
                "source_name":"Butterfree" , 
                "dest_name":"Butterfree" ,   
                "dest_namespace":"Butterfree" ,
                "dest_port": "9200",
                "action":"allow"
                },
                "policies": { "by_tiered_policy":{"buckets":[{"key": "2|__PROFILE__|__PROFILE__.kns.calico-monitoring|allow"}]}},
                "source_labels": { "by_kvpair": { "buckets": [] } },
                "dest_labels": { "by_kvpair": { "buckets": [{ "key": "k8s-app=cnx-manager" }]}}
        },
        { "key": {
                "reporter":"src",
                "source_type": "wep",
                "dest_type": "wep",
                "source_namespace":"Pidgeot" ,
                "source_name":"Pidgeot" , 
                "dest_name":"Pidgeot" ,   
                "dest_namespace":"Pidgeot" ,
                "dest_port": "9200",
                "action":"allow"
                },
                "policies": { "by_tiered_policy":{"buckets":[]}},
                "source_labels": { "by_kvpair": { "buckets": [] } },
                "dest_labels": { "by_kvpair": { "buckets": [] } }

        } 
    ]`
