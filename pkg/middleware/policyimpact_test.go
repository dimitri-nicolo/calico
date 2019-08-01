package middleware_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/tigera/es-proxy/pkg/middleware"
	"github.com/tigera/es-proxy/pkg/pip"
)

var _ = Describe("ExtractPolicyImpactParamsFromRequest properly modifies requests and extracts params", func() {
	DescribeTable("Extracts the correct parameters from the request",
		func(requestMethod string, originalReqBody string, expectedModifiedBody string, expectedParams string) {

			//build the request
			b := []byte(originalReqBody)
			req, err := http.NewRequest(requestMethod, "", bytes.NewBuffer(b))
			Expect(err).NotTo(HaveOccurred())

			//pass it to the function under test
			p, err := middleware.ExtractPolicyImpactParamsFromRequest("", req)
			Expect(err).NotTo(HaveOccurred())

			//read the request body after the call
			ob, err := ioutil.ReadAll(req.Body)
			Expect(err).NotTo(HaveOccurred())
			obs := string(ob)

			//check the final request body
			Expect(obs).To(MatchJSON(expectedModifiedBody))

			//check expected params
			if expectedParams == "" {
				Expect(p).To(BeNil())
			} else {
				paramsJson := pipParamsToJson(*p)
				Expect(paramsJson).To(MatchJSON(expectedParams))
			}

		},
		Entry("Not a pip request", http.MethodPost, nullESQuery, nullESQuery, ""),
		Entry("Calico pip request", http.MethodPost, pipCalicoRequestWithQuery, nullESQuery, pipCalicoExpectedParams),
		Entry("Global pip request", http.MethodPost, pipGlobalRequestWithQuery, nullESQuery, pipGlobalExpectedParams),
		Entry("K8s pip request", http.MethodPost, pipK8sRequestWithQuery, nullESQuery, pipK8sExpectedParams),
	)
})

var _ = Describe("Time parsing works", func() {
	It("Parses now without error", func() {
		now := time.Now()
		s := "now"
		t, err := middleware.ParseElasticsearchTime(now, &s)
		Expect(err).NotTo(HaveOccurred())
		Expect(t).NotTo(BeNil())
		Expect(now.Sub(*t)).To(BeZero())
	})

	It("Parses now - 0 without error", func() {
		now := time.Now()
		s := "now - 0"
		t, err := middleware.ParseElasticsearchTime(now, &s)
		Expect(err).NotTo(HaveOccurred())
		Expect(t).NotTo(BeNil())
		Expect(now.Sub(*t)).To(BeZero())
	})

	It("Parses now - 15m without error", func() {
		now := time.Now()
		s := "now - 15m"
		t, err := middleware.ParseElasticsearchTime(now, &s)
		Expect(err).NotTo(HaveOccurred())
		Expect(t).NotTo(BeNil())
		Expect(now.Sub(*t)).To(Equal(15 * time.Minute))
	})

	It("Parses now-10m without error", func() {
		now := time.Now()
		s := "now-10m"
		t, err := middleware.ParseElasticsearchTime(now, &s)
		Expect(err).NotTo(HaveOccurred())
		Expect(t).NotTo(BeNil())
		Expect(now.Sub(*t)).To(Equal(10 * time.Minute))
	})

	It("Parses now-100h without error", func() {
		now := time.Now()
		s := "now-100h"
		t, err := middleware.ParseElasticsearchTime(now, &s)
		Expect(err).NotTo(HaveOccurred())
		Expect(t).NotTo(BeNil())
		Expect(now.Sub(*t)).To(Equal(100 * time.Hour))
	})

	It("Parses now-3d without error", func() {
		now := time.Now()
		s := "now-3d"
		t, err := middleware.ParseElasticsearchTime(now, &s)
		Expect(err).NotTo(HaveOccurred())
		Expect(t).NotTo(BeNil())
		Expect(now.Sub(*t)).To(Equal(3 * 24 * time.Hour))
	})

	It("Does not parse now-32", func() {
		now := time.Now()
		s := "now-32"
		t, err := middleware.ParseElasticsearchTime(now, &s)
		Expect(err).To(HaveOccurred())
		Expect(t).To(BeNil())
	})

	It("Does not parse now-xxx", func() {
		now := time.Now()
		s := "now-xxx"
		t, err := middleware.ParseElasticsearchTime(now, &s)
		Expect(err).To(HaveOccurred())
		Expect(t).To(BeNil())
	})

	It("Parses an RFC3339 format time", func() {
		now := time.Now().UTC()
		s := now.Add(-5 * time.Second).UTC().Format(time.RFC3339)
		t, err := middleware.ParseElasticsearchTime(now, &s)
		Expect(err).NotTo(HaveOccurred())
		Expect(t).NotTo(BeNil())
		Expect(now.Sub(*t) / time.Second).To(BeEquivalentTo(5)) // Avoids ms accuracy in `now` but not in `t`.
	})
})

