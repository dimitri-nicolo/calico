// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package fv_test

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("PolicyimpactFV Elasticsearch", func() {

	proxyScheme := getEnvOrDefaultString("TEST_PROXY_SCHEME", "https")
	proxyHost := getEnvOrDefaultString("TEST_PROXY_HOST", "127.0.0.1:8000")

	var client *http.Client

	BeforeEach(func() {
		// Scripts have already launched Elasticsearch and other containers
		// Just need to execute tests at this point

		tr := &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    30 * time.Second,
			DisableCompression: true,
			TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
		}
		client = &http.Client{Transport: tr}
	})

	// We only verify access from the clients point of view.
	DescribeTable("Users can only preview policy changes they can perform ",
		func(userAuth authInjector, postRequestBody string, expectedStatusCode int) {

			//build the request
			// for policy impact the es query is always a post to flows
			requestVerb := http.MethodPost
			reqPath := "tigera_secure_ee_flows*/_search"
			urlStr := fmt.Sprintf("%s://%s/%s", proxyScheme, proxyHost, reqPath)
			bodyreader := strings.NewReader(postRequestBody)
			req, err := http.NewRequest(requestVerb, urlStr, bodyreader)
			Expect(err).To(BeNil())

			//add the content and auth headers
			req.Header.Add("content-length", fmt.Sprintf("%d", len(postRequestBody)))
			req.Header.Add("Content-Type", "application/json")
			userAuth.setAuthHeader(req)

			//make the request (the item under test)
			resp, err := client.Do(req)

			//check expected response
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(expectedStatusCode))

		},
		Entry("Malformed request errors correctly", authFullCRUDDefault, badBody, http.StatusBadRequest),

		Entry("Full CRUD user can preview k8s policy create in default", authFullCRUDDefault, bodyCreateDefaultK8s, http.StatusOK),
		Entry("Full CRUD user can preview k8s policy update in default", authFullCRUDDefault, bodyUpdateDefaultK8s, http.StatusOK),
		Entry("Full CRUD user can preview k8s policy delete in default", authFullCRUDDefault, bodyDeleteDefaultK8s, http.StatusOK),

		Entry("Read Only user cannot preview k8s policy create in default", authReadOnlyDefault, bodyCreateDefaultK8s, http.StatusUnauthorized),
		Entry("Read Only user cannot preview k8s policy update in default", authReadOnlyDefault, bodyUpdateDefaultK8s, http.StatusUnauthorized),
		Entry("Read Only user cannot preview k8s policy delete in default", authReadOnlyDefault, bodyDeleteDefaultK8s, http.StatusUnauthorized),

		Entry("Read Create user can preview k8s policy create in default", authReadCreateDefault, bodyCreateDefaultK8s, http.StatusOK),
		Entry("Read Create user cannot preview k8s policy update in default", authReadCreateDefault, bodyUpdateDefaultK8s, http.StatusUnauthorized),
		Entry("Read Create user cannot preview k8s policy delete in default", authReadCreateDefault, bodyDeleteDefaultK8s, http.StatusUnauthorized),

		Entry("Read Update user cannot preview k8s policy create in default", authReadUpdateDefault, bodyCreateDefaultK8s, http.StatusUnauthorized),
		Entry("Read Update user can preview k8s policy update in default", authReadUpdateDefault, bodyUpdateDefaultK8s, http.StatusOK),
		Entry("Read Update user cannot preview k8s policy delete in default", authReadUpdateDefault, bodyDeleteDefaultK8s, http.StatusUnauthorized),

		Entry("Read Delete user cannot preview k8s policy create in default", authReadDeleteDefault, bodyCreateDefaultK8s, http.StatusUnauthorized),
		Entry("Read Delete user cannot preview k8s policy update in default", authReadDeleteDefault, bodyUpdateDefaultK8s, http.StatusUnauthorized),
		Entry("Read Delete user can preview k8s policy delete in default", authReadDeleteDefault, bodyDeleteDefaultK8s, http.StatusOK),

		Entry("Full CRUD user cannot preview k8s policy create in alt-ns", authFullCRUDDefault, bodyCreateAltNSK8s, http.StatusUnauthorized),
		Entry("Full CRUD user cannot preview k8s policy update in alt-ns", authFullCRUDDefault, bodyUpdateAltNSK8s, http.StatusUnauthorized),
		Entry("Full CRUD user cannot preview k8s policy delete in alt-ns", authFullCRUDDefault, bodyDeleteAltNSK8s, http.StatusUnauthorized),

		Entry("Read Create user cannot preview k8s policy create in alt-ns", authReadCreateDefault, bodyCreateAltNSK8s, http.StatusUnauthorized),
		Entry("Read Update user cannot preview k8s policy update in alt-ns", authReadUpdateDefault, bodyUpdateAltNSK8s, http.StatusUnauthorized),
		Entry("Read Delete user cannot preview k8s policy delete in alt-ns", authReadDeleteDefault, bodyDeleteAltNSK8s, http.StatusUnauthorized),

		Entry("Full CRUD user can preview Calico policy create in default", authFullCRUDDefault, bodyCreateDefaultCalico, http.StatusOK),
		Entry("Full CRUD user can preview Calico policy update in default", authFullCRUDDefault, bodyUpdateDefaultCalico, http.StatusOK),
		Entry("Full CRUD user can preview Calico policy delete in default", authFullCRUDDefault, bodyDeleteDefaultCalico, http.StatusOK),

		Entry("Read Only user cannot preview Calico policy create in default", authReadOnlyDefault, bodyCreateDefaultCalico, http.StatusUnauthorized),
		Entry("Read Only user cannot preview Calico policy update in default", authReadOnlyDefault, bodyUpdateDefaultCalico, http.StatusUnauthorized),
		Entry("Read Only user cannot preview Calico policy delete in default", authReadOnlyDefault, bodyDeleteDefaultCalico, http.StatusUnauthorized),

		Entry("Read Create user can preview Calico policy create in default", authReadCreateDefault, bodyCreateDefaultCalico, http.StatusOK),
		Entry("Read Create user cannot preview Calico policy update in default", authReadCreateDefault, bodyUpdateDefaultCalico, http.StatusUnauthorized),
		Entry("Read Create user cannot preview Calico policy delete in default", authReadCreateDefault, bodyDeleteDefaultCalico, http.StatusUnauthorized),

		Entry("Read Update user cannot preview Calico policy 'default.p-name' create in default", authReadUpdateDefault, bodyCreateDefaultCalico, http.StatusUnauthorized),
		Entry("Read Update user can preview Calico policy 'default.p-name' update in default", authReadUpdateDefault, bodyUpdateDefaultCalico, http.StatusOK),
		Entry("Read Update user cannot preview Calico policy 'default.p-name' delete in default", authReadUpdateDefault, bodyDeleteDefaultCalico, http.StatusUnauthorized),

		Entry("Read Update user cannot preview Calico policy 'default.different' create in default", authReadUpdateDefault, bodyCreateDefaultDifferentCalico, http.StatusUnauthorized),
		Entry("Read Update user can preview Calico policy 'default.different' update in default", authReadUpdateDefault, bodyUpdateDefaultDifferentCalico, http.StatusUnauthorized),
		Entry("Read Update user cannot preview Calico policy 'default.different' delete in default", authReadUpdateDefault, bodyDeleteDefaultDifferentCalico, http.StatusUnauthorized),

		Entry("Read Delete user cannot preview Calico policy create in default", authReadDeleteDefault, bodyCreateDefaultCalico, http.StatusUnauthorized),
		Entry("Read Delete user cannot preview Calico policy update in default", authReadDeleteDefault, bodyUpdateDefaultCalico, http.StatusUnauthorized),
		Entry("Read Delete user can preview Calico policy delete in default", authReadDeleteDefault, bodyDeleteDefaultCalico, http.StatusOK),

		Entry("Full CRUD user cannot preview Calico policy create in alt-ns", authFullCRUDDefault, bodyCreateAltNSCalico, http.StatusUnauthorized),
		Entry("Full CRUD user cannot preview Calico policy update in alt-ns", authFullCRUDDefault, bodyUpdateAltNSCalico, http.StatusUnauthorized),
		Entry("Full CRUD user cannot preview Calico policy delete in alt-ns", authFullCRUDDefault, bodyDeleteAltNSCalico, http.StatusUnauthorized),

		Entry("Read Create user cannot preview Calico policy create in alt-ns", authReadCreateDefault, bodyCreateAltNSCalico, http.StatusUnauthorized),
		Entry("Read Update user cannot preview Calico policy update in alt-ns", authReadUpdateDefault, bodyUpdateAltNSCalico, http.StatusUnauthorized),
		Entry("Read Delete user cannot preview Calico policy delete in alt-ns", authReadDeleteDefault, bodyDeleteAltNSCalico, http.StatusUnauthorized),

		Entry("Full CRUD user can preview Global policy create in default", authFullCRUDDefault, bodyCreateDefaultGlobal, http.StatusOK),
		Entry("Full CRUD user can preview Global policy update in default", authFullCRUDDefault, bodyUpdateDefaultGlobal, http.StatusOK),
		Entry("Full CRUD user can preview Global policy delete in default", authFullCRUDDefault, bodyDeleteDefaultGlobal, http.StatusOK),

		Entry("Read Only user cannot preview Global policy create", authReadOnlyDefault, bodyCreateDefaultGlobal, http.StatusUnauthorized),
		Entry("Read Only user cannot preview Global policy update", authReadOnlyDefault, bodyUpdateDefaultGlobal, http.StatusUnauthorized),
		Entry("Read Only user cannot preview Global policy delete", authReadOnlyDefault, bodyDeleteDefaultGlobal, http.StatusUnauthorized),

		Entry("Read Create user can preview Global policy create", authReadCreateDefault, bodyCreateDefaultGlobal, http.StatusOK),
		Entry("Read Create user cannot preview Global policy update", authReadCreateDefault, bodyUpdateDefaultGlobal, http.StatusUnauthorized),
		Entry("Read Create user cannot preview Global policy delete", authReadCreateDefault, bodyDeleteDefaultGlobal, http.StatusUnauthorized),

		Entry("Read Update user cannot preview Global policy create", authReadUpdateDefault, bodyCreateDefaultGlobal, http.StatusUnauthorized),
		Entry("Read Update user can preview Global policy update", authReadUpdateDefault, bodyUpdateDefaultGlobal, http.StatusOK),
		Entry("Read Update user cannot preview Global policy delete", authReadUpdateDefault, bodyDeleteDefaultGlobal, http.StatusUnauthorized),

		Entry("Read Delete user cannot preview Global policy create", authReadDeleteDefault, bodyCreateDefaultGlobal, http.StatusUnauthorized),
		Entry("Read Delete user cannot preview Global policy update", authReadDeleteDefault, bodyUpdateDefaultGlobal, http.StatusUnauthorized),
		Entry("Read Delete user can preview Global policy delete", authReadDeleteDefault, bodyDeleteDefaultGlobal, http.StatusOK),
	)
})

