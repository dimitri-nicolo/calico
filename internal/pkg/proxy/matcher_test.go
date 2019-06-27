// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package proxy_test

import (
	"net/http"
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/tigera/voltron/internal/pkg/proxy"
	"github.com/tigera/voltron/internal/pkg/targets"
)

var _ = Describe("Selector", func() {
	const RegexStartsWithApi = "^/api"
	const RegexApi = "api"
	const ApiUrl = "http://abc.com/api"
	const ApisUrl = "http://abc.com/apis"

	DescribeTable("matches path regex from any HTTP request",
		func(request *http.Request, tgts map[string]*url.URL, expectedValue string) {
			pathExtractor := proxy.NewPathMatcher(targets.New(tgts))
			path, err := pathExtractor.Match(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(path).To(Equal(parse(expectedValue)))
		},
		Entry("should match GET using localhost and port with /api",
			request("http://localhost:123/api"),
			map[string]*url.URL{RegexStartsWithApi: parse("http://localhost:123")},
			"http://localhost:123"),
		Entry("should match GET with /api",
			request(ApiUrl),
			map[string]*url.URL{RegexStartsWithApi: parse(ApiUrl)},
			ApiUrl),
	)

	DescribeTable("not matches path regex from HTTP request",
		func(request *http.Request, tgts map[string]*url.URL) {
			pathExtractor := proxy.NewPathMatcher(targets.New(tgts))
			_, err := pathExtractor.Match(request)
			Expect(err).To(HaveOccurred())
		},
		Entry("should return error when request has no path defined",
			request("http://abc.com"),
			map[string]*url.URL{"/v1/api": parse(ApiUrl)}),
	)

	const header = "x-target"
	headers := map[string][]string{header: {"api"}}

	DescribeTable("matches header regex from any HTTP request",
		func(request *http.Request, tgts map[string]*url.URL, expectedValue string) {
			headerMatcher := proxy.NewHeaderMatcher(targets.New(tgts), header)
			path, err := headerMatcher.Match(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(path).To(Equal(parse(expectedValue)))
		},
		Entry("should match GET with /api",
			requestWithHeader(headers),
			map[string]*url.URL{RegexApi: parse(ApiUrl)},
			ApiUrl),
	)

	DescribeTable("not matches header regex from any HTTP request",
		func(request *http.Request, tgts map[string]*url.URL) {
			headerMatcher := proxy.NewHeaderMatcher(targets.New(tgts), header)
			_, err := headerMatcher.Match(request)
			Expect(err).To(HaveOccurred())
		},
		Entry("should return an error when passing multiple headers",
			requestWithHeader(map[string][]string{header: {"api", "abc"}}),
			map[string]*url.URL{RegexApi: parse(ApiUrl)}),
	)

	DescribeTable("matches regex",
		func(value string, tgts map[string]*url.URL, expectedValue string) {
			target, err := proxy.Match(value, tgts)
			Expect(err).NotTo(HaveOccurred())
			Expect(target).To(Equal(parse(expectedValue)))
		},
		Entry("should match /apis",
			"/apis",
			map[string]*url.URL{RegexStartsWithApi: parse(ApisUrl)},
			ApisUrl),
		Entry("should match /apis and ignore invalid regex",
			"/apis",
			map[string]*url.URL{`(?!\/)`: parse(ApiUrl), RegexStartsWithApi: parse(ApisUrl)},
			ApisUrl),
	)

	DescribeTable("not matches regex",
		func(value string, tgts map[string]*url.URL) {
			_, err := proxy.Match(value, tgts)
			Expect(err).To(HaveOccurred())
		},
		Entry("should return error when targets are not set",
			"anyValue",
			map[string]*url.URL{}),
		Entry("should return error when matching empty string",
			"",
			map[string]*url.URL{}),
		Entry("should return error when targets are not matched",
			"anyValue",
			map[string]*url.URL{"/v1/api": parse(ApiUrl)}),
	)
})

func request(rawUrl string) *http.Request {
	return &http.Request{URL: parse(rawUrl)}
}

func requestWithHeader(headers map[string][]string) *http.Request {
	request := &http.Request{Header: http.Header{}}
	for key, values := range headers {
		for _, value := range values {
			request.Header.Set(key, value)
		}
	}
	return request
}

func parse(rawUrl string) *url.URL {
	value, err := url.Parse(rawUrl)
	if err != nil {
		panic(err)
	}

	return value
}
