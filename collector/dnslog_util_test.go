package collector

import (
	"net"

	"github.com/google/gopacket/layers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DNS log utility functions", func() {
	Describe("canonicalizeDNSName", func() {
		Context("noop", func() {
			It("returns the input string", func() {
				in := "tigera.io"
				Expect(canonicalizeDNSName([]byte(in))).Should(Equal(in))
			})
		})
		Context("remove superfluous dots", func() {
			It("strips the dots from the left and right", func() {
				Expect(canonicalizeDNSName([]byte(".tigera.io."))).Should(Equal("tigera.io"))
			})
			It("removes extra dots", func() {
				Expect(canonicalizeDNSName([]byte("..tigera..io.."))).Should(Equal("tigera.io"))
			})
		})
		Context("normalizes characters", func() {
			It("corrects case", func() {
				Expect(canonicalizeDNSName([]byte("tIgeRa.Io"))).Should(Equal("tigera.io"))
			})
		})
	})

	Describe("getRRDecoded", func() {
		It("returns a net.IP for A", func() {
			decoded := net.ParseIP("127.0.0.1")
			v := getRRDecoded(layers.DNSResourceRecord{Type: layers.DNSTypeA, IP: decoded})
			Expect(v).Should(BeAssignableToTypeOf(net.IP{}))
			Expect(v).Should(Equal(decoded))
		})
		It("returns a net.IP for AAAA", func() {
			decoded := net.ParseIP("::1")
			v := getRRDecoded(layers.DNSResourceRecord{Type: layers.DNSTypeAAAA, IP: decoded})
			Expect(v).Should(BeAssignableToTypeOf(net.IP{}))
			Expect(v).Should(Equal(decoded))
		})
		It("returns a string for NS", func() {
			decoded := []byte("tigera.io")
			v := getRRDecoded(layers.DNSResourceRecord{Type: layers.DNSTypeNS, NS: decoded})
			Expect(v).Should(BeAssignableToTypeOf(""))
			Expect(v).Should(Equal(string(decoded)))
		})
		It("returns a string for CNAME", func() {
			decoded := []byte("tigera.io")
			v := getRRDecoded(layers.DNSResourceRecord{Type: layers.DNSTypeCNAME, CNAME: decoded})
			Expect(v).Should(BeAssignableToTypeOf(""))
			Expect(v).Should(Equal(string(decoded)))
		})
		It("returns a string for PTR", func() {
			decoded := []byte("tigera.io")
			v := getRRDecoded(layers.DNSResourceRecord{Type: layers.DNSTypePTR, PTR: decoded})
			Expect(v).Should(BeAssignableToTypeOf(""))
			Expect(v).Should(Equal(string(decoded)))
		})
		It("returns a [][]byte for TXT", func() {
			decoded := [][]byte{[]byte("tigera."), []byte("io")}
			v := getRRDecoded(layers.DNSResourceRecord{Type: layers.DNSTypeTXT, TXTs: decoded})
			Expect(v).Should(BeAssignableToTypeOf([][]byte{}))
			Expect(v).Should(Equal(decoded))
		})
		It("returns a layers.DNSSOA for SOA", func() {
			decoded := layers.DNSSOA{
				MName: []byte("tigera.io."),
			}
			v := getRRDecoded(layers.DNSResourceRecord{Type: layers.DNSTypeSOA, SOA: decoded})
			Expect(v).Should(BeAssignableToTypeOf(layers.DNSSOA{}))
			Expect(v).Should(Equal(decoded))
		})
		It("returns a layers.DNSSRV for SRV", func() {
			decoded := layers.DNSSRV{
				Priority: 10,
			}
			v := getRRDecoded(layers.DNSResourceRecord{Type: layers.DNSTypeSRV, SRV: decoded})
			Expect(v).Should(BeAssignableToTypeOf(layers.DNSSRV{}))
			Expect(v).Should(Equal(decoded))
		})
		It("returns a layers.DNSMX for MX", func() {
			decoded := layers.DNSMX{
				Preference: 10,
			}
			v := getRRDecoded(layers.DNSResourceRecord{Type: layers.DNSTypeMX, MX: decoded})
			Expect(v).Should(BeAssignableToTypeOf(layers.DNSMX{}))
			Expect(v).Should(Equal(decoded))
		})
		It("returns a []byte for unknown", func() {
			raw := []byte("raw")
			v := getRRDecoded(layers.DNSResourceRecord{Type: 0, Data: raw})
			Expect(v).Should(BeAssignableToTypeOf([]byte{}))
			Expect(v).Should(Equal(raw))
		})
	})
})
