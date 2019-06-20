// Copyright (c) 2016-2019 Tigera, Inc. All rights reserved.
package middleware_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/projectcalico/libcalico-go/lib/middleware"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

type basicAuthAssertingHandler struct {
	expectedUsername string
	expectedPassword string

	handlerCalled bool
}

func (b *basicAuthAssertingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b.handlerCalled = true
	username, password, ok := r.BasicAuth()
	Expect(ok).To(BeTrue())
	Expect(username).To(Equal(b.expectedUsername))
	Expect(password).To(Equal(b.expectedPassword))
}

var _ = Describe("Basic auth injector middleware", func() {
	DescribeTable("Basic auth middleware tests",
		func(user, pass string, expectHandlerCalled bool) {
			w := httptest.NewRecorder()
			req := &http.Request{Header: make(http.Header)}
			ah := &basicAuthAssertingHandler{expectedUsername: user, expectedPassword: pass}
			bah := middleware.BasicAuthHeaderInjector(user, pass, ah)
			bah.ServeHTTP(w, req)

			Expect(ah.handlerCalled).To(Equal(expectHandlerCalled))
		},
		Entry("Inject basic auth header", "user", "pass", true),
		Entry("No basic auth header", "", "", false))

})
