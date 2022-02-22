// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package middleware

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Middleware utility tests", func() {
	Context("Parse cluster name from request header", func() {
		It("should parse cluster name when x-cluster-id is set in header", func() {
			req, err := http.NewRequest("GET", "http://some-url", nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Add("x-cluster-id", "test-cluster-name")
			clusterName := MaybeParseClusterNameFromRequest(req)
			Expect(clusterName).To(Equal("test-cluster-name"))
		})

		It("should return default cluster name when x-cluster-id is not set in header", func() {
			req, err := http.NewRequest("GET", "http://some-url", nil)
			Expect(err).NotTo(HaveOccurred())
			clusterName := MaybeParseClusterNameFromRequest(req)
			Expect(clusterName).To(Equal("cluster"))
		})

		It("should return default cluster name when request is nil", func() {
			clusterName := MaybeParseClusterNameFromRequest(nil)
			Expect(clusterName).To(Equal("cluster"))
		})
	})
})
