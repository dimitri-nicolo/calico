// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package auth

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/tigera/apiserver/pkg/authentication"

	"k8s.io/apiserver/pkg/endpoints/request"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	validHeader = "Bearer jane"
	validUser = "jane"
	validGroup = "system:authenticated"
)

var _ = Describe("Test authentication", func() {

	var (
		cfg  rest.Config
		ki   k8s.Interface
		req  *http.Request
		ka   *k8sauth
		stat int
		err  error
	)

	BeforeEach(func() {
		cfg = rest.Config{}
		ki, err = k8s.NewForConfig(&cfg)
		Expect(err).NotTo(HaveOccurred())

		authenticator := authentication.NewFakeAuthenticator()
		authenticator.AddValidApiResponse(validHeader, validUser, []string{validGroup})
		ka = &k8sauth{k8sApi: ki, authenticator: authenticator}
		req, err = http.NewRequest("GET", "https://tigera-elasticsearch/flowLogs", nil)
		Expect(err).NotTo(HaveOccurred())
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
		Expect(usr.GetName()).To(Equal(validUser))
		Expect(usr.GetGroups()[0]).To(Equal(validGroup))
	})
})
