// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package auth_test

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"github.com/tigera/lma/pkg/auth"

	"github.com/tigera/apiserver/pkg/authentication"
	"k8s.io/apiserver/pkg/endpoints/request"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var _ = Describe("Test authentication", func() {
	const (
		iss        = "https://127.0.0.1:9443/dex"
		exp        = 9600964803 //Very far in the future
		email      = "rene@tigera.io"
		clientID   = "tigera-manager"
		validGroup = "system:authenticated"
		k8sIss     = "kubernetes/serviceaccount"
	)
	var (
		cfg            rest.Config
		ki             k8s.Interface
		req            *http.Request
		ka             auth.K8sAuthInterface
		stat           int
		err            error
		validHeader    string
		validDexHeader string
	)

	BeforeEach(func() {
		cfg = rest.Config{}
		ki, err = k8s.NewForConfig(&cfg)
		Expect(err).NotTo(HaveOccurred())
		var payload []byte
		validHeader, _ = authHeader(k8sIss, email, clientID, exp)
		validDexHeader, payload = authHeader(iss, email, clientID, exp)
		keyset := &testKeySet{}
		keyset.On("VerifySignature", mock.Anything, mock.Anything).Return(payload, nil)
		authenticator := authentication.NewFakeAuthenticator()
		authenticator.AddValidApiResponse(validHeader, email, []string{validGroup})

		opts := []auth.DexOption{
			auth.WithGroupsClaim("groups"),
			auth.WithUsernamePrefix("-"),
			auth.WithKeySet(keyset),
		}
		dex, err := auth.NewDexAuthenticator(iss, clientID, "email", opts...)
		Expect(err).NotTo(HaveOccurred())
		ka = auth.NewK8sAuth(ki, auth.NewAggregateAuthenticator(dex, nil, authenticator))
		req, err = http.NewRequest("GET", "https://tigera-elasticsearch/flowLogs", nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(req.Header).NotTo(BeNil())
	})

	It("should reject requests without an auth header", func() {
		req, stat, err = ka.Authenticate(req)
		Expect(err).To(HaveOccurred())
		Expect(stat).To(Equal(http.StatusUnauthorized))
		usr, ok := request.UserFrom(req.Context())
		Expect(ok).To(BeFalse())
		Expect(usr).To(BeNil())
	})

	It("should authenticate existing user", func() {
		req, err = http.NewRequest("GET", "https://tigera-elasticsearch/flowLogs", nil)
		req.Header.Set(authentication.AuthorizationHeader, validHeader)
		req, stat, err = ka.Authenticate(req)
		Expect(err).NotTo(HaveOccurred())
		Expect(stat).To(Equal(http.StatusOK))
		usr, ok := request.UserFrom(req.Context())
		Expect(ok).To(BeTrue())
		Expect(usr).NotTo(BeNil())
		Expect(usr.GetName()).To(Equal(email))
		Expect(usr.GetGroups()[0]).To(Equal(validGroup))
	})

	It("should authenticate dex user", func() {
		req, err = http.NewRequest("GET", "https://tigera-elasticsearch/flowLogs", nil)
		req.Header.Set(authentication.AuthorizationHeader, validDexHeader)
		req, stat, err = ka.Authenticate(req)
		Expect(err).NotTo(HaveOccurred())
		Expect(stat).To(Equal(http.StatusOK))
		usr, ok := request.UserFrom(req.Context())
		Expect(ok).To(BeTrue())
		Expect(usr).NotTo(BeNil())
		Expect(usr.GetName()).To(Equal(email))
		Expect(usr.GetGroups()[0]).To(Equal("admins"))
	})
})
