// Copyright (c) 2019-2022 Tigera, Inc. All rights reserved.

package common

import (
	"fmt"
	"net"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/felix/ip"
	"github.com/projectcalico/calico/felix/proto"
	"github.com/projectcalico/calico/felix/testutils"
)

type mockDomainInfoChangeHandler struct {
	dataplaneSyncNeeded bool
	domainsChanged      []string
}

func (m *mockDomainInfoChangeHandler) OnDomainChange(name string) (dataplaneSyncNeeded bool) {
	m.domainsChanged = append(m.domainsChanged, name)
	return m.dataplaneSyncNeeded
}

var _ = Describe("Domain Info Store", func() {
	var (
		handler             *mockDomainInfoChangeHandler
		domainStore         *DomainInfoStore
		mockDNSRecA1        = testutils.MakeA("a.com", "10.0.0.10")
		mockDNSRecA2        = testutils.MakeA("b.com", "10.0.0.20")
		mockDNSRecA2Caps    = testutils.MakeA("B.cOm", "10.0.0.20")
		mockDNSRecAAAA1     = testutils.MakeAAAA("aaaa.com", "fe80:fe11::1")
		mockDNSRecAAAA2     = testutils.MakeAAAA("bbbb.com", "fe80:fe11::2")
		mockDNSRecAAAA3Caps = testutils.MakeAAAA("mIxEdCaSe.CoM", "fe80:fe11::3")
		invalidDNSRec       = layers.DNSResourceRecord{
			Name:       []byte("invalid#rec.com"),
			Type:       layers.DNSTypeMX,
			Class:      layers.DNSClassAny,
			TTL:        2147483648,
			DataLength: 0,
			Data:       []byte("999.000.999.000"),
			IP:         net.ParseIP("999.000.999.000"),
		}
		mockDNSRecCNAME = []layers.DNSResourceRecord{
			testutils.MakeCNAME("cname1.com", "cname2.com"),
			testutils.MakeCNAME("cNAME2.com", "cname3.com"),
			testutils.MakeCNAME("cname3.com", "a.com"),
		}
		mockDNSRecCNAMEUnderscore = []layers.DNSResourceRecord{
			testutils.MakeCNAME("cname_1.com", "cname2.com"),
			testutils.MakeCNAME("cNAME2.com", "cname_3.com"),
			testutils.MakeCNAME("cname_3.com", "a.com"),
		}
		lastTTL time.Duration

		// Callback indication. Tests using callbacks must emulate the dataplane by calling back into the
		// DomainInfoStore to notify when the dataplane is programmed.
		callbackIdsMutex sync.Mutex
		callbackIds      []string
	)

	BeforeEach(func() {
		// Reset the callback IDs.
		callbackIdsMutex.Lock()
		defer callbackIdsMutex.Unlock()
		callbackIds = nil
	})

	// Program a DNS record as an "answer" type response.
	programDNSAnswer := func(domainStore *DomainInfoStore, dnsPacket layers.DNSResourceRecord, callbackId ...string) {
		var layerDNS layers.DNS
		layerDNS.Answers = append(layerDNS.Answers, dnsPacket)
		var cb func()
		if len(callbackId) == 1 {
			cb = func() {
				callbackIdsMutex.Lock()
				defer callbackIdsMutex.Unlock()
				callbackIds = append(callbackIds, callbackId[0])
			}
		}
		domainStore.processDNSPacket(&layerDNS, cb)
	}

	// Program a DNS record as an "additionals" type response.
	programDNSAdditionals := func(domainStore *DomainInfoStore, dnsPacket layers.DNSResourceRecord, callbackId ...string) {
		var layerDNS layers.DNS
		layerDNS.Additionals = append(layerDNS.Additionals, dnsPacket)
		var cb func()
		if len(callbackId) == 1 {
			cb = func() {
				callbackIdsMutex.Lock()
				defer callbackIdsMutex.Unlock()
				callbackIds = append(callbackIds, callbackId[0])
			}
		}
		domainStore.processDNSPacket(&layerDNS, cb)
	}

	// Assert that the domain store accepted and signaled the given record (and reason).
	AssertDomainChanged := func(domainStore *DomainInfoStore, d string, r string) {
		Expect(domainStore.UpdatesReadyChannel()).Should(Receive())
		log.Info("Domain updates ready to handle")
		domainStore.HandleUpdates()
		// DomainInfoStore stores domains in lowercase.
		Expect(handler.domainsChanged).To(ConsistOf(strings.ToLower(d)))

		// Reset the domains changed ready for the next test.
		handler.domainsChanged = nil
	}

	// Assert that the domain store registered the given record and then process its expiration.
	AssertValidRecord := func(dnsRec layers.DNSResourceRecord) {
		It("should result in a domain entry", func() {
			Expect(domainStore.GetDomainIPs(string(dnsRec.Name))).To(Equal([]string{dnsRec.IP.String()}))
		})
		It("should expire and signal a domain change", func() {
			domainStore.processMappingExpiry(strings.ToLower(string(dnsRec.Name)), dnsRec.IP.String())
			AssertDomainChanged(domainStore, string(dnsRec.Name), "mapping expired")
			Expect(domainStore.collectGarbage()).To(Equal(1))
		})
	}

	defaultConfig := &DnsConfig{
		DNSCacheFile:         "/dnsinfo",
		DNSCacheSaveInterval: time.Minute,
	}

	// Create a new datastore.
	domainStoreCreateEx := func(capacity int, config *DnsConfig) {
		handler = &mockDomainInfoChangeHandler{
			// For most tests assume the dataplane does need to be sync'd.
			dataplaneSyncNeeded: true,
		}
		// For UT purposes, don't actually run any expiry timers, but arrange that mappings
		// always appear to have expired when UT code calls processMappingExpiry.
		domainStore = newDomainInfoStoreWithShims(
			config,
			func(ttl time.Duration, _ func()) *time.Timer {
				lastTTL = ttl
				return nil
			},
			func(time.Time) bool { return true },
		)
		domainStore.RegisterHandler(handler)
	}
	domainStoreCreate := func() {
		// Create domain info store with 100 capacity for changes channel.
		domainStoreCreateEx(100, defaultConfig)
	}

	// Basic validation tests that add/expire one or two DNS records of A and AAAA type to the data store.
	domainStoreTestValidRec := func(dnsRec1, dnsRec2 layers.DNSResourceRecord) {
		Describe("receiving a DNS packet", func() {
			BeforeEach(func() {
				domainStoreCreate()
			})

			Context("with a valid type A DNS answer record", func() {
				BeforeEach(func() {
					programDNSAnswer(domainStore, dnsRec1)
					AssertDomainChanged(domainStore, string(dnsRec1.Name), "mapping added")
				})
				AssertValidRecord(dnsRec1)
			})

			Context("with a valid type A DNS additional record", func() {
				BeforeEach(func() {
					programDNSAdditionals(domainStore, dnsRec1)
					AssertDomainChanged(domainStore, string(dnsRec1.Name), "mapping added")
				})
				AssertValidRecord(dnsRec1)
			})

			Context("with two valid type A DNS answer records", func() {
				BeforeEach(func() {
					programDNSAnswer(domainStore, dnsRec1)
					AssertDomainChanged(domainStore, string(dnsRec1.Name), "mapping added")
					programDNSAnswer(domainStore, dnsRec2)
					AssertDomainChanged(domainStore, string(dnsRec2.Name), "mapping added")
				})
				AssertValidRecord(dnsRec1)
				AssertValidRecord(dnsRec2)
			})

			Context("with two valid type A DNS additional records", func() {
				BeforeEach(func() {
					programDNSAdditionals(domainStore, dnsRec1)
					AssertDomainChanged(domainStore, string(dnsRec1.Name), "mapping added")
					programDNSAdditionals(domainStore, dnsRec2)
					AssertDomainChanged(domainStore, string(dnsRec2.Name), "mapping added")
				})
				AssertValidRecord(dnsRec1)
				AssertValidRecord(dnsRec2)
			})
		})
	}

	// Check that a malformed DNS record will not be accepted.
	domainStoreTestInvalidRec := func(dnsRec layers.DNSResourceRecord) {
		Context("with an invalid DNS record", func() {
			BeforeEach(func() {
				domainStoreCreate()
				programDNSAnswer(domainStore, dnsRec)
			})
			It("should return nil", func() {
				Expect(domainStore.GetDomainIPs(string(dnsRec.Name))).To(BeNil())
			})
		})
	}

	// Check that a chain of CNAME records with one A record results in a domain change only for the latter.
	domainStoreTestCNAME := func(CNAMErecs []layers.DNSResourceRecord, aRec layers.DNSResourceRecord) {
		Context("with a chain of CNAME records", func() {
			BeforeEach(func() {
				// Check that we receive signals when there are updates ready.
				domainStoreCreate()
				for _, r := range CNAMErecs {
					programDNSAnswer(domainStore, r)
					Expect(domainStore.UpdatesReadyChannel()).Should(Receive())
					domainStore.HandleUpdates()
				}
				programDNSAnswer(domainStore, aRec)
				Expect(domainStore.UpdatesReadyChannel()).Should(Receive())
				domainStore.HandleUpdates()
			})
			It("should result in a CNAME->A mapping", func() {
				Expect(domainStore.GetDomainIPs(string(CNAMErecs[0].Name))).To(Equal([]string{aRec.IP.String()}))
			})
		})
	}

	domainStoreTestValidRec(mockDNSRecA1, mockDNSRecA2)
	domainStoreTestValidRec(mockDNSRecA1, mockDNSRecA2Caps)
	domainStoreTestValidRec(mockDNSRecAAAA1, mockDNSRecAAAA2)
	domainStoreTestValidRec(mockDNSRecAAAA1, mockDNSRecAAAA3Caps)
	domainStoreTestInvalidRec(invalidDNSRec)
	domainStoreTestCNAME(mockDNSRecCNAME, mockDNSRecA1)
	domainStoreTestCNAME(mockDNSRecCNAMEUnderscore, mockDNSRecA1)

	handleUpdatesAndExpectChangesFor := func(domains ...string) {
		// For a set of changes we should get a single update ready notification.
		ExpectWithOffset(1, domainStore.UpdatesReadyChannel()).To(Receive())
		ExpectWithOffset(1, domainStore.UpdatesReadyChannel()).NotTo(Receive())
		log.Debug("Updates ready to handle")

		// Handle the updates - this synchronously invokes the OnDomainChange callbacks.
		domainStore.HandleUpdates()

		// We shouldn't care if _more_ domains are signaled than we expect.  Just check that
		// the expected ones _are_ signaled.
		for _, domain := range domains {
			ExpectWithOffset(1, handler.domainsChanged).To(ContainElement(domain),
				fmt.Sprintf("Expected domain %v to be signalled but it wasn't", domain))
		}
		handler.domainsChanged = nil
	}

	// Assert that the expected callbacks have been received. This resets the callback Ids.
	expectCallbacks := func(expectedCallbackIds ...string) {
		getCallbackIds := func() []string {
			callbackIdsMutex.Lock()
			defer callbackIdsMutex.Unlock()
			if len(callbackIds) == 0 {
				return nil
			}
			callbackIdsCopy := make([]string, len(callbackIds))
			copy(callbackIdsCopy, callbackIds)
			return callbackIdsCopy
		}
		if len(expectedCallbackIds) == 0 {
			ConsistentlyWithOffset(1, getCallbackIds).Should(HaveLen(0))
		} else {
			EventuallyWithOffset(1, getCallbackIds).Should(ConsistOf(expectedCallbackIds))
			ConsistentlyWithOffset(1, getCallbackIds).Should(ConsistOf(expectedCallbackIds))
		}

		callbackIdsMutex.Lock()
		defer callbackIdsMutex.Unlock()
		ExpectWithOffset(1, callbackIds).To(ConsistOf(expectedCallbackIds))
		callbackIds = nil
	}

	Context("with monitor thread", func() {

		var (
			expectedSeen      bool
			expectedDomainIPs []string
			monitorMutex      sync.Mutex
			killMonitor       chan struct{}
			monitorRunning    sync.WaitGroup
		)

		monitor := func(domain string) {
			defer monitorRunning.Done()
			for {
			loop:
				for {
					select {
					case <-killMonitor:
						return
					case <-domainStore.UpdatesReadyChannel():
						log.Debug("Updates ready to handle")
						domainStore.HandleUpdates()
						monitorMutex.Lock()
						for _, signalDomain := range handler.domainsChanged {
							if signalDomain == domain {
								expectedSeen = true
								expectedDomainIPs = domainStore.GetDomainIPs(domain)
								break
							}
						}
						monitorMutex.Unlock()
					default:
						break loop
					}
				}
			}
		}

		checkMonitor := func(expectedIPs []string) {
			Eventually(func() bool {
				monitorMutex.Lock()
				defer monitorMutex.Unlock()
				result := expectedSeen && reflect.DeepEqual(expectedDomainIPs, expectedIPs)
				if result {
					expectedSeen = false
				}
				return result
			}).Should(BeTrue())
		}

		BeforeEach(func() {
			expectedSeen = false
			killMonitor = make(chan struct{})
			domainStoreCreateEx(0, defaultConfig)
			monitorRunning.Add(1)
			go monitor("*.microsoft.com")
		})

		AfterEach(func() {
			close(killMonitor)
			monitorRunning.Wait()
		})

		It("microsoft case", func() {
			Expect(domainStore.GetDomainIPs("*.microsoft.com")).To(Equal([]string(nil)))
			programDNSAnswer(domainStore, testutils.MakeCNAME("www.microsoft.com", "www.microsoft.com-c-3.edgekey.net"))
			checkMonitor(nil)
			programDNSAnswer(domainStore, testutils.MakeCNAME("www.microsoft.com-c-3.edgekey.net", "www.microsoft.com-c-3.edgekey.net.globalredir.akadns.net"))
			programDNSAnswer(domainStore, testutils.MakeCNAME("www.microsoft.com-c-3.edgekey.net.globalredir.akadns.net", "e13678.dspb.akamaiedge.net"))
			programDNSAnswer(domainStore, testutils.MakeA("e13678.dspb.akamaiedge.net", "104.75.174.50"))
			checkMonitor([]string{"104.75.174.50"})
		})
	})

	// Test where:
	// - a1.com and a2.com are both CNAMEs for b.com
	// - b.com is a CNAME for c.com
	// - c.com resolves to an IP address
	// The ipsets manager is interested in both a1.com and a2.com.
	//
	// The key point is that when the IP address for c.com changes, the ipsets manager
	// should be notified that domain info has changed for both a1.com and a2.com.
	It("should handle a branched DNS graph", func() {
		domainStoreCreate()
		programDNSAnswer(domainStore, testutils.MakeCNAME("a1.com", "b.com"))
		programDNSAnswer(domainStore, testutils.MakeCNAME("a2.com", "b.com"))
		programDNSAnswer(domainStore, testutils.MakeCNAME("b.com", "c.com"))
		programDNSAnswer(domainStore, testutils.MakeA("c.com", "3.4.5.6"))
		handleUpdatesAndExpectChangesFor("a1.com", "a2.com", "b.com", "c.com")
		Expect(domainStore.GetDomainIPs("a1.com")).To(Equal([]string{"3.4.5.6"}))
		Expect(domainStore.GetDomainIPs("a2.com")).To(Equal([]string{"3.4.5.6"}))
		programDNSAnswer(domainStore, testutils.MakeA("c.com", "7.8.9.10"))
		handleUpdatesAndExpectChangesFor("a1.com", "a2.com", "c.com")
		Expect(domainStore.GetDomainIPs("a1.com")).To(ConsistOf("3.4.5.6", "7.8.9.10"))
		Expect(domainStore.GetDomainIPs("a2.com")).To(ConsistOf("3.4.5.6", "7.8.9.10"))
		domainStore.processMappingExpiry("c.com", "3.4.5.6")
		handleUpdatesAndExpectChangesFor("a1.com", "a2.com", "c.com")
		Expect(domainStore.GetDomainIPs("a1.com")).To(Equal([]string{"7.8.9.10"}))
		Expect(domainStore.GetDomainIPs("a2.com")).To(Equal([]string{"7.8.9.10"}))
		// No garbage yet, because c.com still has a value and is the RHS of other mappings.
		Expect(domainStore.collectGarbage()).To(Equal(0))
	})

	It("is not vulnerable to CNAME loops", func() {
		domainStoreCreate()
		programDNSAnswer(domainStore, testutils.MakeCNAME("a.com", "b.com"))
		programDNSAnswer(domainStore, testutils.MakeCNAME("b.com", "c.com"))
		programDNSAnswer(domainStore, testutils.MakeCNAME("c.com", "a.com"))
		Expect(domainStore.GetDomainIPs("a.com")).To(BeEmpty())
	})

	It("0.0.0.0 is ignored", func() {
		domainStoreCreate()
		// 0.0.0.0 should be ignored.
		programDNSAnswer(domainStore, testutils.MakeA("a.com", "0.0.0.0"))
		Expect(domainStore.GetDomainIPs("a.com")).To(BeEmpty())
		// But not any other IP.
		programDNSAnswer(domainStore, testutils.MakeA("a.com", "0.0.0.1"))
		Expect(domainStore.GetDomainIPs("a.com")).To(HaveLen(1))
	})

	DescribeTable("it should identify wildcards",
		func(domain string, expectedIsWildcard bool) {
			Expect(isWildcard(domain)).To(Equal(expectedIsWildcard))
		},
		Entry("*.com",
			"*.com", true),
		Entry(".com",
			".com", false),
		Entry("google.com",
			"google.com", false),
		Entry("*.google.com",
			"*.google.com", true),
		Entry("update.*.tigera.io",
			"update.*.tigera.io", true),
		Entry("cpanel.blog.org",
			"cpanel.blog.org", false),
	)

	DescribeTable("it should build correct wildcard regexps",
		func(wildcard, expectedRegexp string) {
			Expect(wildcardToRegexpString(wildcard)).To(Equal(expectedRegexp))
		},
		Entry("*.com",
			"*.com", "^.*\\.com$"),
		Entry("*.google.com",
			"*.google.com", "^.*\\.google\\.com$"),
		Entry("update.*.tigera.io",
			"update.*.tigera.io", "^update\\..*\\.tigera\\.io$"),
	)

	DescribeTable("wildcards match as expected",
		func(wildcard, name string, expectedMatch bool) {
			regex, err := regexp.Compile(wildcardToRegexpString(wildcard))
			Expect(err).NotTo(HaveOccurred())
			Expect(regex.MatchString(name)).To(Equal(expectedMatch))
		},
		Entry("*.com",
			"*.com", "google.com", true),
		Entry("*.com",
			"*.com", "www.google.com", true),
		Entry("*.com",
			"*.com", "com", false),
		Entry("*.com",
			"*.com", "tigera.io", false),
		Entry("*.google.com",
			"*.google.com", "www.google.com", true),
		Entry("*.google.com",
			"*.google.com", "ipv6.google.com", true),
		Entry("*.google.com",
			"*.google.com", "ipv6google.com", false),
		Entry("*.google.com",
			"*.google.com", "ipv6.experimental.google.com", true),
		Entry("update.*.tigera.io",
			"update.*.tigera.io", "update.calico.tigera.io", true),
		Entry("update.*.tigera.io",
			"update.*.tigera.io", "update.tsee.tigera.io", true),
		Entry("update.*.tigera.io",
			"update.*.tigera.io", "update.security.tsee.tigera.io", true),
		Entry("update.*.tigera.io",
			"update.*.tigera.io", "update.microsoft.com", false),
	)

	Context("wildcard handling", func() {
		BeforeEach(func() {
			domainStoreCreate()
		})

		// Test where wildcard is configured in the data model before we have any DNS
		// information that matches it.
		Context("with client interested in *.google.com", func() {
			BeforeEach(func() {
				Expect(domainStore.GetDomainIPs("*.google.com")).To(BeEmpty())
			})

			Context("with IP for update.google.com", func() {
				BeforeEach(func() {
					programDNSAnswer(domainStore, testutils.MakeA("update.google.com", "1.2.3.5"))
				})

				It("should update *.google.com", func() {
					handleUpdatesAndExpectChangesFor("*.google.com")
					Expect(domainStore.GetDomainIPs("*.google.com")).To(Equal([]string{"1.2.3.5"}))
				})
			})
		})

		// Test where wildcard is configured in the data model when we already have DNS
		// information that matches it.
		Context("with IP for update.google.com", func() {
			BeforeEach(func() {
				programDNSAnswer(domainStore, testutils.MakeA("update.google.com", "1.2.3.5"))
			})

			It("should get that IP for *.google.com", func() {
				Expect(domainStore.GetDomainIPs("*.google.com")).To(Equal([]string{"1.2.3.5"}))
			})

			It("should handle reverse lookup when no IP was requested", func() {
				ipb, _ := ip.ParseIPAs16Byte("1.2.3.5")
				Expect(domainStore.GetWatchedDomainForIP(ipb)).To(Equal(""))
			})

			It("should handle reverse lookup when IP was requested as update.google.com", func() {
				ipb, _ := ip.ParseIPAs16Byte("1.2.3.5")
				Expect(domainStore.GetDomainIPs("update.google.com")).To(Equal([]string{"1.2.3.5"}))
				Expect(domainStore.GetWatchedDomainForIP(ipb)).To(Equal("update.google.com"))
			})

			It("should handle reverse lookup when IP was requested as *.google.com", func() {
				ipb, _ := ip.ParseIPAs16Byte("1.2.3.5")
				Expect(domainStore.GetDomainIPs("*.google.com")).To(Equal([]string{"1.2.3.5"}))
				Expect(domainStore.GetWatchedDomainForIP(ipb)).To(Equal("*.google.com"))
			})

			It("should handle reverse lookup when IP was requested as update.google.com and *.google.com", func() {
				ipb, _ := ip.ParseIPAs16Byte("1.2.3.5")
				Expect(domainStore.GetDomainIPs("*.google.com")).To(Equal([]string{"1.2.3.5"}))
				Expect(domainStore.GetDomainIPs("update.google.com")).To(Equal([]string{"1.2.3.5"}))
				Expect(domainStore.GetWatchedDomainForIP(ipb)).To(BeElementOf("update.google.com", "*.google.com"))
			})
		})
	})

	Context("with 10s extra TTL", func() {
		BeforeEach(func() {
			domainStoreCreateEx(100, &DnsConfig{
				DNSExtraTTL:   10 * time.Second,
				DNSCacheEpoch: 1,
			})
			programDNSAnswer(domainStore, testutils.MakeA("update.google.com", "1.2.3.5"))
		})

		It("delivers the IP when queried", func() {
			Expect(domainStore.GetDomainIPs("update.google.com")).To(Equal([]string{"1.2.3.5"}))
		})

		It("created the mapping with 10s expiry", func() {
			Expect(lastTTL).To(Equal(10 * time.Second))
		})

		Context("with an epoch change", func() {
			BeforeEach(func() {
				domainStore.OnUpdate(&proto.ConfigUpdate{
					Config: map[string]string{"DNSCacheEpoch": "2"},
				})
				log.Info("Injected epoch change")
			})

			It("quickly removes the mapping", func() {
				// Note, Eventually by default allows up to 1 second.
				Eventually(func() []string {
					domainStore.loopIteration(nil, nil)
					return domainStore.GetDomainIPs("update.google.com")
				}).Should(BeEmpty())
			})
		})
	})

	Context("with dynamic config update for 1h extra TTL", func() {
		BeforeEach(func() {
			domainStoreCreate()
			domainStore.OnUpdate(&proto.ConfigUpdate{
				Config: map[string]string{"DNSExtraTTL": "3600"},
			})
			log.Info("Updated extra TTL to 1h")
			programDNSAnswer(domainStore, testutils.MakeA("update.google.com", "1.2.3.5"))
		})

		It("delivers the IP when queried", func() {
			Expect(domainStore.GetDomainIPs("update.google.com")).To(Equal([]string{"1.2.3.5"}))
		})

		It("created the mapping with 1h expiry", func() {
			Expect(lastTTL).To(Equal(1 * time.Hour))
		})
	})

	// Test callbacks are invoked after dataplane programming is completed.
	It("should handle DNS updates with callbacks", func() {
		domainStoreCreate()

		// Program answer.  Handle updates and expect domain updates. No callbacks should be invoked until updates
		// are applied.
		programDNSAnswer(domainStore, testutils.MakeCNAME("a1.com", "b.com"), "cb1")
		handleUpdatesAndExpectChangesFor("a1.com")
		expectCallbacks()
		domainStore.UpdatesApplied()
		expectCallbacks("cb1")

		programDNSAnswer(domainStore, testutils.MakeCNAME("a2.com", "b.com"), "cb2")
		handleUpdatesAndExpectChangesFor("a2.com")

		// We are already waiting for the dataplane updates for the previous two messages to be applied. In the meantime
		// send in more updates, the last is a repeat of the previous message.
		programDNSAnswer(domainStore, testutils.MakeCNAME("b.com", "c.com"), "cb3")
		programDNSAnswer(domainStore, testutils.MakeA("c.com", "3.4.5.6"), "cb4")
		programDNSAnswer(domainStore, testutils.MakeCNAME("a2.com", "b.com"), "cb5")

		// Apply the dataplane changes. We should get the callbacks for cb2 and cb5 once the changes are applied.
		expectCallbacks()
		domainStore.UpdatesApplied()
		expectCallbacks("cb2", "cb5")

		// Handle the remaining changes and apply the updates, we should get remaining callbacks invoked.
		handleUpdatesAndExpectChangesFor("b.com", "c.com")
		domainStore.UpdatesApplied()
		expectCallbacks("cb3", "cb4")
		handler.dataplaneSyncNeeded = true

		// Get IPs for domains a1.com and a2.com and then update c.com.
		Expect(domainStore.GetDomainIPs("a1.com")).To(Equal([]string{"3.4.5.6"}))
		Expect(domainStore.GetDomainIPs("a2.com")).To(Equal([]string{"3.4.5.6"}))

		// Have the handler indicate that no dataplane updates are required.  In this case the callbacks should happen
		// immediately without waiting for UpdatesApplied().
		handler.dataplaneSyncNeeded = false
		programDNSAnswer(domainStore, testutils.MakeA("c.com", "7.8.9.10"), "cb6")
		handleUpdatesAndExpectChangesFor("a1.com", "a2.com", "c.com")
		expectCallbacks("cb6")
		domainStore.UpdatesApplied()
		expectCallbacks()
		handler.dataplaneSyncNeeded = true

		// Repeat a message that is already programmed. We should get no further changes and the callback should happen
		// without any further dataplane involvement.
		programDNSAnswer(domainStore, testutils.MakeCNAME("a2.com", "b.com"), "cb7")
		Expect(domainStore.UpdatesReadyChannel()).ShouldNot(Receive())
		expectCallbacks("cb7")
	})

	It("should not panic because of an IPv4 packet with no transport header", func() {
		domainStoreCreate()

		pkt := gopacket.NewSerializeBuffer()
		err := gopacket.SerializeLayers(
			pkt,
			gopacket.SerializeOptions{ComputeChecksums: true},
			&layers.IPv4{
				Version:  4,
				IHL:      5,
				TTL:      64,
				Flags:    layers.IPv4DontFragment,
				SrcIP:    net.IPv4(172, 31, 11, 2),
				DstIP:    net.IPv4(172, 31, 21, 5),
				Protocol: layers.IPProtocolTCP,
				Length:   5 * 4,
			},
		)
		Expect(err).NotTo(HaveOccurred())

		domainStore.MsgChannel() <- DataWithTimestamp{
			Data: pkt.Bytes(),
		}
		saveTimerC := make(chan time.Time)
		gcTimerC := make(chan time.Time)
		Expect(func() {
			domainStore.loopIteration(saveTimerC, gcTimerC)
		}).NotTo(Panic())
	})
})
