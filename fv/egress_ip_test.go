// Copyright (c) 2020 Tigera, Inc. All rights reserved.
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

// +build fvtests

package fv_test

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/felix/fv/infrastructure"
	"github.com/projectcalico/felix/fv/workload"
	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/ipam"
	"github.com/projectcalico/libcalico-go/lib/net"
	"github.com/projectcalico/libcalico-go/lib/options"
)

var _ = infrastructure.DatastoreDescribe("Egress IP", []apiconfig.DatastoreType{apiconfig.Kubernetes}, func(getInfra infrastructure.InfraFactory) {
	var (
		infra        infrastructure.DatastoreInfra
		felix        *infrastructure.Felix
		client       client.Interface
		err          error
		supportLevel string
	)

	makeGateway := func(wIP, wName string) *workload.Workload {
		err := client.IPAM().AssignIP(context.Background(), ipam.AssignIPArgs{
			IP:       net.MustParseIP(wIP),
			HandleID: &wName,
			Attrs: map[string]string{
				ipam.AttributeNode: felix.Hostname,
			},
			Hostname: felix.Hostname,
		})
		Expect(err).NotTo(HaveOccurred())
		gw := workload.Run(felix, wName, "default", wIP, "8055", "tcp")
		gw.WorkloadEndpoint.Labels["egress-code"] = "red"
		gw.ConfigureInInfra(infra)
		return gw
	}

	makeClient := func(wIP, wName string) *workload.Workload {
		err := client.IPAM().AssignIP(context.Background(), ipam.AssignIPArgs{
			IP:       net.MustParseIP(wIP),
			HandleID: &wName,
			Attrs: map[string]string{
				ipam.AttributeNode: felix.Hostname,
			},
			Hostname: felix.Hostname,
		})
		Expect(err).NotTo(HaveOccurred())
		app := workload.Run(felix, wName, "default", wIP, "8055", "tcp")
		app.WorkloadEndpoint.Spec.EgressGateway = &api.EgressSpec{
			Selector: "egress-code == 'red'",
		}
		app.ConfigureInInfra(infra)
		return app
	}

	getIPRules := func() map[string]string {
		rules, err := felix.ExecOutput("ip", "rule")
		log.WithError(err).Infof("ip rule said:\n%v", rules)
		Expect(err).NotTo(HaveOccurred())
		mappings := map[string]string{}
		fwmarkRE := regexp.MustCompile(`from ([0-9.]+) fwmark [^ ]+ lookup ([0-9]+)`)
		for _, line := range strings.Split(rules, "\n") {
			match := fwmarkRE.FindStringSubmatch(line)
			if len(match) < 3 {
				continue
			}
			mappings[match[1]] = match[2]
		}
		log.Infof("Found mappings: %v", mappings)
		return mappings
	}

	getIPRoute := func(table string) string {
		route, err := felix.ExecOutput("ip", "r", "l", "table", table)
		log.WithError(err).Infof("ip r l said:\n%v", route)
		Expect(err).NotTo(HaveOccurred())
		return strings.TrimSpace(route)
	}

	getIPNeigh := func() map[string]string {
		neigh, err := felix.ExecOutput("ip", "neigh", "show", "dev", "egress.calico")
		log.WithError(err).Infof("ip neigh said:\n%v", neigh)
		Expect(err).NotTo(HaveOccurred())
		mappings := map[string]string{}
		lladdrRE := regexp.MustCompile(`([0-9.]+) lladdr ([0-9a-f:]+)`)
		for _, line := range strings.Split(neigh, "\n") {
			match := lladdrRE.FindStringSubmatch(line)
			if len(match) < 3 {
				continue
			}
			mappings[match[1]] = match[2]
		}
		log.Infof("Found mappings: %v", mappings)
		return mappings
	}

	getBridgeFDB := func() map[string]string {
		fdb, err := felix.ExecOutput("bridge", "fdb", "show", "dev", "egress.calico")
		log.WithError(err).Infof("bridge fdb said:\n%v", fdb)
		Expect(err).NotTo(HaveOccurred())
		mappings := map[string]string{}
		fdbRE := regexp.MustCompile(`([0-9a-f:]+) dst ([0-9.]+)`)
		for _, line := range strings.Split(fdb, "\n") {
			match := fdbRE.FindStringSubmatch(line)
			if len(match) < 3 {
				continue
			}
			mappings[match[1]] = match[2]
		}
		log.Infof("Found mappings: %v", mappings)
		return mappings
	}

	JustBeforeEach(func() {
		infra = getInfra()
		topologyOptions := infrastructure.DefaultTopologyOptions()
		topologyOptions.IPIPEnabled = false
		topologyOptions.IPIPRoutesEnabled = false
		topologyOptions.ExtraEnvVars["FELIX_EGRESSIPSUPPORT"] = supportLevel
		felix, client = infrastructure.StartSingleNodeTopology(topologyOptions, infra)

		// Install a default profile that allows all ingress and egress, in the absence of any Policy.
		infra.AddDefaultAllow()

		// Create the normal IP pool.
		ctx := context.Background()
		ippool := api.NewIPPool()
		ippool.Name = "test-pool"
		ippool.Spec.CIDR = "10.65.0.0/16"
		ippool.Spec.NATOutgoing = false
		_, err = client.IPPools().Create(ctx, ippool, options.SetOptions{})
		Expect(err).NotTo(HaveOccurred())

		// Create an egress IP pool.
		ippool = api.NewIPPool()
		ippool.Name = "egress-pool"
		ippool.Spec.CIDR = "10.10.10.0/29"
		ippool.Spec.NATOutgoing = false
		ippool.Spec.BlockSize = 29
		ippool.Spec.NodeSelector = "!all()"
		_, err = client.IPPools().Create(ctx, ippool, options.SetOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	expectedRoute := func(ips ...string) string {
		if len(ips) == 0 {
			return "unreachable default scope link"
		} else if len(ips) == 1 {
			return "default via " + ips[0] + " dev egress.calico onlink"
		} else {
			r := "default onlink \n"
			for _, ip := range ips {
				r += "\tnexthop via " + ip + " dev egress.calico weight 1 onlink \n"
			}
			return strings.TrimSpace(r)
		}
	}

	Context("EnabledPerNamespaceOrPerPod", func() {
		BeforeEach(func() {
			supportLevel = "EnabledPerNamespaceOrPerPod"
		})

		It("keeps gateway device route when client goes away", func() {
			By("Create a gateway and client")
			gw := makeGateway("10.10.10.1", "gw1")
			defer gw.Stop()
			app := makeClient("10.65.0.2", "app")
			appExists := true
			defer func() {
				if appExists {
					app.Stop()
				}
			}()

			By("Check gateway route exists")
			checkGatewayRoute := func() (err error) {
				routes, err := felix.ExecOutput("ip", "r")
				if err != nil {
					return
				}
				for _, route := range strings.Split(routes, "\n") {
					if matched, _ := regexp.MatchString("^10.10.10.1 dev cali", route); matched {
						return
					}
				}
				return fmt.Errorf("10.10.10.1 device route is not present in:\n%v", routes)
			}
			Eventually(checkGatewayRoute, "10s", "1s").Should(Succeed())

			By("Remove the client again")
			app.RemoveFromInfra(infra)
			app.Stop()
			appExists = false

			By("Check gateway route still present")
			Expect(checkGatewayRoute()).To(Succeed())
			Consistently(checkGatewayRoute, "5s", "1s").Should(Succeed())
		})

		It("updates rules and routing as gateways are added and removed", func() {
			By("Create a gateway.")
			gw := makeGateway("10.10.10.1", "gw1")
			defer gw.Stop()

			By("No egress ip rules expected yet.")
			Consistently(getIPRules).Should(BeEmpty())

			By("Create a client.")
			app := makeClient("10.65.0.2", "app")
			defer app.Stop()

			By("Check ip rules.")
			Eventually(getIPRules, "10s", "1s").Should(HaveLen(1))
			Eventually(getIPRules, "10s", "1s").Should(HaveKey("10.65.0.2"))
			table1 := getIPRules()["10.65.0.2"]

			By("Check ip routes.")
			Eventually(func() string {
				return getIPRoute(table1)
			}, "10s", "1s").Should(Equal(expectedRoute("10.10.10.1")))
			Consistently(func() string {
				return getIPRoute(table1)
			}).Should(Equal(expectedRoute("10.10.10.1")))

			By("Check L2.")
			Expect(getIPNeigh()).To(Equal(map[string]string{
				"10.10.10.1": "a2:2a:0a:0a:0a:01",
			}))
			Expect(getBridgeFDB()).To(Equal(map[string]string{
				"a2:2a:0a:0a:0a:01": "10.10.10.1",
			}))

			By("Create another client.")
			app2 := makeClient("10.65.0.3", "app2")
			defer app2.Stop()

			By("Check ip rules and routes.")
			Eventually(getIPRules, "10s", "1s").Should(Equal(map[string]string{"10.65.0.2": table1, "10.65.0.3": table1}))
			Consistently(func() string {
				return getIPRoute(table1)
			}).Should(Equal(expectedRoute("10.10.10.1")))

			By("Check L2.")
			Expect(getIPNeigh()).To(Equal(map[string]string{
				"10.10.10.1": "a2:2a:0a:0a:0a:01",
			}))
			Expect(getBridgeFDB()).To(Equal(map[string]string{
				"a2:2a:0a:0a:0a:01": "10.10.10.1",
			}))

			By("Create another gateway.")
			gw2 := makeGateway("10.10.10.2", "gw2")
			defer gw2.Stop()

			By("Check ip rules and routes.")
			Eventually(getIPRules, "10s", "1s").Should(Equal(map[string]string{"10.65.0.2": table1, "10.65.0.3": table1}))
			Consistently(func() string {
				return getIPRoute(table1)
			}).Should(Equal(expectedRoute("10.10.10.1", "10.10.10.2")))

			By("Check L2.")
			Expect(getIPNeigh()).To(Equal(map[string]string{
				"10.10.10.1": "a2:2a:0a:0a:0a:01",
				"10.10.10.2": "a2:2a:0a:0a:0a:02",
			}))
			Expect(getBridgeFDB()).To(Equal(map[string]string{
				"a2:2a:0a:0a:0a:01": "10.10.10.1",
				"a2:2a:0a:0a:0a:02": "10.10.10.2",
			}))

			By("Create 3rd gateway.")
			gw3 := makeGateway("10.10.10.3", "gw3")
			defer gw3.Stop()

			By("Check ip rules and routes.")
			Eventually(getIPRules, "10s", "1s").Should(Equal(map[string]string{"10.65.0.2": table1, "10.65.0.3": table1}))
			Eventually(func() string {
				return getIPRoute(table1)
			}, "10s", "1s").Should(Equal(expectedRoute("10.10.10.1", "10.10.10.2", "10.10.10.3")))

			By("Check L2.")
			Expect(getIPNeigh()).To(Equal(map[string]string{
				"10.10.10.1": "a2:2a:0a:0a:0a:01",
				"10.10.10.2": "a2:2a:0a:0a:0a:02",
				"10.10.10.3": "a2:2a:0a:0a:0a:03",
			}))
			Expect(getBridgeFDB()).To(Equal(map[string]string{
				"a2:2a:0a:0a:0a:01": "10.10.10.1",
				"a2:2a:0a:0a:0a:02": "10.10.10.2",
				"a2:2a:0a:0a:0a:03": "10.10.10.3",
			}))

			By("Remove 3rd gateway again.")
			gw3.RemoveFromInfra(infra)

			By("Check ip rules and routes.")
			Eventually(getIPRules, "10s", "1s").Should(Equal(map[string]string{"10.65.0.2": table1, "10.65.0.3": table1}))
			Eventually(func() string {
				return getIPRoute(table1)
			}, "10s", "1s").Should(Equal(expectedRoute("10.10.10.1", "10.10.10.2")))

			By("Check L2.")
			Expect(getIPNeigh()).To(Equal(map[string]string{
				"10.10.10.1": "a2:2a:0a:0a:0a:01",
				"10.10.10.2": "a2:2a:0a:0a:0a:02",
			}))
			Expect(getBridgeFDB()).To(Equal(map[string]string{
				"a2:2a:0a:0a:0a:01": "10.10.10.1",
				"a2:2a:0a:0a:0a:02": "10.10.10.2",
			}))

			By("Remove the first gateway.")
			gw.RemoveFromInfra(infra)

			By("Check ip rules and routes.")
			Eventually(getIPRules, "10s", "1s").Should(Equal(map[string]string{"10.65.0.2": table1, "10.65.0.3": table1}))
			Eventually(func() string {
				return getIPRoute(table1)
			}, "10s", "1s").Should(Equal(expectedRoute("10.10.10.2")))

			By("Check L2.")
			Expect(getIPNeigh()).To(Equal(map[string]string{
				"10.10.10.2": "a2:2a:0a:0a:0a:02",
			}))
			Expect(getBridgeFDB()).To(Equal(map[string]string{
				"a2:2a:0a:0a:0a:02": "10.10.10.2",
			}))

			By("Remove the second gateway.")
			gw2.RemoveFromInfra(infra)

			By("Check ip rules and routes.")
			Consistently(getIPRules, "5s", "1s").Should(Equal(map[string]string{"10.65.0.2": table1, "10.65.0.3": table1}))
			Eventually(func() string {
				return getIPRoute(table1)
			}, "10s", "1s").Should(Equal(expectedRoute()))

			By("Check L2.")
			Expect(getIPNeigh()).To(Equal(map[string]string{}))
			Expect(getBridgeFDB()).To(Equal(map[string]string{}))
		})
	})

	Context("Disabled", func() {
		BeforeEach(func() {
			supportLevel = "Disabled"
		})

		It("does nothing when egress IP is disabled", func() {
			By("Create a gateway.")
			gw := makeGateway("10.10.10.1", "gw1")
			defer gw.Stop()

			By("Create a client.")
			app := makeClient("10.65.0.2", "app")
			defer app.Stop()

			By("Should be no ip rules.")
			Consistently(getIPRules, "5s", "1s").Should(BeEmpty())
		})
	})

	Context("EnabledPerNamespace", func() {
		BeforeEach(func() {
			supportLevel = "EnabledPerNamespace"
		})

		It("honours namespace annotations but not per-pod", func() {
			By("Create a gateway.")
			gw := makeGateway("10.10.10.1", "gw1")
			defer gw.Stop()

			By("Create a client.")
			app := makeClient("10.65.0.2", "app")
			defer app.Stop()

			By("Should be no ip rules.")
			Consistently(getIPRules, "5s", "1s").Should(BeEmpty())

			By("Add egress annotations to the default namespace.")
			coreV1 := infra.(*infrastructure.K8sDatastoreInfra).K8sClient.CoreV1()
			ns, err := coreV1.Namespaces().Get(context.Background(), app.WorkloadEndpoint.Namespace, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			if ns.Annotations == nil {
				ns.Annotations = map[string]string{}
			}
			ns.Annotations["egress.projectcalico.org/selector"] = "egress-code == 'red'"
			_, err = coreV1.Namespaces().Update(context.Background(), ns, metav1.UpdateOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Check ip rules.")
			// (In this example the gateway is also in the default namespace, but is
			// prevented from looping around to itself (or to any other gateway) because
			// it is an egress gateway itself.)
			Eventually(getIPRules, "10s", "1s").Should(HaveLen(1))
			rules := getIPRules()
			Expect(rules).To(HaveKey("10.65.0.2"))
			table1 := rules["10.65.0.2"]

			By("Check ip routes.")
			Eventually(func() string {
				return getIPRoute(table1)
			}, "10s", "1s").Should(Equal(expectedRoute("10.10.10.1")))
			Consistently(func() string {
				return getIPRoute(table1)
			}).Should(Equal(expectedRoute("10.10.10.1")))
		})
	})

	AfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			felix.Exec("iptables-save", "-c")
			felix.Exec("ipset", "list")
			felix.Exec("ip", "r")
			felix.Exec("ip", "a")
		}

		felix.Stop()

		if CurrentGinkgoTestDescription().Failed {
			infra.DumpErrorData()
		}
		infra.Stop()
	})
})
