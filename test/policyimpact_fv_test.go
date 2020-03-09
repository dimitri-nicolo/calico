// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package fv_test

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("PolicyimpactFV Elasticsearch PIP", func() {

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
		func(userAuth authInjector, params url.Values, expectedStatusCode int) {

			//build the request
			// for policy impact the es query is always a post to flows
			requestVerb := http.MethodGet
			reqPath := "flowLogs"
			urlStr := fmt.Sprintf("%s://%s/%s?%s", proxyScheme, proxyHost, reqPath, params.Encode())
			req, err := http.NewRequest(requestVerb, urlStr, nil)
			Expect(err).To(BeNil())

			//add the auth header
			userAuth.setAuthHeader(req)

			//make the request (the item under test)
			resp, err := client.Do(req)

			//check expected response
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(expectedStatusCode))

		},

		Entry("Full CRUD user can preview Calico policy create in default", authFullCRUDDefault, bodyCreateDefaultCalico, http.StatusOK),
		Entry("Full CRUD user can preview Calico policy update in default", authFullCRUDDefault, bodyUpdateDefaultCalico, http.StatusOK),
		Entry("Full CRUD user can preview Calico policy delete in default", authFullCRUDDefault, bodyDeleteDefaultCalico, http.StatusOK),

		Entry("Malformed request errors correctly", authFullCRUDDefault, badParams, http.StatusBadRequest),
		Entry("Invalid action errors correctly", authReadCreateDefault, bodyInvalidAction, http.StatusUnprocessableEntity),
		Entry("Missing action errors correctly", authReadCreateDefault, bodyMissingAction, http.StatusUnprocessableEntity),

		Entry("Full CRUD user can preview k8s policy create in default", authFullCRUDDefault, bodyCreateDefaultK8s, http.StatusOK),
		Entry("Full CRUD user can preview k8s policy update in default", authFullCRUDDefault, bodyUpdateDefaultK8s, http.StatusOK),
		Entry("Full CRUD user can preview k8s policy delete in default", authFullCRUDDefault, bodyDeleteDefaultK8s, http.StatusOK),

		Entry("Read Only user cannot preview k8s policy create in default", authReadOnlyDefault, bodyCreateDefaultK8s, http.StatusForbidden),
		Entry("Read Only user cannot preview k8s policy update in default", authReadOnlyDefault, bodyUpdateDefaultK8s, http.StatusForbidden),
		Entry("Read Only user cannot preview k8s policy delete in default", authReadOnlyDefault, bodyDeleteDefaultK8s, http.StatusForbidden),

		Entry("Read Create user can preview k8s policy create in default", authReadCreateDefault, bodyCreateDefaultK8s, http.StatusOK),
		Entry("Read Create user cannot preview k8s policy update in default", authReadCreateDefault, bodyUpdateDefaultK8s, http.StatusForbidden),
		Entry("Read Create user cannot preview k8s policy delete in default", authReadCreateDefault, bodyDeleteDefaultK8s, http.StatusForbidden),

		Entry("Read Update user cannot preview k8s policy create in default", authReadUpdateDefault, bodyCreateDefaultK8s, http.StatusForbidden),
		Entry("Read Update user can preview k8s policy update in default", authReadUpdateDefault, bodyUpdateDefaultK8s, http.StatusOK),
		Entry("Read Update user cannot preview k8s policy delete in default", authReadUpdateDefault, bodyDeleteDefaultK8s, http.StatusForbidden),

		Entry("Read Delete user cannot preview k8s policy create in default", authReadDeleteDefault, bodyCreateDefaultK8s, http.StatusForbidden),
		Entry("Read Delete user cannot preview k8s policy update in default", authReadDeleteDefault, bodyUpdateDefaultK8s, http.StatusForbidden),
		Entry("Read Delete user can preview k8s policy delete in default", authReadDeleteDefault, bodyDeleteDefaultK8s, http.StatusOK),

		Entry("Full CRUD user cannot preview k8s policy create in alt-ns", authFullCRUDDefault, bodyCreateAltNSK8s, http.StatusForbidden),
		Entry("Full CRUD user cannot preview k8s policy update in alt-ns", authFullCRUDDefault, bodyUpdateAltNSK8s, http.StatusForbidden),
		Entry("Full CRUD user cannot preview k8s policy delete in alt-ns", authFullCRUDDefault, bodyDeleteAltNSK8s, http.StatusForbidden),

		Entry("Read Create user cannot preview k8s policy create in alt-ns", authReadCreateDefault, bodyCreateAltNSK8s, http.StatusForbidden),
		Entry("Read Update user cannot preview k8s policy update in alt-ns", authReadUpdateDefault, bodyUpdateAltNSK8s, http.StatusForbidden),
		Entry("Read Delete user cannot preview k8s policy delete in alt-ns", authReadDeleteDefault, bodyDeleteAltNSK8s, http.StatusForbidden),

		Entry("Full CRUD user can preview Calico policy create in default", authFullCRUDDefault, bodyCreateDefaultCalico, http.StatusOK),
		Entry("Full CRUD user can preview Calico policy update in default", authFullCRUDDefault, bodyUpdateDefaultCalico, http.StatusOK),
		Entry("Full CRUD user can preview Calico policy delete in default", authFullCRUDDefault, bodyDeleteDefaultCalico, http.StatusOK),

		Entry("Read Only user cannot preview Calico policy create in default", authReadOnlyDefault, bodyCreateDefaultCalico, http.StatusForbidden),
		Entry("Read Only user cannot preview Calico policy update in default", authReadOnlyDefault, bodyUpdateDefaultCalico, http.StatusForbidden),
		Entry("Read Only user cannot preview Calico policy delete in default", authReadOnlyDefault, bodyDeleteDefaultCalico, http.StatusForbidden),

		Entry("Read Create user can preview Calico policy create in default", authReadCreateDefault, bodyCreateDefaultCalico, http.StatusOK),
		Entry("Read Create user cannot preview Calico policy update in default", authReadCreateDefault, bodyUpdateDefaultCalico, http.StatusForbidden),
		Entry("Read Create user cannot preview Calico policy delete in default", authReadCreateDefault, bodyDeleteDefaultCalico, http.StatusForbidden),

		Entry("Read Update user cannot preview Calico policy 'default.p-name' create in default", authReadUpdateDefault, bodyCreateDefaultCalico, http.StatusForbidden),
		Entry("Read Update user can preview Calico policy 'default.p-name' update in default", authReadUpdateDefault, bodyUpdateDefaultCalico, http.StatusOK),
		Entry("Read Update user cannot preview Calico policy 'default.p-name' delete in default", authReadUpdateDefault, bodyDeleteDefaultCalico, http.StatusForbidden),

		Entry("Read Update user cannot preview Calico policy 'default.different' create in default", authReadUpdateDefault, bodyCreateDefaultDifferentCalico, http.StatusForbidden),
		Entry("Read Update user can preview Calico policy 'default.different' update in default", authReadUpdateDefault, bodyUpdateDefaultDifferentCalico, http.StatusForbidden),
		Entry("Read Update user cannot preview Calico policy 'default.different' delete in default", authReadUpdateDefault, bodyDeleteDefaultDifferentCalico, http.StatusForbidden),

		Entry("Read Delete user cannot preview Calico policy create in default", authReadDeleteDefault, bodyCreateDefaultCalico, http.StatusForbidden),
		Entry("Read Delete user cannot preview Calico policy update in default", authReadDeleteDefault, bodyUpdateDefaultCalico, http.StatusForbidden),
		Entry("Read Delete user can preview Calico policy delete in default", authReadDeleteDefault, bodyDeleteDefaultCalico, http.StatusOK),

		Entry("Full CRUD user cannot preview Calico policy create in alt-ns", authFullCRUDDefault, bodyCreateAltNSCalico, http.StatusForbidden),
		Entry("Full CRUD user cannot preview Calico policy update in alt-ns", authFullCRUDDefault, bodyUpdateAltNSCalico, http.StatusForbidden),
		Entry("Full CRUD user cannot preview Calico policy delete in alt-ns", authFullCRUDDefault, bodyDeleteAltNSCalico, http.StatusForbidden),

		Entry("Read Create user cannot preview Calico policy create in alt-ns", authReadCreateDefault, bodyCreateAltNSCalico, http.StatusForbidden),
		Entry("Read Update user cannot preview Calico policy update in alt-ns", authReadUpdateDefault, bodyUpdateAltNSCalico, http.StatusForbidden),
		Entry("Read Delete user cannot preview Calico policy delete in alt-ns", authReadDeleteDefault, bodyDeleteAltNSCalico, http.StatusForbidden),

		Entry("Full CRUD user can preview Global policy create in default", authFullCRUDDefault, bodyCreateDefaultGlobal, http.StatusOK),
		Entry("Full CRUD user can preview Global policy update in default", authFullCRUDDefault, bodyUpdateDefaultGlobal, http.StatusOK),
		Entry("Full CRUD user can preview Global policy delete in default", authFullCRUDDefault, bodyDeleteDefaultGlobal, http.StatusOK),

		Entry("Read Only user cannot preview Global policy create", authReadOnlyDefault, bodyCreateDefaultGlobal, http.StatusForbidden),
		Entry("Read Only user cannot preview Global policy update", authReadOnlyDefault, bodyUpdateDefaultGlobal, http.StatusForbidden),
		Entry("Read Only user cannot preview Global policy delete", authReadOnlyDefault, bodyDeleteDefaultGlobal, http.StatusForbidden),

		Entry("Read Create user can preview Global policy create", authReadCreateDefault, bodyCreateDefaultGlobal, http.StatusOK),
		Entry("Read Create user cannot preview Global policy update", authReadCreateDefault, bodyUpdateDefaultGlobal, http.StatusForbidden),
		Entry("Read Create user cannot preview Global policy delete", authReadCreateDefault, bodyDeleteDefaultGlobal, http.StatusForbidden),

		Entry("Read Update user cannot preview Global policy create", authReadUpdateDefault, bodyCreateDefaultGlobal, http.StatusForbidden),
		Entry("Read Update user can preview Global policy update", authReadUpdateDefault, bodyUpdateDefaultGlobal, http.StatusOK),
		Entry("Read Update user cannot preview Global policy delete", authReadUpdateDefault, bodyDeleteDefaultGlobal, http.StatusForbidden),

		Entry("Read Delete user cannot preview Global policy create", authReadDeleteDefault, bodyCreateDefaultGlobal, http.StatusForbidden),
		Entry("Read Delete user cannot preview Global policy update", authReadDeleteDefault, bodyUpdateDefaultGlobal, http.StatusForbidden),
		Entry("Read Delete user can preview Global policy delete", authReadDeleteDefault, bodyDeleteDefaultGlobal, http.StatusOK),
	)
})

