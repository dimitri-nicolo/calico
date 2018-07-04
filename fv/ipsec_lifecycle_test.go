// +build fvtests

// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package fv_test

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/felix/fv/infrastructure"
	"github.com/projectcalico/felix/fv/utils"
	"github.com/projectcalico/felix/fv/workload"
	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"
)

var _ = infrastructure.DatastoreDescribe("IPsec lifecycle tests", []apiconfig.DatastoreType{apiconfig.EtcdV3, apiconfig.Kubernetes}, func(getInfra infrastructure.InfraFactory) {

	var (
		infra   infrastructure.DatastoreInfra
		felixes []*infrastructure.Felix
		client  client.Interface
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
		topologyOptions.ExtraEnvVars["FELIX_IPSECREKEYTIME"] = "20"
		topologyOptions.IPIPEnabled = false

		felixes, client = infrastructure.StartNNodeTopology(2, topologyOptions, infra)

		// Install a default profile that allows all ingress and egress, in the absence of any Policy.
		err = infra.AddDefaultAllow()
		Expect(err).NotTo(HaveOccurred())

		// Create workloads, using that profile.  One on each "host".
		for ii := range w {
			wIP := fmt.Sprintf("10.65.%d.2", ii)
			wName := fmt.Sprintf("w%d", ii)
			w[ii] = workload.Run(felixes[ii], wName, "default", wIP, "8055", "udp")
			w[ii].ConfigureInDatastore(infra)

			hostW[ii] = workload.Run(felixes[ii], fmt.Sprintf("host%d", ii), "", felixes[ii].IP, "8055", "udp")
		}

		// Wait for Felix to program the IPsec policy.  Otherwise, we might see some unencrypted traffic at
		// start-of-day.  There's not much we can do about that in general since we don't know the workload's IP
		// to blacklist it until we hear about the workload.
		const numPoliciesPerWep = 3
		for i, f := range felixes {
			for j := range felixes {
				if i == j {
					continue
				}

				polCount := func() int {
					out, err := f.ExecOutput("ip", "xfrm", "policy")
					Expect(err).NotTo(HaveOccurred())
					return strings.Count(out, w[j].IP)
				}
				// Felix might restart during set up, causing a 2s delay here.
				Eventually(polCount, "5s", "100ms").Should(Equal(numPoliciesPerWep),
					fmt.Sprintf("Expected to see %d IPsec policies for workload IP %s in felix container %s",
						numPoliciesPerWep, w[j].IP, f.Name))
			}
		}

		cc = &workload.ConnectivityChecker{Protocol: "udp"}
	})

	AfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			utils.Run("docker", "ps", "-a")
			for _, felix := range felixes {
				felix.Exec("swanctl", "--list-sas")
				felix.Exec("ip", "-s", "xfrm", "state")
				felix.Exec("ip", "-s", "xfrm", "policy")
			}
		}

		for _, wl := range w {
			wl.Stop()
		}
		for _, wl := range hostW {
			wl.Stop()
		}
		for _, felix := range felixes {
			felix.Stop()
		}

		if CurrentGinkgoTestDescription().Failed {
			infra.DumpErrorData()
		}
		infra.Stop()
	})

	// Function to get number of SAs for connection (src->dest) and SPI for first SA on destination workload.
	getDestSaCountAndSPI := func(src, dest *infrastructure.Felix) (int, string) {
		output, err := dest.ExecOutput("ip", "xfrm", "state")
		Expect(err).NotTo(HaveOccurred())

		saDirectionInfo := fmt.Sprintf("src %s dst %s", src.IP, dest.IP)
		count := strings.Count(output, saDirectionInfo)
		Expect(count).NotTo(Equal(0))

		i := strings.Index(output, saDirectionInfo)
		spi := regexp.MustCompile(`0x[a-f0-9]+`).FindString(output[i:])
		Expect(spi).NotTo(BeEmpty())

		return count, spi
	}

	saExists := func(felix *infrastructure.Felix, sa string) bool {
		output, err := felix.ExecOutput("ip", "xfrm", "state")
		Expect(err).NotTo(HaveOccurred())

		return strings.Contains(output, sa)
	}

	It("Should rekey properly and cause acceptable packet loss", func() {
		// Start packet loss test and monitor SA changes of a destination workload.

		// Start a connection test first.
		// We do not want to spend time on packet loss test if a simple connection test fails.
		cc.ExpectSome(w[0], w[1])
		cc.CheckConnectivity()
		cc.ResetExpectations()

		var count int
		var spi, startSPI string
		Eventually(func() int {
			count, spi = getDestSaCountAndSPI(felixes[0], felixes[1])
			return count
		}).Should(Equal(1), "Number of start SA")
		startSPI = spi

		cc.ExpectLoss(w[0], w[1], 30*time.Second, -1, 20)
		cc.CheckConnectivity()

		// It is possible Felix got config update and restart itself. This is because of a race in the set-up logic;
		// the Node resource gets set up at the same time that Felix is starting.
		// We could have two IKEs setup both with one or two child SAs.
		// It is not feasible to assert on number of child SAs we got at the end of the test.
		// But we should not see the original SA on both hosts.
		Expect(saExists(felixes[0], startSPI)).To(BeFalse())
		Expect(saExists(felixes[1], startSPI)).To(BeFalse())
	})

	It("Felix should restart if charon daemon exits", func() {
		felix := felixes[0]

		// Get felix/charon's PID so we can check that it restarts...
		felixPID := felix.GetFelixPID()
		charonPID := felix.GetSinglePID("/usr/lib/strongswan/charon")

		// Kill charon daemon
		killProcess(felix, fmt.Sprint(charonPID))

		Eventually(felix.GetFelixPID, "5s", "100ms").ShouldNot(Equal(felixPID),
			"Felix failed to restart after killing the charon")

		Eventually(func() int {
			return felix.GetSinglePID("/usr/lib/strongswan/charon")
		}, "3s").ShouldNot(Equal(charonPID), "New charon process")
	})
})

func killProcess(felix *infrastructure.Felix, pidString string) {
	_, err := felix.ExecOutput("kill", "-9", pidString)
	Expect(err).NotTo(HaveOccurred())
}
