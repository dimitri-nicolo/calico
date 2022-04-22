// Copyright 2019 Tigera Inc. All rights reserved.

package elastic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	url2 "net/url"
	"os"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/araddon/dateparse"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiV3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/projectcalico/calico/intrusion-detection/controller/pkg/db"
	"github.com/projectcalico/calico/intrusion-detection/controller/pkg/util"
	lma "github.com/projectcalico/calico/lma/pkg/elastic"
)

const (
	baseURI = "http://127.0.0.1:9200"
)

var (
	oneMinuteAgo time.Time
)

func TestElastic_GetIPSet(t *testing.T) {
	g := NewGomegaWithT(t)

	u, err := url2.Parse(baseURI)
	g.Expect(err).ShouldNot(HaveOccurred())
	client := &http.Client{
		Transport: http.RoundTripper(&testRoundTripper{}),
	}

	lmaESCli, err := lma.New(client, u, "", "", "", 1, 0, false, 0, 0)
	g.Expect(err).Should(BeNil())
	e := NewService(lmaESCli, DefaultIndexSettings())

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

	lmaESCli, err := lma.New(client, u, "", "", "", 1, 0, false, 0, 0)
	g.Expect(err).Should(BeNil())
	e := NewService(lmaESCli, DefaultIndexSettings())

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
	e := NewService(lmaESCli, DefaultIndexSettings())

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
		vals = append(vals, val.Source)
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
	e := NewService(lmaESCli, DefaultIndexSettings())

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	oneMinuteAgo := time.Now().Add(-1 * time.Minute)
	toBeUpdated := &apiV3.GlobalThreatFeed{}
	toBeUpdated.Name = "test"
	toBeUpdated.Status.LastSuccessfulSearch = &metav1.Time{Time: oneMinuteAgo}

	cachedIpSet, err := e.GetIPSet(ctx, "test1")
	toBeUpdated.SetAnnotations(map[string]string{db.IpSetHashKey: util.ComputeSha256Hash(cachedIpSet)})

	roundTripper.params = make(map[string]interface{})
	roundTripper.params["fromTimeStamp"] = oneMinuteAgo.Format(time.RFC3339Nano)

	itr, _, err := e.QueryIPSet(ctx, toBeUpdated)
	g.Expect(err).ShouldNot(HaveOccurred())

	c := 0
	vals := make([]interface{}, 0)
	for itr.Next() {
		c++
		_, val := itr.Value()
		vals = append(vals, val.Source)
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
	e := NewService(lmaESCli, DefaultIndexSettings())

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	testFeed := &apiV3.GlobalThreatFeed{}
	testFeed.Name = "test_big"
	i, _, err := e.QueryIPSet(ctx, testFeed)
	g.Expect(err).ShouldNot(HaveOccurred())

	itr := i.(*queryIterator)

	g.Expect(itr.scrollers).Should(HaveLen(4), "Input was split into 2x2 arrays")
	g.Expect(itr.scrollers[0].terms).Should(HaveLen(MaxClauseCount))
	g.Expect(itr.scrollers[1].terms).Should(HaveLen(MaxClauseCount))
	g.Expect(itr.scrollers[2].terms).Should(HaveLen(256))
	g.Expect(itr.scrollers[3].terms).Should(HaveLen(256))
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
	e := NewService(lmaESCli, DefaultIndexSettings())

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
	e := NewService(lmaESCli, DefaultIndexSettings())

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	close(e.ipSetMappingCreated)

	err = e.PutIPSet(ctx, "test1", db.IPSetSpec{"1.2.3.4"})
	g.Expect(err).ToNot(HaveOccurred())

	close(e.domainNameSetMappingCreated)

	err = e.PutDomainNameSet(ctx, "test1", db.DomainNameSetSpec{"hackers.and.badguys"})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestSplitIPSetToInterface(t *testing.T) {
	g := NewGomegaWithT(t)

	mul := 2
	offset := 11

	var input db.IPSetSpec
	for i := 0; i < mul*MaxClauseCount+offset; i++ {
		input = append(input, fmt.Sprintf("%d", i))
	}

	output := splitIPSetToInterface(input)

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
	e := NewService(lmaESCli, DefaultIndexSettings())

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	err = e.DeleteIPSet(ctx, db.Meta{Name: "test"})
	g.Expect(err).ToNot(HaveOccurred())

	three := int64(3)
	four := int64(4)
	err = e.DeleteDomainNameSet(ctx, db.Meta{Name: "test", SeqNo: &three, PrimaryTerm: &four})
	g.Expect(err).ToNot(HaveOccurred())
}

type testRoundTripper struct {
	u            *url.URL
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
				Body:       ioutil.NopCloser(strings.NewReader("")),
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
		b, _ := ioutil.ReadAll(req.Body)
		_ = req.Body.Close()
		body := string(b)
		req.Body = ioutil.NopCloser(bytes.NewReader(b))

		switch req.URL.String() {
		// QueryIPSet
		case baseURI + "/tigera_secure_ee_flows.cluster.%2A/_search?scroll=5m&size=1000":
			switch body {
			// QueryIPSet source_ip query
			case mustGetString("test_files/3.1.q.json"):
				return &http.Response{
					StatusCode: 200,
					Request:    req,
					Body:       mustOpen("test_files/3.1.r.json"),
				}, nil

			// QueryIPSet dest_ip query
			case mustGetString("test_files/3.3.q.json"):
				return &http.Response{
					StatusCode: 200,
					Request:    req,
					Body:       mustOpen("test_files/3.3.r.json"),
				}, nil

			case mustGetTemplate("test_files/source_ip_with_timestamp_query.json", t.params):
				return &http.Response{
					StatusCode: 200,
					Request:    req,
					Body:       mustOpen("test_files/source_ip_with_timestamp_result.json"),
				}, nil

			case mustGetTemplate("test_files/dest_ip_with_timestamp_query.json", t.params):
				return &http.Response{
					StatusCode: 200,
					Request:    req,
					Body:       mustOpen("test_files/dest_ip_with_timestamp_result.json"),
				}, nil
			}
		case baseURI + "/_search/scroll":
			switch body {
			// QueryIPSet source_ip query
			case mustGetString("test_files/3.2.q.json"):
				return &http.Response{
					StatusCode: 200,
					Request:    req,
					Body:       mustOpen("test_files/3.2.r.json"),
				}, nil
			// QueryIPSet dest_ip query
			case mustGetString("test_files/3.4.q.json"):
				return &http.Response{
					StatusCode: 200,
					Request:    req,
					Body:       mustOpen("test_files/3.4.r.json"),
				}, nil
			}

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
				Body:       ioutil.NopCloser(strings.NewReader("")),
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
			b, _ := ioutil.ReadAll(req.Body)
			_ = req.Body.Close()
			body := string(b)
			req.Body = ioutil.NopCloser(bytes.NewReader(b))
			_, _ = fmt.Fprintln(os.Stderr, body)
		}
	}

	return &http.Response{
		Request:    req,
		StatusCode: 500,
		Body:       ioutil.NopCloser(strings.NewReader("")),
	}, nil
}

func mustOpen(name string) io.ReadCloser {
	f, err := os.Open(name)
	if err != nil {
		panic(err)
	}
	return f
}

func mustGetString(name string) string {
	f, err := os.Open(name)
	if err != nil {
		panic(err)
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}
	err = f.Close()
	if err != nil {
		panic(err)
	}

	return strings.Trim(string(b), " \r\n\t")
}

func mustGetTemplate(fileName string, replacements map[string]interface{}) string {
	jsonTemplate, err := template.ParseFiles(fileName)
	if err != nil {
		panic(err)
	}

	buf := bytes.Buffer{}
	if jsonTemplate != nil {
		err = jsonTemplate.Execute(&buf, replacements)
		if err != nil {
			panic(err)
		}
	}

	compact := bytes.Buffer{}
	err = json.Compact(&compact, buf.Bytes())
	if err != nil {
		panic(err)
	}

	return strings.Trim(compact.String(), " \r\n\t")
}
