// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package auth_test

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	authnv1 "k8s.io/api/authentication/v1"
	authzv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"

	"github.com/tigera/lma/pkg/auth"
	"github.com/tigera/lma/pkg/auth/testing"
)

var _ = Describe("Test dex username prefixes", func() {

	const (
		iss           = "https://127.0.0.1:9443/dex"
		clientID      = "tigera-manager"
		usernameClaim = "email"
		clusterIssuer = "https://kubernetes.default.svc"
	)

	var fakeK8sCli *fake.Clientset

	var (
		jwtAuth          auth.JWTAuth
		keySet           *testKeySet
		jwt              = testing.NewFakeServiceAccountJWT()
		impersonatingJWT = testing.NewFakeJWT(clusterIssuer, clientID)
	)

	BeforeEach(func() {
		keySet = &testKeySet{}
		opts := []auth.DexOption{
			auth.WithKeySet(keySet),
		}
		dex, err := auth.NewDexAuthenticator(iss, clientID, usernameClaim, opts...)
		Expect(err).NotTo(HaveOccurred())

		fakeK8sCli = new(fake.Clientset)

		jwtAuth, err = auth.NewJWTAuth(&rest.Config{BearerToken: impersonatingJWT.ToString()}, fakeK8sCli, auth.WithAuthenticator(iss, dex))
		Expect(err).NotTo(HaveOccurred())
	})

	It("Should authenticate a service account token", func() {
		addTokenReviewsReactor(fakeK8sCli, true, jwt.ToString())
		hdrs := http.Header{}
		hdrs.Set("Authorization", jwt.BearerTokenHeader())
		req := &http.Request{Header: hdrs}

		usr, stat, err := jwtAuth.Authenticate(req)
		Expect(err).NotTo(HaveOccurred())
		Expect(stat).To(Equal(200))
		Expect(usr.GetName()).To(Equal("default"))
	})

	It("Should reject an invalid service account token", func() {
		addTokenReviewsReactor(fakeK8sCli, false, jwt.ToString())
		hdrs := http.Header{}
		hdrs.Set("Authorization", jwt.BearerTokenHeader())
		req := &http.Request{Header: hdrs}

		_, stat, err := jwtAuth.Authenticate(req)
		Expect(err).To(HaveOccurred())
		Expect(stat).To(Equal(401))
	})

	It("Should refuse a missing jwtAuth header", func() {
		req := &http.Request{}
		usr, stat, err := jwtAuth.Authenticate(req)
		Expect(err).To(HaveOccurred())
		Expect(stat).To(Equal(401))
		Expect(usr).To(BeNil())
	})

	It("Should authenticate and impersonate", func() {
		addTokenReviewsReactor(fakeK8sCli, true, impersonatingJWT.ToString())
		addAccessReviewsReactor(fakeK8sCli, true, &user.DefaultInfo{
			Name: jwt.ServiceAccountPayload.KubernetesIoServiceaccountServiceAccountName,
		})
		hdrs := http.Header{}
		hdrs.Set("Authorization", impersonatingJWT.BearerTokenHeader())
		hdrs.Set(authnv1.ImpersonateUserHeader, "jane")
		hdrs.Set(authnv1.ImpersonateGroupHeader, "admin")
		req := &http.Request{Header: hdrs}

		usr, stat, err := jwtAuth.Authenticate(req)
		Expect(err).NotTo(HaveOccurred())
		Expect(stat).To(Equal(200))
		Expect(usr.GetName()).To(Equal("jane"))
		Expect(usr.GetGroups()).To(HaveLen(1))
		Expect(usr.GetGroups()[0]).To(Equal("admin"))
	})

	It("Should refuse service account that is not allowed to impersonate", func() {
		addTokenReviewsReactor(fakeK8sCli, true, impersonatingJWT.ToString())
		addAccessReviewsReactor(fakeK8sCli, false, &user.DefaultInfo{
			Name: jwt.ServiceAccountPayload.KubernetesIoServiceaccountServiceAccountName,
		})
		hdrs := http.Header{}
		hdrs.Set("Authorization", impersonatingJWT.BearerTokenHeader())
		hdrs.Set(authnv1.ImpersonateUserHeader, "jane")
		hdrs.Set(authnv1.ImpersonateGroupHeader, "admin")
		req := &http.Request{Header: hdrs}

		_, stat, err := jwtAuth.Authenticate(req)
		Expect(err).To(HaveOccurred())
		Expect(stat).To(Equal(401))
	})
})

func addTokenReviewsReactor(fakeK8sCli *fake.Clientset, authenticated bool, token string) {
	fakeK8sCli.AddReactor("create", "tokenreviews", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		createAction, ok := action.(k8stesting.CreateAction)
		Expect(ok).To(BeTrue())
		review, ok := createAction.GetObject().(*authnv1.TokenReview)
		Expect(ok).To(BeTrue())
		Expect(review.Spec).To(Equal(authnv1.TokenReviewSpec{
			Token: token,
		}))
		return true, &authnv1.TokenReview{Status: authnv1.TokenReviewStatus{User: authnv1.UserInfo{Username: "default"}, Authenticated: authenticated}}, nil
	})
}

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
		Expect(review.Spec.ResourceAttributes.Name).To(BeElementOf("jane", "admin"))
		Expect(review.Spec.ResourceAttributes.Resource).To(BeElementOf("users", "groups"))
		Expect(review.Spec.ResourceAttributes.Verb).To(Equal("impersonate"))
		return true, &authzv1.SubjectAccessReview{Status: authzv1.SubjectAccessReviewStatus{Allowed: authorized}}, nil
	})
}
