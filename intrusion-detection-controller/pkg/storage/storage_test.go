// Copyright 2019 Tigera Inc. All rights reserved.

package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	url2 "net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/projectcalico/calico/linseed/pkg/client/rest"

	lsclient "github.com/projectcalico/calico/linseed/pkg/client"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"

	"github.com/araddon/dateparse"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/util"
	lma "github.com/projectcalico/calico/lma/pkg/elastic"

	apiV3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

const (
	baseURI = "http://127.0.0.1:9200"
)

var oneMinuteAgo time.Time

func TestElastic_GetIPSet(t *testing.T) {
	g := NewGomegaWithT(t)

	u, err := url2.Parse(baseURI)
	g.Expect(err).ShouldNot(HaveOccurred())
	client := &http.Client{
		Transport: http.RoundTripper(&testRoundTripper{}),
	}

	lmaESCli, err := lma.New(client, u, "", "", "", 1, 0, false, 0, 0)
	g.Expect(err).Should(BeNil())
	// mock linseed client
	lsc := lsclient.NewMockClient("")
	e := NewService(lmaESCli, lsc, "cluster", DefaultIndexSettings())

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

	_, err = e.GetIPSet(ctx, "unknown")
	g.Expect(err).Should(HaveOccurred(), "Elastic error")
}

func TestElastic_GetIPSetModified(t *testing.T) {
	g := NewGomegaWithT(t)

	u, err := url2.Parse(baseURI)
	g.Expect(err).ShouldNot(HaveOccurred())
	client := &http.Client{
		Transport: http.RoundTripper(&testRoundTripper{}),
	}

	lmaESCli, err := lma.New(client, u, "", "", "", 1, 0, false, 0, 0)
	g.Expect(err).Should(BeNil())
	// mock linseed client
	lsc := lsclient.NewMockClient("")
	e := NewService(lmaESCli, lsc, "cluster", DefaultIndexSettings())

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	tm, err := e.GetIPSetModified(ctx, "test")
	g.Expect(err).ShouldNot(HaveOccurred(), "Proper response")
	g.Expect(tm).Should(BeTemporally("==", dateparse.MustParse("2019-03-18T12:29:18.590008-03:00")))

	tm, err = e.GetIPSetModified(ctx, "test2")
	g.Expect(err).ShouldNot(HaveOccurred(), "String integer time")
	g.Expect(tm).Should(BeTemporally("==", dateparse.MustParse("2019-03-20T14:40:52-03:00")))

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

	lmaESCli, err := lma.New(client, u, "", "", "", 1, 0, false, 0, 0)
	g.Expect(err).Should(BeNil())
	results := []rest.MockResult{}
	results = append(results, rest.MockResult{
		Body: v1.List[v1.FlowLog]{
			Items: []v1.FlowLog{
				{
					SourceIP: strPtr("35.32.82.134"),
					DestIP:   strPtr("10.10.1.20"),
					ID:       "BQ15nGkBixKz5K3LBMRy",
				},
				{
					SourceIP: strPtr("35.32.82.134"),
					DestIP:   strPtr("10.10.1.20"),
					ID:       "BQ15nGkBixKz5K3LBMRz",
				},
			},
			TotalHits: 2,
		},
	})
	results = append(results, rest.MockResult{
		Body: v1.List[v1.FlowLog]{
			Items: []v1.FlowLog{
				{
					SourceIP: strPtr("35.32.82.134"),
					DestIP:   strPtr("10.10.1.20"),
					ID:       "BQ15nGkBixKz5K3LBMRy",
				},
				{
					SourceIP: strPtr("35.32.82.134"),
					DestIP:   strPtr("10.10.1.20"),
					ID:       "BQ15nGkBixKz5K3LBMRz",
				},
			},
			TotalHits: 2,
		},
	})

	lsc := lsclient.NewMockClient("", results...)
	e := NewService(lmaESCli, lsc, "cluster", DefaultIndexSettings())

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	oneMinuteAgo = time.Now().Add(-1 * time.Minute)
	toBeUpdated := &apiV3.GlobalThreatFeed{}
	toBeUpdated.Name = "test"
	toBeUpdated.Status.LastSuccessfulSearch = &metav1.Time{Time: oneMinuteAgo}

	itr, _, err := e.QueryIPSet(ctx, toBeUpdated)
	g.Expect(err).ShouldNot(HaveOccurred())

	c := 0
	vals := make([]interface{}, 0)
	for itr.Next() {
		c++
		_, val := itr.Value()
		vals = append(vals, val)
	}
	g.Expect(itr.Err()).ShouldNot(HaveOccurred())
	g.Expect(c).Should(Equal(4))
	g.Expect(len(vals)).Should(Equal(4))
}

