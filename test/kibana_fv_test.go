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

var _ = Describe("Kibana Elasticsearch", func() {

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
	DescribeTable("Users can only access kibana data for indexes they have access to ",
		func(userAuth authInjector, postRequestBody string, expectedStatusCode int) {

			//build the request
			requestVerb := http.MethodPost
			reqPath := ".kibana/_search"
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
		Entry("flows index pattern is accessible by user all", basciUserAll, kibanaBody("tigera_secure_ee_flows"), http.StatusOK),
		Entry("audit index pattern is accessible by user all", basciUserAll, kibanaBody("tigera_secure_ee_audit"), http.StatusOK),
		Entry("events index patern is accessible by user all", basciUserAll, kibanaBody("tigera_secure_ee_events"), http.StatusOK),

		Entry("flows index pattern is accessible by flow only user", basicFlowOnlyUser, kibanaBody("tigera_secure_ee_flows"), http.StatusOK),
		Entry("audit index pattern is not accessible by flow only user", basicFlowOnlyUser, kibanaBody("tigera_secure_ee_audit"), http.StatusForbidden),
		Entry("events index patern is not accessible by flow only user", basicFlowOnlyUser, kibanaBody("tigera_secure_ee_events"), http.StatusForbidden),

		Entry("flows index pattern is not accessible by audit only user", basicAuditOnlyUser, kibanaBody("tigera_secure_ee_flows"), http.StatusForbidden),
		Entry("audit index pattern is accessible by audit only user", basicAuditOnlyUser, kibanaBody("tigera_secure_ee_audit"), http.StatusOK),
		Entry("events index patern is not accessible by audit only user", basicAuditOnlyUser, kibanaBody("tigera_secure_ee_events"), http.StatusForbidden),
	)
})

func kibanaBody(indexPattern string) string {
	return strings.Replace(kibanaReqBody, "{{.IndexPatternTitle}}", indexPattern, -1)
}

const kibanaReqBody = `{ "query": { "bool": { "filter": [ { "match": { "index-pattern.title":"{{.IndexPatternTitle}}" } } ] } } }`

var (
	basciUserAll       = basicAuthMech{"basicuserall", "basicpw"}
	basicAuditOnlyUser = basicAuthMech{"basicuserauditonly", "basicpwaa"}
	basicFlowOnlyUser  = basicAuthMech{"basicuserflowonly", "basicpwf"}
)
