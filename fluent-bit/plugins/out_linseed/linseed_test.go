// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package main

import (
	"bytes"
	_ "embed"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"time"
	"unsafe"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Linseed out plugin tests", func() {
	var (
		f                 *os.File
		pluginConfigKeyFn PluginConfigKeyFunc
		ndjsonBuffer      bytes.Buffer
	)

	BeforeEach(func() {
		var err error
		f, err = os.CreateTemp("", "kubeconfig")
		Expect(err).NotTo(HaveOccurred())

		pluginConfigKeyFn = func(plugin unsafe.Pointer, key string) string {
			if key == "tls.verify" {
				return "true"
			}
			return ""
		}

		ndjsonBuffer.Write([]byte(`{"record":1}\n{"record":2}\n`))
	})

	AfterEach(func() {
		os.Remove(f.Name())
	})

	Context("http request tests", func() {
		It("should send expected requests", func() {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.URL.Path).To(Equal("/ingestion/api/v1/flows/logs/bulk"))
				Expect(r.Header).To(HaveKeyWithValue("Authorization", []string{"Bearer some-token"}))
				Expect(r.Header).To(HaveKeyWithValue("Content-Type", []string{"application/x-ndjson"}))

				bytes, err := io.ReadAll(r.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(bytes)).To(Equal(`{"record":1}\n{"record":2}\n`))

				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			_, err := f.WriteString(validKubeconfig)
			f.Close()
			Expect(err).NotTo(HaveOccurred())

			err = os.Setenv("KUBECONFIG", f.Name())
			Expect(err).NotTo(HaveOccurred())
			err = os.Setenv("ENDPOINT", server.URL)
			Expect(err).NotTo(HaveOccurred())

			cfg, err := NewConfig(nil, pluginConfigKeyFn)
			Expect(err).NotTo(HaveOccurred())

			cfg.serviceAccountName = "some-sa"
			cfg.expiration = time.Now().Add(1 * time.Hour) // must not be expired
			cfg.token = "some-token"

			err = doRequest(cfg, &ndjsonBuffer, "flows")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return error when log type is unexpected", func() {
			_, err := f.WriteString(validKubeconfig)
			f.Close()
			Expect(err).NotTo(HaveOccurred())

			err = os.Setenv("KUBECONFIG", f.Name())
			Expect(err).NotTo(HaveOccurred())
			err = os.Setenv("ENDPOINT", "https://1.2.3.4:5678")
			Expect(err).NotTo(HaveOccurred())

			cfg, err := NewConfig(nil, pluginConfigKeyFn)
			Expect(err).NotTo(HaveOccurred())

			cfg.serviceAccountName = "some-sa"
			cfg.expiration = time.Now().Add(1 * time.Hour) // must not be expired
			cfg.token = "some-token"

			err = doRequest(cfg, &ndjsonBuffer, "unknown-log-type")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`unknown log type "unknown-log-type"`))
		})

		It("should return error when http response is not ok", func() {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			}))
			defer server.Close()

			_, err := f.WriteString(validKubeconfig)
			f.Close()
			Expect(err).NotTo(HaveOccurred())

			err = os.Setenv("KUBECONFIG", f.Name())
			Expect(err).NotTo(HaveOccurred())
			err = os.Setenv("ENDPOINT", server.URL)
			Expect(err).NotTo(HaveOccurred())

			cfg, err := NewConfig(nil, pluginConfigKeyFn)
			Expect(err).NotTo(HaveOccurred())

			cfg.serviceAccountName = "some-sa"
			cfg.expiration = time.Now().Add(1 * time.Hour) // must not be expired
			cfg.token = "some-token"

			err = doRequest(cfg, &ndjsonBuffer, "flows")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error response from server"))
		})
	})
})
