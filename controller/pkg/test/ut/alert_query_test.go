// Copyright (c) 2019 Tigera Inc. All rights reserved.

package ut

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	calicoQuery "github.com/projectcalico/libcalico-go/lib/validator/v3/query"

	"github.com/tigera/intrusion-detection/controller/pkg/alert/query"
	idsElastic "github.com/tigera/intrusion-detection/controller/pkg/elastic"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/events"
)

/*
Most of the query gen logic is tested in the TestDNSQuery function.
*/

func TestAuditQuery(t *testing.T) {
	g := NewWithT(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Install the audit log mapping
	template := mustGetString("test_files/audit_template.json")
	_, err := elasticClient.IndexPutTemplate("audit_logs").BodyString(template).Do(ctx)
	g.Expect(err).ToNot(HaveOccurred())

	// Index some audit logs
	index := "tigera_secure_ee_audit_kube.cluster.testauditquery"
	i := elasticClient.Index().Index(index)
	var logs []interface{}
	b, err := ioutil.ReadFile("test_files/audit_data.json")
	err = json.Unmarshal(b, &logs)

	var logIds []string
	for _, l := range logs {
		resp, err := i.BodyJson(l).Do(ctx)
		g.Expect(err).ToNot(HaveOccurred())
		logIds = append(logIds, resp.Id)
	}
	defer func() {
		_, err := elasticClient.DeleteIndex(index).Do(ctx)
		g.Expect(err).ToNot(HaveOccurred())
	}()

	// Wait until they are indexed
	g.Eventually(func() int64 {
		s, err := elasticClient.Search(index).Do(ctx)
		g.Expect(err).ShouldNot(HaveOccurred())
		res := s.TotalHits()
		if res < int64(len(logs)) {
			time.Sleep(10 * time.Millisecond)
		}
		return res
	}, 30*time.Second).Should(BeNumerically("==", len(logs)))

	runTest := func(input string, expected []string) func(*testing.T) {
		return func(t *testing.T) {
			g := NewWithT(t)

			q, err := calicoQuery.ParseQuery(input)
			g.Expect(err).ShouldNot(HaveOccurred())

			err = calicoQuery.Validate(q, calicoQuery.IsValidAuditAtom)
			g.Expect(err).ShouldNot(HaveOccurred())

			c := query.NewAuditConverter()
			eq := c.Convert(q)
			g.Expect(eq).ShouldNot(BeNil())
			g.Expect(eq).ShouldNot(HaveLen(0))

			b, err := json.Marshal(&eq)
			g.Expect(err).ShouldNot(HaveOccurred())
			fmt.Println(string(b))

			response, err := elasticClient.Search(index).Source(map[string]interface{}{"query": eq}).Do(ctx)
			g.Expect(err).ShouldNot(HaveOccurred())

			var actual []string
			for _, hit := range response.Hits.Hits {
				actual = append(actual, hit.Id)
			}
			g.Expect(actual).Should(ConsistOf(expected))
		}
	}

	t.Run("all", runTest("", logIds))
	t.Run("name", runTest(`"name" = "kube-controller-manager"`, logIds[1:2]))
	t.Run("objectRef.name", runTest(`"objectRef.name" = "kube-scheduler"`, logIds[0:1]))
}

func TestDNSQuery(t *testing.T) {
	g := NewGomegaWithT(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Install the DNS log mapping
	template := mustGetString("test_files/dns_template.json")
	_, err := elasticClient.IndexPutTemplate("dns_logs").BodyString(template).Do(ctx)
	g.Expect(err).ToNot(HaveOccurred())

	// Index some DNS logs
	index := "tigera_secure_ee_dns.cluster.testdnsquery"
	i := elasticClient.Index().Index(index)
	logs := []events.DNSLog{
		{
			StartTime:       idsElastic.Time{Time: time.Unix(123, 0)},
			EndTime:         idsElastic.Time{Time: time.Unix(456, 0)},
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
			Servers: []events.DNSServer{
				{
					Name:      "server-1",
					NameAggr:  "server-*",
					Namespace: "test2",
					IP:        "2.3.4.5",
					Labels: map[string]string{
						"a.b": "c",
					},
				},
			},
		},
		{
			StartTime:       idsElastic.Time{Time: time.Unix(789, 0)},
			EndTime:         idsElastic.Time{Time: time.Unix(101112, 0)},
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
			StartTime:       idsElastic.Time{Time: time.Unix(789, 0)},
			EndTime:         idsElastic.Time{Time: time.Unix(101112, 0)},
			Count:           1,
			ClientName:      "client",
			ClientNamespace: "test",
			ClientLabels: map[string]string{
				"1.2": "3",
			},
			QClass: "IN",
			QType:  "CNAME",
			QName:  "gg.hh.iii",
			RCode:  "NoError",
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
	var logIds []string
	for _, l := range logs {
		resp, err := i.BodyJson(l).Do(ctx)
		g.Expect(err).ToNot(HaveOccurred())
		logIds = append(logIds, resp.Id)
	}
	defer func() {
		_, err := elasticClient.DeleteIndex(index).Do(ctx)
		g.Expect(err).ToNot(HaveOccurred())
	}()

	// Wait until they are indexed
	g.Eventually(func() int64 {
		s, err := elasticClient.Search(index).Do(ctx)
		g.Expect(err).ShouldNot(HaveOccurred())
		res := s.TotalHits()
		if res < int64(len(logs)) {
			time.Sleep(10 * time.Millisecond)
		}
		return res
	}, 30*time.Second).Should(BeNumerically("==", len(logs)))

	runTest := func(input string, expected []string) func(*testing.T) {
		return func(t *testing.T) {
			g := NewWithT(t)

			q, err := calicoQuery.ParseQuery(input)
			g.Expect(err).ShouldNot(HaveOccurred())

			err = calicoQuery.Validate(q, calicoQuery.IsValidDNSAtom)
			g.Expect(err).ShouldNot(HaveOccurred())

			c := query.NewDNSConverter()
			eq := c.Convert(q)
			g.Expect(eq).ShouldNot(BeNil())
			g.Expect(eq).ShouldNot(HaveLen(0))

			b, err := json.Marshal(&eq)
			g.Expect(err).ShouldNot(HaveOccurred())
			fmt.Println(string(b))

			response, err := elasticClient.Search(index).Source(map[string]interface{}{"query": eq}).Do(ctx)
			g.Expect(err).ShouldNot(HaveOccurred())

			var actual []string
			for _, hit := range response.Hits.Hits {
				actual = append(actual, hit.Id)
			}
			g.Expect(actual).Should(ConsistOf(expected))
		}
	}

	t.Run("all", runTest("", logIds))
	t.Run("qtype = A", runTest("qtype = A", logIds[0:2]))
	t.Run("NOT qtype = A", runTest("NOT qtype = A", logIds[2:3]))
	t.Run("qtype = CNAME", runTest("qtype = CNAME", logIds[2:3]))
	t.Run("NOT qtype = CNAME", runTest("NOT qtype = CNAME", logIds[0:2]))
	t.Run("client_labels.*", runTest(`"client_labels.1.2" = 3`, logIds[2:3]))
	t.Run("servers.labels.*", runTest(`"servers.labels.a.b" = c`, logIds[0:1]))
	t.Run("servers.name", runTest(`"servers.name" = "server-1"`, logIds[0:1]))
	t.Run("servers.name_aggr", runTest(`"servers.name_aggr" = "server-*"`, logIds[0:1]))
	t.Run("servers.namespace", runTest(`"servers.namespace" = "test2"`, logIds[0:1]))
	t.Run("servers.ip", runTest(`"servers.ip" = "2.3.4.5"`, logIds[0:1]))
	t.Run("rrsets.name", runTest(`"rrsets.name" = "dd.ee.fff"`, logIds[1:2]))
	t.Run("rrsets.type", runTest(`"rrsets.type" = "CNAME"`, logIds[1:3]))
	t.Run("rrsets.class", runTest(`"rrsets.class" = "IN"`, logIds))
	t.Run("rrsets.rdata", runTest(`"rrsets.rdata" = "5.6.7.8"`, logIds[1:2]))
	t.Run("rrsets.name OR servers.ip",
		runTest(`"rrsets.name" = "dd.ee.fff" OR "servers.ip" = "2.3.4.5"`, logIds[0:2]))
	t.Run("(rrsets.name OR servers.ip) AND NOT servers.labels",
		runTest(`("rrsets.name" = "dd.ee.fff" OR "servers.ip" = "2.3.4.5") AND NOT "servers.labels.a.b" = "c"`, logIds[1:2]))
}

func TestFlowsQuery(t *testing.T) {
	g := NewGomegaWithT(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Install the Flows log mapping
	template := mustGetString("test_files/flows_template.json")
	_, err := elasticClient.IndexPutTemplate("flows_logs").BodyString(template).Do(ctx)
	g.Expect(err).ToNot(HaveOccurred())

	// Index some Flows logs
	index := "tigera_secure_ee_flows.cluster.testflowsquery"
	i := elasticClient.Index().Index(index)

	var logs []interface{}
	b, err := ioutil.ReadFile("test_files/flows_data.json")
	err = json.Unmarshal(b, &logs)

	var logIds []string
	for _, l := range logs {
		resp, err := i.BodyJson(l).Do(ctx)
		g.Expect(err).ToNot(HaveOccurred())
		logIds = append(logIds, resp.Id)
	}
	defer func() {
		_, err := elasticClient.DeleteIndex(index).Do(ctx)
		g.Expect(err).ToNot(HaveOccurred())
	}()

	// Wait until they are indexed
	g.Eventually(func() int64 {
		s, err := elasticClient.Search(index).Do(ctx)
		g.Expect(err).ShouldNot(HaveOccurred())
		res := s.TotalHits()
		if res < int64(len(logs)) {
			time.Sleep(10 * time.Millisecond)
		}
		return res
	}, 30*time.Second).Should(BeNumerically("==", len(logs)))

	runTest := func(input string, expected []string) func(*testing.T) {
		return func(t *testing.T) {
			g := NewWithT(t)

			q, err := calicoQuery.ParseQuery(input)
			g.Expect(err).ShouldNot(HaveOccurred())

			err = calicoQuery.Validate(q, calicoQuery.IsValidFlowsAtom)
			g.Expect(err).ShouldNot(HaveOccurred())

			c := query.NewFlowsConverter()
			eq := c.Convert(q)
			g.Expect(eq).ShouldNot(BeNil())
			g.Expect(eq).ShouldNot(HaveLen(0))

			b, err := json.Marshal(&eq)
			g.Expect(err).ShouldNot(HaveOccurred())
			fmt.Println(string(b))

			response, err := elasticClient.Search(index).Source(map[string]interface{}{"query": eq}).Do(ctx)
			g.Expect(err).ShouldNot(HaveOccurred())

			var actual []string
			for _, hit := range response.Hits.Hits {
				actual = append(actual, hit.Id)
			}
			g.Expect(actual).Should(ConsistOf(expected))
		}
	}

	t.Run("all", runTest("", logIds))
	t.Run("source_ip", runTest(`source_ip = "1.2.3.4"`, logIds[0:1]))
	t.Run("source_labels",
		runTest(`"source_labels.labels" = "k8s-app=tigera-fluentd-node"`, logIds[2:3]))
	t.Run("dest_labels",
		runTest(`"dest_labels.labels"= "common.k8s.elastic.co/type=elasticsearch"`, logIds[2:4]))
	t.Run("policies",
		runTest(`"policies.all_policies"= "0|__PROFILE__|__PROFILE__.kns.calico-monitoring|allow"`,
			append(logIds[0:1], logIds[2:4]...)))
}
