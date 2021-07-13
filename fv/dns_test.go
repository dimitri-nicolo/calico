// +build fvtests

// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.

package fv_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/felix/fv/containers"
	"github.com/projectcalico/felix/fv/infrastructure"
	"github.com/projectcalico/felix/fv/utils"
	"github.com/projectcalico/felix/fv/workload"
	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/numorstring"
	"github.com/projectcalico/libcalico-go/lib/options"
)

const nameserverPrefix = "nameserver "

var localNameservers []string

func GetLocalNameservers() (nameservers []string) {
	if localNameservers == nil {
		// Find out what Docker puts in a container's /etc/resolv.conf.
		resolvConf, err := utils.GetCommandOutput("docker", "run", "--rm", utils.Config.FelixImage, "cat", "/etc/resolv.conf")
		Expect(err).NotTo(HaveOccurred())
		for _, resolvConfLine := range strings.Split(resolvConf, "\n") {
			if strings.HasPrefix(resolvConfLine, nameserverPrefix) {
				localNameservers = append(localNameservers, strings.TrimSpace(resolvConfLine[len(nameserverPrefix):]))
			}
		}
		log.Infof("Discovered nameservers: %v", localNameservers)
	}
	return localNameservers
}

func getDNSLogs(logFile string) ([]string, error) {
	fileExists, err := BeARegularFile().Match(logFile)
	if err != nil {
		return nil, err
	}
	if !fileExists {
		return nil, fmt.Errorf("Expected DNS log file %v does not exist", logFile)
	}
	logBytes, err := ioutil.ReadFile(logFile)
	if err != nil {
		return nil, err
	}
	var logs []string
	for _, log := range strings.Split(string(logBytes), "\n") {
		// Filter out empty strings returned by strings.Split.
		if log != "" {
			logs = append(logs, log)
		}
	}
	return logs, nil
}