func modBody(b string, act string, ns string) string {
	b = strings.Replace(b, "@@ACTION@@", act, -1)
	b = strings.Replace(b, "@@NAMESPACE@@", ns, -1)
	return b
}

func patchVars(b string) string {
	b = strings.Replace(b, "@@QUERY@@", query, -1)
	b = strings.Replace(b, "@@PA_CALICO@@", calicoPolicyActions, -1)
	b = strings.Replace(b, "@@PA_K8S@@", k8sPolicyActions, -1)
	b = strings.Replace(b, "@@PA_GLOBAL@@", globalPolicyActions, -1)
	return b
}

var (
	authReadOnlyDefault   = basicAuthMech{"basicpolicyreadonly", "polreadonlypw"}
	authFullCRUDDefault   = basicAuthMech{"basicpolicycrud", "polcrudpw"}
	authReadCreateDefault = basicAuthMech{"basicpolicyreadcreate", "polreadcreatepw"}
	authReadUpdateDefault = basicAuthMech{"basicpolicyreadupdate", "polreadupdatepw"}
	authReadDeleteDefault = basicAuthMech{"basicpolicyreaddelete", "polreaddeletepw"}
)

var (
	bodyCreateDefaultK8s = modBody(baseBodyK8s, "create", "default")
	bodyUpdateDefaultK8s = modBody(baseBodyK8s, "update", "default")
	bodyDeleteDefaultK8s = modBody(baseBodyK8s, "delete", "default")
	bodyCreateAltNSK8s   = modBody(baseBodyK8s, "create", "alt-ns")
	bodyUpdateAltNSK8s   = modBody(baseBodyK8s, "update", "alt-ns")
	bodyDeleteAltNSK8s   = modBody(baseBodyK8s, "delete", "alt-ns")

	bodyCreateDefaultCalico = modBody(baseBodyCalico, "create", "default")
	bodyUpdateDefaultCalico = modBody(baseBodyCalico, "update", "default")
	bodyDeleteDefaultCalico = modBody(baseBodyCalico, "delete", "default")

	bodyCreateDefaultDifferentCalico = strings.Replace(bodyCreateDefaultCalico, "default.p-name", "default.different", -1)
	bodyUpdateDefaultDifferentCalico = strings.Replace(bodyUpdateDefaultCalico, "default.p-name", "default.different", -1)
	bodyDeleteDefaultDifferentCalico = strings.Replace(bodyDeleteDefaultCalico, "default.p-name", "default.different", -1)

	bodyCreateAltNSCalico = modBody(baseBodyCalico, "create", "alt-ns")
	bodyUpdateAltNSCalico = modBody(baseBodyCalico, "update", "alt-ns")
	bodyDeleteAltNSCalico = modBody(baseBodyCalico, "delete", "alt-ns")

	bodyCreateDefaultGlobal = modBody(baseBodyGlobal, "create", "")
	bodyUpdateDefaultGlobal = modBody(baseBodyGlobal, "update", "")
	bodyDeleteDefaultGlobal = modBody(baseBodyGlobal, "delete", "")
)

