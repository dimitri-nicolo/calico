// +build fvtests

// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package fv_test

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/fv/containers"
	"github.com/projectcalico/felix/fv/infrastructure"
	"github.com/projectcalico/felix/fv/utils"
	"github.com/projectcalico/felix/fv/workload"
	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/numorstring"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/set"
)

var dnsDir string

type mapping struct {
	lhs, rhs string
}

func mappingMatchesLine(m *mapping, line string) bool {
	return strings.Contains(line, "\""+m.lhs+"\"") && strings.Contains(line, "\""+m.rhs+"\"")
}

func fileHasMappingsAndNot(mappings []mapping, notMappings []mapping) func() bool {
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

func fileHasMappings(mappings []mapping) func() bool {
	return fileHasMappingsAndNot(mappings, nil)
}

func fileHasMapping(lname, rname string) func() bool {
	return fileHasMappings([]mapping{{lhs: lname, rhs: rname}})
}

var _ = Describe("DNS Policy", func() {

	var (
		scapyTrusted *containers.Container
		etcd         *containers.Container
		felix        *infrastructure.Felix
		client       client.Interface
		w            [1]*workload.Workload
	)

	BeforeEach(func() {
		opts := infrastructure.DefaultTopologyOptions()
		var err error
		dnsDir, err = ioutil.TempDir("", "dnsinfo")
		Expect(err).NotTo(HaveOccurred())

		// Start scapy first, so we can get its IP and configure Felix to trust it.
		scapyTrusted = containers.Run("scapy",
			containers.RunOpts{AutoRemove: true, WithStdinPipe: true},
			"-i", "--privileged", "tigera-test/scapy")

		// Now start etcd and Felix, with Felix trusting scapy's IP.
		opts.ExtraVolumes[dnsDir] = "/dnsinfo"
		opts.ExtraEnvVars["FELIX_DNSCACHEFILE"] = "/dnsinfo/dnsinfo.txt"
		opts.ExtraEnvVars["FELIX_DNSCACHESAVEINTERVAL"] = "1"
		opts.ExtraEnvVars["FELIX_DNSTRUSTEDSERVERS"] = scapyTrusted.IP
		opts.ExtraEnvVars["FELIX_PolicySyncPathPrefix"] = "/var/run/calico"
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
			Eventually(check, "10s", "2s").Should(BeTrue())
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
				"DNSRR(rrname='microsoft.com',type='A',ttl=24,rdata='10.10.10.16')" +
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
				{lhs: "microsoft.com", rhs: "10.10.10.16"},
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
			}), "10s", "2s").Should(BeTrue())
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
				"-i", "--privileged", "tigera-test/scapy")
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
			}), "10s", "2s").Should(BeTrue())
		})
	})

	workloadCanPingEtcd := func() error {
		out, err := w[0].ExecOutput("ping", "-c", "1", "-W", "1", etcd.IP)
		log.WithError(err).Infof("ping said:\n%v", out)
		if err != nil {
			log.Infof("stderr was:\n%v", string(err.(*exec.ExitError).Stderr))
		}
		return err
	}

	Context("with policy in place first, then connection attempted", func() {
		BeforeEach(func() {
			policy := api.NewGlobalNetworkPolicy()
			policy.Name = "default-deny-egress"
			policy.Spec.Selector = "all()"
			udp := numorstring.ProtocolFromString(numorstring.ProtocolUDP)
			policy.Spec.Egress = []api.Rule{
				{
					Action:   api.Allow,
					Protocol: &udp,
					Destination: api.EntityRule{
						Ports: []numorstring.Port{numorstring.SinglePort(53)},
					},
				},
				{
					Action: api.Deny,
				},
			}
			_, err := client.GlobalNetworkPolicies().Create(utils.Ctx, policy, utils.NoOptions)
			Expect(err).NotTo(HaveOccurred())

			policy = api.NewGlobalNetworkPolicy()
			policy.Name = "allow-xyz"
			order := float64(20)
			policy.Spec.Order = &order
			policy.Spec.Selector = "all()"
			policy.Spec.Egress = []api.Rule{
				{
					Action:      api.Allow,
					Destination: api.EntityRule{Domains: []string{"xyz.com"}},
				},
			}
			_, err = client.GlobalNetworkPolicies().Create(utils.Ctx, policy, utils.NoOptions)
			Expect(err).NotTo(HaveOccurred())

			// Allow 2s for Felix to see and process that policy.
			time.Sleep(2 * time.Second)

			// We use the etcd container as a target IP for the workload to ping, so
			// arrange for it to route back to the workload.
			etcd.Exec("ip", "r", "add", w[0].IP, "via", felix.IP)

			// Create a chain of DNS info that maps xyz.com to that IP.
			dnsServerSetup(scapyTrusted)
			sendDNSResponses(scapyTrusted, []string{
				"DNS(qr=1,qdcount=1,ancount=1,qd=DNSQR(qname='xyz.com',qtype='CNAME'),an=(DNSRR(rrname='xyz.com',type='CNAME',ttl=60,rdata='bob.xyz.com')))",
				"DNS(qr=1,qdcount=1,ancount=1,qd=DNSQR(qname='bob.xyz.com',qtype='CNAME'),an=(DNSRR(rrname='bob.xyz.com',type='CNAME',ttl=10,rdata='server-5.xyz.com')))",
				"DNS(qr=1,qdcount=1,ancount=1,qd=DNSQR(qname='server-5.xyz.com',qtype='A'),an=(DNSRR(rrname='server-5.xyz.com',type='A',ttl=60,rdata='" + etcd.IP + "')))",
			})
			scapyTrusted.Stdin.Close()
		})

		It("workload can ping etcd", func() {
			// Allow 4 seconds for Felix to see the DNS responses and update ipsets.
			time.Sleep(4 * time.Second)
			// Ping should now go through.
			Expect(workloadCanPingEtcd()).NotTo(HaveOccurred())
		})
	})

	Context("with a chain of DNS info for xyz.com", func() {
		BeforeEach(func() {
			// We use the etcd container as a target IP for the workload to ping, so
			// arrange for it to route back to the workload.
			etcd.Exec("ip", "r", "add", w[0].IP, "via", felix.IP)

			// Create a chain of DNS info that maps xyz.com to that IP.
			dnsServerSetup(scapyTrusted)
			sendDNSResponses(scapyTrusted, []string{
				"DNS(qr=1,qdcount=1,ancount=1,qd=DNSQR(qname='xyz.com',qtype='CNAME'),an=(DNSRR(rrname='xyz.com',type='CNAME',ttl=60,rdata='bob.xyz.com')))",
				"DNS(qr=1,qdcount=1,ancount=1,qd=DNSQR(qname='bob.xyz.com',qtype='CNAME'),an=(DNSRR(rrname='bob.xyz.com',type='CNAME',ttl=10,rdata='server-5.xyz.com')))",
				"DNS(qr=1,qdcount=1,ancount=1,qd=DNSQR(qname='server-5.xyz.com',qtype='A'),an=(DNSRR(rrname='server-5.xyz.com',type='A',ttl=60,rdata='" + etcd.IP + "')))",
			})
			scapyTrusted.Stdin.Close()
		})

		It("workload can ping etcd, because there's no policy", func() {
			Expect(workloadCanPingEtcd()).NotTo(HaveOccurred())
		})

		Context("with default-deny egress policy", func() {
			BeforeEach(func() {
				policy := api.NewGlobalNetworkPolicy()
				policy.Name = "default-deny-egress"
				policy.Spec.Selector = "all()"
				policy.Spec.Egress = []api.Rule{{
					Action: api.Deny,
				}}
				_, err := client.GlobalNetworkPolicies().Create(utils.Ctx, policy, utils.NoOptions)
				Expect(err).NotTo(HaveOccurred())
			})

			It("workload cannot ping etcd", func() {
				Eventually(workloadCanPingEtcd, "10s", "2s").Should(HaveOccurred())
			})

			Context("with domain-allow egress policy", func() {
				BeforeEach(func() {
					policy := api.NewGlobalNetworkPolicy()
					policy.Name = "allow-xyz"
					order := float64(20)
					policy.Spec.Order = &order
					policy.Spec.Selector = "all()"
					policy.Spec.Egress = []api.Rule{
						{
							Action:      api.Allow,
							Destination: api.EntityRule{Domains: []string{"xyz.com"}},
						},
					}
					_, err := client.GlobalNetworkPolicies().Create(utils.Ctx, policy, utils.NoOptions)
					Expect(err).NotTo(HaveOccurred())
				})

				It("workload can ping etcd", func() {
					Eventually(workloadCanPingEtcd, "5s", "1s").ShouldNot(HaveOccurred())
				})

				Context("with 11s sleep so that DNS info expires", func() {
					BeforeEach(func() {
						time.Sleep(11 * time.Second)
					})

					It("workload cannot ping etcd", func() {
						Eventually(workloadCanPingEtcd, "5s", "1s").Should(HaveOccurred())
					})
				})

				Context("with a Felix restart", func() {
					BeforeEach(func() {
						felix.Restart()
						// Allow a bit of time for Felix to re-read the
						// persistent file and update the dataplane, but not
						// long enough (8s) for the DNS info to expire.
						time.Sleep(3 * time.Second)
					})

					It("workload can still ping etcd", func() {
						Eventually(workloadCanPingEtcd, "5s", "1s").ShouldNot(HaveOccurred())
					})
				})
			})

			Context("with networkset with allowed egress domains", func() {
				BeforeEach(func() {
					gns := api.NewGlobalNetworkSet()
					gns.Name = "allow-xyz"
					gns.Labels = map[string]string{"thingy": "xyz"}
					gns.Spec.AllowedEgressDomains = []string{"xyz.com"}
					_, err := client.GlobalNetworkSets().Create(utils.Ctx, gns, utils.NoOptions)
					Expect(err).NotTo(HaveOccurred())

					policy := api.NewGlobalNetworkPolicy()
					policy.Name = "allow-xyz"
					order := float64(20)
					policy.Spec.Order = &order
					policy.Spec.Selector = "all()"
					policy.Spec.Egress = []api.Rule{
						{
							Action:      api.Allow,
							Destination: api.EntityRule{Selector: "thingy == 'xyz'"},
						},
					}
					_, err = client.GlobalNetworkPolicies().Create(utils.Ctx, policy, utils.NoOptions)
					Expect(err).NotTo(HaveOccurred())
				})

				Context("with a Felix restart", func() {
					BeforeEach(func() {
						felix.Restart()
						// Allow a bit of time for Felix to re-read the
						// persistent file and update the dataplane, but not
						// long enough (8s) for the DNS info to expire.
						time.Sleep(3 * time.Second)
					})

					It("workload can still ping etcd", func() {
						Eventually(workloadCanPingEtcd, "5s", "1s").ShouldNot(HaveOccurred())
					})

					Context("with 10s sleep so that DNS info expires", func() {
						BeforeEach(func() {
							time.Sleep(10 * time.Second)
						})

						It("workload cannot ping etcd", func() {
							Eventually(workloadCanPingEtcd, "5s", "1s").Should(HaveOccurred())
						})
					})
				})
			})
		})
	})
})

