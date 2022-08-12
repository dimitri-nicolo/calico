// Copyright (c) 2022 Tigera. All rights reserved.
package authhandler_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/libcalico-go/lib/apiconfig"
	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	lmaauth "github.com/projectcalico/calico/lma/pkg/auth"
	"github.com/projectcalico/calico/lma/pkg/auth/testing"
	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/client"
	"github.com/projectcalico/calico/ts-queryserver/queryserver/handlers"
	authhandler "github.com/projectcalico/calico/ts-queryserver/queryserver/handlers/auth"

	authzv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"
)

var _ = Describe("Queryserver query test", func() {
	const (
		iss      = "https://example.com/my-issuer"
		name     = "Gerrit"
		interval = 10 * time.Millisecond
	)

	var (
		authnz     lmaauth.JWTAuth
		c          clientv3.Interface
		mAuth      *mockJWTAuth
		fakeK8sCli *fake.Clientset
		jwt        = testing.NewFakeJWT(iss, name)
		userInfo   = &user.DefaultInfo{Name: "default"}
		qh         handlers.Query
	)

	BeforeEach(func() {
		cfg, err := apiconfig.LoadClientConfig("")
		Expect(err).NotTo(HaveOccurred())
		cfg.Spec.DatastoreType = "etcdv3"
		cfg.Spec.EtcdEndpoints = "http://localhost:2379"
		c, err = clientv3.New(*cfg)
		Expect(err).NotTo(HaveOccurred())

		mAuth = &mockJWTAuth{}
		fakeK8sCli = new(fake.Clientset)
		authnz, err = lmaauth.NewJWTAuth(
			&rest.Config{BearerToken: jwt.ToString()}, fakeK8sCli, lmaauth.WithAuthenticator(iss, mAuth))
		Expect(err).NotTo(HaveOccurred())

		qh = handlers.NewQuery(client.NewQueryInterface(c))
	})

	It("returns a valid handler", func() {
		name := "/endpoints"
		verb := "get"

		By("Defining a new request.")
		req, _ := http.NewRequest(strings.ToUpper(verb), name, nil)

		By("Adding the authorization bearer token to the request header.")
		req.Header.Set("Authorization", jwt.BearerTokenHeader())

		By("Adding access for the user.")
		mAuth.MockJWTAuth.On("Authenticate", req).Return(userInfo, 200, nil)
		addAccessReviewsReactor(fakeK8sCli, true, userInfo)

		user := &user.DefaultInfo{Name: "default", UID: "", Groups: []string(nil), Extra: map[string][]string(nil)}
		resource := &authzv1.ResourceAttributes{Namespace: "", Verb: "get", Group: "", Version: "", Resource: "services/proxy", Subresource: "", Name: "https:tigera-api:8080"}
		nonResource := (*authzv1.NonResourceAttributes)(nil)
		mAuth.MockJWTAuth.On("Authorize", user, resource, nonResource).Return(true, nil)

		By("Defining the authentication handler.")
		handler := authhandler.NewAuthHandler(mAuth)
		ah := handler.AuthenticationHandler(qh.Endpoints)

		By("Calling the handler function.")
		rr := httptest.NewRecorder()
		go ah.ServeHTTP(rr, req)
		time.Sleep(interval)

		By("Verifying an OK status is returned from the auth handler.")
		Expect(rr.Code).To(Equal(http.StatusOK))
		Expect(req.URL.Path).To(Equal(name))
	})

	It("blocks requests without a bearer token, 401", func() {
		name := "/endpoints"
		verb := "get"

		By("Defining a new request.")
		req, _ := http.NewRequest(strings.ToUpper(verb), name, nil)

		By("Defining the authentication handler.")
		handler := authhandler.NewAuthHandler(authnz)
		ah := handler.AuthenticationHandler(qh.Endpoints)

		By("Calling the handler function.")
		rr := httptest.NewRecorder()
		go ah.ServeHTTP(rr, req)
		time.Sleep(interval)

		By("Verifying the recorder error code is 401.")
		Expect(rr.Body.String()).To(Equal("no token present in request\nno token present in request"))
		Expect(rr.Code).To(Equal(http.StatusUnauthorized))
		Expect(req.URL.Path).To(Equal(name))
	})

	It("blocks requests with invalid bearer token, 401", func() {
		name := "/endpoints"
		verb := "get"
		invalidToken := "Bearer FXqRgmleo343ygsl34kl"

		By("Defining a new request.")
		req, _ := http.NewRequest(strings.ToUpper(verb), name, nil)

		By("Adding the authorization bearer token to the request header.")
		req.Header.Set("Authorization", invalidToken)

		By("Defining the authentication handler.")
		handler := authhandler.NewAuthHandler(authnz)

		By("Getting the authentication handler function.")
		ah := handler.AuthenticationHandler(qh.Endpoints)

		By("Calling the handler function.")
		rr := httptest.NewRecorder()
		go ah.ServeHTTP(rr, req)
		time.Sleep(interval)

		By("Verifying the recorder error code is 401.")
		Expect(rr.Code).To(Equal(http.StatusUnauthorized))
		Expect(req.URL.Path).To(Equal(name))
	})

	It("blocks requests with a bearer token with invalid prefix, 401", func() {
		name := "/endpoints"
		verb := "get"
		invalidToken := jwt.BearerTokenHeader()
		prefix := "Bearer "
		invalidToken = strings.TrimPrefix(invalidToken, prefix)

		By("Defining a new request.")
		req, _ := http.NewRequest(strings.ToUpper(verb), name, nil)

		By("Adding the authorization bearer token to the request header.")
		req.Header.Set("Authorization", invalidToken)

		By("Defining the authentication handler.")
		handler := authhandler.NewAuthHandler(authnz)

		By("Getting the authentication handler function.")
		ah := handler.AuthenticationHandler(qh.Endpoints)

		By("Calling the handler function.")
		rr := httptest.NewRecorder()
		go ah.ServeHTTP(rr, req)
		time.Sleep(interval)

		By("Verifying the recorder error code is 401.")
		Expect(rr.Code).To(Equal(http.StatusUnauthorized))
		Expect(req.URL.Path).To(Equal(name))
	})

	It("blocks unauthorized requests", func() {
		name := "/endpoints"
		verb := "get"

		By("Defining a new request.")
		req, _ := http.NewRequest(strings.ToUpper(verb), name, nil)

		By("Adding the authorization bearer token to the request header.")
		req.Header.Set("Authorization", jwt.BearerTokenHeader())

		By("Denying access for the user.")
		isAuthorized := false
		mAuth.MockJWTAuth.On("Authenticate", req).Return(userInfo, 200, nil)
		addAccessReviewsReactor(fakeK8sCli, isAuthorized, userInfo)

		user := &user.DefaultInfo{Name: "default", UID: "", Groups: []string(nil), Extra: map[string][]string(nil)}
		resource := &authzv1.ResourceAttributes{Namespace: "", Verb: "get", Group: "", Version: "", Resource: "services/proxy", Subresource: "", Name: "https:tigera-api:8080"}
		nonResource := (*authzv1.NonResourceAttributes)(nil)
		mAuth.MockJWTAuth.On("Authorize", user, resource, nonResource).Return(false, nil)

		By("Defining the authentication handler.")
		handler := authhandler.NewAuthHandler(mAuth)

		By("Getting the authentication handler function.")
		ah := handler.AuthenticationHandler(qh.Endpoints)

		By("Calling the handler function.")
		rr := httptest.NewRecorder()
		go ah.ServeHTTP(rr, req)
		time.Sleep(interval)

		By("Verifying the recorder error code is 403.")
		Expect(rr.Body.String()).To(Equal("user &{default  [] map[]} is not authorized to perform GET https:tigera-api:8080"))
		Expect(rr.Code).To(Equal(http.StatusForbidden))
		Expect(req.URL.Path).To(Equal(name))
	})

	It("blocks unauthorized requests, due to authorization error", func() {
		name := "/endpoints"
		verb := "get"

		By("Defining a new request.")
		req, _ := http.NewRequest(strings.ToUpper(verb), name, nil)
		By("Adding the authorization bearer token to the request header.")
		req.Header.Set("Authorization", jwt.BearerTokenHeader())

		mAuth.MockJWTAuth.On("Authenticate", req).Return(userInfo, 200, nil)

		user := &user.DefaultInfo{Name: "default", UID: "", Groups: []string(nil), Extra: map[string][]string(nil)}
		resource := &authzv1.ResourceAttributes{Namespace: "", Verb: "get", Group: "", Version: "", Resource: "services/proxy", Subresource: "", Name: "https:tigera-api:8080"}
		nonResource := (*authzv1.NonResourceAttributes)(nil)
		mAuth.MockJWTAuth.On("Authorize", user, resource, nonResource).Return(true, errors.New("internal server error."))

		By("Defining the authentication handler.")
		handler := authhandler.NewAuthHandler(mAuth)

		By("Getting the authentication handler function.")
		ah := handler.AuthenticationHandler(qh.Endpoints)

		By("Calling the handler function.")
		rr := httptest.NewRecorder()
		go ah.ServeHTTP(rr, req)
		time.Sleep(interval)

		By("Verifying an OK status is returned from the auth handler.")
		Expect(rr.Body.String()).To(Equal("internal server error."))
		Expect(rr.Code).To(Equal(http.StatusInternalServerError))
		Expect(req.URL.Path).To(Equal(name))
	})

	It("blocks invalid method", func() {
		name := "/endpoints"
		verb := "git"

		By("Defining a new request.")
		req, _ := http.NewRequest(strings.ToUpper(verb), name, nil)
		By("Adding the authorization bearer token to the request header.")
		req.Header.Set("Authorization", jwt.BearerTokenHeader())

		mAuth.MockJWTAuth.On("Authenticate", req).Return(userInfo, 200, nil)
		addAccessReviewsReactor(fakeK8sCli, false, userInfo)

		By("Defining the authentication handler.")
		handler := authhandler.NewAuthHandler(mAuth)

		By("Getting the authentication handler function.")
		authHandler := handler.AuthenticationHandler(qh.Endpoints)

		By("Calling the handler function.")
		rr := httptest.NewRecorder()
		go authHandler.ServeHTTP(rr, req)
		time.Sleep(interval)

		By("Verifying an OK status is returned from the auth handler.")
		Expect(rr.Body.String()).To(Equal("Method Not Allowed"))
		Expect(rr.Code).To(Equal(http.StatusMethodNotAllowed))
		Expect(req.URL.Path).To(Equal(name))
	})
})

