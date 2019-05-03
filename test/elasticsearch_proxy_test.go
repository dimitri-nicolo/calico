// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package fv_test

import (
	"crypto/tls"
	"fmt"
	"net/http"
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

func (tah tokenAuthMech) setAuthHeader(req *http.Request) {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", tah.token))
}

type basicAuthMech struct {
	username string
	password string
}

func (bah basicAuthMech) setAuthHeader(req *http.Request) {
	req.SetBasicAuth(bah.username, bah.password)
}

const (
	proxyListenHost = "127.0.0.1:8000"
)

var _ = Describe("Elasticsearch access", func() {
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
	verify := func(reqPath string, userAuth authInjector, expectedStatusCode int) {

		urlStr := fmt.Sprintf("%s://%s/%s", proxyScheme, proxyHost, reqPath)

		req, err := http.NewRequest("GET", urlStr, nil)
		Expect(err).To(BeNil())
		userAuth.setAuthHeader(req)

		resp, err := client.Do(req)
		Expect(err).To(BeNil())
		Expect(resp.StatusCode).To(Equal(expectedStatusCode))
	}

	nonExistentAuth := func(ai authInjector) {
		By("checking auth access for non existent user")
		verify("tigera_secure_ee_flows*/_search", ai, http.StatusUnauthorized)
	}

	accessEverything := func(ai authInjector) {
		By("checking auth access for to all indices")
		verify("tigera_secure_ee_flows*/_search", ai, http.StatusOK)
		verify("tigera_secure_ee_audit*/_search", ai, http.StatusOK)
		verify("tigera_secure_ee_events*/_search", ai, http.StatusOK)
	}

	accessFlowOnly := func(ai authInjector) {
		By("checking auth access for flow index only")
		verify("tigera_secure_ee_flows*/_search", ai, http.StatusOK)
		verify("tigera_secure_ee_audit*/_search", ai, http.StatusForbidden)
		verify("tigera_secure_ee_events*/_search", ai, http.StatusForbidden)
	}

	accessAuditOnly := func(ai authInjector) {
		By("checking auth access for audit index only")
		verify("tigera_secure_ee_flows*/_search", ai, http.StatusForbidden)
		verify("tigera_secure_ee_audit*/_search", ai, http.StatusOK)
		verify("tigera_secure_ee_events*/_search", ai, http.StatusForbidden)
	}

	accessNone := func(ai authInjector) {
		By("checking auth access for audit index only")
		verify("tigera_secure_ee_flows*/_search", ai, http.StatusForbidden)
		verify("tigera_secure_ee_audit*/_search", ai, http.StatusForbidden)
		verify("tigera_secure_ee_events*/_search", ai, http.StatusForbidden)
	}

	It("enforces RBAC for basic auth", func() {
		nonExistentAuth(basicAuthMech{"idontexist", "guessme"})
		accessEverything(basicAuthMech{"basicuserall", "basicpw"})
		accessEverything(basicAuthMech{"basicuserallgrp", "basicpwgrp"})
		accessFlowOnly(basicAuthMech{"basicuserflowonly", "basicpwf"})
		accessAuditOnly(basicAuthMech{"basicuserauditonly", "basicpwaa"})
		accessNone(basicAuthMech{"basicusernone", "basicpw0"})
		accessNone(basicAuthMech{"basicusernoselfaccess", "basicpwnos"})
	})
	It("enforces RBAC for token auth", func() {
		nonExistentAuth(tokenAuthMech{"adeadbeef"})
		accessEverything(tokenAuthMech{"deadbeef"})
		accessFlowOnly(tokenAuthMech{"deadbeeff"})
		accessAuditOnly(tokenAuthMech{"deadbeefaa"})
		accessNone(tokenAuthMech{"deadbeef0"})
	})
	It("rejects request to unknown URLS", func() {
		verify("tigera_secure_cloud_edition_flows*/_search", basicAuthMech{"basicuserall", "basicpw"}, http.StatusForbidden)
	})
})
