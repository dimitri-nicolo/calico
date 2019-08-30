// Copyright 2019 Tigera Inc. All rights reserved.

package ut

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	oElastic "github.com/olivere/elastic"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/elastic"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/events"
)

var uut *elastic.Elastic
var elasticClient *oElastic.Client

func TestMain(m *testing.M) {
	d, err := client.NewEnvClient()
	if err != nil {
		panic("could not create Docker client: " + err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create Elastic
	cfg := &container.Config{
		Env:   []string{"discovery.type=single-node"},
		Image: "docker.elastic.co/elasticsearch/elasticsearch:6.4.3",
	}
	result, err := d.ContainerCreate(ctx, cfg, nil, nil, "")
	if err != nil {
		fmt.Println(os.Geteuid())
		fmt.Println(os.Getegid())
		panic("could not create elastic container: " + err.Error())
	}

	err = d.ContainerStart(ctx, result.ID, types.ContainerStartOptions{})
	if err != nil {
		panic("could not start elastic: " + err.Error())
	}

	// get IP
	j, err := d.ContainerInspect(ctx, result.ID)
	if err != nil {
		panic("could not inspect elastic container: " + err.Error())
	}
	host := j.NetworkSettings.IPAddress

	u := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:9200", host),
	}

	// Wait for elastic to start responding
	c := http.Client{}
	to := time.After(1 * time.Minute)
	for {
		_, err := c.Get("http://" + u.Host)
		if err == nil {
			break
		}
		select {
		case <-to:
			panic("elasticsearch didn't come up")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	options := []oElastic.ClientOptionFunc{
		oElastic.SetURL(u.String()),
		oElastic.SetErrorLog(log.StandardLogger()),
		oElastic.SetSniff(false),
		oElastic.SetHealthcheck(false),
		//elastic.SetTraceLog(log.StandardLogger()),
	}
	elasticClient, err = oElastic.NewClient(options...)
	if err != nil {
		panic("could not create elasticClient: " + err.Error())
	}

	uut, err = elastic.NewElastic(&http.Client{}, u, "", "")
	if err != nil {
		panic("could not create unit under test: " + err.Error())
	}

	uut.Run(ctx)

	rc := m.Run()

	timeout := time.Second * 10
	_ = d.ContainerStop(ctx, result.ID, &timeout)
	_ = d.ContainerRemove(ctx, result.ID, types.ContainerRemoveOptions{Force: true})

	os.Exit(rc)
}

func TestGetDomainNameSet_GetDomainNameSetModifed_Exist(t *testing.T) {
	g := NewGomegaWithT(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	input := db.DomainNameSetSpec{"xx.yy.zzz"}
	err := uut.PutDomainNameSet(ctx, "test", input)

	g.Expect(err).ToNot(HaveOccurred())

	defer func() {
		err := uut.DeleteDomainNameSet(ctx, db.Meta{Name: "test"})
		g.Expect(err).ToNot(HaveOccurred())
	}()

	actual, err := uut.GetDomainNameSet(ctx, "test")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(actual).To(Equal(input))

	m, err := uut.GetDomainNameSetModified(ctx, "test")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(m).To(BeTemporally("<", time.Now()))
	g.Expect(m).To(BeTemporally(">", time.Now().Add(-5*time.Second)), "modified in the last 5 seconds")
}

func TestGetDomainNameSet_NotExist(t *testing.T) {
	g := NewGomegaWithT(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := uut.GetDomainNameSet(ctx, "test")
	g.Expect(err).To(Equal(&oElastic.Error{Status: 404}))
}

func TestQueryDomainNameSet_Success(t *testing.T) {
	g := NewGomegaWithT(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Install the DNS log mapping
	template := mustGetString("test_files/dns_template.json")
	_, err := elasticClient.IndexPutTemplate("dns_logs").BodyString(template).Do(ctx)
	g.Expect(err).ToNot(HaveOccurred())

	// Index some DNS logs
	index := "tigera_secure_ee_dns.cluster.testquerydomainnameset_success"
	i := elasticClient.Index().Index(index).Type("fluentd")
	logs := []events.DNSLog{
		{
			StartTime:       123,
			EndTime:         456,
			Count:           1,
			ClientName:      "client",
			ClientNamespace: "test",
			QClass:          "IN",
			QType:           "A",
			QName:           "xx.yy.zzz",
			RCode:           "NoError",
			RRSets: []events.DNSRRSet{
				{
					Name:  "xx.yy.zzz",
					Class: "IN",
					Type:  "A",
					RData: []string{"1.2.3.4"},
				},
			},
		},
		{
			StartTime:       789,
			EndTime:         101112,
			Count:           1,
			ClientName:      "client",
			ClientNamespace: "test",
			QClass:          "IN",
			QType:           "A",
			QName:           "aa.bb.ccc",
			RCode:           "NoError",
			RRSets: []events.DNSRRSet{
				{
					Name:  "aa.bb.ccc",
					Class: "IN",
					Type:  "CNAME",
					RData: []string{"dd.ee.fff"},
				},
				{
					Name:  "dd.ee.fff",
					Class: "IN",
					Type:  "A",
					RData: []string{"5.6.7.8"},
				},
			},
		},
		{
			StartTime:       789,
			EndTime:         101112,
			Count:           1,
			ClientName:      "client",
			ClientNamespace: "test",
			QClass:          "IN",
			QType:           "CNAME",
			QName:           "gg.hh.iii",
			RCode:           "NoError",
			RRSets: []events.DNSRRSet{
				{
					Name:  "gg.hh.iii",
					Class: "IN",
					Type:  "CNAME",
					RData: []string{"jj.kk.lll"},
				},
			},
		},
	}
	for _, l := range logs {
		_, err := i.BodyJson(l).Do(ctx)
		g.Expect(err).ToNot(HaveOccurred())
	}
	defer func() {
		_, err := elasticClient.DeleteIndex(index).Do(ctx)
		g.Expect(err).ToNot(HaveOccurred())
	}()

	// Wait until they are indexed
	to := time.After(30 * time.Second)
	for {
		s, err := elasticClient.Search(index).Do(ctx)
		g.Expect(err).ToNot(HaveOccurred())
		if s.TotalHits() == 3 {
			break
		}
		g.Expect(to).NotTo(Receive(), "wait for log index timed out")
		time.Sleep(10 * time.Millisecond)
	}

	// Run the search
	domains := db.DomainNameSetSpec{"xx.yy.zzz", "dd.ee.fff", "jj.kk.lll"}
	iter, err := uut.QueryDomainNameSet(ctx, "test-feed", domains)
	g.Expect(err).ToNot(HaveOccurred())

	var actual []events.DNSLog
	var keys []string
	for iter.Next() {
		k, h := iter.Value()
		keys = append(keys, k)
		var al events.DNSLog
		err := json.Unmarshal(*h.Source, &al)
		g.Expect(err).ToNot(HaveOccurred())
		actual = append(actual, al)
	}
	g.Expect(keys).To(Equal([]string{"qname", "rrsets.name", "rrsets.name", "rrsets.rdata", "rrsets.rdata"}))

	// Qname query
	g.Expect(actual[0].QName).To(Equal("xx.yy.zzz"))

	// rrsets.name query
	// We identify the results by the QName, which is unique for each log.
	qnames := []string{actual[1].QName, actual[2].QName}
	// Query for xx.yy.zzz has the name xx.yy.zzz in the first RRSet
	g.Expect(qnames).To(ContainElement("xx.yy.zzz"))
	// Query for aa.bb.ccc has the name dd.ee.fff in the second RRSet
	g.Expect(qnames).To(ContainElement("aa.bb.ccc"))

	// rrsets.rdata query
	// We identify the results by the QName, which is unique for each log.
	qnames = []string{actual[3].QName, actual[4].QName}
	// Query for aa.bb.ccc has the data dd.ee.fff in the first rrset
	g.Expect(qnames).To(ContainElement("aa.bb.ccc"))
	// Query for gg.hh.iii has the data jj.kk.lll in the first rrset
	g.Expect(qnames).To(ContainElement("gg.hh.iii"))
}

func TestPutSecurityEvent_DomainName(t *testing.T) {
	g := NewGomegaWithT(t)

	l := events.DNSLog{
		StartTime:       123,
		EndTime:         456,
		Count:           1,
		ClientName:      "client",
		ClientNamespace: "test",
		QClass:          "IN",
		QType:           "A",
		QName:           "xx.yy.zzz",
		RCode:           "NoError",
		RRSets: []events.DNSRRSet{
			{
				Name:  "xx.yy.zzz",
				Class: "IN",
				Type:  "A",
				RData: []string{"1.2.3.4"},
			},
		},
	}
	h := &oElastic.SearchHit{Index: "dns_index", Id: "dns_id"}
	domains := map[string]struct{}{
		"xx.yy.zzz": {},
	}
	e := events.ConvertDNSLog(l, "qname", h, domains, "my-feed", "my-other-feed")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := uut.PutSecurityEvent(ctx, e)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify the event exists
	result, err := elasticClient.Get().Index(elastic.EventIndex).Id(e.ID()).Do(ctx)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Id).To(Equal(e.ID()))
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
