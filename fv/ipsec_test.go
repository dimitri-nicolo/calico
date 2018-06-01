// +build fvtests

// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package fv_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"regexp"

	"context"

	"github.com/projectcalico/felix/fv/containers"
	"github.com/projectcalico/felix/fv/infrastructure"
	"github.com/projectcalico/felix/fv/workload"
	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
)

var _ = infrastructure.DatastoreDescribe("IPsec tests", []apiconfig.DatastoreType{apiconfig.EtcdV3, apiconfig.Kubernetes}, func(getInfra infrastructure.InfraFactory) {

	var (
		infra    infrastructure.DatastoreInfra
		felixes  []*infrastructure.Felix
		tcpdumps []*containers.TCPDump
		client   client.Interface
		// w[n] is a simulated workload for host n.  It has its own network namespace (as if it was a container).
		w [2]*workload.Workload
		// hostW[n] is a simulated host networked workload for host n.  It runs in felix's network namespace.
		hostW [2]*workload.Workload
		cc    *workload.ConnectivityChecker
	)

	BeforeEach(func() {
		var err error

		infra, err = getInfra()
		Expect(err).NotTo(HaveOccurred())
		topologyOptions := infrastructure.DefaultTopologyOptions()
		// Enable IPsec.
		topologyOptions.ExtraEnvVars["FELIX_IPSECMODE"] = "PSK"
		topologyOptions.ExtraEnvVars["FELIX_IPSECPSKFILE"] = "/proc/1/cmdline"
		topologyOptions.ExtraEnvVars["FELIX_IPSECIKEAlGORITHM"] = "aes128gcm16-prfsha256-ecp256"
		topologyOptions.ExtraEnvVars["FELIX_IPSECESPAlGORITHM"] = "aes128gcm16-ecp256"
		topologyOptions.IPIPEnabled = false

		felixes, client = infrastructure.StartNNodeTopology(2, topologyOptions, infra)

		// Install a default profile that allows all ingress and egress, in the absence of any Policy.
		err = infra.AddDefaultAllow()
		Expect(err).NotTo(HaveOccurred())

		// Start tcpdump inside each host container.  Dumping inside the container means that we'll see a lot less
		// noise from the rest of the system.
		tcpdumps = nil
		for _, f := range felixes {
			tcpdump := containers.AttachTCPDump(f.Container, "eth0")
			tcpdump.AddMatcher("numIKEPackets", regexp.MustCompile(`.*isakmp:.*`))
			tcpdump.AddMatcher("numInboundESPPackets", regexp.MustCompile(`.*`+regexp.QuoteMeta("> "+f.IP)+`.*ESP.*`))
			tcpdump.AddMatcher("numOutboundESPPackets", regexp.MustCompile(`.*`+regexp.QuoteMeta(f.IP+" >")+`.*ESP.*`))
			tcpdump.AddMatcher("numInboundWorkloadPackets",
				regexp.MustCompile(`.*`+regexp.QuoteMeta(">")+` 10\.65\.\d+\.2.*`))
			tcpdump.AddMatcher("numInboundWorkloadToHostPackets",
				regexp.MustCompile(`.*10\.65\.\d+\.2.\d+ `+regexp.QuoteMeta("> "+f.IP)))
			tcpdump.Start()
			tcpdumps = append(tcpdumps, tcpdump)
		}

		// Create workloads, using that profile.  One on each "host".
		for ii := range w {
			wIP := fmt.Sprintf("10.65.%d.2", ii)
			wName := fmt.Sprintf("w%d", ii)
			w[ii] = workload.Run(felixes[ii], wName, "default", wIP, "8055", "tcp")
			w[ii].ConfigureInDatastore(infra)

			hostW[ii] = workload.Run(felixes[ii], fmt.Sprintf("host%d", ii), "", felixes[ii].IP, "8055", "tcp")
		}

		cc = &workload.ConnectivityChecker{}
	})

	AfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			for _, felix := range felixes {
				felix.Exec("iptables-save", "-c")
				felix.Exec("ipset", "list")
				felix.Exec("ip", "r")
				felix.Exec("ip", "a")
				felix.Exec("ip", "xfrm", "state")
				felix.Exec("ip", "xfrm", "policy")
			}
		}

		for _, wl := range w {
			wl.Stop()
		}
		for _, wl := range hostW {
			wl.Stop()
		}
		for _, t := range tcpdumps {
			t.Stop()
		}
		for _, felix := range felixes {
			felix.Stop()
		}

		if CurrentGinkgoTestDescription().Failed {
			infra.DumpErrorData()
		}
		infra.Stop()
	})

	tcpdumpMatches := func(felix int, name string) func() int {
		return func() int {
			return tcpdumps[felix].MatchCount(name)
		}
	}

	expectIKE := func() {
		for i := range felixes {
			By(fmt.Sprintf("Doing IKE (felix %v)", i))
			Eventually(tcpdumpMatches(i, "numIKEPackets")).Should(BeNumerically(">", 0),
				"tcpdump didn't record any IKE packets")
		}
	}

	expectNoESP := func() {
		for i := range felixes {
			By(fmt.Sprintf("Doing no ESP (felix %v)", i))
			Eventually(tcpdumpMatches(i, "numInboundESPPackets")).Should(BeNumerically("==", 0),
				"tcpdump saw unexpected inbound ESP packets")
			Eventually(tcpdumpMatches(i, "numOutboundESPPackets")).Should(BeNumerically("==", 0),
				"tcpdump saw unexpected outbound ESP packets")
		}
	}

	expectIKEAndESP := func() {
		expectIKE()

		for i := range felixes {
			By(fmt.Sprintf("Doing ESP (felix %v)", i))
			Eventually(tcpdumpMatches(i, "numInboundESPPackets")).Should(BeNumerically(">", 0),
				"tcpdump didn't record any inbound ESP packets")
			Eventually(tcpdumpMatches(i, "numOutboundESPPackets")).Should(BeNumerically(">", 0),
				"tcpdump didn't record any inbound ESP packets")
		}
	}

	It("workload-to-workload should be allowed and encrypted", func() {
		cc.ExpectSome(w[0], w[1])
		cc.ExpectSome(w[1], w[0])
		cc.CheckConnectivity()

		expectIKEAndESP()

		for i := range felixes {
			By(fmt.Sprintf("Doing IKE and ESP (felix %v)", i))

			// When snooping, tcpdump sees both inbound post-decryption packets as well as both inbound and outbound
			// encrypted packets.  That means we expect the number of unencrypted packets that we see in the capture
			// to be equal to the number of inbound encrypted packets.
			Eventually(func() int {
				return tcpdumpMatches(i, "numInboundWorkloadPackets")() -
					tcpdumpMatches(i, "numInboundESPPackets")()
			}).Should(BeZero(), "Number of inbound unencrypted packets didn't match number of inbound ESP packets")
		}
	})

	It("host-to-workload connections should be encrypted", func() {
		cc.ExpectSome(felixes[0], w[1])
		cc.ExpectSome(felixes[1], w[0])
		cc.CheckConnectivity()

		expectIKEAndESP()

		for i := range felixes {
			By(fmt.Sprintf("Having expected mix of encrypted/unencrypted packets (felix %v)", i))

			// When snooping, tcpdump sees both inbound post-decryption packets as well as both inbound and outbound
			// encrypted packets.  That means we expect the number of unencrypted packets that we see in the capture
			// to be equal to the number of inbound encrypted packets.
			Eventually(func() int {
				return tcpdumpMatches(i, "numInboundWorkloadPackets")() +
					tcpdumpMatches(i, "numInboundWorkloadToHostPackets")() -
					tcpdumpMatches(i, "numInboundESPPackets")()
			}).Should(BeZero(), "Number of inbound unencrypted packets didn't match number of inbound ESP packets")
		}
	})

	It("should have unencrypted host to host connectivity", func() {
		cc.ExpectSome(felixes[0], hostW[1])
		cc.ExpectSome(felixes[1], hostW[0])
		cc.CheckConnectivity()

		expectIKE()
		expectNoESP()
	})

	Context("with host protection policy in place", func() {
		BeforeEach(func() {
			// Make sure host endpoints don't block IPSec traffic (since they deny all traffic by default)
			err := infra.AddAllowToDatastore("host-endpoint=='true'")
			Expect(err).NotTo(HaveOccurred())

			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			for _, f := range felixes {
				hep := api.NewHostEndpoint()
				hep.Name = "eth0-" + f.Name
				hep.Labels = map[string]string{
					"host-endpoint": "true",
				}
				hep.Spec.Node = f.Hostname
				hep.Spec.ExpectedIPs = []string{f.IP}
				_, err := client.HostEndpoints().Create(ctx, hep, options.SetOptions{})
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("should have workload connectivity but not host connectivity", func() {
			// Host endpoints (with no policies) block host-host traffic due to default drop.
			cc.ExpectNone(felixes[0], hostW[1])
			cc.ExpectNone(felixes[1], hostW[0])
			// But the rules to allow IPSec between our hosts let the workload traffic through.
			cc.ExpectSome(w[0], w[1])
			cc.ExpectSome(w[1], w[0])
			cc.CheckConnectivity()
		})
	})

	// FIXME: This doesn't work because we currently allow unencrypted traffic if it's from a workload that we don't have IPsec config for.
	PContext("after removing BGP address from nodes", func() {
		// Simulate having a host send IPsec traffic from an unknown source, should get blocked.
		BeforeEach(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			l, err := client.Nodes().List(ctx, options.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			for _, node := range l.Items {
				node.Spec.BGP = nil
				_, err := client.Nodes().Update(ctx, &node, options.SetOptions{})
				Expect(err).NotTo(HaveOccurred())
			}

			// Removing the BGP config triggers a Felix restart and Felix has a 2s timer during
			// a config restart to ensure that it doesn't tight loop.  Wait for the ipset to be
			// updated as a signal that Felix has restarted.
			for _, f := range felixes {
				Eventually(func() int {
					return getNumIPSetMembers(f.Container, "cali40all-hosts")
				}, "5s", "200ms").Should(BeZero())
			}
		})

		It("should have no workload to workload connectivity", func() {
			cc.ExpectNone(w[0], w[1])
			cc.ExpectNone(w[1], w[0])
			cc.CheckConnectivity()
		})
	})
})
