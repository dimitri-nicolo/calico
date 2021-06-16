// +build fvtests

// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package fv_test

import (
	"context"
	"fmt"
	"strconv"
	"time"

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

var _ = describeTProxyTest(false)
var _ = describeTProxyTest(true)

func describeTProxyTest(ipip bool) bool {

	tunnel := "none"
	if ipip {
		tunnel = "ipip"
	}

	return infrastructure.DatastoreDescribe("tproxy tests tunnel="+tunnel,
		[]apiconfig.DatastoreType{apiconfig.Kubernetes}, func(getInfra infrastructure.InfraFactory) {

			const numNodes = 2

			var (
				infra        infrastructure.DatastoreInfra
				felixes      []*infrastructure.Felix
				proxies      []*tproxy.TProxy
				cc           *Checker
				options      infrastructure.TopologyOptions
				calicoClient client.Interface
				w            [numNodes][2]*workload.Workload
			)

			BeforeEach(func() {
				options = infrastructure.DefaultTopologyOptions()

				cc = &Checker{
					CheckSNAT: true,
				}
				cc.Protocol = "tcp"

				options.FelixLogSeverity = "debug"
				options.NATOutgoingEnabled = true
				options.AutoHEPsEnabled = true

				if !ipip {
					options.IPIPEnabled = false
					options.IPIPRoutesEnabled = false
				}

				options.ExtraEnvVars["FELIX_TPROXYMODE"] = "Enabled"

				// XXX this is temporary until service annotation does the job
				options.ExtraEnvVars["FELIX_IPSETSREFRESHINTERVAL"] = "0"
			})

			createPolicy := func(policy *api.GlobalNetworkPolicy) *api.GlobalNetworkPolicy {
				log.WithField("policy", dumpResource(policy)).Info("Creating policy")
				policy, err := calicoClient.GlobalNetworkPolicies().Create(utils.Ctx, policy, utils.NoOptions)
				Expect(err).NotTo(HaveOccurred())
				return policy
			}

			JustBeforeEach(func() {
				infra = getInfra()

				felixes, calicoClient = infrastructure.StartNNodeTopology(numNodes, options, infra)

				proxies = []*tproxy.TProxy{}
				for _, felix := range felixes {
					proxy := tproxy.New(felix, 16001)
					proxy.Start()
					proxies = append(proxies, proxy)
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
				hundred := float64(100)
				pol.Spec.Order = &hundred

				pol = createPolicy(pol)

				k8sClient = infra.(*infrastructure.K8sDatastoreInfra).K8sClient
				_ = k8sClient
			})

			// XXX this is temporary method until service annotation does the job
			ipsetsUpdate := func(value string) {
				// XXX need to give felix enough time to be executed _after_ it
				// XXX syncs the ipsets for the first time
				time.Sleep(15 * time.Second)
				for _, felix := range felixes {
					err := felix.ExecMayFail("ipset", "add", "cali40tproxy-services", value)
					Expect(err).NotTo(HaveOccurred())
				}
			}

			JustAfterEach(func() {
				for _, p := range proxies {
					p.Stop()
				}

				if CurrentGinkgoTestDescription().Failed {
					for _, felix := range felixes {
						felix.Exec("iptables-save", "-c")
						felix.Exec("ipset", "list", "cali40tproxy-services")
						felix.Exec("ip", "route")
					}
				}
			})

			AfterEach(func() {
				log.Info("AfterEach starting")
				infra.Stop()
				log.Info("AfterEach done")
			})

			Context("Pod-Pod", func() {
				It("should have connectivity from all workloads via w[0][0].IP", func() {
					cc.Expect(Some, w[0][1], w[0][0], ExpectWithPorts(8055))
					cc.Expect(Some, w[1][0], w[0][0], ExpectWithPorts(8055))
					cc.Expect(Some, w[1][1], w[0][0], ExpectWithPorts(8055))
					cc.CheckConnectivity()
				})
			})

			Context("ClusterIP", func() {
				clusterIP := "10.101.0.10"

				var pod, svc string

				JustBeforeEach(func() {
					pod = w[0][0].IP + ":8055"
					svc = clusterIP + ":8090"

					// Mimic the kube-proxy service iptable clusterIP rule.
					for _, f := range felixes {
						f.Exec("iptables", "-t", "nat", "-A", "PREROUTING",
							"-w", "10", // Retry this for 10 seconds, e.g. if something else is holding the lock
							"-W", "100000", // How often to probe the lock in microsecs.
							"-p", "tcp",
							"-d", clusterIP,
							"-m", "tcp", "--dport", "8090",
							"-j", "DNAT", "--to-destination",
							pod)
					}

				})

				It("should have connectivity from all workloads via ClusterIP", func() {
					ipsetsUpdate("10.101.0.10,tcp:8090")

					cc.Expect(Some, w[0][1], TargetIP(clusterIP), ExpectWithPorts(8090))
					cc.Expect(Some, w[1][0], TargetIP(clusterIP), ExpectWithPorts(8090))
					cc.Expect(Some, w[1][1], TargetIP(clusterIP), ExpectWithPorts(8090))
					cc.CheckConnectivity()

					// Connection should be proxied on the pod's local node
					Eventually(proxies[0].ProxiedCount(w[0][1].IP, pod, svc)).Should(BeNumerically(">", 0))
					Eventually(proxies[1].ProxiedCount(w[1][0].IP, pod, svc)).Should(BeNumerically(">", 0))
					Eventually(proxies[1].ProxiedCount(w[1][1].IP, pod, svc)).Should(BeNumerically(">", 0))

					// Connection should not be proxied on the backend pod's node
					Eventually(proxies[0].ProxiedCount(w[1][0].IP, pod, svc)).Should(Equal(0))
					Eventually(proxies[0].ProxiedCount(w[1][1].IP, pod, svc)).Should(Equal(0))
				})

				Context("With ingress traffic denied from w[0][1] and w[1][1]", func() {
					It("should have connectivity only from w[1][0]", func() {
						By("Denying traffic from w[1][1]", func() {
							pol := api.NewGlobalNetworkPolicy()
							pol.Namespace = "fv"
							pol.Name = "policy-deny-1-1"
							pol.Spec.Ingress = []api.Rule{
								{
									Action: "Deny",
									Source: api.EntityRule{
										Selector: "(name=='" + w[1][1].Name + "') || (name=='" + w[0][1].Name + "')",
									},
								},
							}
							pol.Spec.Selector = "name=='" + w[0][0].Name + "'"
							one := float64(1)
							pol.Spec.Order = &one

							pol = createPolicy(pol)
						})

						ipsetsUpdate("10.101.0.10,tcp:8090")

						cc.Expect(None, w[0][1], TargetIP(clusterIP), ExpectWithPorts(8090))
						cc.Expect(Some, w[1][0], TargetIP(clusterIP), ExpectWithPorts(8090))
						cc.Expect(None, w[1][1], TargetIP(clusterIP), ExpectWithPorts(8090))
						cc.CheckConnectivity()

						// Connection should be proxied on the pod's local node

						Eventually(proxies[0].AcceptedCount(w[0][1].IP, pod, svc)).Should(BeNumerically(">", 0))
						Eventually(proxies[1].AcceptedCount(w[1][0].IP, pod, svc)).Should(BeNumerically(">", 0))
						Eventually(proxies[1].AcceptedCount(w[1][1].IP, pod, svc)).Should(BeNumerically(">", 0))

						Eventually(proxies[0].ProxiedCount(w[0][1].IP, pod, svc)).Should(Equal(0))
						Eventually(proxies[1].ProxiedCount(w[1][0].IP, pod, svc)).Should(BeNumerically(">", 0))
						Eventually(proxies[1].ProxiedCount(w[1][1].IP, pod, svc)).Should(Equal(0))
					})
				})

				Context("With egress traffic denied from w[1][1]", func() {
					It("should have connectivity only from w[0][1] and w[1][0]", func() {
						By("Denying traffic from w[1][1]", func() {
							pol := api.NewGlobalNetworkPolicy()
							pol.Namespace = "fv"
							pol.Name = "policy-deny-1-1"
							pol.Spec.Egress = []api.Rule{
								{
									Action: "Deny",
									Destination: api.EntityRule{
										Selector: "name=='" + w[0][0].Name + "'",
									},
								},
							}
							pol.Spec.Selector = "name=='" + w[1][1].Name + "'"
							one := float64(1)
							pol.Spec.Order = &one

							pol = createPolicy(pol)
						})

						ipsetsUpdate("10.101.0.10,tcp:8090")

						cc.Expect(Some, w[0][1], TargetIP(clusterIP), ExpectWithPorts(8090))
						cc.Expect(Some, w[1][0], TargetIP(clusterIP), ExpectWithPorts(8090))
						cc.Expect(None, w[1][1], TargetIP(clusterIP), ExpectWithPorts(8090))
						cc.CheckConnectivity()

						// Connection should be proxied on the pod's local node

						Eventually(proxies[0].AcceptedCount(w[0][1].IP, pod, svc)).Should(BeNumerically(">", 0))
						Eventually(proxies[1].AcceptedCount(w[1][0].IP, pod, svc)).Should(BeNumerically(">", 0))
						Eventually(proxies[1].AcceptedCount(w[1][1].IP, pod, svc)).Should(Equal(0))

						Eventually(proxies[0].ProxiedCount(w[0][1].IP, pod, svc)).Should(BeNumerically(">", 0))
						Eventually(proxies[1].ProxiedCount(w[1][0].IP, pod, svc)).Should(BeNumerically(">", 0))
						Eventually(proxies[1].ProxiedCount(w[1][1].IP, pod, svc)).Should(Equal(0))
					})
				})
			})
		})
}