var _ = Describe("_BPF-SAFE_ DNS Policy", func() {

	var (
		etcd   *containers.Container
		felix  *infrastructure.Felix
		client client.Interface
		infra  infrastructure.DatastoreInfra
		w      [1]*workload.Workload
		dnsDir string

		// Path to the save file from the point of view inside the Felix container.
		// (Whereas dnsDir is the directory outside the container.)
		saveFile                       string
		saveFileMappedOutsideContainer bool

		enableLogs    bool
		enableLatency bool
	)

	BeforeEach(func() {
		saveFile = "/dnsinfo/dnsinfo.txt"
		saveFileMappedOutsideContainer = true
		enableLogs = true
		enableLatency = true
	})

	logAndReport := func(out string, err error) error {
		log.WithError(err).Infof("test-dns said:\n%v", out)
		return err
	}

	wgetMicrosoftErr := func() error {
		w[0].C.EnsureBinary("test-dns")
		out, err := w[0].ExecCombinedOutput("/test-dns", "-", "microsoft.com")
		return logAndReport(out, err)
	}

	canWgetMicrosoft := func() {
		Eventually(wgetMicrosoftErr, "10s", "1s").ShouldNot(HaveOccurred())
		Consistently(wgetMicrosoftErr, "4s", "1s").ShouldNot(HaveOccurred())
	}

	cannotWgetMicrosoft := func() {
		Eventually(wgetMicrosoftErr, "10s", "1s").Should(HaveOccurred())
		Consistently(wgetMicrosoftErr, "4s", "1s").Should(HaveOccurred())
	}

	hostWgetMicrosoftErr := func() error {
		felix.EnsureBinary("test-dns")
		out, err := felix.ExecCombinedOutput("/test-dns", "-", "microsoft.com")
		return logAndReport(out, err)
	}

	hostCanWgetMicrosoft := func() {
		Eventually(hostWgetMicrosoftErr, "10s", "1s").ShouldNot(HaveOccurred())
		Consistently(hostWgetMicrosoftErr, "4s", "1s").ShouldNot(HaveOccurred())
	}

	hostCannotWgetMicrosoft := func() {
		Eventually(hostWgetMicrosoftErr, "10s", "1s").Should(HaveOccurred())
		Consistently(hostWgetMicrosoftErr, "4s", "1s").Should(HaveOccurred())
	}

	Context("with save file in initially non-existent directory", func() {
		BeforeEach(func() {
			saveFile = "/a/b/c/d/e/dnsinfo.txt"
			saveFileMappedOutsideContainer = false
		})

		It("can wget microsoft.com", func() {
			canWgetMicrosoft()
		})
	})

	getLastMicrosoftALog := func() (lastLog string) {
		dnsLogs, err := getDNSLogs(path.Join(dnsDir, "dns.log"))
		Expect(err).NotTo(HaveOccurred())
		for _, log := range dnsLogs {
			if strings.Contains(log, `"qname":"microsoft.com"`) && strings.Contains(log, `"qtype":"A"`) {
				lastLog = log
			}
		}
		return
	}

	Context("after wget microsoft.com", func() {

		JustBeforeEach(func() {
			time.Sleep(time.Second)
			canWgetMicrosoft()
		})

		It("should emit microsoft.com DNS log with latency", func() {
			Eventually(getLastMicrosoftALog, "10s", "1s").Should(MatchRegexp(`"latency_count":[1-9]`))
		})

		Context("with a preceding DNS request that went unresponded", func() {

			if os.Getenv("FELIX_FV_ENABLE_BPF") == "true" {
				// Skip because the following test relies on a HostEndpoint.
				return
			}

			JustBeforeEach(func() {
				hep := api.NewHostEndpoint()
				hep.Name = "felix-eth0"
				hep.Labels = map[string]string{"host-endpoint": "yes"}
				hep.Spec.Node = felix.Hostname
				hep.Spec.InterfaceName = "eth0"
				_, err := client.HostEndpoints().Create(utils.Ctx, hep, utils.NoOptions)
				Expect(err).NotTo(HaveOccurred())

				udp := numorstring.ProtocolFromString("udp")
				policy := api.NewGlobalNetworkPolicy()
				policy.Name = "deny-dns"
				policy.Spec.Selector = "host-endpoint == 'yes'"
				policy.Spec.Egress = []api.Rule{
					{
						Action:   api.Deny,
						Protocol: &udp,
						Destination: api.EntityRule{
							Ports: []numorstring.Port{numorstring.SinglePort(53)},
						},
					},
					{
						Action: api.Allow,
					},
				}
				policy.Spec.ApplyOnForward = true
				_, err = client.GlobalNetworkPolicies().Create(utils.Ctx, policy, utils.NoOptions)
				Expect(err).NotTo(HaveOccurred())

				// DNS should now fail, leaving at least one unresponded DNS
				// request.
				cannotWgetMicrosoft()

				// Delete the policy again.
				_, err = client.GlobalNetworkPolicies().Delete(utils.Ctx, "deny-dns", options.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())

				// Delete the host endpoint again.
				_, err = client.HostEndpoints().Delete(utils.Ctx, "felix-eth0", options.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())

				// Wait 11 seconds so that the unresponded request timestamp is
				// eligible for cleanup.
				time.Sleep(11 * time.Second)

				// Now DNS and outbound connection should work.
				canWgetMicrosoft()
			})

			It("should emit microsoft.com DNS log with latency", func() {
				Eventually(getLastMicrosoftALog, "10s", "1s").Should(MatchRegexp(`"latency_count":[1-9]`))
			})
		})

		Context("with DNS latency disabled", func() {
			BeforeEach(func() {
				enableLatency = false
			})

			It("should emit microsoft.com DNS log without latency", func() {
				Eventually(getLastMicrosoftALog, "10s", "1s").Should(MatchRegexp(`"latency_count":0`))
			})
		})

		Context("with DNS logs disabled", func() {
			BeforeEach(func() {
				enableLogs = false
			})

			It("should not emit DNS logs", func() {
				Consistently(path.Join(dnsDir, "dns.log"), "5s", "1s").ShouldNot(BeARegularFile())
			})
		})
	})

	Context("after host wget microsoft.com", func() {

		JustBeforeEach(func() {
			time.Sleep(time.Second)
			hostCanWgetMicrosoft()
		})

		It("should emit DNS logs", func() {
			Eventually(getLastMicrosoftALog, "10s", "1s").ShouldNot(BeEmpty())
		})

		Context("with DNS logs disabled", func() {
			BeforeEach(func() {
				enableLogs = false
			})

			It("should not emit DNS logs", func() {
				Consistently(path.Join(dnsDir, "dns.log"), "5s", "1s").ShouldNot(BeARegularFile())
			})
		})
	})

	JustBeforeEach(func() {
		opts := infrastructure.DefaultTopologyOptions()
		var err error
		dnsDir, err = ioutil.TempDir("", "dnsinfo")
		Expect(err).NotTo(HaveOccurred())
		opts.ExtraVolumes[dnsDir] = "/dnsinfo"
		opts.ExtraEnvVars["FELIX_DNSCACHEFILE"] = saveFile
		// For this test file, configure DNSCacheSaveInterval to be much longer than any
		// test duration, so we can be sure that the writing of the dnsinfo.txt file is
		// triggered by shutdown instead of by a periodic timer.
		opts.ExtraEnvVars["FELIX_DNSCACHESAVEINTERVAL"] = "3600"
		opts.ExtraEnvVars["FELIX_DNSTRUSTEDSERVERS"] = strings.Join(GetLocalNameservers(), ",")
		opts.ExtraEnvVars["FELIX_PolicySyncPathPrefix"] = "/var/run/calico"
		opts.ExtraEnvVars["FELIX_DNSLOGSFILEDIRECTORY"] = "/dnsinfo"
		opts.ExtraEnvVars["FELIX_DNSLOGSFLUSHINTERVAL"] = "1"
		if enableLogs {
			// Default for this is false.  Set "true" to enable.
			opts.ExtraEnvVars["FELIX_DNSLOGSFILEENABLED"] = "true"
		}
		if !enableLatency {
			// Default for this is true.  Set "false" to disable.
			opts.ExtraEnvVars["FELIX_DNSLOGSLATENCY"] = "false"
		}
		// This file tests that Felix writes out its DNS mappings file on shutdown, so we
		// need to stop Felix gracefully.
		opts.FelixStopGraceful = true
		// Tests in this file require a node IP, so that Felix can attach a BPF program to
		// host interfaces.
		opts.NeedNodeIP = true
		felix, etcd, client, infra = infrastructure.StartSingleNodeEtcdTopology(opts)
		infrastructure.CreateDefaultProfile(client, "default", map[string]string{"default": ""}, "")

		// Create a workload, using that profile.
		for ii := range w {
			iiStr := strconv.Itoa(ii)
			w[ii] = workload.Run(felix, "w"+iiStr, "default", "10.65.0.1"+iiStr, "8055", "tcp")
			w[ii].Configure(client)
		}

		// Allow workloads to connect out to the Internet.
		felix.Exec(
			"iptables", "-w", "-t", "nat",
			"-A", "POSTROUTING",
			"-o", "eth0",
			"-j", "MASQUERADE", "--random-fully",
		)
	})

	// Stop etcd and workloads, collecting some state if anything failed.
	AfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			felix.Exec("calico-bpf", "ipsets", "dump", "--debug")
			felix.Exec("ipset", "list")
			felix.Exec("iptables-save", "-c")
			felix.Exec("ip", "r")
		}

		for ii := range w {
			w[ii].Stop()
		}
		felix.Stop()
		if saveFileMappedOutsideContainer {
			Eventually(path.Join(dnsDir, "dnsinfo.txt"), "10s", "1s").Should(BeARegularFile())
		}

		if CurrentGinkgoTestDescription().Failed {
			etcd.Exec("etcdctl", "ls", "--recursive", "/")
		}
		etcd.Stop()
		infra.Stop()
	})

	It("can wget microsoft.com", func() {
		canWgetMicrosoft()
	})

	It("host can wget microsoft.com", func() {
		hostCanWgetMicrosoft()
	})

	Context("with default-deny egress policy", func() {
		JustBeforeEach(func() {
			policy := api.NewGlobalNetworkPolicy()
			policy.Name = "default-deny-egress"
			policy.Spec.Selector = "all()"
			policy.Spec.Egress = []api.Rule{{
				Action: api.Deny,
			}}
			_, err := client.GlobalNetworkPolicies().Create(utils.Ctx, policy, utils.NoOptions)
			Expect(err).NotTo(HaveOccurred())
		})

		It("cannot wget microsoft.com", func() {
			cannotWgetMicrosoft()
		})

		// There's no HostEndpoint yet, so the policy doesn't affect the host.
		It("host can wget microsoft.com", func() {
			hostCanWgetMicrosoft()
		})

		configureGNPAllowToMicrosoft := func() {
			policy := api.NewGlobalNetworkPolicy()
			policy.Name = "allow-microsoft"
			order := float64(20)
			policy.Spec.Order = &order
			policy.Spec.Selector = "all()"
			udp := numorstring.ProtocolFromString(numorstring.ProtocolUDP)
			policy.Spec.Egress = []api.Rule{
				{
					Action:      api.Allow,
					Destination: api.EntityRule{Domains: []string{"microsoft.com", "www.microsoft.com"}},
				},
				{
					Action:   api.Allow,
					Protocol: &udp,
					Destination: api.EntityRule{
						Ports: []numorstring.Port{numorstring.SinglePort(53)},
					},
				},
			}
			_, err := client.GlobalNetworkPolicies().Create(utils.Ctx, policy, utils.NoOptions)
			Expect(err).NotTo(HaveOccurred())
		}

		Context("with HostEndpoint", func() {

			if os.Getenv("FELIX_FV_ENABLE_BPF") == "true" {
				// Skip because the following test relies on a HostEndpoint.
				return
			}

			JustBeforeEach(func() {
				hep := api.NewHostEndpoint()
				hep.Name = "hep-1"
				hep.Spec.Node = felix.Hostname
				hep.Spec.InterfaceName = "eth0"
				_, err := client.HostEndpoints().Create(utils.Ctx, hep, utils.NoOptions)
				Expect(err).NotTo(HaveOccurred())
			})

			It("host cannot wget microsoft.com", func() {
				hostCannotWgetMicrosoft()
			})

			Context("with domain-allow egress policy", func() {
				JustBeforeEach(configureGNPAllowToMicrosoft)

				It("host can wget microsoft.com", func() {
					hostCanWgetMicrosoft()
				})
			})
		})

		Context("with domain-allow egress policy", func() {
			JustBeforeEach(configureGNPAllowToMicrosoft)

			It("can wget microsoft.com", func() {
				canWgetMicrosoft()
			})
		})

		Context("with namespaced domain-allow egress policy", func() {
			JustBeforeEach(func() {
				policy := api.NewNetworkPolicy()
				policy.Name = "allow-microsoft"
				policy.Namespace = "fv"
				order := float64(20)
				policy.Spec.Order = &order
				policy.Spec.Selector = "all()"
				udp := numorstring.ProtocolFromString(numorstring.ProtocolUDP)
				policy.Spec.Egress = []api.Rule{
					{
						Action:      api.Allow,
						Destination: api.EntityRule{Domains: []string{"microsoft.com", "www.microsoft.com"}},
					},
					{
						Action:   api.Allow,
						Protocol: &udp,
						Destination: api.EntityRule{
							Ports: []numorstring.Port{numorstring.SinglePort(53)},
						},
					},
				}
				_, err := client.NetworkPolicies().Create(utils.Ctx, policy, utils.NoOptions)
				Expect(err).NotTo(HaveOccurred())
			})

			It("can wget microsoft.com", func() {
				canWgetMicrosoft()
			})
		})

		Context("with namespaced domain-allow egress policy in wrong namespace", func() {
			JustBeforeEach(func() {
				policy := api.NewNetworkPolicy()
				policy.Name = "allow-microsoft"
				policy.Namespace = "wibbly-woo"
				order := float64(20)
				policy.Spec.Order = &order
				policy.Spec.Selector = "all()"
				udp := numorstring.ProtocolFromString(numorstring.ProtocolUDP)
				policy.Spec.Egress = []api.Rule{
					{
						Action:      api.Allow,
						Destination: api.EntityRule{Domains: []string{"microsoft.com", "www.microsoft.com"}},
					},
					{
						Action:   api.Allow,
						Protocol: &udp,
						Destination: api.EntityRule{
							Ports: []numorstring.Port{numorstring.SinglePort(53)},
						},
					},
				}
				_, err := client.NetworkPolicies().Create(utils.Ctx, policy, utils.NoOptions)
				Expect(err).NotTo(HaveOccurred())
			})

			It("cannot wget microsoft.com", func() {
				cannotWgetMicrosoft()
			})
		})

		Context("with wildcard domain-allow egress policy", func() {
			JustBeforeEach(func() {
				policy := api.NewGlobalNetworkPolicy()
				policy.Name = "allow-microsoft-wild"
				order := float64(20)
				policy.Spec.Order = &order
				policy.Spec.Selector = "all()"
				udp := numorstring.ProtocolFromString(numorstring.ProtocolUDP)
				policy.Spec.Egress = []api.Rule{
					{
						Action:      api.Allow,
						Destination: api.EntityRule{Domains: []string{"microsoft.*", "*.microsoft.com"}},
					},
					{
						Action:   api.Allow,
						Protocol: &udp,
						Destination: api.EntityRule{
							Ports: []numorstring.Port{numorstring.SinglePort(53)},
						},
					},
				}
				_, err := client.GlobalNetworkPolicies().Create(utils.Ctx, policy, utils.NoOptions)
				Expect(err).NotTo(HaveOccurred())
			})

			It("can wget microsoft.com", func() {
				canWgetMicrosoft()
			})
		})

		Context("with networkset with allowed egress domains", func() {
			JustBeforeEach(func() {
				gns := api.NewGlobalNetworkSet()
				gns.Name = "allow-microsoft"
				gns.Labels = map[string]string{"founder": "billg"}
				gns.Spec.AllowedEgressDomains = []string{"microsoft.com", "www.microsoft.com"}
				_, err := client.GlobalNetworkSets().Create(utils.Ctx, gns, utils.NoOptions)
				Expect(err).NotTo(HaveOccurred())

				policy := api.NewGlobalNetworkPolicy()
				policy.Name = "allow-microsoft"
				order := float64(20)
				policy.Spec.Order = &order
				policy.Spec.Selector = "all()"
				udp := numorstring.ProtocolFromString(numorstring.ProtocolUDP)
				policy.Spec.Egress = []api.Rule{
					{
						Action: api.Allow,
						Destination: api.EntityRule{
							Selector: "founder == 'billg'",
						},
					},
					{
						Action:   api.Allow,
						Protocol: &udp,
						Destination: api.EntityRule{
							Ports: []numorstring.Port{numorstring.SinglePort(53)},
						},
					},
				}
				_, err = client.GlobalNetworkPolicies().Create(utils.Ctx, policy, utils.NoOptions)
				Expect(err).NotTo(HaveOccurred())
			})

			It("can wget microsoft.com", func() {
				canWgetMicrosoft()
			})

			It("handles a domain set update", func() {
				// Create another GNS with same labels as the previous one, so that
				// the destination selector will now match this one as well, and so
				// the domain set membership will change.
				gns := api.NewGlobalNetworkSet()
				gns.Name = "allow-microsoft-2"
				gns.Labels = map[string]string{"founder": "billg"}
				gns.Spec.AllowedEgressDomains = []string{"port25.microsoft.com"}
				_, err := client.GlobalNetworkSets().Create(utils.Ctx, gns, utils.NoOptions)
				Expect(err).NotTo(HaveOccurred())

				time.Sleep(2 * time.Second)
				canWgetMicrosoft()
			})
		})

		Context("with networkset with allowed egress wildcard domains", func() {
			JustBeforeEach(func() {
				gns := api.NewGlobalNetworkSet()
				gns.Name = "allow-microsoft"
				gns.Labels = map[string]string{"founder": "billg"}
				gns.Spec.AllowedEgressDomains = []string{"microsoft.*", "*.microsoft.com"}
				_, err := client.GlobalNetworkSets().Create(utils.Ctx, gns, utils.NoOptions)
				Expect(err).NotTo(HaveOccurred())

				policy := api.NewGlobalNetworkPolicy()
				policy.Name = "allow-microsoft"
				order := float64(20)
				policy.Spec.Order = &order
				policy.Spec.Selector = "all()"
				udp := numorstring.ProtocolFromString(numorstring.ProtocolUDP)
				policy.Spec.Egress = []api.Rule{
					{
						Action: api.Allow,
						Destination: api.EntityRule{
							Selector: "founder == 'billg'",
						},
					},
					{
						Action:   api.Allow,
						Protocol: &udp,
						Destination: api.EntityRule{
							Ports: []numorstring.Port{numorstring.SinglePort(53)},
						},
					},
				}
				_, err = client.GlobalNetworkPolicies().Create(utils.Ctx, policy, utils.NoOptions)
				Expect(err).NotTo(HaveOccurred())
			})

			It("can wget microsoft.com", func() {
				canWgetMicrosoft()
			})

			It("handles a domain set update", func() {
				// Create another GNS with same labels as the previous one, so that
				// the destination selector will now match this one as well, and so
				// the domain set membership will change.
				gns := api.NewGlobalNetworkSet()
				gns.Name = "allow-microsoft-2"
				gns.Labels = map[string]string{"founder": "billg"}
				gns.Spec.AllowedEgressDomains = []string{"port25.microsoft.com"}
				_, err := client.GlobalNetworkSets().Create(utils.Ctx, gns, utils.NoOptions)
				Expect(err).NotTo(HaveOccurred())

				time.Sleep(2 * time.Second)
				canWgetMicrosoft()
			})
		})

		Context("with networkset with allowed egress domains", func() {
			JustBeforeEach(func() {
				ns := api.NewNetworkSet()
				ns.Name = "allow-microsoft"
				ns.Namespace = "fv"
				ns.Labels = map[string]string{"founder": "billg"}
				ns.Spec.AllowedEgressDomains = []string{"microsoft.com", "www.microsoft.com"}
				_, err := client.NetworkSets().Create(utils.Ctx, ns, utils.NoOptions)
				Expect(err).NotTo(HaveOccurred())

				policy := api.NewNetworkPolicy()
				policy.Name = "allow-microsoft"
				policy.Namespace = "fv"
				order := float64(20)
				policy.Spec.Order = &order
				policy.Spec.Selector = "all()"
				udp := numorstring.ProtocolFromString(numorstring.ProtocolUDP)
				policy.Spec.Egress = []api.Rule{
					{
						Action: api.Allow,
						Destination: api.EntityRule{
							Selector: "founder == 'billg'",
						},
					},
					{
						Action:   api.Allow,
						Protocol: &udp,
						Destination: api.EntityRule{
							Ports: []numorstring.Port{numorstring.SinglePort(53)},
						},
					},
				}
				_, err = client.NetworkPolicies().Create(utils.Ctx, policy, utils.NoOptions)
				Expect(err).NotTo(HaveOccurred())
			})

			It("can wget microsoft.com", func() {
				canWgetMicrosoft()
			})

			It("handles a domain set update", func() {
				// Create another NetworkSet with same labels as the previous one, so that
				// the destination selector will now match this one as well, and so
				// the domain set membership will change.
				ns := api.NewNetworkSet()
				ns.Name = "allow-microsoft-2"
				ns.Namespace = "fv"
				ns.Labels = map[string]string{"founder": "billg"}
				ns.Spec.AllowedEgressDomains = []string{"port25.microsoft.com"}
				_, err := client.NetworkSets().Create(utils.Ctx, ns, utils.NoOptions)
				Expect(err).NotTo(HaveOccurred())

				time.Sleep(2 * time.Second)
				canWgetMicrosoft()
			})
		})

		Context("with networkset with allowed egress wildcard domains", func() {
			JustBeforeEach(func() {
				ns := api.NewNetworkSet()
				ns.Name = "allow-microsoft"
				ns.Namespace = "fv"
				ns.Labels = map[string]string{"founder": "billg"}
				ns.Spec.AllowedEgressDomains = []string{"microsoft.*", "*.microsoft.com"}
				_, err := client.NetworkSets().Create(utils.Ctx, ns, utils.NoOptions)
				Expect(err).NotTo(HaveOccurred())

				policy := api.NewNetworkPolicy()
				policy.Name = "allow-microsoft"
				policy.Namespace = "fv"
				order := float64(20)
				policy.Spec.Order = &order
				policy.Spec.Selector = "all()"
				udp := numorstring.ProtocolFromString(numorstring.ProtocolUDP)
				policy.Spec.Egress = []api.Rule{
					{
						Action: api.Allow,
						Destination: api.EntityRule{
							Selector: "founder == 'billg'",
						},
					},
					{
						Action:   api.Allow,
						Protocol: &udp,
						Destination: api.EntityRule{
							Ports: []numorstring.Port{numorstring.SinglePort(53)},
						},
					},
				}
				_, err = client.NetworkPolicies().Create(utils.Ctx, policy, utils.NoOptions)
				Expect(err).NotTo(HaveOccurred())
			})

			It("can wget microsoft.com", func() {
				canWgetMicrosoft()
			})

			It("handles a domain set update", func() {
				// Create another NetworkSet with same labels as the previous one, so that
				// the destination selector will now match this one as well, and so
				// the domain set membership will change.
				ns := api.NewNetworkSet()
				ns.Name = "allow-microsoft-2"
				ns.Namespace = "fv"
				ns.Labels = map[string]string{"founder": "billg"}
				ns.Spec.AllowedEgressDomains = []string{"port25.microsoft.com"}
				_, err := client.NetworkSets().Create(utils.Ctx, ns, utils.NoOptions)
				Expect(err).NotTo(HaveOccurred())

				time.Sleep(2 * time.Second)
				canWgetMicrosoft()
			})
		})
	})
})
