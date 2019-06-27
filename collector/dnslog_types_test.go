package collector

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"

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
			r := DNSRDatas{{Raw: []byte("a")}, {Raw: []byte("b")}}
			e := DNSRDatas{{Raw: []byte("b")}, {Raw: []byte("a")}}

			r.Swap(0, 1)
			It("Swapped", func() {
				Expect(r).Should(Equal(e))
			})

			r.Swap(0, 0)
			It("Swapped same", func() {
				Expect(r).Should(Equal(e))
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
	})

	Describe("DNSSpec", func() {
		Context("Merge", func() {
			It("Merges correctly", func() {
				origCount := uint(2)

				a := DNSSpec{
					Servers: map[EndpointMetadata]DNSLabels{
						{Name: "ns1"}: {"a": "b"},
						{Name: "ns2"}: {"d": "e"},
					},
					ClientLabels: map[string]string{
						"0": "0",
					},
					DNSStats: DNSStats{
						Count: origCount,
					},
				}
				b := DNSSpec{
					Servers: map[EndpointMetadata]DNSLabels{
						{Name: "ns1"}: {"b": "c"},
						{Name: "ns3"}: {"f": "g"},
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
				Expect(a.Servers).Should(Equal(map[EndpointMetadata]DNSLabels{
					{Name: "ns1"}: {"b": "c"},
					{Name: "ns2"}: {"d": "e"},
					{Name: "ns3"}: {"f": "g"},
				}))
			})
		})
		Context("JSON", func() {
			It("Encodes correctly", func() {
				a := DNSSpec{
					RRSets: DNSRRSets{
						{
							Name:  "tigera.io.",
							Class: DNSClass(layers.DNSClassIN),
							Type:  DNSType(layers.DNSTypeCNAME),
						}: {{
							Decoded: "www.tigera.io.",
						}},
					},
					Servers: map[EndpointMetadata]DNSLabels{
						{Name: "ns1"}: {"b": "c"},
					},
					ClientLabels: map[string]string{
						"1": "2",
					},
					DNSStats: DNSStats{
						Count: 2,
					},
				}

				b, err := json.Marshal(&a)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(string(b)).Should(Equal(`{"rrsets":[{"name":"tigera.io.","class":"IN","type":"CNAME","rdata":["www.tigera.io."]}],"servers":[{"type":"","namespace":"","name":"ns1","aggregated_name":"","labels":{"b":"c"}}],"clientLabels":{"1":"2"},"count":2}`))
			})
			It("Decodes correctly", func() {
				b := []byte(`{"rrsets":[{"name":"tigera.io.","class":"IN","type":"CNAME","rdata":["www.tigera.io."]}],"servers":[{"type":"","namespace":"","name":"ns1","aggregated_name":"","labels":{"b":"c"}}],"clientLabels":{"1":"2"},"count":2}`)
				var a DNSSpec
				err := json.Unmarshal(b, &a)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(a).Should(Equal(DNSSpec{
					RRSets: DNSRRSets{
						{
							Name:  "tigera.io.",
							Class: DNSClass(layers.DNSClassIN),
							Type:  DNSType(layers.DNSTypeCNAME),
						}: {{
							Decoded: "www.tigera.io.",
						}},
					},
					Servers: map[EndpointMetadata]DNSLabels{
						{Name: "ns1"}: {"b": "c"},
					},
					ClientLabels: map[string]string{
						"1": "2",
					},
					DNSStats: DNSStats{
						Count: 2,
					},
				}))
			})
		})
	})
})