func TestElastic_QueryIPSet_SameIPSet(t *testing.T) {
	g := NewGomegaWithT(t)

	u, err := url2.Parse(baseURI)
	g.Expect(err).ShouldNot(HaveOccurred())
	roundTripper := testRoundTripper{}
	client := &http.Client{
		Transport: http.RoundTripper(&roundTripper),
	}

	lmaESCli, err := lma.New(client, u, "", "", "", 1, 0, false, 0, 0)
	g.Expect(err).Should(BeNil())
	results := []rest.MockResult{}
	results = append(results, rest.MockResult{
		Body: v1.List[v1.FlowLog]{
			Items: []v1.FlowLog{
				{
					SourceIP:  strPtr("35.32.82.134"),
					DestIP:    strPtr("10.10.1.20"),
					ID:        "BQ15nGkBixKz5K3LBMRy",
					StartTime: 1536897600,
					EndTime:   1536897900,
				},
			},
			TotalHits: 1,
		},
	})
	results = append(results, rest.MockResult{
		Body: v1.List[v1.FlowLog]{
			Items: []v1.FlowLog{
				{
					SourceIP:  strPtr("35.32.82.134"),
					DestIP:    strPtr("10.10.1.20"),
					ID:        "BQ15nGkBixKz5K3LBMRy",
					StartTime: 1536897600,
					EndTime:   1536897900,
				},
			},
			TotalHits: 1,
		},
	})

	lsc := lsclient.NewMockClient("", results...)
	e := NewService(lmaESCli, lsc, "cluster", DefaultIndexSettings())

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	oneMinuteAgo := time.Now().Add(-1 * time.Minute)
	toBeUpdated := &apiV3.GlobalThreatFeed{}
	toBeUpdated.Name = "test"
	toBeUpdated.Status.LastSuccessfulSearch = &metav1.Time{Time: oneMinuteAgo}

	cachedIpSet, err := e.GetIPSet(ctx, "test1")
	g.Expect(err).NotTo(HaveOccurred())
	toBeUpdated.SetAnnotations(map[string]string{IpSetHashKey: util.ComputeSha256Hash(cachedIpSet)})

	roundTripper.params = make(map[string]interface{})
	roundTripper.params["fromTimeStamp"] = oneMinuteAgo.Format(time.RFC3339Nano)

	itr, _, err := e.QueryIPSet(ctx, toBeUpdated)
	g.Expect(err).ShouldNot(HaveOccurred())

	c := 0
	vals := make([]interface{}, 0)
	for itr.Next() {
		c++
		_, val := itr.Value()
		vals = append(vals, val)
	}
	g.Expect(itr.Err()).ShouldNot(HaveOccurred())
	g.Expect(c).Should(Equal(2))
	g.Expect(len(vals)).Should(Equal(2))
}