const (
	apiCalico = "projectcalico.org/v3"
	apiK8s    = "networking.k8s.io/v1"
)

func params(api, kind, verb, ns, name string) url.Values {
	var policy string
	if api == apiCalico {
		policy = strings.Replace(validCalicoPolicy, "API", api, -1)
	} else {
		policy = strings.Replace(validK8sPolicy, "API", api, -1)
	}

	policy = strings.Replace(policy, "KIND", kind, -1)
	policy = strings.Replace(policy, "VERB", verb, -1)
	policy = strings.Replace(policy, "NAMESPACE", ns, -1)
	policy = strings.Replace(policy, "@NAME", name, -1)
	pars := url.Values{
		"policyPreview": {policy},
	}

	return pars
}

var validCalicoPolicy = `{
  "networkPolicy": {
    "apiVersion": "API",
    "kind": "KIND",
    "metadata": {
      "name": "default.@NAME",
      "uid": "",
      "namespace": "NAMESPACE"
    },
    "spec": {
      "ingress": [
        {
          "action": "Deny",
          "destination": {},
          "source": {
            "selector": "all()"
          }
        }
      ],
      "order": 1100,
      "selector": "k8s-app == \"compliance-server\"",
      "tier": "default"
    }
  },
  "verb": "VERB"
}`

