// Copyright 2019 Tigera Inc. All rights reserved.

package elastic

import (
	"bytes"
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

func TestElastic_GetIPSet(t *testing.T) {
	g := NewGomegaWithT(t)

	u, err := url2.Parse(baseURI)
	g.Expect(err).ShouldNot(HaveOccurred())
	client := &http.Client{
		Transport: http.RoundTripper(&testRoundTripper{}),
	}

	e := NewElastic(client, u, "", "")

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	ipSet, err := e.GetIPSet(ctx, "test1")
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(ipSet).Should(ConsistOf("35.32.82.0/24", "10.10.1.20/32"))

	_, err = e.GetIPSet(ctx, "test2")
	g.Expect(err).Should(HaveOccurred(), "Missing source")

	_, err = e.GetIPSet(ctx, "test3")
	g.Expect(err).Should(HaveOccurred(), "Empty source")

	_, err = e.GetIPSet(ctx, "test4")
	g.Expect(err).Should(HaveOccurred(), "Invalid ips type")

	_, err = e.GetIPSet(ctx, "test5")
	g.Expect(err).Should(HaveOccurred(), "Invalid ips element type")

	ipSet, err = e.GetIPSet(ctx, "unknown")
	g.Expect(err).Should(HaveOccurred(), "Elastic error")
}

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
	g.Expect(err).ShouldNot(HaveOccurred(), "Proper response")
	g.Expect(tm).Should(Equal(dateparse.MustParse("2019-03-18T12:29:18.590008-03:00")))

	tm, err = e.GetIPSetModified(ctx, "test2")
	g.Expect(err).ShouldNot(HaveOccurred(), "String integer time")
	g.Expect(tm).Should(Equal(dateparse.MustParse("2019-03-20T14:40:52-03:00")))

	tm, err = e.GetIPSetModified(ctx, "test3")
	g.Expect(err).ShouldNot(HaveOccurred(), "Missing source")
	g.Expect(tm).Should(Equal(time.Time{}))

	tm, err = e.GetIPSetModified(ctx, "test4")
	g.Expect(err).ShouldNot(HaveOccurred(), "Empty source")
	g.Expect(tm).Should(Equal(time.Time{}))

	_, err = e.GetIPSetModified(ctx, "test5")
	g.Expect(err).Should(HaveOccurred(), "Invalid created_at type")

	// synthetic error 500
	_, err = e.GetIPSetModified(ctx, "unknown")
	g.Expect(err).Should(HaveOccurred(), "Elastic error")
}

func TestElastic_QueryIPSet(t *testing.T) {
	g := NewGomegaWithT(t)

	u, err := url2.Parse(baseURI)
	g.Expect(err).ShouldNot(HaveOccurred())
	client := &http.Client{
		Transport: http.RoundTripper(&testRoundTripper{}),
	}

	e := NewElastic(client, u, "", "")

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	itr, err := e.QueryIPSet(ctx, "test")
	g.Expect(err).ShouldNot(HaveOccurred())

	c := 0
	for itr.Next() {
		c++
	}
	g.Expect(itr.Err()).ShouldNot(HaveOccurred())
	g.Expect(c).Should(Equal(2))
}

type testRoundTripper struct {
	u *url.URL
}

