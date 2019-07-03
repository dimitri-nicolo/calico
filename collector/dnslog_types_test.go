package collector

import (
	"encoding/json"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/google/gopacket/layers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DNS log type tests", func() {
	Describe("DNSRData tests", func() {
		Context("With IP address", func() {
			ipstr := "127.0.0.1"
			r := DNSRData{nil, net.ParseIP(ipstr)}
			It("Should return the IP string", func() {
				Expect(r.String()).Should(Equal(ipstr))
			})
		})
		Context("With NS", func() {
			nsstr := "ns1.tigera.io."
			r := DNSRData{nil, nsstr}
			It("Should return the right hostname", func() {
				Expect(r.String()).Should(Equal(nsstr))
			})
		})
		Context("With TXT", func() {
			txt := [][]byte{[]byte("foo"), []byte("bar")}
			r := DNSRData{nil, txt}
			It("Should return the strings joined together", func() {
				Expect(r.String()).Should(Equal("foobar"))
			})
		})
		Context("With SOA", func() {
			soa := layers.DNSSOA{
				MName:   []byte("tigera.io."),
				RName:   []byte("root.tigera.io."),
				Serial:  1,
				Refresh: 3600,
				Retry:   60,
				Expire:  86400,
				Minimum: 1800,
			}
			r := DNSRData{nil, soa}
			It("Should return the zone formatted SOA", func() {
				Expect(r.String()).Should(Equal("tigera.io. root.tigera.io. 1 3600 60 86400 1800"))
			})
		})
		Context("With SRV", func() {
			srv := layers.DNSSRV{
				Priority: 10,
				Weight:   20,
				Port:     53,
				Name:     []byte("ns.tigera.io."),
			}
			r := DNSRData{nil, srv}
			It("Should return the zone formatted SRV", func() {
				Expect(r.String()).Should(Equal("10 20 53 ns.tigera.io."))
			})
		})
		Context("With MX", func() {
			mx := layers.DNSMX{
				Preference: 10,
				Name:       []byte("mail.tigera.io."),
			}
			r := DNSRData{nil, mx}
			It("Should return the zone formatted MX", func() {
				Expect(r.String()).Should(Equal("10 mail.tigera.io."))
			})
		})
		Context("With bytes", func() {
			b := []byte("abc")
			r := DNSRData{nil, b}
			It("Should return the base64 encoded string", func() {
				Expect(r.String()).Should(Equal("YWJj"))
			})
		})
		Context("With unexpected", func() {
			r := DNSRData{}
			It("Should return \"nil\"", func() {
				Expect(r.String()).Should(Equal("<nil>"))
			})
		})
		Context("JSON marshal", func() {
			t := "test"
			r := DNSRData{Raw: []byte("junk"), Decoded: t}
			It("Should only encode the decoded as a string", func() {
				b, err := json.Marshal(&r)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(string(b)).Should(Equal(fmt.Sprintf(`"%s"`, t)))
			})
		})
	})

	Describe("DNSRdatas", func() {
		Context("Len function", func() {
			It("Nil array", func() {
				Expect(DNSRDatas(nil).Len()).Should(Equal(0))
			})
			It("Empty array", func() {
				Expect(DNSRDatas{}.Len()).Should(Equal(0))
			})
			It("Array with stuff in it", func() {
				Expect(DNSRDatas{{}, {}}.Len()).Should(Equal(2))
			})
		})

		Context("Less function", func() {
			It("Less", func() {
				Expect(DNSRDatas{{Raw: []byte("a")}, {Raw: []byte("b")}}.Less(0, 1)).Should(BeTrue())
			})
			It("Equal", func() {
				Expect(DNSRDatas{{Raw: []byte("a")}, {Raw: []byte("a")}}.Less(0, 1)).Should(BeFalse())
			})
			It("Greater", func() {
				Expect(DNSRDatas{{Raw: []byte("a")}, {Raw: []byte("b")}}.Less(1, 0)).Should(BeFalse())
			})
			It("Same", func() {
				Expect(DNSRDatas{{Raw: []byte("a")}, {Raw: []byte("b")}}.Less(0, 0)).Should(BeFalse())
			})
		})

		Context("Swap function", func() {
			var r DNSRDatas
			BeforeEach(func() {
				r = DNSRDatas{{Raw: []byte("a")}, {Raw: []byte("b")}}
			})

			It("Swapped", func() {
				r.Swap(0, 1)
				Expect(r).Should(Equal(DNSRDatas{{Raw: []byte("b")}, {Raw: []byte("a")}}))
			})

			It("Swapped same", func() {
				r.Swap(0, 0)
				Expect(r).Should(Equal(DNSRDatas{{Raw: []byte("a")}, {Raw: []byte("b")}}))
			})
		})

	})

	Describe("DNSName", func() {
		Context("A", func() {
			n := DNSName{"tigera.io.", DNSClass(layers.DNSClassIN), DNSType(layers.DNSTypeA)}
			It("String", func() {
				Expect(n.String()).Should(Equal("tigera.io. IN A"))
			})
		})
		Context("Unknown Class", func() {
			n := DNSName{"tigera.io.", DNSClass(5), DNSType(layers.DNSTypeA)}
			It("String", func() {
				Expect(n.String()).Should(Equal("tigera.io. #5 A"))
			})
		})
		Context("Unknown type", func() {
			n := DNSName{"tigera.io.", DNSClass(layers.DNSClassIN), DNSType(254)}
			It("String", func() {
				Expect(n.String()).Should(Equal("tigera.io. IN #254"))
			})
		})
		Context("Comparator", func() {
			It("Less, same root", func() {
				Expect(DNSName{Name: "a.b."}.Less(DNSName{Name: "b.b."})).Should(BeTrue())
			})
			It("Equal", func() {
				Expect(DNSName{Name: "a.b."}.Less(DNSName{Name: "a.b."})).Should(BeFalse())
			})
			It("Greater, same root", func() {
				Expect(DNSName{Name: "b.b."}.Less(DNSName{Name: "a.b."})).Should(BeFalse())
			})
			It("Less, subdomain on right", func() {
				Expect(DNSName{Name: "a.b."}.Less(DNSName{Name: "c.a.b."})).Should(BeTrue())
			})
			It("Greater, subdomain on right", func() {
				Expect(DNSName{Name: "b.b."}.Less(DNSName{Name: "c.a.b."})).Should(BeFalse())
			})
			It("Less, subdomain on left", func() {
				Expect(DNSName{Name: "c.a.b."}.Less(DNSName{Name: "b.b."})).Should(BeTrue())
			})
			It("Greater, subdomain on left", func() {
				Expect(DNSName{Name: "c.a.b."}.Less(DNSName{Name: "a.b."})).Should(BeFalse())
			})
			It("Longer names", func() {
				Expect(DNSName{Name: "cd.ab."}.Less(DNSName{Name: "abcd."})).Should(BeTrue())
			})
			It("Different classes", func() {
				Expect(DNSName{Name: "a.", Class: DNSClass(1)}.Less(DNSName{Name: "a.", Class: DNSClass(2)})).Should(BeTrue())
				Expect(DNSName{Name: "a.", Class: DNSClass(2)}.Less(DNSName{Name: "a.", Class: DNSClass(1)})).Should(BeFalse())
			})
			It("Different types", func() {
				Expect(DNSName{Name: "a.", Type: DNSType(1)}.Less(DNSName{Name: "a.", Type: DNSType(2)})).Should(BeTrue())
				Expect(DNSName{Name: "a.", Type: DNSType(2)}.Less(DNSName{Name: "a.", Type: DNSType(1)})).Should(BeFalse())
			})
			It("sorts correctly", func() {
				a := DNSNames{
					DNSName{Name: "b."},
					DNSName{Name: "a."},
					DNSName{Name: "c."},
					DNSName{Name: "a.", Type: DNSType(1)},
				}
				sort.Stable(a)
				Expect(a).Should(Equal(DNSNames{
					DNSName{Name: "a."},
					DNSName{Name: "a.", Type: DNSType(1)},
					DNSName{Name: "b."},
					DNSName{Name: "c."},
				}))
			})
		})
		Context("JSON", func() {
			It("All knowns", func() {
				n := DNSName{"tigera.io.", DNSClass(layers.DNSClassIN), DNSType(layers.DNSTypeA)}
				b, err := json.Marshal(n)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(string(b)).Should(Equal(`{"name":"tigera.io.","class":"IN","type":"A"}`))
			})
			It("Unknowns", func() {
				n := DNSName{"tigera.io.", DNSClass(5), DNSType(254)}
				b, err := json.Marshal(n)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(string(b)).Should(Equal(`{"name":"tigera.io.","class":5,"type":254}`))

			})
		})
	})

	Describe("DNSRRSets", func() {
		Context("Empty", func() {
			r := make(DNSRRSets)

			It("Empty", func() {
				Expect(r.String()).Should(Equal(""))
			})
		})

		Context("Populated", func() {
			r := make(DNSRRSets)

			r[DNSName{"tigera.io.", DNSClass(layers.DNSClassIN), DNSType(layers.DNSTypeA)}] = DNSRDatas{
				{Decoded: net.ParseIP("127.0.0.1")},
				{Decoded: net.ParseIP("192.168.0.1")},
			}
			r[DNSName{"cname.tigera.io.", DNSClass(layers.DNSClassIN), DNSType(layers.DNSTypeCNAME)}] = DNSRDatas{
				{Decoded: "www.tigera.io."},
			}

			It("Populated", func() {
				Expect(r.String()).Should(Equal(strings.Join([]string{
					"tigera.io. IN A 127.0.0.1",
					"tigera.io. IN A 192.168.0.1",
					"cname.tigera.io. IN CNAME www.tigera.io.",
				}, "\n")))
			})
		})

		Context("Add function", func() {
			It("Empty set should add a record", func() {
				r := make(DNSRRSets)
				name := DNSName{"tigera.io", 1, 1}

				r.Add(name, DNSRData{Raw: []byte("2"), Decoded: "test1"})
				Expect(r).Should(HaveLen(1))
				Expect(r[name]).Should(HaveLen(1))
			})

			It("Set with a duplicate key should add records in the right order", func() {
				r := make(DNSRRSets)
				name := DNSName{"tigera.io", 1, 1}

				r.Add(name, DNSRData{Raw: []byte("2"), Decoded: "test1"})
				r.Add(name, DNSRData{Raw: []byte("1"), Decoded: "test2"})
				r.Add(name, DNSRData{Raw: []byte("3"), Decoded: "test3"})

				Expect(r).Should(HaveLen(1))
				Expect(r[name]).Should(Equal(DNSRDatas{
					{Raw: []byte("1"), Decoded: "test2"},
					{Raw: []byte("2"), Decoded: "test1"},
					{Raw: []byte("3"), Decoded: "test3"},
				}))
			})

			It("Set with a different key should add a second record", func() {
				r := make(DNSRRSets)

				r.Add(DNSName{"tigera.io", 1, 1}, DNSRData{Raw: []byte("1"), Decoded: "test1"})
				r.Add(DNSName{"tigera.io", 1, 2}, DNSRData{Raw: []byte("2"), Decoded: "test2"})

				Expect(r).Should(HaveLen(2))
			})
		})
	})

	Describe("DNSData", func() {
		var d DNSData
		BeforeEach(func() {
			d = DNSData{
				DNSMeta: DNSMeta{},
				DNSSpec: DNSSpec{
					Servers: map[EndpointMetadataWithIP]DNSLabels{
						{}: {
							"c": "d",
						},
					},
					ClientLabels: map[string]string{
						"a": "b",
					},
				},
			}
		})

		It("includes labels when desired", func() {
			l := d.ToDNSLog(time.Time{}, time.Time{}, true)
			Expect(l.ClientLabels).ShouldNot(HaveLen(0))
			for _, s := range l.Servers {
				Expect(s.Labels).ShouldNot(HaveLen(0))
			}
		})

		It("excludes labels when desired", func() {
			l := d.ToDNSLog(time.Time{}, time.Time{}, false)
			Expect(l.ClientLabels).Should(HaveLen(0))
			for _, s := range l.Servers {
				Expect(s.Labels).Should(HaveLen(0))
			}
		})

		It("excluding labels has no side effects", func() {
			d.ToDNSLog(time.Time{}, time.Time{}, false)
			Expect(d.ClientLabels).ShouldNot(HaveLen(0))
			for _, l := range d.Servers {
				Expect(l).ShouldNot(HaveLen(0))
			}
		})
	})

	Describe("DNSSpec", func() {
		Context("Merge", func() {
			It("Merges correctly", func() {
				origCount := uint(2)

				a := DNSSpec{
					Servers: map[EndpointMetadataWithIP]DNSLabels{
						{EndpointMetadata: EndpointMetadata{Name: "ns1"}}: {"a": "b"},
						{EndpointMetadata: EndpointMetadata{Name: "ns2"}}: {"d": "e"},
					},
					ClientLabels: map[string]string{
						"0": "0",
					},
					DNSStats: DNSStats{
						Count: origCount,
					},
				}
				b := DNSSpec{
					Servers: map[EndpointMetadataWithIP]DNSLabels{
						{EndpointMetadata: EndpointMetadata{Name: "ns1"}}: {"b": "c"},
						{EndpointMetadata: EndpointMetadata{Name: "ns3"}}: {"f": "g"},
					},
					ClientLabels: map[string]string{
						"1": "2",
					},
					DNSStats: DNSStats{
						Count: 5,
					},
				}

				a.Merge(b)
				Expect(a.ClientLabels).Should(Equal(b.ClientLabels))
				Expect(a.Count).Should(Equal(origCount + b.Count))
				Expect(a.Servers).Should(Equal(map[EndpointMetadataWithIP]DNSLabels{
					{EndpointMetadata: EndpointMetadata{Name: "ns1"}}: {"b": "c"},
					{EndpointMetadata: EndpointMetadata{Name: "ns2"}}: {"d": "e"},
					{EndpointMetadata: EndpointMetadata{Name: "ns3"}}: {"f": "g"},
				}))
			})
		})
	})

	Describe("DNSLog Tests", func() {
		It("marshals correctly", func() {
			t := time.Date(2019, 07, 02, 0, 0, 0, 0, time.UTC)

			l := &DNSLog{
				StartTime:       t,
				EndTime:         t.Add(time.Minute),
				Count:           5,
				ClientName:      "test-1",
				ClientNameAggr:  "test-*",
				ClientNamespace: "test-ns",
				ClientIP:        "127.0.0.1",
				ClientLabels: map[string]string{
					"t1": "a",
				},
				Servers: []DNSServer{
					{
						EndpointMetadataWithIP: EndpointMetadataWithIP{
							EndpointMetadata: EndpointMetadata{
								Type:           "Pod",
								Namespace:      "test2-ns",
								Name:           "test-2",
								AggregatedName: "test-*",
							},
							IP: "192.168.0.1",
						},
						Labels: map[string]string{
							"t2": "b",
						},
					},
				},
				QName:  "tigera.io",
				QClass: DNSClass(layers.DNSClassIN),
				QType:  DNSType(layers.DNSTypeA),
				RCode:  DNSResponseCode(layers.DNSResponseCodeNoErr),
				RRSets: DNSRRSets{
					{
						Name:  "tigera.io",
						Class: DNSClass(layers.DNSClassIN),
						Type:  DNSType(layers.DNSTypeA),
					}: {
						{Decoded: net.ParseIP("127.0.0.1")},
						{Decoded: net.ParseIP("127.0.0.2")},
					},
				},
			}

			b, err := json.Marshal(l)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(string(b)).Should(Equal(`{"start_time":"2019-07-02T00:00:00Z","end_time":"2019-07-02T00:01:00Z","count":5,"client_name":"test-1","client_name_aggr":"test-*","client_namespace":"test-ns","client_ip":"127.0.0.1","client_labels":{"t1":"a"},"servers":[{"name":"test-2","name_aggr":"test-*","namespace":"test2-ns","ip":"192.168.0.1"}],"qname":"tigera.io","qclass":"IN","qtype":"A","rcode":"NoError","rrsets":[{"name":"tigera.io","class":"IN","type":"A","rdata":["127.0.0.1","127.0.0.2"]}]}`))
		})

	})
})
