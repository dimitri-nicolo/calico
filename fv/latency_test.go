// +build fvtests

// Copyright (c) 2017-2018 Tigera, Inc. All rights reserved.
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
	"fmt"
	"os"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/fv/containers"
	"github.com/projectcalico/felix/fv/infrastructure"
	"github.com/projectcalico/felix/fv/utils"
	"github.com/projectcalico/felix/fv/workload"
	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"
)

type latencyConfig struct {
	ipVersion   int
	generateIPs func(n int) (result []string)
}

func (c latencyConfig) workloadIP(workloadIdx int) string {
	if c.ipVersion == 4 {
		// Each IP is in its own /24.
		return fmt.Sprintf("10.65.1.%d", workloadIdx)
	}
	// Each IP gets its own /64.
	return fmt.Sprintf("fdc6:3dbc:e983:cbc%x::1", workloadIdx)
}

var _ = Context("Latency tests with initialized Felix and etcd datastore", func() {

	var (
		etcd   *containers.Container
		felix  *infrastructure.Felix
		client client.Interface

		resultsFile *os.File
	)

	BeforeEach(func() {
		topologyOptions := infrastructure.DefaultTopologyOptions()
		topologyOptions.EnableIPv6 = true

		felix, etcd, client = infrastructure.StartSingleNodeEtcdTopology(topologyOptions)
		_ = felix.GetFelixPID()

		// Install the hping tool, which we use for latency measurments.
		felix.Exec("sh", "-c", "echo http://dl-cdn.alpinelinux.org/alpine/edge/testing >> /etc/apk/repositories")
		felix.Exec("apk", "update")
		felix.Exec("apk", "add", "hping3")

		var err error
		resultsFile, err = os.OpenFile("latency.log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		err := resultsFile.Close()
		if err != nil {
			log.WithError(err).Error("Close returned error")
		}

		if CurrentGinkgoTestDescription().Failed {
			felix.Exec("iptables-save", "-c")
		}
		felix.Stop()

		if CurrentGinkgoTestDescription().Failed {
			etcd.Exec("etcdctl", "ls", "--recursive", "/")
		}
		etcd.Stop()
	})

	describeLatencyTests := func(c latencyConfig) {
		var (
			w   [2]*workload.Workload
			cc  *workload.ConnectivityChecker
			pol *api.GlobalNetworkPolicy
		)

		createPolicy := func(policy *api.GlobalNetworkPolicy) *api.GlobalNetworkPolicy {
			log.WithField("policy", dumpResource(policy)).Info("Creating policy")
			policy, err := client.GlobalNetworkPolicies().Create(utils.Ctx, policy, utils.NoOptions)
			Expect(err).NotTo(HaveOccurred())
			return policy
		}

		updatePolicy := func(policy *api.GlobalNetworkPolicy) *api.GlobalNetworkPolicy {
			log.WithField("policy", dumpResource(policy)).Info("Updating policy")
			policy, err := client.GlobalNetworkPolicies().Update(utils.Ctx, policy, utils.NoOptions)
			Expect(err).NotTo(HaveOccurred())
			return policy
		}

		BeforeEach(func() {
			for ii := range w {
				iiStr := strconv.Itoa(ii)
				var ports string

				ports = "3000"
				w[ii] = workload.Run(
					felix,
					"w"+iiStr,
					"fv",
					c.workloadIP(ii),
					ports,
					"tcp",
				)

				w[ii].DefaultPort = "3000"
				w[ii].Configure(client)
			}

			cc = &workload.ConnectivityChecker{
				Protocol: "tcp",
			}

			pol = api.NewGlobalNetworkPolicy()
			pol.Namespace = "fv"
			pol.Name = "policy-1"
			pol.Spec.Ingress = []api.Rule{
				{
					Action: "Allow",
				},
			}
			pol.Spec.Egress = []api.Rule{
				{
					Action: "Allow",
				},
			}
			pol.Spec.Selector = "all()"

			pol = createPolicy(pol)

			cc.ExpectSome(w[0], w[1])
			cc.ExpectSome(w[1], w[0])
			cc.CheckConnectivity()
		})

		It("with allow-all should have good latency", func() {
			meanRtt := w[0].LatencyTo(w[1].IP, w[1].DefaultPort)
			_, err := fmt.Fprintf(resultsFile, "allow-all: %v\n", meanRtt)
			Expect(meanRtt).To(BeNumerically("<", 10*time.Millisecond))
			Expect(err).NotTo(HaveOccurred())
		})

		Describe("with all() source selector", func() {
			const (
				sourceSelector = "all()"
			)
			BeforeEach(func() {
				pol.Spec.Ingress[0].Source.Selector = sourceSelector
				pol = updatePolicy(pol)
			})

			It("should have good latency", func() {
				meanRtt := w[0].LatencyTo(w[1].IP, w[1].DefaultPort)
				_, err := fmt.Fprintf(resultsFile, "all-selector: %v\n", meanRtt)
				Expect(meanRtt).To(BeNumerically("<", 10*time.Millisecond))
				Expect(err).NotTo(HaveOccurred())
			})

			Describe("with 10k IPs in an IP set", func() {
				BeforeEach(func() {
					// Add an extra 10k IPs to one of the workload endpoints.
					w[1].WorkloadEndpoint.Spec.IPNetworks = append(w[1].WorkloadEndpoint.Spec.IPNetworks,
						c.generateIPs(10000)...)
					wep := w[1].WorkloadEndpoint
					wep.Namespace = "fv"
					_, err := client.WorkloadEndpoints().Update(utils.Ctx, wep, utils.NoOptions)
					Expect(err).NotTo(HaveOccurred())

					// The all() selector should now map to an IP set with 10,002 IPs in it.
					ipSetName := utils.IPSetNameForSelector(c.ipVersion, sourceSelector)
					Eventually(func() int {
						return getNumIPSetMembers(
							felix.Container,
							ipSetName,
						)
					}, "100s", "1000ms").Should(Equal(10002))
				})

				It("should have good latency", func() {
					meanRtt := w[0].LatencyTo(w[1].IP, w[1].DefaultPort)
					_, err := fmt.Fprintf(resultsFile, "all-selector-10k: %v\n", meanRtt)
					Expect(meanRtt).To(BeNumerically("<", 10*time.Millisecond))
					Expect(err).NotTo(HaveOccurred())
				})
			})
		})

		AfterEach(func() {
			for ii := range w {
				w[ii].Stop()
			}
		})
	}

	Context("IPv4: Network sets tests with initialized Felix and etcd datastore", func() {
		describeLatencyTests(latencyConfig{ipVersion: 4, generateIPs: generateIPv4s})
	})

	// Unfortunately, hping3 doesn't support IPv6.
	//Context("IPv6: Network sets tests with initialized Felix and etcd datastore", func() {
	//	describeLatencyTests(latencyConfig{ipVersion: 6, generateIPs: generateIPv6s})
	//})
})

func generateIPv4s(n int) (result []string) {
	for a := 0; a < 256; a++ {
		for b := 0; b < 256; b++ {
			for c := 0; c < 256; c++ {
				if n <= 0 {
					return
				}
				result = append(result, fmt.Sprintf("11.%d.%d.%d", a, b, c))
				n--
			}
		}
	}
	panic("too many IPs")
}

func generateIPv6s(n int) (result []string) {
	for a := 0; a < 256; a++ {
		for b := 0; b < 256; b++ {
			for c := 0; c < 256; c++ {
				if n <= 0 {
					return
				}
				result = append(result, fmt.Sprintf("fdc6:3dbc:e983:cbcf:%x:%x:%x::1", a, b, c))
				n--
			}
		}
	}
	panic("too many IPs")
}
