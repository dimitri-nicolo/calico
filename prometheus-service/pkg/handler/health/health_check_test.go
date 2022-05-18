// Copyright (c) 2021 Tigera. All rights reserved.
package handler_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	health "github.com/tigera/prometheus-service/pkg/handler/health"
)

var _ = Describe("Health Check test", func() {

	It("returns UP on health endpoint", func() {

		healthCheck := health.HealthCheck()

		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health", nil)

		healthCheck.ServeHTTP(rr, req)

		bodyBytes, err := ioutil.ReadAll(rr.Body)

		Expect(err).NotTo(HaveOccurred())
		Expect(rr.Result().StatusCode).To(Equal(200))
		Expect(string(bodyBytes)).To(Equal("UP"))
	})
})
