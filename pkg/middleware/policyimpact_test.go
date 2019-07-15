package middleware_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/tigera/es-proxy/pkg/middleware"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"net/http"
)

var _ = Describe("The PolicyImpactRequestProcessor properly modifies requests and extracts params", func() {

	DescribeTable("Extracts the correct parameters from the request",
		func(originalReqBody string, expectedModifiedBody string, expectedParams string) {

			//build the request
			b := []byte(originalReqBody)
			req, err := http.NewRequest(http.MethodPost, "", bytes.NewBuffer(b))
			Expect(err).NotTo(HaveOccurred())

			//pass it to the function under test
			p, err := middleware.PolicyImpactRequestProcessor(req)
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

				log.Info("Params JSON", paramsJson)

				Expect(paramsJson).To(MatchJSON(expectedParams))
			}

		},
		Entry("Not a pip request", nullESQuery, nullESQuery, ""),
		Entry("Calico pip request", pipCalicoRequestWithQuery, nullESQuery, pipCalicoExpectedParams),
		Entry("Global pip request", pipGlobalRequestWithQuery, nullESQuery, pipGlobalExpectedParams),
		Entry("K8s pip request", pipK8sRequestWithQuery, nullESQuery, pipK8sExpectedParams),
	)

})

func pipParamsToJson(pipParams middleware.PolicyImpactParams) string {
	RegisterFailHandler(Fail)
	b, err := json.Marshal(pipParams)
	Expect(err).ToNot(HaveOccurred())
	return string(b)
}

func jsonToPipParams(j string) *middleware.PolicyImpactParams {
	RegisterFailHandler(Fail)
	var data middleware.PolicyImpactParams
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
