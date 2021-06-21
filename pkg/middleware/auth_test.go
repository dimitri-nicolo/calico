// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package middleware_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	lmaauth "github.com/tigera/lma/pkg/auth"
	"github.com/tigera/packetcapture-api/pkg/cache"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	authzv1 "k8s.io/api/authorization/v1"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/endpoints/request"

	"github.com/tigera/packetcapture-api/pkg/middleware"
)

var _ = Describe("Auth", func() {
	var req *http.Request
	var noOpHandler http.HandlerFunc
	var anyError = fmt.Errorf("any error")
	var resAtr = &authzv1.ResourceAttributes{
		Verb:        "get",
		Group:       "projectcalico.org",
		Resource:    "packetcaptures",
		Subresource: "files",
		Name:        "name",
		Namespace:   "ns",
	}

	BeforeEach(func() {
		// Create a new request
		var err error
		req, err = http.NewRequest("GET", "/download/ns/name?files.zip", nil)
		Expect(err).NotTo(HaveOccurred())
		// Set the Authorization header
		req.Header.Set("Authorization", "cluster")
		// Create a noOp handler func for the middleware
		noOpHandler = func(w http.ResponseWriter, r *http.Request) {}

		// Setup the variables on the context to be used for authN/authZ
		req = req.WithContext(middleware.WithClusterID(req.Context(), "cluster"))
		req = req.WithContext(middleware.WithNamespace(req.Context(), "ns"))
		req = req.WithContext(middleware.WithCaptureName(req.Context(), "name"))
		req = req.WithContext(request.WithUser(req.Context(), &user.DefaultInfo{}))
	})

	It("Fails to authenticate user", func() {
		// Bootstrap the authenticator
		var mockAuthenticator = &mockAuthenticator{}
		mockAuthenticator.On("Authenticate", "cluster").Return(&user.DefaultInfo{},
			http.StatusUnauthorized, anyError)
		var auth = middleware.NewAuth(mockAuthenticator, &cache.MockClientCache{})

		// Bootstrap the http recorder
		recorder := httptest.NewRecorder()
		handler := auth.Authenticate(noOpHandler)
		handler.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
		Expect(recorder.Body.String()).To(Equal("any error\n"))
	})

	It("Authenticate user", func() {
		// Bootstrap the authenticator
		var user = &user.DefaultInfo{}
		var mockAuthenticator = &mockAuthenticator{}
		mockAuthenticator.On("Authenticate", "cluster").Return(user, http.StatusOK, nil)
		var auth = middleware.NewAuth(mockAuthenticator, &cache.MockClientCache{})

		// Bootstrap the http recorder
		recorder := httptest.NewRecorder()
		handler := auth.Authenticate(noOpHandler)
		handler.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusOK))
		Expect(recorder.Body.String()).To(Equal(""))
	})

	It("Fails to create client for authorizer", func() {
		// Bootstrap the authorizer
		var mockCache = &cache.MockClientCache{}
		mockCache.On("GetAuthorizer", "cluster").Return(nil, anyError)
		var auth = middleware.NewAuth(&mockAuthenticator{}, mockCache)

		// Bootstrap the http recorder
		recorder := httptest.NewRecorder()
		handler := auth.Authorize(noOpHandler)
		handler.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
		Expect(recorder.Body.String()).To(Equal("any error\n"))
	})

	It("Fails to authorize user", func() {
		// Bootstrap the authorizer
		var mockCache = &cache.MockClientCache{}
		var mockAuth = &lmaauth.MockRBACAuthorizer{}
		mockCache.On("GetAuthorizer", "cluster").Return(mockAuth, nil)
		mockAuth.On("Authorize", &user.DefaultInfo{}, resAtr, (*authzv1.NonResourceAttributes)(nil)).Return(false, anyError)
		var auth = middleware.NewAuth(&mockAuthenticator{}, mockCache)

		// Bootstrap the http recorder
		recorder := httptest.NewRecorder()
		handler := auth.Authorize(noOpHandler)
		handler.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
		Expect(recorder.Body.String()).To(Equal("any error\n"))
	})

	It("User is not authorized", func() {
		// Bootstrap the authorizer
		var mockCache = &cache.MockClientCache{}
		var mockAuth = &lmaauth.MockRBACAuthorizer{}
		mockCache.On("GetAuthorizer", "cluster").Return(mockAuth, nil)
		mockAuth.On("Authorize", &user.DefaultInfo{}, resAtr, (*authzv1.NonResourceAttributes)(nil)).Return(false, anyError)
		var auth = middleware.NewAuth(&mockAuthenticator{}, mockCache)

		// Bootstrap the http recorder
		recorder := httptest.NewRecorder()
		handler := auth.Authorize(noOpHandler)
		handler.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
		Expect(recorder.Body.String()).To(Equal("any error\n"))
	})

	It("Authorizes user", func() {
		// Bootstrap the authorizer
		var mockCache = &cache.MockClientCache{}
		var mockAuth = &lmaauth.MockRBACAuthorizer{}
		mockCache.On("GetAuthorizer", "cluster").Return(mockAuth, nil)
		mockAuth.On("Authorize", &user.DefaultInfo{}, resAtr, (*authzv1.NonResourceAttributes)(nil)).Return(true, nil)
		var auth = middleware.NewAuth(&mockAuthenticator{}, mockCache)

		// Bootstrap the http recorder
		recorder := httptest.NewRecorder()
		handler := auth.Authorize(noOpHandler)
		handler.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusOK))
		Expect(recorder.Body.String()).To(Equal(""))
	})
})

type mockAuthenticator struct {
	mock.Mock
}

func (m *mockAuthenticator) Authenticate(token string) (user.Info, int, error) {
	args := m.Called(token)
	return args.Get(0).(user.Info), args.Int(1), args.Error(2)
}
