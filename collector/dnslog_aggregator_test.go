// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package collector

import (
	"fmt"
	"net"
	"time"

	"github.com/google/gopacket/layers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DNS log aggregator", func() {
	var l *dnsLogAggregator
	BeforeEach(func() {
		l = NewDNSLogAggregator().(*dnsLogAggregator)
	})

	Describe("constructor", func() {
		It("initializes the dns store", func() {
			Expect(l.dnsStore).ShouldNot(BeNil())
		})
		It("sets the start time", func() {
			Expect(l.aggregationStartTime).Should(BeTemporally(">", time.Time{}))
		})
	})

	Describe("settings", func() {
		It("include labels", func() {
			Expect(l.includeLabels).Should(BeFalse())
			Expect(l.IncludeLabels(true)).Should(Equal(l))
			Expect(l.includeLabels).Should(BeTrue())
			Expect(l.IncludeLabels(false)).Should(Equal(l))
			Expect(l.includeLabels).Should(BeFalse())
		})

		It("aggregate over", func() {
			Expect(l.kind).Should(Equal(Default))
			Expect(l.AggregateOver(SourcePort)).Should(Equal(l))
			Expect(l.kind).Should(Equal(SourcePort))
			Expect(l.AggregateOver(PrefixName)).Should(Equal(l))
			Expect(l.kind).Should(Equal(PrefixName))
			Expect(l.AggregateOver(Default)).Should(Equal(l))
			Expect(l.kind).Should(Equal(Default))
		})
	})

	Describe("feed update", func() {
		BeforeEach(func() {
			err := l.FeedUpdate(&layers.DNS{
				ResponseCode: layers.DNSResponseCodeNoErr,
				Questions: []layers.DNSQuestion{
					{Name: []byte("tigera.io."), Type: layers.DNSTypeA, Class: layers.DNSClassIN},
				},
				Answers: []layers.DNSResourceRecord{
					{Name: []byte("tigera.io."), Type: layers.DNSTypeA, Class: layers.DNSClassIN, IP: net.ParseIP("127.0.0.1")},
				},
			})

			Expect(err).ShouldNot(HaveOccurred())
			Expect(l.dnsStore).Should(HaveLen(1))
		})

		It("new entry", func() {
			err := l.FeedUpdate(&layers.DNS{
				ResponseCode: layers.DNSResponseCodeNoErr,
				Questions: []layers.DNSQuestion{
					{Name: []byte("tigera.io."), Type: layers.DNSTypeAAAA, Class: layers.DNSClassIN},
				},
				Answers: []layers.DNSResourceRecord{
					{Name: []byte("tigera.io."), Type: layers.DNSTypeAAAA, Class: layers.DNSClassIN, IP: net.ParseIP("::1")},
				},
			})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(l.dnsStore).Should(HaveLen(2))
			for _, v := range l.dnsStore {
				Expect(v.Count).Should(BeNumerically("==", 1))
			}
		})

		It("update with same rdata", func() {
			err := l.FeedUpdate(&layers.DNS{
				ResponseCode: layers.DNSResponseCodeNoErr,
				Questions: []layers.DNSQuestion{
					{Name: []byte("tigera.io."), Type: layers.DNSTypeA, Class: layers.DNSClassIN},
				},
				Answers: []layers.DNSResourceRecord{
					{Name: []byte("tigera.io."), Type: layers.DNSTypeA, Class: layers.DNSClassIN, IP: net.ParseIP("127.0.0.1")},
				},
			})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(l.dnsStore).Should(HaveLen(1))
			for _, v := range l.dnsStore {
				Expect(v.Count).Should(BeNumerically("==", 2))
			}
		})

		It("update with different rdata", func() {
			err := l.FeedUpdate(&layers.DNS{
				ResponseCode: layers.DNSResponseCodeNoErr,
				Questions: []layers.DNSQuestion{
					{Name: []byte("tigera.io."), Type: layers.DNSTypeA, Class: layers.DNSClassIN},
				},
				Answers: []layers.DNSResourceRecord{
					{Name: []byte("tigera.io."), Type: layers.DNSTypeA, Class: layers.DNSClassIN, IP: net.ParseIP("127.0.0.2")},
				},
			})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(l.dnsStore).Should(HaveLen(2))
			for _, v := range l.dnsStore {
				Expect(v.Count).Should(BeNumerically("==", 1))
			}
		})
	})

	Describe("get", func() {
		It("empty", func() {
			Expect(l.Get()).Should(HaveLen(0))
		})

		Describe("populated", func() {
			BeforeEach(func() {
				err := l.FeedUpdate(&layers.DNS{
					ResponseCode: layers.DNSResponseCodeNoErr,
					Questions: []layers.DNSQuestion{
						{Name: []byte("tigera.io."), Type: layers.DNSTypeA, Class: layers.DNSClassIN},
					},
					Answers: []layers.DNSResourceRecord{
						{Name: []byte("tigera.io."), Type: layers.DNSTypeA, Class: layers.DNSClassIN, IP: net.ParseIP("127.0.0.1")},
					},
				})
				Expect(err).ShouldNot(HaveOccurred())

				err = l.FeedUpdate(&layers.DNS{
					ResponseCode: layers.DNSResponseCodeNoErr,
					Questions: []layers.DNSQuestion{
						{Name: []byte("tigera.io."), Type: layers.DNSTypeA, Class: layers.DNSClassIN},
					},
					Answers: []layers.DNSResourceRecord{
						{Name: []byte("tigera.io."), Type: layers.DNSTypeA, Class: layers.DNSClassIN, IP: net.ParseIP("127.0.0.2")},
					},
				})
				Expect(err).ShouldNot(HaveOccurred())
			})

			for _, b := range []bool{false, true} {
				withLabels := b
				var logs []*DNSLog

				Describe(fmt.Sprintf("withLabels: %t", withLabels), func() {
					BeforeEach(func() {
						l.IncludeLabels(withLabels)
						logs = l.Get()
						Expect(logs).Should(HaveLen(2))
					})

					It("sets startTime correctly", func() {
						for _, log := range logs {
							Expect(log.StartTime).Should(BeTemporally("==", l.aggregationStartTime))
						}
					})

					It("sets endTime correctly", func() {
						for _, log := range logs {
							Expect(log.EndTime).Should(BeTemporally("~", time.Now(), time.Minute))
						}
					})

					switch withLabels {
					case true:
						It("includes labels", func() {
							// TODO
						})
					case false:
						It("excludes labels", func() {
							for _, log := range logs {
								Expect(log.ClientLabels).Should(HaveLen(0))
								for _, server := range log.Servers {
									Expect(server.Labels).Should(HaveLen(0))
								}
							}
						})
					}
				})
			}
		})

	})
})
