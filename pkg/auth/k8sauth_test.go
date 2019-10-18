// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package auth

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

type DummyHttpHandler struct {
	serveCalled bool
}

func (dhh *DummyHttpHandler) ServeHTTP(http.ResponseWriter, *http.Request) {
	dhh.serveCalled = true
}

var _ = Describe("Test request parsing", func() {
	DescribeTable("Test invalid Authorization Headers",
		func(req *http.Request) {
			var ka k8sauth
			ka.delegateAuthentication = true
			h := &DummyHttpHandler{serveCalled: false}
			w := httptest.NewRecorder()

			uut := ka.KubernetesAuthnAuthz(h)
			uut.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusUnauthorized))
		},

		Entry("No authorization header", &http.Request{}),
		Entry("No token or basic in header",
			&http.Request{Header: http.Header{"Authorization": []string{"Bearer"}}}),
		Entry("Bad token: bear token",
			&http.Request{Header: http.Header{"Authorization": []string{"bear token"}}}),
		Entry("Bad token: Bearer: token",
			&http.Request{Header: http.Header{"Authorization": []string{"Bearer: token"}}}),
		Entry("No authorization header and empty string for impersonate user header",
			&http.Request{Header: http.Header{"Authorization": []string{"Bearer"}, "Impersonate-User": []string{""}}}),
		Entry("No authorization header and empty impersonate user header",
			&http.Request{Header: http.Header{"Authorization": []string{"Bearer"}, "Impersonate-User": []string{}}}),
	)
})
