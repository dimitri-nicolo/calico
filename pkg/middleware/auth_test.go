// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package middleware_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	lmak8s "github.com/tigera/lma/pkg/k8s"
	authenticationv1 "k8s.io/api/authentication/v1"
	"k8s.io/apiserver/pkg/endpoints/request"

	lmaauth "github.com/tigera/lma/pkg/auth"
	"github.com/tigera/packetcapture-api/pkg/cache"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"github.com/tigera/packetcapture-api/pkg/middleware"
	authzv1 "k8s.io/api/authorization/v1"
	"k8s.io/apiserver/pkg/authentication/user"
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
		req, err = http.NewRequest("GET", "/download/ns/name/files.zip", nil)
		Expect(err).NotTo(HaveOccurred())
		// Set the Authorization header
		req.Header.Set("Authorization", "token")
		// Create a noOp handler func for the middleware
		noOpHandler = func(w http.ResponseWriter, r *http.Request) {}

		// Setup the variables on the context to be used for authN/authZ
		req = req.WithContext(middleware.WithClusterID(req.Context(), lmak8s.DefaultCluster))
		req = req.WithContext(middleware.WithNamespace(req.Context(), "ns"))
		req = req.WithContext(middleware.WithCaptureName(req.Context(), "name"))
	})

	It("Fails to authenticate user", func() {
		// Bootstrap the authenticator
		var mockAuthenticator = &mockAuthenticator{}
		mockAuthenticator.On("Authenticate", "token").Return(&user.DefaultInfo{},
			http.StatusUnauthorized, anyError)
		var auth = middleware.NewAuth(mockAuthenticator, &cache.MockClientCache{})

		// Bootstrap the http recorder
		recorder := httptest.NewRecorder()
		handler := auth.Authenticate(noOpHandler)
		handler.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
		Expect(recorder.Body.String()).To(Equal("any error\n"))
	})

	It("Authenticate user without checking impersonation", func() {
		// Bootstrap the authenticator
		var user = &user.DefaultInfo{}
		var mockAuthenticator = &mockAuthenticator{}
		mockAuthenticator.On("Authenticate", "token").Return(user, http.StatusOK, nil)
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
		mockCache.On("GetAuthorizer", lmak8s.DefaultCluster).Return(nil, anyError)
		var auth = middleware.NewAuth(&mockAuthenticator{}, mockCache)

		// Bootstrap the http recorder
		recorder := httptest.NewRecorder()
		handler := auth.Authorize(noOpHandler)
		handler.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
		Expect(recorder.Body.String()).To(Equal("any error\n"))
	})

	It("Fails to authorize user", func() {
		req = req.WithContext(request.WithUser(req.Context(), &user.DefaultInfo{}))

		// Bootstrap the authorizer
		var mockCache = &cache.MockClientCache{}
		var mockAuth = &lmaauth.MockRBACAuthorizer{}
		mockCache.On("GetAuthorizer", lmak8s.DefaultCluster).Return(mockAuth, nil)
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
		req = req.WithContext(request.WithUser(req.Context(), &user.DefaultInfo{}))

		// Bootstrap the authorizer
		var mockCache = &cache.MockClientCache{}
		var mockAuth = &lmaauth.MockRBACAuthorizer{}
		mockCache.On("GetAuthorizer", lmak8s.DefaultCluster).Return(mockAuth, nil)
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
		req = req.WithContext(request.WithUser(req.Context(), &user.DefaultInfo{}))

		// Bootstrap the authorizer
		var mockCache = &cache.MockClientCache{}
		var mockAuth = &lmaauth.MockRBACAuthorizer{}
		mockCache.On("GetAuthorizer", lmak8s.DefaultCluster).Return(mockAuth, nil)
		mockAuth.On("Authorize", &user.DefaultInfo{}, resAtr, (*authzv1.NonResourceAttributes)(nil)).Return(true, nil)
		var auth = middleware.NewAuth(&mockAuthenticator{}, mockCache)

		// Bootstrap the http recorder
		recorder := httptest.NewRecorder()
		handler := auth.Authorize(noOpHandler)
		handler.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusOK))
		Expect(recorder.Body.String()).To(Equal(""))
	})

	type expectedImpersonationReq struct {
		resAttr *authzv1.ResourceAttributes
		allowed bool
		err     error
	}

	DescribeTable("Authenticate user based on impersonation headers",
		func(token, impersonateUser string, impersonateGroups []string, extras map[string][]string,
			expectedImpersonationReq []expectedImpersonationReq, expectedStatus int, expectedBody string, expectedUser *user.DefaultInfo) {
			// Setup the token of the service account that will be doing the impersonation
			req.Header.Set("Authorization", token)
			// Setup up impersonation headers
			req.Header.Set(authenticationv1.ImpersonateUserHeader, impersonateUser)
			for _, group := range impersonateGroups {
				req.Header.Add(authenticationv1.ImpersonateGroupHeader, group)
			}
			for extraKey, values := range extras {
				for _, value := range values {
					req.Header.Add(fmt.Sprintf("%s%s", authenticationv1.ImpersonateUserExtraHeaderPrefix, extraKey), value)
				}
			}

			// Bootstrap test handler
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Expect authentication information to be set on the context
				usr, ok := request.UserFrom(r.Context())
				Expect(ok).To(BeTrue())
				Expect(usr).To(Equal(expectedUser))
			})

			// Bootstrap the serviceAccount that will be doing impersonation
			var user = &user.DefaultInfo{
				Name: "guardian",
			}

			// Mock authorization
			var mockCache = &cache.MockClientCache{}
			var mockAuth = &lmaauth.MockRBACAuthorizer{}
			mockCache.On("GetAuthorizer", lmak8s.DefaultCluster).Return(mockAuth, nil)
			for _, impReq := range expectedImpersonationReq {
				mockAuth.On("Authorize", user, impReq.resAttr, (*authzv1.NonResourceAttributes)(nil)).Return(impReq.allowed, impReq.err)
			}

			// Mock authentication for the service account doing the impersonation
			var mockAuthenticator = &mockAuthenticator{}
			mockAuthenticator.On("Authenticate", "guardian").Return(user, http.StatusOK, nil)

			var auth = middleware.NewAuth(mockAuthenticator, mockCache)

			// Bootstrap the http recorder
			recorder := httptest.NewRecorder()
			handler := auth.Authenticate(testHandler)
			handler.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(expectedStatus))
			Expect(strings.Trim(recorder.Body.String(), "\n")).To(Equal(expectedBody))
		},
		Entry("Impersonate serviceAccount", "guardian", "system:serviceaccount:default:jane",
			[]string{"system:serviceaccounts", "system:serviceaccounts:default", "system:authenticated"},
			make(map[string][]string), []expectedImpersonationReq{
				{
					resAttr: &authzv1.ResourceAttributes{
						Verb:      "impersonate",
						Resource:  "serviceaccounts",
						Name:      "jane",
						Namespace: "default",
					},
					allowed: true,
				},
				{
					resAttr: &authzv1.ResourceAttributes{
						Verb:     "impersonate",
						Resource: "groups",
						Name:     "system:serviceaccounts",
					},
					allowed: true,
				},
				{
					resAttr: &authzv1.ResourceAttributes{
						Verb:     "impersonate",
						Resource: "groups",
						Name:     "system:serviceaccounts:default",
					},
					allowed: true,
				},
				{
					resAttr: &authzv1.ResourceAttributes{
						Verb:     "impersonate",
						Resource: "groups",
						Name:     "system:authenticated",
					},
					allowed: true,
				},
			}, http.StatusOK, "", &user.DefaultInfo{
				Name:   "system:serviceaccount:default:jane",
				Groups: []string{"system:serviceaccounts", "system:serviceaccounts:default", "system:authenticated"},
				Extra:  map[string][]string{},
			}),
		Entry("Impersonate user", "guardian", "jane",
			[]string{"system:authenticated"},
			make(map[string][]string), []expectedImpersonationReq{
				{
					resAttr: &authzv1.ResourceAttributes{
						Verb:     "impersonate",
						Resource: "users",
						Name:     "jane",
					},
					allowed: true,
				},
				{
					resAttr: &authzv1.ResourceAttributes{
						Verb:     "impersonate",
						Resource: "groups",
						Name:     "system:authenticated",
					},
					allowed: true,
				},
			}, http.StatusOK, "", &user.DefaultInfo{
				Name:   "jane",
				Groups: []string{"system:authenticated"},
				Extra:  map[string][]string{},
			}),
		Entry("Impersonate extra scopes", "guardian", "jane",
			[]string{"system:authenticated"},
			map[string][]string{
				"scopes":             {"view", "deployment"},
				"acme.com%2Fproject": {"some-project"},
			}, []expectedImpersonationReq{
				{
					resAttr: &authzv1.ResourceAttributes{
						Verb:     "impersonate",
						Resource: "users",
						Name:     "jane",
					},
					allowed: true,
				},
				{
					resAttr: &authzv1.ResourceAttributes{
						Verb:     "impersonate",
						Resource: "groups",
						Name:     "system:authenticated",
					},
					allowed: true,
				},
				{
					resAttr: &authzv1.ResourceAttributes{
						Verb:        "impersonate",
						Resource:    "userextras",
						Subresource: "scopes",
						Name:        "view",
					},
					allowed: true,
				},
				{
					resAttr: &authzv1.ResourceAttributes{
						Verb:        "impersonate",
						Resource:    "userextras",
						Subresource: "scopes",
						Name:        "deployment",
					},
					allowed: true,
				},
				{
					resAttr: &authzv1.ResourceAttributes{
						Verb:        "impersonate",
						Resource:    "userextras",
						Subresource: "acme.com/project",
						Name:        "some-project",
					},
					allowed: true,
				},
			}, http.StatusOK, "", &user.DefaultInfo{
				Name:   "jane",
				Groups: []string{"system:authenticated"},
				Extra: map[string][]string{
					"scopes":           {"view", "deployment"},
					"acme.com/project": {"some-project"}},
			}),
		Entry("Missing user impersonation header", "guardian", "",
			[]string{"system:authenticated"},
			map[string][]string{},
			[]expectedImpersonationReq{}, http.StatusBadRequest, "missing impersonation headers", &user.DefaultInfo{}),
		Entry("Token not allowed to impersonate user", "guardian", "jane",
			[]string{"system:authenticated"},
			make(map[string][]string), []expectedImpersonationReq{
				{
					resAttr: &authzv1.ResourceAttributes{
						Verb:     "impersonate",
						Resource: "users",
						Name:     "jane",
					},
					allowed: false,
				},
			}, http.StatusUnauthorized, "guardian is not authorized to impersonate for users", &user.DefaultInfo{}),
		Entry("Failure to impersonate users", "guardian", "jane",
			[]string{"system:authenticated"},
			make(map[string][]string), []expectedImpersonationReq{
				{
					resAttr: &authzv1.ResourceAttributes{
						Verb:     "impersonate",
						Resource: "users",
						Name:     "jane",
					},
					allowed: false,
					err:     fmt.Errorf("any error"),
				},
			}, http.StatusUnauthorized, "any error", &user.DefaultInfo{}),
	)
})

type mockAuthenticator struct {
	mock.Mock
}

func (m *mockAuthenticator) Authenticate(token string) (user.Info, int, error) {
	args := m.Called(token)
	return args.Get(0).(user.Info), args.Int(1), args.Error(2)
}
