// Copyright (c) 2021 Tigera. All rights reserved.
package handler_test

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	handler "github.com/tigera/prometheus-service/pkg/handler/proxy"
)

var _ = Describe("Prometheus Proxy Query test", func() {

	testUrl, _ := url.Parse("http://test-service:9090")

	It("passes the request to the Proxy", func() {

		var requestReceived *http.Request

		mockRevProxy := httputil.NewSingleHostReverseProxy(testUrl)
		mockRevProxy.Director = func(req *http.Request) {
			requestReceived = req
		}

		proxy := handler.Proxy(mockRevProxy)

		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test-endpoint", nil)

		proxy.ServeHTTP(rr, req)

		Expect(requestReceived).ToNot(BeNil())
		Expect(requestReceived.Method).To(Equal("GET"))
		Expect(requestReceived.URL.Path).To(Equal("/test-endpoint"))
	})
})
