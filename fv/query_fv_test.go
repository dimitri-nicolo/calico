// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package fv_test

import (
	"context"
	"encoding/json"

	"net/http"
	"net/url"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Prometheus Proxy Test", func() {

	const httpScheme = "http://"
	const mockPrometheuServicesUrl = "127.0.0.1:9090"
	const proxyServicesUrl = "127.0.0.1:8090"

	const testPrometheusQuery = "sum(http_requests_total{method=\"GET\"} offset 5m)"
	const testStep = "15s"

	var mockPrometheusService *http.Server

	BeforeEach(func() {
		// setup mock prometheus service
		mockPrometheusServiceMux := http.NewServeMux()
		mockPrometheusServiceMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			requestParam := struct {
				Host   string
				Method string
				Path   string
			}{
				r.Host,
				r.Method,
				r.URL.Path,
			}

			err := json.NewEncoder(w).Encode(requestParam)
			if err != nil {
				log.Errorf("JSON Encoder error: %s", err)
			}
		})
		mockPrometheusService = &http.Server{
			Addr:    mockPrometheuServicesUrl,
			Handler: mockPrometheusServiceMux,
		}
		go func() {
			err := mockPrometheusService.ListenAndServe()
			if err != nil {
				log.Warnf("Mock Prometheus Service: %s", err)
			}
		}()
	})

	AfterSuite(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := mockPrometheusService.Shutdown(ctx); err != nil {
			Fail("Unable to continue, server does not shutdown")
		}
	})

	It("should proxy http requests to the promeheus service as it was received", func() {
		http_proxy_url, err := url.Parse(httpScheme + proxyServicesUrl)
		Expect(err).NotTo(HaveOccurred())
		http_proxy_url.Path = "/api/v1/query_range"

		req, err := http.NewRequest("GET", http_proxy_url.String(), nil)
		Expect(err).NotTo(HaveOccurred())

		req_query := req.URL.Query()
		req_query.Add("query", testPrometheusQuery)
		t := time.Now()
		start := strconv.FormatInt(t.Unix(), 10)
		end := strconv.FormatInt(t.Unix()+60, 10)
		req_query.Add("start", start)
		req_query.Add("end", end)
		req_query.Add("step", testStep)

		client := &http.Client{}
		resp, err := client.Do(req)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(200))

		var data map[string]string
		err = json.NewDecoder(resp.Body).Decode(&data)
		Expect(err).NotTo(HaveOccurred())
		Expect(data["Method"]).To(Equal(req.Method))
		Expect(data["Path"]).To(Equal(req.URL.Path))
		// arrived from the proxy
		Expect(data["Host"]).To(Equal(proxyServicesUrl))
	})
})
