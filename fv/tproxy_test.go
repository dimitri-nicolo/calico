// +build fvtests

// Copyright (c) 2017-2021 Tigera, Inc. All rights reserved.
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
	"context"
	"fmt"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"

	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/ipam"
	cnet "github.com/projectcalico/libcalico-go/lib/net"

	. "github.com/projectcalico/felix/fv/connectivity"
	"github.com/projectcalico/felix/fv/infrastructure"
	"github.com/projectcalico/felix/fv/tproxy"
	"github.com/projectcalico/felix/fv/utils"
	"github.com/projectcalico/felix/fv/workload"
)

var _ = infrastructure.DatastoreDescribe("tproxy tests",
	[]apiconfig.DatastoreType{apiconfig.Kubernetes}, func(getInfra infrastructure.InfraFactory) {

		var (
			infra        infrastructure.DatastoreInfra
			felixes      []*infrastructure.Felix
			proxies      []*tproxy.TProxy
			cc           *Checker
			options      infrastructure.TopologyOptions
			calicoClient client.Interface
		)

		BeforeEach(func() {
			infra = getInfra()

			cc = &Checker{
				CheckSNAT: true,
			}
			cc.Protocol = "tcp"

			options = infrastructure.DefaultTopologyOptions()
			options.FelixLogSeverity = "debug"
			options.NATOutgoingEnabled = true
			options.AutoHEPsEnabled = true
			// override IPIP being enabled by default
			options.IPIPEnabled = false
			options.IPIPRoutesEnabled = false

			options.ExtraEnvVars["FELIX_TPROXYMODE"] = "Enabled"
		})

		JustAfterEach(func() {
			if CurrentGinkgoTestDescription().Failed {
				for _, felix := range felixes {
					felix.Exec("iptables-save", "-c")
					felix.Exec("ipset", "list", "cali40tproxy-services")
					felix.Exec("ip", "rule")
					felix.Exec("ip", "route")
					felix.Exec("ip", "route", "show", "table", "224")
					felix.Exec("ip", "route", "show", "cached")
				}
			}
		})

		AfterEach(func() {
			log.Info("AfterEach starting")
			infra.Stop()
			log.Info("AfterEach done")
		})

		const numNodes = 2

		var (
			w [numNodes][2]*workload.Workload
		)

		BeforeEach(func() {
			felixes, calicoClient = infrastructure.StartNNodeTopology(numNodes, options, infra)

			for _, felix := range felixes {
				proxy := tproxy.New(felix, 16001)
				proxy.Start()
				proxies = append(proxies, proxy)
			}

			createPolicy := func(policy *api.GlobalNetworkPolicy) *api.GlobalNetworkPolicy {
				log.WithField("policy", dumpResource(policy)).Info("Creating policy")
				policy, err := calicoClient.GlobalNetworkPolicies().Create(utils.Ctx, policy, utils.NoOptions)
				Expect(err).NotTo(HaveOccurred())
				return policy
			}

			addWorkload := func(run bool, ii, wi, port int, labels map[string]string) *workload.Workload {
				if labels == nil {
					labels = make(map[string]string)
				}

				wIP := fmt.Sprintf("10.65.%d.%d", ii, wi+2)
				wName := fmt.Sprintf("w%d%d", ii, wi)

				w := workload.New(felixes[ii], wName, "default",
					wIP, strconv.Itoa(port), "tcp")
				if run {
					w.Start()
				}

				labels["name"] = w.Name
				labels["workload"] = "regular"

				w.WorkloadEndpoint.Labels = labels
				w.ConfigureInInfra(infra)
				if options.UseIPPools {
					// Assign the workload's IP in IPAM, this will trigger calculation of routes.
					err := calicoClient.IPAM().AssignIP(context.Background(), ipam.AssignIPArgs{
						IP:       cnet.MustParseIP(wIP),
						HandleID: &w.Name,
						Attrs: map[string]string{
							ipam.AttributeNode: felixes[ii].Hostname,
						},
						Hostname: felixes[ii].Hostname,
					})
					Expect(err).NotTo(HaveOccurred())
				}

				return w
			}

			for ii := range felixes {
				// Two workloads on each host so we can check the same host and other host cases.
				w[ii][0] = addWorkload(true, ii, 0, 8055, nil)
				w[ii][1] = addWorkload(true, ii, 1, 8055, nil)
			}

			var (
				pol       *api.GlobalNetworkPolicy
				k8sClient *kubernetes.Clientset
			)

			pol = api.NewGlobalNetworkPolicy()
			pol.Namespace = "fv"
			pol.Name = "policy-1"
			pol.Spec.Ingress = []api.Rule{
				{
					Action: "Allow",
					Source: api.EntityRule{
						Selector: "workload=='regular'",
					},
				},
			}
			pol.Spec.Egress = []api.Rule{
				{
					Action: "Allow",
					Source: api.EntityRule{
						Selector: "workload=='regular'",
					},
				},
			}
			pol.Spec.Selector = "workload=='regular'"

			pol = createPolicy(pol)

			k8sClient = infra.(*infrastructure.K8sDatastoreInfra).K8sClient
			_ = k8sClient
		})

		Context("ClusterIP", func() {
			clusterIP := "10.101.0.10"
			BeforeEach(func() {
				// Mimic the kube-proxy service iptable clusterIP rule.
				for _, f := range felixes {
					f.Exec("iptables", "-t", "nat", "-A", "PREROUTING",
						"-p", "tcp",
						"-d", clusterIP,
						"-m", "tcp", "--dport", "8090",
						"-j", "DNAT", "--to-destination",
						w[0][0].IP+":8055")
				}
			})

			It("connectivity from all workloads via ClusterIP", func() {
				cc.ExpectSome(w[0][1], TargetIP(clusterIP), 8090)
				cc.ExpectSome(w[1][0], TargetIP(clusterIP), 8090)
				cc.ExpectSome(w[1][1], TargetIP(clusterIP), 8090)
				cc.CheckConnectivity()

				for _, p := range proxies {
					p.Stop()
				}

				// Connection should be proxied on the pod's local node
				Expect(proxies[0].ConnCount(w[0][1].IP, w[0][0].IP+":8055", clusterIP+":8090")).To(BeNumerically(">", 0))
				Expect(proxies[1].ConnCount(w[1][0].IP, w[0][0].IP+":8055", clusterIP+":8090")).To(BeNumerically(">", 0))
				Expect(proxies[1].ConnCount(w[1][1].IP, w[0][0].IP+":8055", clusterIP+":8090")).To(BeNumerically(">", 0))

				// Connection should not be proxied on the backend pod's node
				Expect(proxies[0].ConnCount(w[1][0].IP, w[0][0].IP+":8055", clusterIP+":8090")).To(Equal(0))
				Expect(proxies[0].ConnCount(w[1][1].IP, w[0][0].IP+":8055", clusterIP+":8090")).To(Equal(0))
			})
		})
	})
