// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package auth_test

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	auth "github.com/tigera/voltron/internal/pkg/auth"
)

var _ = Describe("Token", func() {
	Describe("Extracts tokens", func() {
		Context("from http request", func() {
			It("should extract Bearer token", func() {
				token, tokenType := auth.Extract(requestWithHeader(map[string][]string{"Authorization": {"Bearer Token"}}))
				Expect(token).To(Equal("Token"))
				Expect(tokenType).To(Equal(auth.Bearer))
			})

			It("should not extract missing Bearer Token", func() {
				token, tokenType := auth.Extract(requestWithHeader(map[string][]string{"Authorization": {"Bearer"}}))
				Expect(token).To(Equal(""))
				Expect(tokenType).To(Equal(auth.Unknown))
			})

			It("should not extract empty Authorization header", func() {
				token, tokenType := auth.Extract(requestWithHeader(map[string][]string{"Authorization": {""}}))
				Expect(token).To(Equal(""))
				Expect(tokenType).To(Equal(auth.Unknown))
			})

			It("should not extract invalid token {Bearer Invalid }", func() {
				token, tokenType := auth.Extract(requestWithHeader(map[string][]string{"Authorization": {"Bearer Invalid "}}))
				Expect(token).To(Equal(""))
				Expect(tokenType).To(Equal(auth.Unknown))
			})

			It("should extract Basic Token {Basic:Pwd}", func() {
				token, tokenType := auth.Extract(requestWithHeader(map[string][]string{"Authorization": {"Basic Token"}}))
				Expect(token).To(Equal("Token"))
				Expect(tokenType).To(Equal(auth.Basic))
			})

			It("should not extract invalid Basic Token", func() {
				token, tokenType := auth.Extract(requestWithHeader(map[string][]string{"Authorization": {"Basic  "}}))
				Expect(token).To(Equal(""))
				Expect(tokenType).To(Equal(auth.Unknown))
			})

			It("should not extract tokens from empty headers", func() {
				token, tokenType := auth.Extract(requestWithHeader(map[string][]string{}))
				Expect(token).To(Equal(""))
				Expect(tokenType).To(Equal(auth.Unknown))
			})

		})
	})

})

func requestWithHeader(headers map[string][]string) *http.Request {
	request := &http.Request{Header: http.Header{}}
	for key, values := range headers {
		for _, value := range values {
			request.Header.Set(key, value)
		}
	}
	return request
}
