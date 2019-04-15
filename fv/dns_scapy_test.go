// +build fvtests

// Copyright (c) 2019 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fv_test

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/fv/containers"
	"github.com/projectcalico/felix/fv/infrastructure"
	"github.com/projectcalico/felix/fv/workload"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/set"
)

type mapping struct {
	lhs, rhs string
}

var _ = Describe("DNS Policy", func() {

	var (
		scapyTrusted *containers.Container
		etcd         *containers.Container
		felix        *infrastructure.Felix
		client       client.Interface
		w            [1]*workload.Workload
		dnsDir       string
	)

	BeforeEach(func() {
		opts := infrastructure.DefaultTopologyOptions()
		var err error
		dnsDir, err = ioutil.TempDir("", "dnsinfo")
		Expect(err).NotTo(HaveOccurred())

		// Start scapy first, so we can get its IP and configure Felix to trust it.
		scapyTrusted = containers.Run("scapy",
			containers.RunOpts{AutoRemove: true, WithStdinPipe: true},
			"-i", "--privileged", "scapy")
		scapyTrusted.WaitUntilRunning()

		// Now start etcd and Felix, with Felix trusting scapy's IP.
		opts.ExtraVolumes[dnsDir] = "/dnsinfo"
		opts.ExtraEnvVars["FELIX_DOMAININFOSTORE"] = "/dnsinfo/dnsinfo.txt"
		opts.ExtraEnvVars["FELIX_DOMAININFOSAVEINTERVAL"] = "1"
		opts.ExtraEnvVars["FELIX_DOMAININFOTRUSTEDSERVERS"] = scapyTrusted.IP
		felix, etcd, client = infrastructure.StartSingleNodeEtcdTopology(opts)
		infrastructure.CreateDefaultProfile(client, "default", map[string]string{"default": ""}, "")

		// Create a workload, using that profile.
		for ii := range w {
			iiStr := strconv.Itoa(ii)
			w[ii] = workload.Run(felix, "w"+iiStr, "default", "10.65.0.1"+iiStr, "8055", "tcp")
			w[ii].Configure(client)
		}
	})

	// Stop etcd and workloads, collecting some state if anything failed.
	AfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			felix.Exec("ipset", "list")
			felix.Exec("iptables-save", "-c")
			felix.Exec("ip", "r")
			felix.Exec("conntrack", "-L")
		}

		for ii := range w {
			w[ii].Stop()
		}
		felix.Stop()

		if CurrentGinkgoTestDescription().Failed {
			etcd.Exec("etcdctl", "ls", "--recursive", "/")
		}
		etcd.Stop()
	})

	mappingMatchesLine := func(m *mapping, line string) bool {
		return strings.Contains(line, "\""+m.lhs+"\"") && strings.Contains(line, "\""+m.rhs+"\"")
	}

	fileHasMappingsAndNot := func(mappings []mapping, notMappings []mapping) func() bool {
		mset := set.FromArray(mappings)
		notset := set.FromArray(notMappings)
		return func() bool {
			f, err := os.Open(path.Join(dnsDir, "dnsinfo.txt"))
			if err == nil {
				scanner := bufio.NewScanner(f)
				for scanner.Scan() {
					line := scanner.Text()
					mset.Iter(func(item interface{}) error {
						m := item.(mapping)
						if mappingMatchesLine(&m, line) {
							return set.RemoveItem
						}
						return nil
					})
					foundWrongMapping := false
					notset.Iter(func(item interface{}) error {
						m := item.(mapping)
						if mappingMatchesLine(&m, line) {
							log.Infof("Found wrong mapping: %v", m)
							foundWrongMapping = true
						}
						return nil
					})
					if foundWrongMapping {
						return false
					}
				}
				if mset.Len() == 0 {
					log.Info("All expected mappings found")
					return true
				} else {
					log.Infof("Missing %v expected mappings", mset.Len())
					mset.Iter(func(item interface{}) error {
						m := item.(mapping)
						log.Infof("Missed mapping: %v", m)
						return nil
					})
				}
			}
			log.Info("Returning false by default")
			return false
		}
	}

	fileHasMappings := func(mappings []mapping) func() bool {
		return fileHasMappingsAndNot(mappings, nil)
	}

	fileHasMapping := func(lname, rname string) func() bool {
		return fileHasMappings([]mapping{{lhs: lname, rhs: rname}})
	}

	dnsServerSetup := func(scapy *containers.Container) {
		// Establish conntrack state, in Felix, as though the workload just sent a DNS
		// request to the specified scapy.
		felix.Exec("conntrack", "-I", "-s", w[0].IP, "-d", scapy.IP, "-p", "UDP", "-t", "10", "--sport", "53", "--dport", "53")

		// Allow scapy to route back to the workload.
		io.WriteString(scapy.Stdin,
			fmt.Sprintf("conf.route.add(host='%v',gw='%v')\n", w[0].IP, felix.IP))
	}

	sendDNSResponses := func(scapy *containers.Container, dnsSpecs []string) {
		// Drive scapy.
		for _, dnsSpec := range dnsSpecs {
			io.WriteString(scapy.Stdin,
				fmt.Sprintf("send(IP(dst='%v')/UDP(sport=53)/%v)\n", w[0].IP, dnsSpec))
		}
	}

	DescribeTable("DNS response processing",
		func(dnsSpecs []string, check func() bool) {
			dnsServerSetup(scapyTrusted)
			sendDNSResponses(scapyTrusted, dnsSpecs)
			scapyTrusted.Stdin.Close()
			Eventually(check, "5s", "1s").Should(BeTrue())
		},

		Entry("A record", []string{
			"DNS(qr=1,qdcount=1,ancount=1,qd=DNSQR(qname='bankofsteve.com',qtype='A'),an=(DNSRR(rrname='bankofsteve.com',type='A',ttl=36000,rdata='192.168.56.1')))",
		},
			fileHasMapping("bankofsteve.com", "192.168.56.1"),
		),
		Entry("AAAA record", []string{
			"DNS(qr=1,qdcount=1,ancount=1,qd=DNSQR(qname='bankofsteve.com',qtype='AAAA'),an=(DNSRR(rrname='bankofsteve.com',type='AAAA',ttl=36000,rdata='fdf5:8944::3')))",
		},
			fileHasMapping("bankofsteve.com", "fdf5:8944::3"),
		),
		Entry("CNAME record", []string{
			"DNS(qr=1,qdcount=1,ancount=1,qd=DNSQR(qname='bankofsteve.com',qtype='CNAME'),an=(DNSRR(rrname='bankofsteve.com',type='CNAME',ttl=36000,rdata='my.home.server')))",
		},
			fileHasMapping("bankofsteve.com", "my.home.server"),
		),
		Entry("3 A records", []string{
			"DNS(qr=1,qdcount=1,ancount=3,qd=DNSQR(qname='microsoft.com',qtype='A'),an=(" +
				"DNSRR(rrname='microsoft.com',type='A',ttl=24,rdata='19.16.5.102')/" +
				"DNSRR(rrname='microsoft.com',type='A',ttl=36,rdata='10.146.25.132')/" +
				"DNSRR(rrname='microsoft.com',type='A',ttl=48,rdata='35.5.5.199')" +
				"))",
		},
			fileHasMappings([]mapping{
				{lhs: "microsoft.com", rhs: "19.16.5.102"},
				{lhs: "microsoft.com", rhs: "10.146.25.132"},
				{lhs: "microsoft.com", rhs: "35.5.5.199"},
			}),
		),
		Entry("as many A records as can fit in 512 bytes", []string{
			// 19 answers => 590 bytes of UDP payload
			// 17 answers => 532 bytes of UDP payload
			// 16 answers => 503 bytes of UDP payload
			"DNS(qr=1,qdcount=1,ancount=16,qd=DNSQR(qname='microsoft.com',qtype='A'),an=(" +
				"DNSRR(rrname='microsoft.com',type='A',ttl=24,rdata='10.10.10.1')/" +
				"DNSRR(rrname='microsoft.com',type='A',ttl=24,rdata='10.10.10.2')/" +
				"DNSRR(rrname='microsoft.com',type='A',ttl=24,rdata='10.10.10.3')/" +
				"DNSRR(rrname='microsoft.com',type='A',ttl=24,rdata='10.10.10.4')/" +
				"DNSRR(rrname='microsoft.com',type='A',ttl=24,rdata='10.10.10.5')/" +
				"DNSRR(rrname='microsoft.com',type='A',ttl=24,rdata='10.10.10.6')/" +
				"DNSRR(rrname='microsoft.com',type='A',ttl=24,rdata='10.10.10.7')/" +
				"DNSRR(rrname='microsoft.com',type='A',ttl=24,rdata='10.10.10.8')/" +
				"DNSRR(rrname='microsoft.com',type='A',ttl=24,rdata='10.10.10.9')/" +
				"DNSRR(rrname='microsoft.com',type='A',ttl=24,rdata='10.10.10.10')/" +
				"DNSRR(rrname='microsoft.com',type='A',ttl=24,rdata='10.10.10.11')/" +
				"DNSRR(rrname='microsoft.com',type='A',ttl=24,rdata='10.10.10.12')/" +
				"DNSRR(rrname='microsoft.com',type='A',ttl=24,rdata='10.10.10.13')/" +
				"DNSRR(rrname='microsoft.com',type='A',ttl=24,rdata='10.10.10.14')/" +
				"DNSRR(rrname='microsoft.com',type='A',ttl=24,rdata='10.10.10.15')/" +
				//"DNSRR(rrname='microsoft.com',type='A',ttl=24,rdata='10.10.10.16')/" +
				//"DNSRR(rrname='microsoft.com',type='A',ttl=24,rdata='10.10.10.17')/" +
				//"DNSRR(rrname='microsoft.com',type='A',ttl=24,rdata='10.10.10.18')/" +
				"DNSRR(rrname='microsoft.com',type='A',ttl=24,rdata='10.10.10.19')" +
				"))",
		},
			fileHasMappings([]mapping{
				{lhs: "microsoft.com", rhs: "10.10.10.1"},
				{lhs: "microsoft.com", rhs: "10.10.10.2"},
				{lhs: "microsoft.com", rhs: "10.10.10.3"},
				{lhs: "microsoft.com", rhs: "10.10.10.4"},
				{lhs: "microsoft.com", rhs: "10.10.10.5"},
				{lhs: "microsoft.com", rhs: "10.10.10.6"},
				{lhs: "microsoft.com", rhs: "10.10.10.7"},
				{lhs: "microsoft.com", rhs: "10.10.10.8"},
				{lhs: "microsoft.com", rhs: "10.10.10.9"},
				{lhs: "microsoft.com", rhs: "10.10.10.10"},
				{lhs: "microsoft.com", rhs: "10.10.10.11"},
				{lhs: "microsoft.com", rhs: "10.10.10.12"},
				{lhs: "microsoft.com", rhs: "10.10.10.13"},
				{lhs: "microsoft.com", rhs: "10.10.10.14"},
				{lhs: "microsoft.com", rhs: "10.10.10.15"},
				//{lhs: "microsoft.com", rhs: "10.10.10.16"},
				//{lhs: "microsoft.com", rhs: "10.10.10.17"},
				//{lhs: "microsoft.com", rhs: "10.10.10.18"},
				{lhs: "microsoft.com", rhs: "10.10.10.19"},
			}),
		),
	)

	DescribeTable("Benign DNS responses",
		// Various responses that we don't expect Felix to extract any information from, but
		// that should not cause any problem.
		func(dnsSpec string) {
			dnsServerSetup(scapyTrusted)
			sendDNSResponses(scapyTrusted, []string{
				"DNS(qr=1,qdcount=1,ancount=1,qd=DNSQR(qname='bankofsteve.com',qtype='A'),an=(DNSRR(rrname='bankofsteve.com',type='A',ttl=36000,rdata='192.168.56.1')))",
				dnsSpec,
				"DNS(qr=1,qdcount=1,ancount=1,qd=DNSQR(qname='fidget.com',qtype='A'),an=(DNSRR(rrname='fidget.com',type='A',ttl=36000,rdata='2.3.4.5')))",
			})
			scapyTrusted.Stdin.Close()
			Eventually(fileHasMappings([]mapping{
				{lhs: "bankofsteve.com", rhs: "192.168.56.1"},
				{lhs: "fidget.com", rhs: "2.3.4.5"},
			}), "5s", "1s").Should(BeTrue())
		},
		Entry("MX",
			"DNS(qr=1,qdcount=1,ancount=1,qd=DNSQR(qname='bankofsteve.com',qtype='MX'),an=(DNSRR(rrname='bankofsteve.com',type='MX',ttl=36000,rdata='mail.bankofsteve.com')))",
		),
		Entry("TXT",
			"DNS(qr=1,qdcount=1,ancount=1,qd=DNSQR(qname='bankofsteve.com',qtype='TXT'),an=(DNSRR(rrname='bankofsteve.com',type='TXT',ttl=36000,rdata='v=spf1 ~all')))",
		),
		Entry("SRV",
			"DNS(qr=1,qdcount=1,ancount=1,qd=DNSQR(qname='_sip._tcp.bankofsteve.com',qtype='SRV'),an=(DNSRR(rrname='_sip._tcp.bankofsteve.com',type='SRV',ttl=36000,rdata='sipserver.bankofsteve.com')))",
		),
		Entry("PTR",
			"DNS(qr=1,qdcount=1,ancount=1,qd=DNSQR(qname='20',qtype='PTR'),an=(DNSRR(rrname='20',type='PTR',ttl=36000,rdata='sipserver.bankofsteve.com')))",
		),
		Entry("SOA",
			"DNS(qr=1,qdcount=1,ancount=1,qd=DNSQR(qname='dnsimple.com',qtype='SOA'),an=(DNSRR(rrname='dnsimple.com',type='SOA',ttl=36000,rdata='ns1.dnsimple.com admin.dnsimple.com 2013022001 86400 7200 604800 300')))",
		),
		Entry("ALIAS",
			"DNS(qr=1,qdcount=1,ancount=1,qd=DNSQR(qname='bankofsteve.com',qtype='ALIAS'),an=(DNSRR(rrname='bankofsteve.com',type='ALIAS',ttl=36000,rdata='example.server')))",
		),
		Entry("Class CH",
			"DNS(qr=1,qdcount=1,ancount=1,qd=DNSQR(qname='microsoft.com',qclass='CH',qtype='A'),an=(DNSRR(rrname='bankofsteve.com',rclass='CH',type='A',ttl=36000,rdata='10.10.10.10')))",
		),
		Entry("Class HS",
			"DNS(qr=1,qdcount=1,ancount=1,qd=DNSQR(qname='microsoft.com',qclass='HS',qtype='A'),an=(DNSRR(rrname='bankofsteve.com',rclass='HS',type='A',ttl=36000,rdata='10.10.10.10')))",
		),
		Entry("NXDOMAIN",
			"DNS(qr=1,qdcount=1,rcode=3,qd=DNSQR(qname='microsoft.com',qtype='A'))",
		),
		Entry("response that claims to have 3 answers but doesn't",
			"DNS(qr=1,qdcount=1,ancount=3,qd=DNSQR(qname='microsoft.com',qtype='A'))",
		),
	)

	Context("with an untrusted DNS server", func() {
		var scapyUntrusted *containers.Container

		BeforeEach(func() {
			// Start another scapy.  This one's IP won't be trusted by Felix.
			scapyUntrusted = containers.Run("scapy",
				containers.RunOpts{AutoRemove: true, WithStdinPipe: true},
				"-i", "--privileged", "scapy")
		})

		It("s DNS information should be ignored", func() {
			dnsServerSetup(scapyTrusted)
			dnsServerSetup(scapyUntrusted)
			sendDNSResponses(scapyTrusted, []string{
				"DNS(qr=1,qdcount=1,ancount=1,qd=DNSQR(qname='alice.com',qtype='A'),an=(DNSRR(rrname='alice.com',type='A',ttl=36000,rdata='10.10.10.1')))",
			})
			sendDNSResponses(scapyUntrusted, []string{
				"DNS(qr=1,qdcount=1,ancount=1,qd=DNSQR(qname='alice.com',qtype='A'),an=(DNSRR(rrname='alice.com',type='A',ttl=36000,rdata='10.10.10.2')))",
			})
			sendDNSResponses(scapyTrusted, []string{
				"DNS(qr=1,qdcount=1,ancount=1,qd=DNSQR(qname='alice.com',qtype='A'),an=(DNSRR(rrname='alice.com',type='A',ttl=36000,rdata='10.10.10.3')))",
			})
			sendDNSResponses(scapyUntrusted, []string{
				"DNS(qr=1,qdcount=1,ancount=1,qd=DNSQR(qname='alice.com',qtype='A'),an=(DNSRR(rrname='alice.com',type='A',ttl=36000,rdata='10.10.10.4')))",
			})
			sendDNSResponses(scapyTrusted, []string{
				"DNS(qr=1,qdcount=1,ancount=1,qd=DNSQR(qname='alice.com',qtype='A'),an=(DNSRR(rrname='alice.com',type='A',ttl=36000,rdata='10.10.10.5')))",
			})
			sendDNSResponses(scapyUntrusted, []string{
				"DNS(qr=1,qdcount=1,ancount=1,qd=DNSQR(qname='alice.com',qtype='A'),an=(DNSRR(rrname='alice.com',type='A',ttl=36000,rdata='10.10.10.6')))",
			})
			sendDNSResponses(scapyTrusted, []string{
				"DNS(qr=1,qdcount=1,ancount=1,qd=DNSQR(qname='alice.com',qtype='A'),an=(DNSRR(rrname='alice.com',type='A',ttl=36000,rdata='10.10.10.7')))",
			})
			scapyUntrusted.Stdin.Close()
			scapyTrusted.Stdin.Close()
			Eventually(fileHasMappingsAndNot([]mapping{
				{lhs: "alice.com", rhs: "10.10.10.1"},
				{lhs: "alice.com", rhs: "10.10.10.3"},
				{lhs: "alice.com", rhs: "10.10.10.5"},
				{lhs: "alice.com", rhs: "10.10.10.7"},
			}, []mapping{
				{lhs: "alice.com", rhs: "10.10.10.2"},
				{lhs: "alice.com", rhs: "10.10.10.4"},
				{lhs: "alice.com", rhs: "10.10.10.6"},
			}), "5s", "1s").Should(BeTrue())
		})
	})
})