var validK8sPolicy = `{
  "networkPolicy": {
    "apiVersion": "API",
    "kind": "KIND",
    "metadata": {
      "name": "default.@NAME",
      "uid": "",
      "namespace": "NAMESPACE"
    },
    "spec": {
      "podSelector": {}
    }
  },
  "verb": "VERB"
}`

var (
	authReadOnlyDefault   = basicAuthMech{"basicpolicyreadonly", "polreadonlypw"}
	authFullCRUDDefault   = basicAuthMech{"basicpolicycrud", "polcrudpw"}
	authReadCreateDefault = basicAuthMech{"basicpolicyreadcreate", "polreadcreatepw"}
	authReadUpdateDefault = basicAuthMech{"basicpolicyreadupdate", "polreadupdatepw"}
	authReadDeleteDefault = basicAuthMech{"basicpolicyreaddelete", "polreaddeletepw"}
)

var (
	bodyCreateDefaultK8s = params(apiK8s, "NetworkPolicy", "create", "default", "p-name")
	bodyUpdateDefaultK8s = params(apiK8s, "NetworkPolicy", "update", "default", "p-name")
	bodyDeleteDefaultK8s = params(apiK8s, "NetworkPolicy", "delete", "default", "p-name")

	bodyCreateAltNSK8s = params(apiK8s, "NetworkPolicy", "create", "alt-ns", "p-name")
	bodyUpdateAltNSK8s = params(apiK8s, "NetworkPolicy", "update", "alt-ns", "p-name")
	bodyDeleteAltNSK8s = params(apiK8s, "NetworkPolicy", "delete", "alt-ns", "p-name")

	bodyCreateDefaultCalico = params(apiCalico, "NetworkPolicy", "create", "default", "p-name")
	bodyUpdateDefaultCalico = params(apiCalico, "NetworkPolicy", "update", "default", "p-name")
	bodyDeleteDefaultCalico = params(apiCalico, "NetworkPolicy", "delete", "default", "p-name")

	bodyCreateDefaultDifferentCalico = params(apiCalico, "NetworkPolicy", "create", "default", "different")
	bodyUpdateDefaultDifferentCalico = params(apiCalico, "NetworkPolicy", "update", "default", "different")
	bodyDeleteDefaultDifferentCalico = params(apiCalico, "NetworkPolicy", "delete", "default", "different")

	bodyCreateAltNSCalico = params(apiCalico, "NetworkPolicy", "create", "alt-ns", "p-name")
	bodyUpdateAltNSCalico = params(apiCalico, "NetworkPolicy", "update", "alt-ns", "p-name")
	bodyDeleteAltNSCalico = params(apiCalico, "NetworkPolicy", "delete", "alt-ns", "p-name")

	bodyCreateDefaultGlobal = params(apiCalico, "GlobalNetworkPolicy", "create", "default", "p-name")
	bodyUpdateDefaultGlobal = params(apiCalico, "GlobalNetworkPolicy", "update", "default", "p-name")
	bodyDeleteDefaultGlobal = params(apiCalico, "GlobalNetworkPolicy", "delete", "default", "p-name")

	bodyMissingAction = params(apiCalico, "NetworkPolicy", "", "default", "p-name")
	bodyInvalidAction = params(apiCalico, "NetworkPolicy", "pancakes", "default", "p-name")
)

var badParams = url.Values{
	"policyPreview": {`{"query":{"bool":{"must":[{"match_all":{}}],"must_not":[],"should":[]}},"from":0,"size":10,"sort":[],"aggs":{},
	"resourceActions":[{"resource":{ "apiVersion": "projectcalico.org/v3","kind":"NetworkPolicy", "spec":{ "order":"xyz" } } ,"action":"create"}] }`},
}
