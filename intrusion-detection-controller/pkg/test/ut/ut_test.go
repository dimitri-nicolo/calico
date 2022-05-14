package ut

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	oElastic "github.com/olivere/elastic/v7"

	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/db"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/elastic"
	idsElastic "github.com/projectcalico/calico/intrusion-detection-controller/pkg/elastic"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/feeds/events"
	lma "github.com/projectcalico/calico/lma/pkg/elastic"

	apiV3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

var _ = Describe("Elasticsearch UT", func() {

	var uut *elastic.Elastic
	var elasticClient *oElastic.Client
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()

		u := &url.URL{
			Scheme: "http",
			Host:   "localhost:9200",
		}

		indexSettings := elastic.DefaultIndexSettings()
		lmaESCli, err := lma.New(&http.Client{}, u, "", "", "cluster", 1, 0, true, indexSettings.Replicas, indexSettings.Shards)
		if err != nil {
			panic("could not create unit under test: " + err.Error())
		}
		elasticClient = lmaESCli.Backend()
		err = lmaESCli.CreateEventsIndex(ctx)
		Expect(err).ShouldNot(HaveOccurred())
		uut = elastic.NewService(lmaESCli, indexSettings)
		uut.Run(ctx)
	})

	AfterEach(func() {
		uut.Close()
	})

	Context("Domain name set", func() {

		It("Get existing domain name set", func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			input := db.DomainNameSetSpec{"xx.yy.zzz"}
			err := uut.PutDomainNameSet(ctx, "test", input)

			Expect(err).ToNot(HaveOccurred())

			defer func() {
				err := uut.DeleteDomainNameSet(ctx, db.Meta{Name: "test"})
				Expect(err).ToNot(HaveOccurred())
			}()

			actual, err := uut.GetDomainNameSet(ctx, "test")
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal(input))

			m, err := uut.GetDomainNameSetModified(ctx, "test")
			Expect(err).ToNot(HaveOccurred())
			Expect(m).To(BeTemporally("<", time.Now()))
			Expect(m).To(BeTemporally(">", time.Now().Add(-5*time.Second)), "modified in the last 5 seconds")
		})

		It("Get non-existing domain set", func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			_, err := uut.GetDomainNameSet(ctx, "test")
			Expect(err).To(Equal(&oElastic.Error{Status: 404}))
		})

		It("Query domain name set", func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Install the DNS log mapping
			template := mustGetString("test_files/dns_template.json")
			_, err := elasticClient.IndexPutTemplate("dns_logs").BodyString(template).Do(ctx)
			Expect(err).ToNot(HaveOccurred())

			// Index some DNS logs
			index := "tigera_secure_ee_dns.cluster.testquerydomainnameset_success"
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
				Expect(err).ToNot(HaveOccurred())
			}
			defer func() {
				_, err := elasticClient.DeleteIndex(index).Do(ctx)
				Expect(err).ToNot(HaveOccurred())
			}()

			// Wait until they are indexed
			to := time.After(30 * time.Second)
			for {
				s, err := elasticClient.Search(index).Do(ctx)
				Expect(err).ToNot(HaveOccurred())
				if s.TotalHits() == 3 {
					break
				}
				Expect(to).NotTo(Receive(), "wait for log index timed out")
				time.Sleep(10 * time.Millisecond)
			}

			// Run the search
			domains := db.DomainNameSetSpec{"xx.yy.zzz", "dd.ee.fff", "jj.kk.lll"}
			testFeed := &apiV3.GlobalThreatFeed{}
			testFeed.Name = "test-feed"
			iter, _, err := uut.QueryDomainNameSet(ctx, domains, testFeed)
			Expect(err).ToNot(HaveOccurred())

			var actual []events.DNSLog
			var keys []db.QueryKey
			for iter.Next() {
				k, h := iter.Value()
				keys = append(keys, k)
				var al events.DNSLog
				err := json.Unmarshal(h.Source, &al)
				Expect(err).ToNot(HaveOccurred())
				actual = append(actual, al)
			}
			Expect(keys).To(Equal([]db.QueryKey{
				db.QueryKeyDNSLogQName,
				db.QueryKeyDNSLogRRSetsName, db.QueryKeyDNSLogRRSetsName,
				db.QueryKeyDNSLogRRSetsRData, db.QueryKeyDNSLogRRSetsRData,
			}))

			// Qname query
			Expect(actual[0].QName).To(Equal("xx.yy.zzz"))

			// rrsets.name query
			// We identify the results by the QName, which is unique for each log.
			qnames := []string{actual[1].QName, actual[2].QName}
			// Query for xx.yy.zzz has the name xx.yy.zzz in the first RRSet
			Expect(qnames).To(ContainElement("xx.yy.zzz"))
			// Query for aa.bb.ccc has the name dd.ee.fff in the second RRSet
			Expect(qnames).To(ContainElement("aa.bb.ccc"))

			// rrsets.rdata query
			// We identify the results by the QName, which is unique for each log.
			qnames = []string{actual[3].QName, actual[4].QName}
			// Query for aa.bb.ccc has the data dd.ee.fff in the first rrset
			Expect(qnames).To(ContainElement("aa.bb.ccc"))
			// Query for gg.hh.iii has the data jj.kk.lll in the first rrset
			Expect(qnames).To(ContainElement("gg.hh.iii"))
		})

		It("Add security events for domain names", func() {
			l := events.DNSLog{
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
			}
			h := &oElastic.SearchHit{Index: "dns_index", Id: "dns_id"}
			domains := map[string]struct{}{
				"xx.yy.zzz": {},
			}
			e := events.ConvertDNSLog(l, db.QueryKeyDNSLogQName, h, domains, "my-feed", "my-other-feed")

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			err := uut.PutSecurityEventWithID(ctx, e)
			Expect(err).ToNot(HaveOccurred())

			// Verify the event exists
			result, err := elasticClient.Get().Index(elastic.EventIndex).Id(e.GetID()).Do(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Id).To(Equal(e.GetID()))
		})
	})
})

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