func pipParamsToJson(pipParams pip.PolicyImpactParams) string {
	RegisterFailHandler(Fail)
	b, err := json.Marshal(pipParams)
	Expect(err).ToNot(HaveOccurred())
	return string(b)
}

func jsonToPipParams(j string) *pip.PolicyImpactParams {
	RegisterFailHandler(Fail)
	var data pip.PolicyImpactParams
	err := json.Unmarshal([]byte(j), &data)
	Expect(err).ToNot(HaveOccurred())
	return &data
}

func patchVars(b string) string {
	b = strings.Replace(b, "@@QUERY@@", query, -1)
	b = strings.Replace(b, "@@PA_CALICO@@", calicoPolicyActions, -1)
	b = strings.Replace(b, "@@PA_K8S@@", k8sPolicyActions, -1)
	b = strings.Replace(b, "@@PA_GLOBAL@@", globalPolicyActions, -1)
	return b
}

var nullESQuery = patchVars("{@@QUERY@@}")

var pipCalicoRequestWithQuery = patchVars("{@@QUERY@@,@@PA_CALICO@@}")

var pipCalicoExpectedParams = patchVars("{@@PA_CALICO@@}")

var pipGlobalRequestWithQuery = patchVars("{@@QUERY@@,@@PA_GLOBAL@@}")

var pipGlobalExpectedParams = patchVars("{@@PA_GLOBAL@@}")

var pipK8sRequestWithQuery = patchVars("{@@QUERY@@,@@PA_K8S@@}")

var pipK8sExpectedParams = patchVars("{@@PA_K8S@@}")

var query = `"query":{"bool":{"must":[{"match_all":{}}],"must_not":[],"should":[]}},"from":0,"size":10,"sort":[],"aggs":{}`

var calicoPolicyActions = `"resourceActions":[{"resource":{
	"apiVersion": "projectcalico.org/v3",
	"kind":"NetworkPolicy",
	"metadata":{
		"name":"default.p-name",
		"generateName":"p-gen-name",
		"namespace":"p-name-space",
		"selfLink":"p-self-link",
		"resourceVersion":"p-res-ver",
		"creationTimestamp": null
	},
	"spec":{
		"tier":"default",
		"order":1,
		"selector":"a|bogus|selector|string"
	}
}
,"action":"create"}]`

var k8sPolicyActions = `"resourceActions":[{"resource":{
	"apiVersion": "networking.k8s.io/v1",
	"kind": "NetworkPolicy",
	"metadata": {
		"name": "a-kubernetes-network-policy",
		"uid": "7dfbb617-a1ea-11e9-bd43-001c42e3cabd",
		"namespace": "default",
		"resourceVersion": "758945",
		 "creationTimestamp": null
	},
	"spec": {
		"podSelector": {},
		"ingress": [
		{"from": [{"podSelector": {"matchLabels": {"color": "blue"}}}],
			"ports": [{"port": 111,"protocol": "TCP"}]},
		{"from": [{"namespaceSelector": {"matchExpressions": [{
			"key": "name","operator": "In","values": ["es-client-tigera-elasticsearch"]}
		]}}, {"podSelector": {}}]}],
		"policyTypes": ["Ingress"]
	}
}
,"action":"create"}]`

var globalPolicyActions = `"resourceActions":[{"resource":{
	"apiVersion": "projectcalico.org/v3",
	"kind": "GlobalNetworkPolicy",
	"metadata": {
		"creationTimestamp": null,
		"name": "test.a-global-policy"
	},
	"spec": {
		"tier": "test",
		"order": 100,
		"selector": "all()",
		"ingress": [{
			"action": "Allow",
			"source": {	"namespaceSelector": "name == \"kibana-tigera-elasticsearch\"" },
			"destination": {}
		}],
		"types": ["Ingress"]
	}
}
,"action":"create"}]`