func TestElastic_QueryIPSet_Big(t *testing.T) {
	g := NewGomegaWithT(t)

	u, err := url2.Parse(baseURI)
	g.Expect(err).ShouldNot(HaveOccurred())
	client := &http.Client{
		Transport: http.RoundTripper(&testRoundTripper{}),
	}

	lmaESCli, err := lma.New(client, u, "", "", "", 1, 0, false, 0, 0)
	g.Expect(err).Should(BeNil())
	results := []rest.MockResult{}
	results = append(results, rest.MockResult{
		Body: v1.List[v1.FlowLog]{
			Items: []v1.FlowLog{},
		},
	})
	results = append(results, rest.MockResult{
		Body: v1.List[v1.FlowLog]{
			Items: []v1.FlowLog{},
		},
	})

	lsc := lsclient.NewMockClient("", results...)

	e := NewService(lmaESCli, lsc, "cluster", DefaultIndexSettings())

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	testFeed := &apiV3.GlobalThreatFeed{}
	testFeed.Name = "test_big"
	i, _, err := e.QueryIPSet(ctx, testFeed)
	g.Expect(err).ShouldNot(HaveOccurred())

	itr := i.(*queryIterator[v1.FlowLog, v1.FlowLogParams])

	g.Expect(itr.queries).Should(HaveLen(4), "Input was split into 2x2 arrays")
	g.Expect(itr.queries[0].queryParams.IPMatches).Should(HaveLen(1))
	g.Expect(itr.queries[0].queryParams.IPMatches[0].IPs).Should(HaveLen(MaxClauseCount))
	g.Expect(itr.queries[1].queryParams.IPMatches).Should(HaveLen(1))
	g.Expect(itr.queries[1].queryParams.IPMatches[0].IPs).Should(HaveLen(MaxClauseCount))
	g.Expect(itr.queries[2].queryParams.IPMatches).Should(HaveLen(1))
	g.Expect(itr.queries[2].queryParams.IPMatches[0].IPs).Should(HaveLen(256))
	g.Expect(itr.queries[3].queryParams.IPMatches).Should(HaveLen(1))
	g.Expect(itr.queries[3].queryParams.IPMatches[0].IPs).Should(HaveLen(256))
}

func TestElastic_ListSets(t *testing.T) {
	g := NewGomegaWithT(t)

	u, err := url2.Parse(baseURI)
	g.Expect(err).ShouldNot(HaveOccurred())
	rt := &testRoundTripper{}
	client := &http.Client{
		Transport: http.RoundTripper(rt),
	}

	lmaESCli, err := lma.New(client, u, "", "", "", 1, 0, false, 0, 0)
	g.Expect(err).Should(BeNil())
	// mock linseed client
	lsc := lsclient.NewMockClient("")
	e := NewService(lmaESCli, lsc, "cluster", DefaultIndexSettings())

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	rt.listRespFile = "test_files/list.1.r.json"
	rt.listStatus = 200
	metas, err := e.ListIPSets(ctx)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(metas).To(HaveLen(0))

	rt.listRespFile = "test_files/list.2.r.json"
	rt.listStatus = 404
	metas, err = e.ListIPSets(ctx)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(metas).To(HaveLen(0))

	rt.listRespFile = "test_files/list.1.r.json"
	rt.listStatus = 200
	metas, err = e.ListDomainNameSets(ctx)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(metas).To(HaveLen(0))

	rt.listRespFile = "test_files/list.2.r.json"
	rt.listStatus = 404
	metas, err = e.ListDomainNameSets(ctx)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(metas).To(HaveLen(0))
}

