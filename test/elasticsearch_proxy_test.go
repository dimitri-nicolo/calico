// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package fv_test

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type authInjector interface {
	setAuthHeader(req *http.Request)
}

type tokenAuthMech struct {
	token string
}

func (tam tokenAuthMech) setAuthHeader(req *http.Request) {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", tam.token))
}

type basicAuthMech struct {
	username string
	password string
}

func (bam basicAuthMech) setAuthHeader(req *http.Request) {
	req.SetBasicAuth(bam.username, bam.password)
}

const (
	proxyListenHost = "127.0.0.1:8000"
)

var _ = Describe("Elasticsearch access", func() {
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
	verify := func(reqPath string, userAuth authInjector, expectedStatusCode int, requestVerb string) {

		urlStr := fmt.Sprintf("%s://%s/%s", proxyScheme, proxyHost, reqPath)

		var bodyreader io.Reader
		if requestVerb == http.MethodPost {
			bodyreader = strings.NewReader(postRequestBody)
		}

		req, err := http.NewRequest(requestVerb, urlStr, bodyreader)
		Expect(err).To(BeNil())
		if requestVerb == http.MethodPost {
			req.Header.Add("content-length", fmt.Sprintf("%d", len(postRequestBody)))
			req.Header.Add("Content-Type", "application/json")
		}
		userAuth.setAuthHeader(req)

		resp, err := client.Do(req)
		Expect(err).To(BeNil())
		Expect(resp.StatusCode).To(Equal(expectedStatusCode))
	}

	nonExistentAuth := func(ai authInjector, requestVerb string) {
		By("checking auth access for non existent user")
		verify("tigera_secure_ee_flows*/_search", ai, http.StatusUnauthorized, requestVerb)
	}

	accessEverything := func(ai authInjector, requestVerb string) {
		By("checking auth access for to all indices")
		verify("tigera_secure_ee_flows*/_search", ai, http.StatusOK, requestVerb)
		verify("tigera_secure_ee_audit*/_search", ai, http.StatusOK, requestVerb)
		verify("tigera_secure_ee_events*/_search", ai, http.StatusOK, requestVerb)
	}

	accessFlowOnly := func(ai authInjector, requestVerb string) {
		By("checking auth access for flow index only")
		verify("tigera_secure_ee_flows*/_search", ai, http.StatusOK, requestVerb)
		verify("tigera_secure_ee_audit*/_search", ai, http.StatusForbidden, requestVerb)
		verify("tigera_secure_ee_events*/_search", ai, http.StatusForbidden, requestVerb)
	}

	accessAuditOnly := func(ai authInjector, requestVerb string) {
		By("checking auth access for audit index only")
		verify("tigera_secure_ee_flows*/_search", ai, http.StatusForbidden, requestVerb)
		verify("tigera_secure_ee_audit*/_search", ai, http.StatusOK, requestVerb)
		verify("tigera_secure_ee_events*/_search", ai, http.StatusForbidden, requestVerb)
	}

	accessNone := func(ai authInjector, requestVerb string) {
		By("checking auth access for audit index only")
		verify("tigera_secure_ee_flows*/_search", ai, http.StatusForbidden, requestVerb)
		verify("tigera_secure_ee_audit*/_search", ai, http.StatusForbidden, requestVerb)
		verify("tigera_secure_ee_events*/_search", ai, http.StatusForbidden, requestVerb)
	}

	It("enforces RBAC for basic auth GET requests", func() {
		nonExistentAuth(basicAuthMech{"idontexist", "guessme"}, http.MethodGet)
		accessEverything(basicAuthMech{"basicuserall", "basicpw"}, http.MethodGet)
		accessEverything(basicAuthMech{"basicuserallgrp", "basicpwgrp"}, http.MethodGet)
		accessFlowOnly(basicAuthMech{"basicuserflowonly", "basicpwf"}, http.MethodGet)
		accessAuditOnly(basicAuthMech{"basicuserauditonly", "basicpwaa"}, http.MethodGet)
		accessNone(basicAuthMech{"basicusernone", "basicpw0"}, http.MethodGet)
		accessNone(basicAuthMech{"basicusernoselfaccess", "basicpwnos"}, http.MethodGet)
	})
	It("enforces RBAC for token auth GET requests", func() {
		nonExistentAuth(tokenAuthMech{"adeadbeef"}, http.MethodGet)
		accessEverything(tokenAuthMech{"deadbeef"}, http.MethodGet)
		accessFlowOnly(tokenAuthMech{"deadbeeff"}, http.MethodGet)
		accessAuditOnly(tokenAuthMech{"deadbeefaa"}, http.MethodGet)
		accessNone(tokenAuthMech{"deadbeef0"}, http.MethodGet)
	})
	It("rejects GET request to unknown URLS", func() {
		verify("tigera_secure_cloud_edition_flows*/_search", basicAuthMech{"basicuserall", "basicpw"}, http.StatusForbidden, http.MethodGet)
	})

	It("enforces RBAC for basic auth POST requests", func() {
		nonExistentAuth(basicAuthMech{"idontexist", "guessme"}, http.MethodPost)
		accessEverything(basicAuthMech{"basicuserall", "basicpw"}, http.MethodPost)
		accessEverything(basicAuthMech{"basicuserallgrp", "basicpwgrp"}, http.MethodPost)
		accessFlowOnly(basicAuthMech{"basicuserflowonly", "basicpwf"}, http.MethodPost)
		accessAuditOnly(basicAuthMech{"basicuserauditonly", "basicpwaa"}, http.MethodPost)
		accessNone(basicAuthMech{"basicusernone", "basicpw0"}, http.MethodPost)
		accessNone(basicAuthMech{"basicusernoselfaccess", "basicpwnos"}, http.MethodPost)
	})
	It("enforces RBAC for token auth POST requests", func() {
		nonExistentAuth(tokenAuthMech{"adeadbeef"}, http.MethodPost)
		accessEverything(tokenAuthMech{"deadbeef"}, http.MethodPost)
		accessFlowOnly(tokenAuthMech{"deadbeeff"}, http.MethodPost)
		accessAuditOnly(tokenAuthMech{"deadbeefaa"}, http.MethodPost)
		accessNone(tokenAuthMech{"deadbeef0"}, http.MethodPost)
	})
	It("rejects POST request to unknown URLS", func() {
		verify("tigera_secure_cloud_edition_flows*/_search", basicAuthMech{"basicuserall", "basicpw"}, http.StatusForbidden, http.MethodPost)
	})

})

var postRequestBody = `{"query":{"bool":{"must":[{"match_all":{}}],"must_not":[],"should":[]}},"from":0,"size":10,"sort":[],"aggs":{}}`
