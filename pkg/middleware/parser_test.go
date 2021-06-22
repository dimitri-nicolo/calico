// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package middleware_test

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	lmak8s "github.com/tigera/lma/pkg/k8s"

	"net/http"
	"net/http/httptest"

	"github.com/tigera/packetcapture-api/pkg/middleware"
)

var _ = Describe("Parser", func() {
	type expectedResponse struct {
		ns        string
		name      string
		clusterID string
		body      string
		status    int
	}
	DescribeTable("Validate requests",
		func(url, xClusterID string, expected expectedResponse) {
			var req, err = http.NewRequest("GET", url, nil)
			req.Header.Set(lmak8s.XClusterIDHeader, xClusterID)
			Expect(err).NotTo(HaveOccurred())

			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Expect namespace, name and cluster ID to be set on the context
				Expect(middleware.NamespaceFromContext(r.Context())).To(Equal(expected.ns))
				Expect(middleware.CaptureNameFromContext(r.Context())).To(Equal(expected.name))
				Expect(middleware.ClusterIDFromContext(r.Context())).To(Equal(expected.clusterID))
			})

			// Bootstrap the http recorder
			recorder := httptest.NewRecorder()
			handler := middleware.Parse(testHandler)
			handler.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(expected.status))
			Expect(strings.Trim(recorder.Body.String(), "\n")).To(Equal(expected.body))
		},
		Entry("Malformed request", "/$534/$  ", "",
			expectedResponse{ns: "", name: "", clusterID: lmak8s.DefaultCluster,
				body: "request URL is malformed", status: http.StatusBadRequest}),
		Entry("Missing prefix", "/", "",
			expectedResponse{ns: "", name: "", clusterID: lmak8s.DefaultCluster,
				body: "request URL is malformed", status: http.StatusBadRequest}),
		Entry("Missing namespace and name", "/download", "",
			expectedResponse{ns: "", name: "", clusterID: lmak8s.DefaultCluster,
				body: "request URL is malformed", status: http.StatusBadRequest}),
		Entry("Missing namespace", "/download/ns", "",
			expectedResponse{ns: "ns", name: "", clusterID: lmak8s.DefaultCluster,
				body: "request URL is malformed", status: http.StatusBadRequest}),
		Entry("Missing query", "/download/ns/name", "",
			expectedResponse{ns: "ns", name: "name", clusterID: lmak8s.DefaultCluster,
				body: "request URL is malformed", status: http.StatusBadRequest}),
		Entry("Invalid query", "/download/ns/name/file=abc", "",
			expectedResponse{ns: "ns", name: "name", clusterID: lmak8s.DefaultCluster,
				body: "request URL is malformed", status: http.StatusBadRequest}),
		Entry("Ok for default cluster", "/download/ns/name/files.zip", "",
			expectedResponse{ns: "ns", name: "name", clusterID: lmak8s.DefaultCluster,
				body: "", status: http.StatusOK}),
		Entry("Ok for other cluster", "/download/ns/name/files.zip", "otherCluster",
			expectedResponse{ns: "ns", name: "name", clusterID: "otherCluster",
				body: "", status: http.StatusOK}),
	)
})
