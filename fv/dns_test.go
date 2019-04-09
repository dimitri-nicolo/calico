// +build fvtests

// Copyright (c) 2017-2019 Tigera, Inc. All rights reserved.
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
	"io/ioutil"
	"path"
	"strconv"

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
)

var _ = Describe("DNS Policy", func() {

	var (
		etcd   *containers.Container
		felix  *infrastructure.Felix
		client client.Interface
		w      [1]*workload.Workload
		dnsDir string
	)

	BeforeEach(func() {
		opts := infrastructure.DefaultTopologyOptions()
		var err error
		dnsDir, err = ioutil.TempDir("", "dnsinfo")
		Expect(err).NotTo(HaveOccurred())
		opts.ExtraVolumes[dnsDir] = "/dnsinfo"
		opts.ExtraEnvVars["FELIX_DOMAININFOSTORE"] = "/dnsinfo/dnsinfo.txt"
		felix, etcd, client = infrastructure.StartSingleNodeEtcdTopology(opts)
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
			"-j", "MASQUERADE",
		)
	})

	// Stop etcd and workloads, collecting some state if anything failed.
	AfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			felix.Exec("ipset", "list")
			felix.Exec("iptables-save", "-c")
			felix.Exec("ip", "r")
		}

		for ii := range w {
			w[ii].Stop()
		}
		felix.Stop()
		Eventually(path.Join(dnsDir, "dnsinfo.txt"), "10s", "1s").Should(BeARegularFile())

		if CurrentGinkgoTestDescription().Failed {
			etcd.Exec("etcdctl", "ls", "--recursive", "/")
		}
		etcd.Stop()
	})

	It("can wget microsoft.com", func() {
		out, err := w[0].ExecOutput("wget", "-T", "10", "microsoft.com")
		log.WithError(err).Infof("wget said:\n%v", out)
		Expect(err).NotTo(HaveOccurred())
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

		It("cannot wget microsoft.com", func() {
			out, err := w[0].ExecOutput("wget", "-T", "10", "microsoft.com")
			log.WithError(err).Infof("wget said:\n%v", out)
			Expect(err).To(HaveOccurred())
		})

		Context("with domain-allow egress policy", func() {
			BeforeEach(func() {
				policy := api.NewGlobalNetworkPolicy()
				policy.Name = "allow-microsoft"
				order := float64(20)
				policy.Spec.Order = &order
				policy.Spec.Selector = "all()"
				tcp := numorstring.ProtocolFromString(numorstring.ProtocolTCP)
				udp := numorstring.ProtocolFromString(numorstring.ProtocolUDP)
				policy.Spec.Egress = []api.Rule{
					{
						Action:      api.Allow,
						Destination: api.EntityRule{Domains: []string{"microsoft.com", "www.microsoft.com"}},
					},
					{
						Action:   api.Allow,
						Protocol: &tcp,
						Destination: api.EntityRule{
							Ports: []numorstring.Port{numorstring.SinglePort(53)},
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
				_, err := client.GlobalNetworkPolicies().Create(utils.Ctx, policy, utils.NoOptions)
				Expect(err).NotTo(HaveOccurred())
			})

			It("can wget microsoft.com", func() {
				out, err := w[0].ExecOutput("wget", "-T", "10", "microsoft.com")
				log.WithError(err).Infof("wget said:\n%v", out)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("with networkset with allowed egress domains", func() {
			BeforeEach(func() {
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
				tcp := numorstring.ProtocolFromString(numorstring.ProtocolTCP)
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
						Protocol: &tcp,
						Destination: api.EntityRule{
							Ports: []numorstring.Port{numorstring.SinglePort(53)},
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
				out, err := w[0].ExecOutput("wget", "-T", "10", "microsoft.com")
				log.WithError(err).Infof("wget said:\n%v", out)
				Expect(err).NotTo(HaveOccurred())
			})
		})

	})
})