var _ = Describe("DNS Policy with server on host", func() {

	var (
		scapyTrusted *containers.Container
		etcd         *containers.Container
		felix        *infrastructure.Felix
		client       client.Interface
		w            [1]*workload.Workload
	)

	BeforeEach(func() {
		opts := infrastructure.DefaultTopologyOptions()
		var err error
		dnsDir, err = ioutil.TempDir("", "dnsinfo")
		Expect(err).NotTo(HaveOccurred())

		// Start etcd and Felix, with no trusted DNS server IPs yet.
		opts.ExtraVolumes[dnsDir] = "/dnsinfo"
		opts.ExtraEnvVars["FELIX_DNSCACHEFILE"] = "/dnsinfo/dnsinfo.txt"
		opts.ExtraEnvVars["FELIX_DNSCACHESAVEINTERVAL"] = "1"
		opts.ExtraEnvVars["FELIX_PolicySyncPathPrefix"] = "/var/run/calico"
		felix, etcd, client = infrastructure.StartSingleNodeEtcdTopology(opts)
		infrastructure.CreateDefaultProfile(client, "default", map[string]string{"default": ""}, "")

		// Create a workload, using that profile.
		for ii := range w {
			iiStr := strconv.Itoa(ii)
			w[ii] = workload.Run(felix, "w"+iiStr, "default", "10.65.0.1"+iiStr, "8055", "tcp")
			w[ii].Configure(client)
		}

		// Start scapy, in the same namespace as Felix.
		scapyTrusted = containers.Run("scapy",
			containers.RunOpts{AutoRemove: true, WithStdinPipe: true, SameNamespace: felix.Container},
			"-i", "--privileged", "tigera-test/scapy")

		// Configure Felix to trust its own IP as a DNS server.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		c := api.NewFelixConfiguration()
		c.Name = "default"
		c.Spec.DNSTrustedServers = &[]string{felix.IP}
		_, err = client.FelixConfigurations().Create(ctx, c, options.SetOptions{})
		Expect(err).NotTo(HaveOccurred())

		// Allow time for Felix to restart before we send the DNS response from scapy.
		time.Sleep(3 * time.Second)
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

	dnsServerSetup := func(scapy *containers.Container) {
		// Establish conntrack state, in Felix, as though the workload just sent a DNS
		// request to the specified scapy.
		felix.Exec("conntrack", "-I", "-s", w[0].IP, "-d", felix.IP, "-p", "UDP", "-t", "10", "--sport", "53", "--dport", "53")
	}

	sendDNSResponses := func(scapy *containers.Container, dnsSpecs []string) {
		// Drive scapy.
		for _, dnsSpec := range dnsSpecs {
			// Because we're sending from scapy in the same network namespace as Felix,
			// we need to use normal Linux sending instead of scapy's send function, as
			// the latter bypasses iptables.  We just use scapy to build the DNS
			// payload.
			io.WriteString(scapy.Stdin,
				fmt.Sprintf("dns = %v\n", dnsSpec))
			io.WriteString(scapy.Stdin, "import socket\n")
			io.WriteString(scapy.Stdin, "sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)\n")
			io.WriteString(scapy.Stdin,
				fmt.Sprintf("sock.bind(('%v', 53))\n", felix.IP))
			io.WriteString(scapy.Stdin,
				fmt.Sprintf("sock.sendto(dns.__bytes__(), ('%v', 53))\n", w[0].IP))
		}
	}

	DescribeTable("DNS response processing",
		func(dnsSpecs []string, check func() bool) {
			dnsServerSetup(scapyTrusted)
			sendDNSResponses(scapyTrusted, dnsSpecs)
			scapyTrusted.Stdin.Close()
			Eventually(check, "10s", "2s").Should(BeTrue())
		},

		Entry("A record", []string{
			"DNS(qr=1,qdcount=1,ancount=1,qd=DNSQR(qname='bankofsteve.com',qtype='A'),an=(DNSRR(rrname='bankofsteve.com',type='A',ttl=36000,rdata='192.168.56.1')))",
		},
			func() bool {
				return fileHasMapping("bankofsteve.com", "192.168.56.1")()
			},
		),
	)
})