func (*testRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	switch req.Method {
	case "HEAD":
		switch req.URL.String() {
		case baseURI:
			return &http.Response{
				StatusCode: 200,
				Request:    req,
				Body:       ioutil.NopCloser(strings.NewReader("")),
			}, nil
		}
	case "GET":
		switch req.URL.String() {
		// GetIPSet
		case baseURI + "/.tigera.ipset/_doc/test1":
			return &http.Response{
				StatusCode: 200,
				Request:    req,
				Body:       ioutil.NopCloser(strings.NewReader(`{"_index":".tigera.ipset","_type":"_doc","_id":"test1","_version":5,"found":true,"_source":{"ips":["35.32.82.0/24","10.10.1.20/32"]}}`)),
			}, nil
		case baseURI + "/.tigera.ipset/_doc/test2":
			return &http.Response{
				StatusCode: 200,
				Request:    req,
				Body:       ioutil.NopCloser(strings.NewReader(`{"_index":".tigera.ipset","_type":"_doc","_id":"test2","_version":5,"found":true}`)),
			}, nil
		case baseURI + "/.tigera.ipset/_doc/test3":
			return &http.Response{
				StatusCode: 200,
				Request:    req,
				Body:       ioutil.NopCloser(strings.NewReader(`{"_index":".tigera.ipset","_type":"_doc","_id":"test3","_version":5,"found":true,"_source":{}}`)),
			}, nil
		case baseURI + "/.tigera.ipset/_doc/test4":
			return &http.Response{
				StatusCode: 200,
				Request:    req,
				Body:       ioutil.NopCloser(strings.NewReader(`{"_index":".tigera.ipset","_type":"_doc","_id":"test4","_version":5,"found":true,"_source":{"ips":"123"}}`)),
			}, nil
		case baseURI + "/.tigera.ipset/_doc/test5":
			return &http.Response{
				StatusCode: 200,
				Request:    req,
				Body:       ioutil.NopCloser(strings.NewReader(`{"_index":".tigera.ipset","_type":"_doc","_id":"test4","_version":5,"found":true,"_source":{"ips":[123]}}`)),
			}, nil

		// GetIPSetModified
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

		}
	case "POST":
		b, _ := ioutil.ReadAll(req.Body)
		_ = req.Body.Close()
		body := string(b)
		req.Body = ioutil.NopCloser(bytes.NewReader(b))

		switch req.URL.String() {
		case baseURI + "/tigera_secure_ee_flows%2A/_search?scroll=5m&size=1000":
			switch body {
			// QueryIPSet source_ip query
			case `{"query":{"terms":{"source_ip":{"id":"test","index":".tigera.ipset","path":"ips","type":"_doc"}}},"sort":["_doc"]}`:
				return &http.Response{
					StatusCode: 200,
					Request:    req,
					Body:       ioutil.NopCloser(strings.NewReader(`{"_scroll_id":"DnF1ZXJ5VGhlbkZldGNoDwAAAAAABD7nFm9CTGZ2OWNZU2E2UjJtdGNYOVJpcUEAAAAAAAQ-5hZvQkxmdjljWVNhNlIybXRjWDlSaXFBAAAAAAAEPugWb0JMZnY5Y1lTYTZSMm10Y1g5UmlxQQAAAAAABD7pFm9CTGZ2OWNZU2E2UjJtdGNYOVJpcUEAAAAAAAQ-6hZvQkxmdjljWVNhNlIybXRjWDlSaXFBAAAAAAAEPuwWb0JMZnY5Y1lTYTZSMm10Y1g5UmlxQQAAAAAABD7rFm9CTGZ2OWNZU2E2UjJtdGNYOVJpcUEAAAAAAAQ-8xZvQkxmdjljWVNhNlIybXRjWDlSaXFBAAAAAAAEPu0Wb0JMZnY5Y1lTYTZSMm10Y1g5UmlxQQAAAAAABD7uFm9CTGZ2OWNZU2E2UjJtdGNYOVJpcUEAAAAAAAQ-7xZvQkxmdjljWVNhNlIybXRjWDlSaXFBAAAAAAAEPvAWb0JMZnY5Y1lTYTZSMm10Y1g5UmlxQQAAAAAABD7xFm9CTGZ2OWNZU2E2UjJtdGNYOVJpcUEAAAAAAAQ-8hZvQkxmdjljWVNhNlIybXRjWDlSaXFBAAAAAAAEPvQWb0JMZnY5Y1lTYTZSMm10Y1g5UmlxQQ==","took":18,"timed_out":false,"_shards":{"total":15,"successful":15,"skipped":0,"failed":0},"hits":{"total":1,"max_score":null,"hits":[{"_index":"tigera_secure_ee_flows.flowsynth.20180914","_type":"fluentd","_id":"BQ15nGkBixKz5K3LBMRy","_score":null,"_source":{"start_time":1536897600,"end_time":1536897900,"source_ip":"35.32.82.134","source_name":"-","source_name_aggr":"pub","source_namespace":"-","source_port":null,"source_type":"net","source_labels":null,"dest_ip":"10.10.1.20","dest_name":"basic-6tn46e","dest_name_aggr":"basic-*","dest_namespace":"default","dest_port":0,"dest_type":"wep","dest_labels":null,"proto":"tcp","action":"allow","reporter":"dst","policies":null,"bytes_in":18966,"bytes_out":10048,"num_flows":1,"num_flows_started":1,"num_flows_completed":1,"packets_in":172,"packets_out":84,"http_requests_allowed_in":0,"http_requests_denied_in":0,"host":"node02"},"sort":[0]}]}}`)),
				}, nil

			// QueryIPSet dest_ip query
			case `{"query":{"terms":{"dest_ip":{"id":"test","index":".tigera.ipset","path":"ips","type":"_doc"}}},"sort":["_doc"]}`:
				return &http.Response{
					StatusCode: 200,
					Request:    req,
					Body:       ioutil.NopCloser(strings.NewReader(`{"_scroll_id":"DnF1ZXJ5VGhlbkZldGNoDwAAAAAABEEmFm9CTGZ2OWNZU2E2UjJtdGNYOVJpcUEAAAAAAARBIBZvQkxmdjljWVNhNlIybXRjWDlSaXFBAAAAAAAEQSEWb0JMZnY5Y1lTYTZSMm10Y1g5UmlxQQAAAAAABEEiFm9CTGZ2OWNZU2E2UjJtdGNYOVJpcUEAAAAAAARBIxZvQkxmdjljWVNhNlIybXRjWDlSaXFBAAAAAAAEQSQWb0JMZnY5Y1lTYTZSMm10Y1g5UmlxQQAAAAAABEElFm9CTGZ2OWNZU2E2UjJtdGNYOVJpcUEAAAAAAARBJxZvQkxmdjljWVNhNlIybXRjWDlSaXFBAAAAAAAEQSgWb0JMZnY5Y1lTYTZSMm10Y1g5UmlxQQAAAAAABEEpFm9CTGZ2OWNZU2E2UjJtdGNYOVJpcUEAAAAAAARBKhZvQkxmdjljWVNhNlIybXRjWDlSaXFBAAAAAAAEQSsWb0JMZnY5Y1lTYTZSMm10Y1g5UmlxQQAAAAAABEEsFm9CTGZ2OWNZU2E2UjJtdGNYOVJpcUEAAAAAAARBLRZvQkxmdjljWVNhNlIybXRjWDlSaXFBAAAAAAAEQS4Wb0JMZnY5Y1lTYTZSMm10Y1g5UmlxQQ==","took":10,"timed_out":false,"_shards":{"total":15,"successful":15,"skipped":0,"failed":0},"hits":{"total":1,"max_score":null,"hits":[{"_index":"tigera_secure_ee_flows.flowsynth.20180914","_type":"fluentd","_id":"BQ15nGkBixKz5K3LBMRy","_score":null,"_source":{"start_time":1536897600,"end_time":1536897900,"source_ip":"35.32.82.134","source_name":"-","source_name_aggr":"pub","source_namespace":"-","source_port":null,"source_type":"net","source_labels":null,"dest_ip":"10.10.1.20","dest_name":"basic-6tn46e","dest_name_aggr":"basic-*","dest_namespace":"default","dest_port":0,"dest_type":"wep","dest_labels":null,"proto":"tcp","action":"allow","reporter":"dst","policies":null,"bytes_in":18966,"bytes_out":10048,"num_flows":1,"num_flows_started":1,"num_flows_completed":1,"packets_in":172,"packets_out":84,"http_requests_allowed_in":0,"http_requests_denied_in":0,"host":"node02"},"sort":[0]}]}} `)),
				}, nil
			}
		case baseURI + "/_search/scroll":
			switch body {
			// QueryIPSet source_ip query
			case `{"scroll":"5m","scroll_id":"DnF1ZXJ5VGhlbkZldGNoDwAAAAAABD7nFm9CTGZ2OWNZU2E2UjJtdGNYOVJpcUEAAAAAAAQ-5hZvQkxmdjljWVNhNlIybXRjWDlSaXFBAAAAAAAEPugWb0JMZnY5Y1lTYTZSMm10Y1g5UmlxQQAAAAAABD7pFm9CTGZ2OWNZU2E2UjJtdGNYOVJpcUEAAAAAAAQ-6hZvQkxmdjljWVNhNlIybXRjWDlSaXFBAAAAAAAEPuwWb0JMZnY5Y1lTYTZSMm10Y1g5UmlxQQAAAAAABD7rFm9CTGZ2OWNZU2E2UjJtdGNYOVJpcUEAAAAAAAQ-8xZvQkxmdjljWVNhNlIybXRjWDlSaXFBAAAAAAAEPu0Wb0JMZnY5Y1lTYTZSMm10Y1g5UmlxQQAAAAAABD7uFm9CTGZ2OWNZU2E2UjJtdGNYOVJpcUEAAAAAAAQ-7xZvQkxmdjljWVNhNlIybXRjWDlSaXFBAAAAAAAEPvAWb0JMZnY5Y1lTYTZSMm10Y1g5UmlxQQAAAAAABD7xFm9CTGZ2OWNZU2E2UjJtdGNYOVJpcUEAAAAAAAQ-8hZvQkxmdjljWVNhNlIybXRjWDlSaXFBAAAAAAAEPvQWb0JMZnY5Y1lTYTZSMm10Y1g5UmlxQQ=="}`:
				return &http.Response{
					StatusCode: 200,
					Request:    req,
					Body:       ioutil.NopCloser(strings.NewReader(`{"_scroll_id":"DnF1ZXJ5VGhlbkZldGNoDwAAAAAABD7nFm9CTGZ2OWNZU2E2UjJtdGNYOVJpcUEAAAAAAAQ-5hZvQkxmdjljWVNhNlIybXRjWDlSaXFBAAAAAAAEPugWb0JMZnY5Y1lTYTZSMm10Y1g5UmlxQQAAAAAABD7pFm9CTGZ2OWNZU2E2UjJtdGNYOVJpcUEAAAAAAAQ-6hZvQkxmdjljWVNhNlIybXRjWDlSaXFBAAAAAAAEPuwWb0JMZnY5Y1lTYTZSMm10Y1g5UmlxQQAAAAAABD7rFm9CTGZ2OWNZU2E2UjJtdGNYOVJpcUEAAAAAAAQ-8xZvQkxmdjljWVNhNlIybXRjWDlSaXFBAAAAAAAEPu0Wb0JMZnY5Y1lTYTZSMm10Y1g5UmlxQQAAAAAABD7uFm9CTGZ2OWNZU2E2UjJtdGNYOVJpcUEAAAAAAAQ-7xZvQkxmdjljWVNhNlIybXRjWDlSaXFBAAAAAAAEPvAWb0JMZnY5Y1lTYTZSMm10Y1g5UmlxQQAAAAAABD7xFm9CTGZ2OWNZU2E2UjJtdGNYOVJpcUEAAAAAAAQ-8hZvQkxmdjljWVNhNlIybXRjWDlSaXFBAAAAAAAEPvQWb0JMZnY5Y1lTYTZSMm10Y1g5UmlxQQ==","took":1,"timed_out":false,"_shards":{"total":15,"successful":15,"skipped":0,"failed":0},"hits":{"total":1,"max_score":null,"hits":[]}}`)),
				}, nil
			// QueryIPSet dest_ip query
			case `{"scroll":"5m","scroll_id":"DnF1ZXJ5VGhlbkZldGNoDwAAAAAABEEmFm9CTGZ2OWNZU2E2UjJtdGNYOVJpcUEAAAAAAARBIBZvQkxmdjljWVNhNlIybXRjWDlSaXFBAAAAAAAEQSEWb0JMZnY5Y1lTYTZSMm10Y1g5UmlxQQAAAAAABEEiFm9CTGZ2OWNZU2E2UjJtdGNYOVJpcUEAAAAAAARBIxZvQkxmdjljWVNhNlIybXRjWDlSaXFBAAAAAAAEQSQWb0JMZnY5Y1lTYTZSMm10Y1g5UmlxQQAAAAAABEElFm9CTGZ2OWNZU2E2UjJtdGNYOVJpcUEAAAAAAARBJxZvQkxmdjljWVNhNlIybXRjWDlSaXFBAAAAAAAEQSgWb0JMZnY5Y1lTYTZSMm10Y1g5UmlxQQAAAAAABEEpFm9CTGZ2OWNZU2E2UjJtdGNYOVJpcUEAAAAAAARBKhZvQkxmdjljWVNhNlIybXRjWDlSaXFBAAAAAAAEQSsWb0JMZnY5Y1lTYTZSMm10Y1g5UmlxQQAAAAAABEEsFm9CTGZ2OWNZU2E2UjJtdGNYOVJpcUEAAAAAAARBLRZvQkxmdjljWVNhNlIybXRjWDlSaXFBAAAAAAAEQS4Wb0JMZnY5Y1lTYTZSMm10Y1g5UmlxQQ=="}`:
				return &http.Response{
					StatusCode: 200,
					Request:    req,
					Body:       ioutil.NopCloser(strings.NewReader(`{"_scroll_id":"DnF1ZXJ5VGhlbkZldGNoDwAAAAAABEEmFm9CTGZ2OWNZU2E2UjJtdGNYOVJpcUEAAAAAAARBIBZvQkxmdjljWVNhNlIybXRjWDlSaXFBAAAAAAAEQSEWb0JMZnY5Y1lTYTZSMm10Y1g5UmlxQQAAAAAABEEiFm9CTGZ2OWNZU2E2UjJtdGNYOVJpcUEAAAAAAARBIxZvQkxmdjljWVNhNlIybXRjWDlSaXFBAAAAAAAEQSQWb0JMZnY5Y1lTYTZSMm10Y1g5UmlxQQAAAAAABEElFm9CTGZ2OWNZU2E2UjJtdGNYOVJpcUEAAAAAAARBJxZvQkxmdjljWVNhNlIybXRjWDlSaXFBAAAAAAAEQSgWb0JMZnY5Y1lTYTZSMm10Y1g5UmlxQQAAAAAABEEpFm9CTGZ2OWNZU2E2UjJtdGNYOVJpcUEAAAAAAARBKhZvQkxmdjljWVNhNlIybXRjWDlSaXFBAAAAAAAEQSsWb0JMZnY5Y1lTYTZSMm10Y1g5UmlxQQAAAAAABEEsFm9CTGZ2OWNZU2E2UjJtdGNYOVJpcUEAAAAAAARBLRZvQkxmdjljWVNhNlIybXRjWDlSaXFBAAAAAAAEQS4Wb0JMZnY5Y1lTYTZSMm10Y1g5UmlxQQ==","took":3,"timed_out":false,"_shards":{"total":15,"successful":15,"skipped":0,"failed":0},"hits":{"total":1,"max_score":null,"hits":[]}} `)),
				}, nil
			}
		}
	}

	return &http.Response{
		StatusCode: 500,
	}, nil
}