func addAccessReviewsReactor(fakeK8sCli *fake.Clientset, authorized bool, userInfo user.Info) {
	fakeK8sCli.AddReactor("create", "subjectaccessreviews", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		extra := make(map[string]authzv1.ExtraValue)
		for k, v := range userInfo.GetExtra() {
			extra[k] = v
		}
		createAction, ok := action.(k8stesting.CreateAction)
		Expect(ok).To(BeTrue())
		review, ok := createAction.GetObject().(*authzv1.SubjectAccessReview)
		Expect(ok).To(BeTrue())
		Expect(review.Spec.User).To(Equal(userInfo.GetName()))
		Expect(review.Spec.UID).To(Equal(userInfo.GetUID()))
		Expect(review.Spec.Groups).To(Equal(userInfo.GetGroups()))
		Expect(review.Spec.Extra).To(Equal(extra))
		Expect(review.Spec.ResourceAttributes.Name).To(BeElementOf("https:tigera-api:8080",
			"/endpoints", "/endpoints/", "/policies", "/policies/", "/nodes", "/nodes/",
			"/summary", "/version", "/license"))
		Expect(review.Spec.ResourceAttributes.Resource).To(Equal("services/proxy"))
		Expect(review.Spec.ResourceAttributes.Verb).To(Equal("get"))
		return true, &authzv1.SubjectAccessReview{Status: authzv1.SubjectAccessReviewStatus{Allowed: authorized}}, nil
	})
}

type mockJWTAuth struct {
	lmaauth.MockJWTAuth
}