func TestElastic_Put_Set(t *testing.T) {
	g := NewGomegaWithT(t)

	u, err := url2.Parse(baseURI)
	g.Expect(err).ShouldNot(HaveOccurred())
	rt := &testRoundTripper{}
	client := &http.Client{
		Transport: http.RoundTripper(rt),
	}

	lmaESCli, err := lma.New(client, u, "", "", "", 1, 0, false, 0, 0)
	g.Expect(err).Should(BeNil())
	lsc := lsclient.NewMockClient("")
	e := NewService(lmaESCli, lsc, "cluster", DefaultIndexSettings())

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	close(e.ipSetMappingCreated)

	err = e.PutIPSet(ctx, "test1", IPSetSpec{"1.2.3.4"})
	g.Expect(err).ToNot(HaveOccurred())

	close(e.domainNameSetMappingCreated)

	err = e.PutDomainNameSet(ctx, "test1", DomainNameSetSpec{"hackers.and.badguys"})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestSplitIPSetToInterface(t *testing.T) {
	g := NewGomegaWithT(t)

	mul := 2
	offset := 11

	var input IPSetSpec
	for i := 0; i < mul*MaxClauseCount+offset; i++ {
		input = append(input, fmt.Sprintf("%d", i))
	}

	output := splitIPSet(input)

	g.Expect(len(output)).Should(Equal(mul + 1))
	for i := 0; i < mul; i++ {
		g.Expect(len(output[i])).Should(Equal(MaxClauseCount))
		for idx, v := range output[i] {
			g.Expect(v).Should(Equal(fmt.Sprintf("%d", i*MaxClauseCount+idx)))
		}
	}
	g.Expect(len(output[mul])).Should(Equal(offset))
	for idx, v := range output[mul] {
		g.Expect(v).Should(Equal(fmt.Sprintf("%d", mul*MaxClauseCount+idx)))
	}
}

func TestElastic_Delete_Set(t *testing.T) {
	g := NewGomegaWithT(t)

	u, err := url2.Parse(baseURI)
	g.Expect(err).ShouldNot(HaveOccurred())
	rt := &testRoundTripper{}
	client := &http.Client{
		Transport: http.RoundTripper(rt),
	}

	lmaESCli, err := lma.New(client, u, "", "", "", 1, 0, false, 0, 0)
	g.Expect(err).Should(BeNil())
	lsc := lsclient.NewMockClient("")
	e := NewService(lmaESCli, lsc, "cluster", DefaultIndexSettings())

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	err = e.DeleteIPSet(ctx, Meta{Name: "test"})
	g.Expect(err).ToNot(HaveOccurred())

	three := int64(3)
	four := int64(4)
	err = e.DeleteDomainNameSet(ctx, Meta{Name: "test", SeqNo: &three, PrimaryTerm: &four})
	g.Expect(err).ToNot(HaveOccurred())
}

type testRoundTripper struct {
	e            error
	listRespFile string
	listStatus   int
	params       map[string]interface{}
}

func (t *testRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.e != nil {
		return nil, t.e
	}
	switch req.Method {
	case "HEAD":
		switch req.URL.String() {
		case baseURI:
			return &http.Response{
				StatusCode: 200,
				Request:    req,
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		}
	case "GET":
		switch req.URL.String() {
		// QueryIPSet
		case baseURI + "/.tigera.ipset.cluster/_doc/test":
			return &http.Response{
				StatusCode: 200,
				Request:    req,
				Body:       mustOpen("test_files/3.ipset.json"),
			}, nil
		// QueryIPSet
		case baseURI + "/.tigera.ipset.cluster/_doc/test_big":
			return &http.Response{
				StatusCode: 200,
				Request:    req,
				Body:       mustOpen("test_files/big_ipset.json"),
			}, nil

		// GetIPSet
		case baseURI + "/.tigera.ipset.cluster/_doc/test1":
			return &http.Response{
				StatusCode: 200,
				Request:    req,
				Body:       mustOpen("test_files/1.1.json"),
			}, nil
		case baseURI + "/.tigera.ipset.cluster/_doc/test2":
			return &http.Response{
				StatusCode: 200,
				Request:    req,
				Body:       mustOpen("test_files/1.2.json"),
			}, nil
		case baseURI + "/.tigera.ipset.cluster/_doc/test3":
			return &http.Response{
				StatusCode: 200,
				Request:    req,
				Body:       mustOpen("test_files/1.3.json"),
			}, nil
		case baseURI + "/.tigera.ipset.cluster/_doc/test4":
			return &http.Response{
				StatusCode: 200,
				Request:    req,
				Body:       mustOpen("test_files/1.4.json"),
			}, nil
		case baseURI + "/.tigera.ipset.cluster/_doc/test5":
			return &http.Response{
				StatusCode: 200,
				Request:    req,
				Body:       mustOpen("test_files/1.5.json"),
			}, nil

		// GetIPSetModified
		case baseURI + "/.tigera.ipset.cluster/_doc/test?_source_includes=created_at":
			return &http.Response{
				StatusCode: 200,
				Request:    req,
				Body:       mustOpen("test_files/2.1.json"),
			}, nil
		case baseURI + "/.tigera.ipset.cluster/_doc/test2?_source_includes=created_at":
			return &http.Response{
				StatusCode: 200,
				Request:    req,
				Body:       mustOpen("test_files/2.2.json"),
			}, nil
		case baseURI + "/.tigera.ipset.cluster/_doc/test3?_source_includes=created_at":
			return &http.Response{
				StatusCode: 200,
				Request:    req,
				Body:       mustOpen("test_files/2.3.json"),
			}, nil
		case baseURI + "/.tigera.ipset.cluster/_doc/test4?_source_includes=created_at":
			return &http.Response{
				StatusCode: 200,
				Request:    req,
				Body:       mustOpen("test_files/2.4.json"),
			}, nil
		case baseURI + "/.tigera.ipset.cluster/_doc/test5?_source_includes=created_at":
			return &http.Response{
				StatusCode: 200,
				Request:    req,
				Body:       mustOpen("test_files/2.5.json"),
			}, nil

		}
	case "POST":
		switch req.URL.String() {
		case baseURI + "/.tigera.ipset.cluster/_search?scroll=5m":
			return &http.Response{
				StatusCode: t.listStatus,
				Request:    req,
				Body:       mustOpen(t.listRespFile),
			}, nil
		case baseURI + "/.tigera.domainnameset.cluster/_search?scroll=5m":
			return &http.Response{
				StatusCode: t.listStatus,
				Request:    req,
				Body:       mustOpen(t.listRespFile),
			}, nil
		}
	case "PUT":
		switch u := req.URL.String(); {
		case strings.HasPrefix(u, baseURI+"/.tigera.ipset.cluster/_doc"):
			return &http.Response{
				StatusCode: 200,
				Request:    req,
				Body:       mustOpen("test_files/put.ipset.1.r.json"),
			}, nil
		case strings.HasPrefix(u, baseURI+"/.tigera.domainnameset.cluster/_doc"):
			return &http.Response{
				StatusCode: 200,
				Request:    req,
				Body:       mustOpen("test_files/put.domainnameset.1.r.json"),
			}, nil
		default:
			return &http.Response{
				StatusCode: 404,
				Request:    req,
			}, nil
		}
	case "DELETE":
		switch u := req.URL.String(); {
		case strings.HasPrefix(u, baseURI+"/.tigera.ipset.cluster/_doc"):
			return &http.Response{
				StatusCode: 200,
				Request:    req,
				Body:       mustOpen("test_files/delete.ipset.1.r.json"),
			}, nil
		case strings.HasPrefix(u, baseURI+"/.tigera.domainnameset.cluster/_doc"):
			return &http.Response{
				StatusCode: 200,
				Request:    req,
				Body:       mustOpen("test_files/delete.domainnameset.1.r.json"),
			}, nil
		case u == baseURI+"/_search/scroll":
			return &http.Response{
				StatusCode: 200,
				Request:    req,
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		default:
			return &http.Response{
				StatusCode: 404,
				Request:    req,
			}, nil
		}
	}

	if os.Getenv("ELASTIC_TEST_DEBUG") == "yes" {
		_, _ = fmt.Fprintf(os.Stderr, "%s %s\n", req.Method, req.URL)
		if req.Body != nil {
			b, _ := io.ReadAll(req.Body)
			_ = req.Body.Close()
			body := string(b)
			req.Body = io.NopCloser(bytes.NewReader(b))
			_, _ = fmt.Fprintln(os.Stderr, body)
		}
	}

	return &http.Response{
		Request:    req,
		StatusCode: 500,
		Body:       io.NopCloser(strings.NewReader("")),
	}, nil
}

func mustOpen(name string) io.ReadCloser {
	f, err := os.Open(name)
	if err != nil {
		panic(err)
	}
	return f
}

func strPtr(val string) *string {
	return &val
}
