// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package auth_test

import (
	"encoding/base64"

	"github.com/tigera/voltron/internal/pkg/test"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/tigera/voltron/internal/pkg/auth"
)

var _ = Describe("Authenticator", func() {
	Describe("authenticates Bearer token", func() {
		Context("against k8s api", func() {

			client := test.NewK8sSimpleFakeClient(nil, nil)
			authenticator := auth.NewBearerAuthenticator(client)

			It("should not authenticate empty token ", func() {
				_, err := authenticator.Authenticate("")
				Expect(err).To(HaveOccurred())
			})

			It("should not authenticate invalid token ", func() {
				_, err := authenticator.Authenticate("$#%")
				Expect(err).To(HaveOccurred())
			})

			It("should authenticate a valid token for jane", func() {
				client.AddJaneIdentity()
				user, err := authenticator.Authenticate(test.JaneBearerToken)
				Expect(err).NotTo(HaveOccurred())
				Expect(user.Name).To(Equal(test.Jane))
				Expect(user.Groups).To(Equal([]string{test.Developers}))
			})

			It("should not authenticate a valid token for bob", func() {
				client.AddBobIdentity()
				_, err := authenticator.Authenticate(test.BobBearerToken)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("authenticates Basic token", func() {
		Context("against k8s api", func() {
			apiGen := test.NewFakeK8sClientGenerator()
			authenticator := auth.NewBasicAuthenticator(apiGen)

			It("should not authenticate empty token ", func() {
				_, err := authenticator.Authenticate("")
				Expect(err).To(HaveOccurred())
			})

			It("should not authenticate invalid token jane", func() {
				_, err := authenticator.Authenticate("jane")
				Expect(err).To(HaveOccurred())
			})

			It("should not authenticate invalid token :jane", func() {
				_, err := authenticator.Authenticate(":jane")
				Expect(err).To(HaveOccurred())
			})

			It("should not authenticate invalid token jane:password:extra", func() {
				_, err := authenticator.Authenticate("jane:password:extra")
				Expect(err).To(HaveOccurred())
			})

			It("should authenticate a valid token jane:password", func() {
				apiGen.AddJaneAccessReview()
				user, err := authenticator.Authenticate(base64.StdEncoding.EncodeToString([]byte(test.JanePassword)))
				Expect(err).NotTo(HaveOccurred())
				Expect(user.Name).To(Equal("jane"))
				Expect(user.Groups).To(Equal([]string{"system:authenticated"}))
			})

			It("should not authenticate an unknown user bob", func() {
				apiGen.AddBobAccessReview()
				_, err := authenticator.Authenticate(base64.StdEncoding.EncodeToString([]byte(test.BobPassword)))
				Expect(err).To(HaveOccurred())
			})

			It("should not authenticate an user if K8s cannot respond", func() {
				apiGen.AddErrorAccessReview()
				_, err := authenticator.Authenticate(base64.StdEncoding.EncodeToString([]byte(test.AnyUserPassword)))
				Expect(err).To(HaveOccurred())
			})

			It("should not authenticate an user if K8s Api cannot be tailored per user", func() {
				_, err := authenticator.Authenticate(base64.StdEncoding.EncodeToString([]byte("missingUser:password")))
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
