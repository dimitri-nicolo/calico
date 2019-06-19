// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package fv_test

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Policyimpact", func() {
	proxyScheme := "https"
	proxyHost := "127.0.0.1:8000"
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
	verify := func(reqPath string, userAuth authInjector, reqbody string, expectedStatusCode int) {

		urlStr := fmt.Sprintf("%s://%s/%s", proxyScheme, proxyHost, reqPath)

		bodyreader := strings.NewReader(reqbody)

		req, err := http.NewRequest("POST", urlStr, bodyreader)

		Expect(err).To(BeNil())
		userAuth.setAuthHeader(req)

		resp, err := client.Do(req)
		Expect(err).To(BeNil())
		Expect(resp.StatusCode).To(Equal(expectedStatusCode))
	}

	It("does not error on pip request", func() {
		path := "tigera_secure_ee_flows*/_search"
		auth := basicAuthMech{"basicuserflowonly", "basicpwf"}
		body := `{"policyActions":[{"policy":{
			"metadata":{
				"name":"p-name",
				"generateName":"p-gen-name",
				"namespace":"p-name-space",
				"selfLink":"p-self-link",
				"resourceVersion":"p-res-ver"
			},
			"spec":{
				"tier":"default",
				"order":1,
				"selector":"a|bogus|selector|string"
			}		
		}
		,"action":"create"}]}`
		verify(path, auth, body, http.StatusOK)
	})

})