var (
	baseBodyK8s    = patchVars("{@@QUERY@@,@@PA_K8S@@}")
	baseBodyCalico = patchVars("{@@QUERY@@,@@PA_CALICO@@}")
	baseBodyGlobal = patchVars("{@@QUERY@@,@@PA_GLOBAL@@}")
)

var badBody = `{"query":{"bool":{"must":[{"match_all":{}}],"must_not":[],"should":[]}},"from":0,"size":10,"sort":[],"aggs":{},
"resourceActions":[{"resource":{ "apiVersion": "projectcalico.org/v3","kind":"NetworkPolicy", "spec":{ "order":"xyz" } } ,"action":"create"}] }`

var query = `"query":{"bool":{"must":[{"match_all":{}}],"must_not":[],"should":[]}},"from":0,"size":10,"sort":[],"aggs":{}`

var calicoPolicyActions = `"resourceActions":[{"resource":{
	"apiVersion": "projectcalico.org/v3",
	"kind":"NetworkPolicy",
	"metadata":{
		"name":"default.p-name",
		"generateName":"p-gen-name",
		"namespace":"@@NAMESPACE@@",
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
,"action":"@@ACTION@@"}]`

var k8sPolicyActions = `"resourceActions":[{"resource":{
	"apiVersion": "networking.k8s.io/v1",
	"kind": "NetworkPolicy",
	"metadata": {
		"name": "a-kubernetes-network-policy",
		"uid": "7dfbb617-a1ea-11e9-bd43-001c42e3cabd",
		"namespace": "@@NAMESPACE@@",
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
,"action":"@@ACTION@@"}]`

var globalPolicyActions = `"resourceActions":[{"resource":{
	"apiVersion": "projectcalico.org/v3",
	"kind": "GlobalNetworkPolicy",
	"metadata": {
		"name": "test.a-global-policy",
		"creationTimestamp": null
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
,"action":"@@ACTION@@"}]`
