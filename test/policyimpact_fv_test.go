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
	AfterEach(func() {
	})

	// We only verify access from the clients point of view.
	DescribeTable("permissions for policy impact requests should be validated ",
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

		Entry("Full CRUD user can preview create in default", authFullCRUDDefault, bodyCreateDefault, http.StatusOK),
		Entry("Full CRUD user can preview update in default", authFullCRUDDefault, bodyUpdateDefault, http.StatusOK),
		Entry("Full CRUD user can preview delete in default", authFullCRUDDefault, bodyDeleteDefault, http.StatusOK),

		Entry("Read Only user cannot preview create in default", authReadOnlyDefault, bodyCreateDefault, http.StatusUnauthorized),
		Entry("Read Only user cannot preview update in default", authReadOnlyDefault, bodyUpdateDefault, http.StatusUnauthorized),
		Entry("Read Only user cannot preview delete in default", authReadOnlyDefault, bodyDeleteDefault, http.StatusUnauthorized),

		Entry("Read Create user can preview create in default", authReadCreateDefault, bodyCreateDefault, http.StatusOK),
		Entry("Read Create user cannot preview update in default", authReadCreateDefault, bodyUpdateDefault, http.StatusUnauthorized),
		Entry("Read Create user cannot preview delete in default", authReadCreateDefault, bodyDeleteDefault, http.StatusUnauthorized),

		Entry("Read Update user cannot preview create in default", authReadUpdateDefault, bodyCreateDefault, http.StatusUnauthorized),
		Entry("Read Update user can preview update in default", authReadUpdateDefault, bodyUpdateDefault, http.StatusOK),
		Entry("Read Update user cannot preview delete in default", authReadUpdateDefault, bodyDeleteDefault, http.StatusUnauthorized),

		Entry("Read Delete user cannot preview create in default", authReadDeleteDefault, bodyCreateDefault, http.StatusUnauthorized),
		Entry("Read Delete user cannot preview update in default", authReadDeleteDefault, bodyUpdateDefault, http.StatusUnauthorized),
		Entry("Read Delete user can preview delete in default", authReadDeleteDefault, bodyDeleteDefault, http.StatusOK),

		Entry("Full CRUD user cannot preview create in alt-ns", authFullCRUDDefault, bodyCreateAltNS, http.StatusUnauthorized),
		Entry("Full CRUD user cannot preview update in alt-ns", authFullCRUDDefault, bodyUpdateAltNS, http.StatusUnauthorized),
		Entry("Full CRUD user cannot preview delete in alt-ns", authFullCRUDDefault, bodyDeleteAltNS, http.StatusUnauthorized),

		Entry("Read Create user cannot preview create in alt-ns", authReadCreateDefault, bodyCreateAltNS, http.StatusUnauthorized),
		Entry("Read Update user cannot preview update in alt-ns", authReadUpdateDefault, bodyUpdateAltNS, http.StatusUnauthorized),
		Entry("Read Delete user cannot preview delete in alt-ns", authReadDeleteDefault, bodyDeleteAltNS, http.StatusUnauthorized),
	)

})

func modBody(b string, act string, ns string) string {
	b = strings.Replace(b, "@@ACTION@@", act, -1)
	b = strings.Replace(b, "@@NAMESPACE@@", ns, -1)
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
	bodyCreateDefault = modBody(body, "create", "default")
	bodyUpdateDefault = modBody(body, "update", "default")
	bodyDeleteDefault = modBody(body, "delete", "default")
	bodyCreateAltNS   = modBody(body, "create", "alt-ns")
	bodyUpdateAltNS   = modBody(body, "update", "alt-ns")
	bodyDeleteAltNS   = modBody(body, "delete", "alt-ns")
)

var badBody = `{"query":{"bool":{"must":[{"match_all":{}}],"must_not":[],"should":[]}},"from":0,"size":10,"sort":[],"aggs":{},
"policyActions":[{"policy":{ "spec":{ "order":"xyz" } } ,"action":"create"}] }`

var body = `{"query":{"bool":{"must":[{"match_all":{}}],"must_not":[],"should":[]}},"from":0,"size":10,"sort":[],"aggs":{},
"policyActions":[{"policy":{
			"metadata":{
				"name":"p-name",
				"generateName":"p-gen-name",
				"namespace":"@@NAMESPACE@@",
				"selfLink":"p-self-link",
				"resourceVersion":"p-res-ver"
			},
			"spec":{
				"tier":"default",
				"order":1,
				"selector":"a|bogus|selector|string"
			}
		}
		,"action":"@@ACTION@@"}]

}`
