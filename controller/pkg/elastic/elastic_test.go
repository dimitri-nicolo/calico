// Copyright 2019 Tigera Inc. All rights reserved.

package elastic

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/url"
	url2 "net/url"
	"strings"
	"testing"
	"time"

	"github.com/araddon/dateparse"
	. "github.com/onsi/gomega"
)

const (
	baseURI = "http://127.0.0.1:9200"
)

func TestElastic_GetIPSetModified(t *testing.T) {
	g := NewGomegaWithT(t)

	u, err := url2.Parse(baseURI)
	g.Expect(err).ShouldNot(HaveOccurred())
	client := &http.Client{
		Transport: http.RoundTripper(&testRoundTripper{}),
	}

	e := NewElastic(client, u, "", "")

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	tm, err := e.GetIPSetModified(ctx, "test")
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(tm).Should(Equal(dateparse.MustParse("2019-03-18T12:29:18.590008-03:00")))

	tm, err = e.GetIPSetModified(ctx, "test2")
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(tm).Should(Equal(dateparse.MustParse("2019-03-20T14:40:52-03:00")))

	tm, err = e.GetIPSetModified(ctx, "test3")
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(tm).Should(Equal(time.Time{}))

	tm, err = e.GetIPSetModified(ctx, "test4")
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(tm).Should(Equal(time.Time{}))

	_, err = e.GetIPSetModified(ctx, "test5")
	g.Expect(err).Should(HaveOccurred())

	_, err = e.GetIPSetModified(ctx, "test6")
	g.Expect(err).Should(HaveOccurred())
}

type testRoundTripper struct {
	u *url.URL
}

func (*testRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	switch req.URL.String() {
	case baseURI:
		return &http.Response{
			StatusCode: 200,
			Request:    req,
			Body:       ioutil.NopCloser(strings.NewReader("")),
		}, nil
	case baseURI + "/.tigera.ipset/_doc/test?_source_include=created_at":
		return &http.Response{
			StatusCode: 200,
			Request:    req,
			Body:       ioutil.NopCloser(strings.NewReader(`{"_index":".tigera.ipset","_type":"_doc","_id":"test","_version":2,"found":true,"_source":{"created_at":"2019-03-18T12:29:18.590008-03:00"}}`)),
		}, nil
	case baseURI + "/.tigera.ipset/_doc/test2?_source_include=created_at":
		return &http.Response{
			StatusCode: 200,
			Request:    req,
			Body:       ioutil.NopCloser(strings.NewReader(`{"_index":".tigera.ipset","_type":"_doc","_id":"test2","_version":2,"found":true,"_source":{"created_at":"1553103652"}}`)),
		}, nil
	case baseURI + "/.tigera.ipset/_doc/test3?_source_include=created_at":
		return &http.Response{
			StatusCode: 200,
			Request:    req,
			Body:       ioutil.NopCloser(strings.NewReader(`{"_index":".tigera.ipset","_type":"_doc","_id":"test3","_version":2,"found":true}`)),
		}, nil
	case baseURI + "/.tigera.ipset/_doc/test4?_source_include=created_at":
		return &http.Response{
			StatusCode: 200,
			Request:    req,
			Body:       ioutil.NopCloser(strings.NewReader(`{"_index":".tigera.ipset","_type":"_doc","_id":"test4","_version":2,"found":true,"_source":{}}`)),
		}, nil
	case baseURI + "/.tigera.ipset/_doc/test5?_source_include=created_at":
		return &http.Response{
			StatusCode: 200,
			Request:    req,
			Body:       ioutil.NopCloser(strings.NewReader(`{"_index":".tigera.ipset","_type":"_doc","_id":"test5","_version":2,"found":true,"_source":{"created_at": 1553103652}}`)),
		}, nil
	case baseURI + "/.tigera.ipset/_doc/test6?_source_include=created_at":
		return &http.Response{
			StatusCode: 200,
			Request:    req,
			Body:       ioutil.NopCloser(strings.NewReader(`{"_index":".tigera.ipset","_type":"_doc","_id":"test6","_version":2,"found":true,"_source":{}`)),
		}, nil
	default:
		panic("unknown URI")
	}
}
