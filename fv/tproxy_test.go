// +build fvtests
//Copyright (c) 2021 Tigera, Inc. All rights reserved.

package fv_test

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	. "github.com/projectcalico/felix/fv/connectivity"
	"github.com/projectcalico/felix/fv/infrastructure"
	"github.com/projectcalico/felix/fv/tproxy"
	"github.com/projectcalico/felix/fv/utils"
	"github.com/projectcalico/felix/fv/workload"
	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	api "github.com/projectcalico/tigera/api/pkg/apis/projectcalico/v3"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/ipam"
	cnet "github.com/projectcalico/libcalico-go/lib/net"
)

var _ = describeTProxyTest(false, "Enabled")
var _ = describeTProxyTest(true, "Enabled")
var _ = describeTProxyTest(false, "EnabledAllServices")

func describeTProxyTest(ipip bool, TPROXYMode string) bool {
	const l7LoggingAnnotation = "projectcalico.org/l7-logging"
	const TPROXYServicesIPSetV4 = "cali40tproxy-services"
	const TPROXYPodToSelf = "cali40tproxy-pod-self"

	var expectedFelix0IP ExpectationOption

	tunnel := "none"
	if ipip {
		tunnel = "ipip"
	}

	return infrastructure.DatastoreDescribe("tproxy tests tunnel="+tunnel+" TPROXYMode="+TPROXYMode,
		[]apiconfig.DatastoreType{apiconfig.Kubernetes}, func(getInfra infrastructure.InfraFactory) {

			const numNodes = 2
			const clusterIP = "10.101.0.10"

			var (
				infra        infrastructure.DatastoreInfra
				felixes      []*infrastructure.Felix
				proxies      []*tproxy.TProxy
				cc           *Checker
				options      infrastructure.TopologyOptions
				calicoClient client.Interface
				w            [numNodes][2]*workload.Workload
				clientset    *kubernetes.Clientset
			)

			createPolicy := func(policy *api.GlobalNetworkPolicy) *api.GlobalNetworkPolicy {
				log.WithField("policy", dumpResource(policy)).Info("Creating policy")
				policy, err := calicoClient.GlobalNetworkPolicies().Create(utils.Ctx, policy, utils.NoOptions)
				Expect(err).NotTo(HaveOccurred())
				return policy
			}

			createService := func(service *v1.Service, client *kubernetes.Clientset) *v1.Service {
				log.WithField("service", dumpResource(service)).Info("Creating service")
				svc, err := client.CoreV1().Services(service.ObjectMeta.Namespace).Create(context.Background(), service, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
				Eventually(k8sGetEpsForServiceFunc(client, service), "10s").Should(HaveLen(1),
					"Service endpoints didn't get created? Is controller-manager happy?")
				log.WithField("service", dumpResource(svc)).Info("created service")

				return svc
			}

			assertIPPortInIPSet := func(ipSetID string, ip string, port string, felixes []*infrastructure.Felix, exists bool) {
				results := make(map[*infrastructure.Felix]struct{}, len(felixes))
				Eventually(func() bool {
					for _, felix := range felixes {
						out, err := felix.ExecOutput("ipset", "list", ipSetID)
						log.Infof("felix ipset list output %s", out)
						Expect(err).NotTo(HaveOccurred())
						if (strings.Contains(out, ip) && strings.Contains(out, port)) == exists {
							results[felix] = struct{}{}
						}
					}
					return len(results) == len(felixes)
				}, "90s", "5s").Should(BeTrue())
			}

			BeforeEach(func() {
				options = infrastructure.DefaultTopologyOptions()

				cc = &Checker{
					CheckSNAT: true,
				}
				cc.Protocol = "tcp"

				options.NATOutgoingEnabled = true
				options.AutoHEPsEnabled = true

				// XXX until we can safely remove roting rules and not break other tests
				options.EnableIPv6 = false

				if !ipip {
					options.IPIPEnabled = false
					options.IPIPRoutesEnabled = false
				}

				options.ExtraEnvVars["FELIX_TPROXYMODE"] = TPROXYMode
				// XXX until we can safely remove roting rules and not break other tests
				options.EnableIPv6 = false

				infra = getInfra()
				clientset = infra.(*infrastructure.K8sDatastoreInfra).K8sClient

				felixes, calicoClient = infrastructure.StartNNodeTopology(numNodes, options, infra)

				if !ipip {
					expectedFelix0IP = ExpectWithSrcIPs(felixes[0].IP)
				} else {
					expectedFelix0IP = ExpectWithSrcIPs(felixes[0].ExpectedIPIPTunnelAddr)
				}

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

				// Make sure the ipsets exist before we do testing. This means
				// that we can sync with the content of the maps.
				Eventually(func() bool {
					for _, felix := range felixes {
						if _, err := felix.ExecOutput("ipset", "list", "cali40tproxy-services"); err != nil {
							return false
						}
					}
					return true
				}, "20s", "1s").Should(BeTrue())

			})

			JustAfterEach(func() {
				for _, p := range proxies {
					p.Stop()
				}

				if CurrentGinkgoTestDescription().Failed {
					for _, felix := range felixes {
						felix.Exec("iptables-save", "-c")
						felix.Exec("ipset", "list", TPROXYServicesIPSetV4)
						felix.Exec("ipset", "list", TPROXYPodToSelf)
						felix.Exec("ip", "route")
					}
				}
			})

			AfterEach(func() {
				log.Info("AfterEach starting")
				for _, f := range felixes {
					f.Stop()
				}
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

				var pod, svc string

				JustBeforeEach(func() {
					pod = w[0][0].IP + ":8055"
					svc = clusterIP + ":8090"

					for _, f := range felixes {
						// Mimic the kube-proxy service iptable clusterIP rule.
						f.Exec("iptables", "-t", "nat", "-A", "PREROUTING",
							"-w", "10", // Retry this for 10 seconds, e.g. if something else is holding the lock
							"-W", "100000", // How often to probe the lock in microsecs.
							"-p", "tcp",
							"-d", clusterIP,
							"-m", "tcp", "--dport", "8090",
							"-j", "DNAT", "--to-destination",
							pod)
						// Mimic the kube-proxy MASQ rule based on the kube-proxy bit
						f.Exec("iptables", "-t", "nat", "-A", "POSTROUTING",
							"-w", "10", // Retry this for 10 seconds, e.g. if something else is holding the lock
							"-W", "100000", // How often to probe the lock in microsecs.
							"-m", "mark", "--mark", "0x4000/0x4000", // 0x4000 is the deault --iptables-masquerade-bit
							"-j", "MASQUERADE", "--random-fully")
					}
					// for this context create service before each test
					v1Svc := k8sService("service-with-annotation", clusterIP, w[0][0], 8090, 8055, 0, "tcp")
					if TPROXYMode == "Enabled" {
						v1Svc.ObjectMeta.Annotations = map[string]string{l7LoggingAnnotation: "true"}
					}
					createService(v1Svc, clientset)
				})

				It("should have connectivity from all workloads via ClusterIP", func() {

					By("asserting that ipaddress, port exists in ipset ")
					assertIPPortInIPSet(TPROXYServicesIPSetV4, clusterIP, "8090", felixes, true)

					cc.Expect(Some, w[0][0], TargetIP(clusterIP), ExpectWithPorts(8090), expectedFelix0IP)
					cc.Expect(Some, w[0][1], TargetIP(clusterIP), ExpectWithPorts(8090))
					cc.Expect(Some, w[1][0], TargetIP(clusterIP), ExpectWithPorts(8090))
					cc.Expect(Some, w[1][1], TargetIP(clusterIP), ExpectWithPorts(8090))
					cc.CheckConnectivity()

					// Connection should be proxied on the pod's local node
					Eventually(proxies[0].ProxiedCountFn(w[0][0].IP, pod, svc)).Should(BeNumerically(">", 0))
					Eventually(proxies[0].ProxiedCountFn(w[0][1].IP, pod, svc)).Should(BeNumerically(">", 0))
					Eventually(proxies[1].ProxiedCountFn(w[1][0].IP, pod, svc)).Should(BeNumerically(">", 0))
					Eventually(proxies[1].ProxiedCountFn(w[1][1].IP, pod, svc)).Should(BeNumerically(">", 0))

					// Connection should not be proxied on the backend pod's node
					Eventually(proxies[0].ProxiedCountFn(w[1][0].IP, pod, svc)).Should(Equal(0))
					Eventually(proxies[0].ProxiedCountFn(w[1][1].IP, pod, svc)).Should(Equal(0))
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
						By("asserting that ipaddress, port exists in ipset ")
						assertIPPortInIPSet(TPROXYServicesIPSetV4, clusterIP, "8090", felixes, true)

						cc.Expect(Some, w[0][0], TargetIP(clusterIP), ExpectWithPorts(8090), expectedFelix0IP)
						cc.Expect(None, w[0][1], TargetIP(clusterIP), ExpectWithPorts(8090))
						cc.Expect(Some, w[1][0], TargetIP(clusterIP), ExpectWithPorts(8090))
						cc.Expect(None, w[1][1], TargetIP(clusterIP), ExpectWithPorts(8090))
						cc.CheckConnectivity()

						// Connection should be proxied on the pod's local node

						Eventually(proxies[0].AcceptedCountFn(w[0][0].IP, pod, svc)).Should(BeNumerically(">", 0))
						Eventually(proxies[0].AcceptedCountFn(w[0][1].IP, pod, svc)).Should(BeNumerically(">", 0))
						Eventually(proxies[1].AcceptedCountFn(w[1][0].IP, pod, svc)).Should(BeNumerically(">", 0))
						Eventually(proxies[1].AcceptedCountFn(w[1][1].IP, pod, svc)).Should(BeNumerically(">", 0))

						Eventually(proxies[0].ProxiedCountFn(w[0][0].IP, pod, svc)).Should(BeNumerically(">", 0))
						Eventually(proxies[0].ProxiedCountFn(w[0][1].IP, pod, svc)).Should(Equal(0))
						Eventually(proxies[1].ProxiedCountFn(w[1][0].IP, pod, svc)).Should(BeNumerically(">", 0))
						Eventually(proxies[1].ProxiedCountFn(w[1][1].IP, pod, svc)).Should(Equal(0))
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

						By("asserting that ipaddress, port exists in ipset ")
						assertIPPortInIPSet(TPROXYServicesIPSetV4, clusterIP, "8090", felixes, true)

						cc.Expect(Some, w[0][0], TargetIP(clusterIP), ExpectWithPorts(8090), expectedFelix0IP)
						cc.Expect(Some, w[0][1], TargetIP(clusterIP), ExpectWithPorts(8090))
						cc.Expect(Some, w[1][0], TargetIP(clusterIP), ExpectWithPorts(8090))
						cc.Expect(None, w[1][1], TargetIP(clusterIP), ExpectWithPorts(8090))
						cc.CheckConnectivity()

						// Connection should be proxied on the pod's local node

						Eventually(proxies[0].AcceptedCountFn(w[0][0].IP, pod, svc)).Should(BeNumerically(">", 0))
						Eventually(proxies[0].AcceptedCountFn(w[0][1].IP, pod, svc)).Should(BeNumerically(">", 0))
						Eventually(proxies[1].AcceptedCountFn(w[1][0].IP, pod, svc)).Should(BeNumerically(">", 0))
						Eventually(proxies[1].AcceptedCountFn(w[1][1].IP, pod, svc)).Should(Equal(0))

						Eventually(proxies[0].ProxiedCountFn(w[0][0].IP, pod, svc)).Should(BeNumerically(">", 0))
						Eventually(proxies[0].ProxiedCountFn(w[0][1].IP, pod, svc)).Should(BeNumerically(">", 0))
						Eventually(proxies[1].ProxiedCountFn(w[1][0].IP, pod, svc)).Should(BeNumerically(">", 0))
						Eventually(proxies[1].ProxiedCountFn(w[1][1].IP, pod, svc)).Should(Equal(0))
					})
				})

				Context("With ingress traffic denied from w[0]0] to self", func() {
					It("should have connectivity from all but self", func() {
						By("Denying traffic from w[1][1]", func() {
							pol := api.NewGlobalNetworkPolicy()
							pol.Namespace = "fv"
							pol.Name = "policy-deny-1-1"
							pol.Spec.Ingress = []api.Rule{
								{
									Action: "Deny",
									Source: api.EntityRule{
										Selector: "(name=='" + w[0][0].Name + "')",
									},
								},
							}
							pol.Spec.Selector = "name=='" + w[0][0].Name + "'"
							one := float64(1)
							pol.Spec.Order = &one

							pol = createPolicy(pol)
						})
						By("asserting that ipaddress, port exists in ipset ")
						assertIPPortInIPSet(TPROXYServicesIPSetV4, clusterIP, "8090", felixes, true)

						cc.Expect(None, w[0][0], TargetIP(clusterIP), ExpectWithPorts(8090))
						cc.Expect(Some, w[0][1], TargetIP(clusterIP), ExpectWithPorts(8090))
						cc.Expect(Some, w[1][0], TargetIP(clusterIP), ExpectWithPorts(8090))
						cc.Expect(Some, w[1][1], TargetIP(clusterIP), ExpectWithPorts(8090))
						cc.CheckConnectivity()

						// Connection should be proxied on the pod's local node

						Eventually(proxies[0].AcceptedCountFn(w[0][0].IP, pod, svc)).Should(BeNumerically(">", 0))
						Eventually(proxies[0].AcceptedCountFn(w[0][1].IP, pod, svc)).Should(BeNumerically(">", 0))
						Eventually(proxies[1].AcceptedCountFn(w[1][0].IP, pod, svc)).Should(BeNumerically(">", 0))
						Eventually(proxies[1].AcceptedCountFn(w[1][1].IP, pod, svc)).Should(BeNumerically(">", 0))

						Eventually(proxies[0].ProxiedCountFn(w[0][0].IP, pod, svc)).Should(Equal(0))
						Eventually(proxies[0].ProxiedCountFn(w[0][1].IP, pod, svc)).Should(BeNumerically(">", 0))
						Eventually(proxies[1].ProxiedCountFn(w[1][0].IP, pod, svc)).Should(BeNumerically(">", 0))
						Eventually(proxies[1].ProxiedCountFn(w[1][1].IP, pod, svc)).Should(BeNumerically(">", 0))
					})
				})

				Context("With egress traffic denied from w[0]0] to self", func() {
					It("should have connectivity from all but self", func() {
						By("Denying traffic from w[1][1]", func() {
							pol := api.NewGlobalNetworkPolicy()
							pol.Namespace = "fv"
							pol.Name = "policy-deny-1-1"
							pol.Spec.Egress = []api.Rule{
								{
									Action: "Deny",
									Source: api.EntityRule{
										Selector: "(name=='" + w[0][0].Name + "')",
									},
								},
							}
							pol.Spec.Selector = "name=='" + w[0][0].Name + "'"
							one := float64(1)
							pol.Spec.Order = &one

							pol = createPolicy(pol)
						})
						By("asserting that ipaddress, port exists in ipset ")
						assertIPPortInIPSet(TPROXYServicesIPSetV4, clusterIP, "8090", felixes, true)

						cc.Expect(None, w[0][0], TargetIP(clusterIP), ExpectWithPorts(8090))
						cc.Expect(Some, w[0][1], TargetIP(clusterIP), ExpectWithPorts(8090))
						cc.Expect(Some, w[1][0], TargetIP(clusterIP), ExpectWithPorts(8090))
						cc.Expect(Some, w[1][1], TargetIP(clusterIP), ExpectWithPorts(8090))
						cc.CheckConnectivity()

						// Connection should be proxied on the pod's local node

						Eventually(proxies[0].AcceptedCountFn(w[0][0].IP, pod, svc)).Should(Equal(0))
						Eventually(proxies[0].AcceptedCountFn(w[0][1].IP, pod, svc)).Should(BeNumerically(">", 0))
						Eventually(proxies[1].AcceptedCountFn(w[1][0].IP, pod, svc)).Should(BeNumerically(">", 0))
						Eventually(proxies[1].AcceptedCountFn(w[1][1].IP, pod, svc)).Should(BeNumerically(">", 0))

						Eventually(proxies[0].ProxiedCountFn(w[0][0].IP, pod, svc)).Should(Equal(0))
						Eventually(proxies[0].ProxiedCountFn(w[0][1].IP, pod, svc)).Should(BeNumerically(">", 0))
						Eventually(proxies[1].ProxiedCountFn(w[1][0].IP, pod, svc)).Should(BeNumerically(">", 0))
						Eventually(proxies[1].ProxiedCountFn(w[1][1].IP, pod, svc)).Should(BeNumerically(">", 0))
					})
				})
			})

			Context("Select Traffic ClusterIP", func() {
				servicePort := "8090"

				It("Should propagate annotated service update and deletions to tproxy ip set", func() {

					By("setting up annotated service for the end points ")
					//create service resource that has annotation at creation
					v1Svc := k8sService("l7-service", clusterIP, w[0][0], 8090, 8055, 0, "tcp")
					v1Svc.ObjectMeta.Annotations = map[string]string{l7LoggingAnnotation: "true"}
					annotatedSvc := createService(v1Svc, clientset)

					By("asserting that ipaddress, port of service updated in ipset ")
					assertIPPortInIPSet(TPROXYServicesIPSetV4, clusterIP, servicePort, felixes, true)

					By("updating the service to not have l7 annotation ")
					annotatedSvc.ObjectMeta.Annotations = map[string]string{}
					_, err := clientset.CoreV1().Services(annotatedSvc.ObjectMeta.Namespace).Update(context.Background(), annotatedSvc, metav1.UpdateOptions{})
					Expect(err).NotTo(HaveOccurred())

					By("asserting that ip, port exists for EnabledAllServices case and doesn't exist for others case when l7 annotation removed")
					assertIPPortInIPSet(TPROXYServicesIPSetV4, clusterIP, servicePort, felixes, TPROXYMode == "EnabledAllServices")

					By("deleting the now unannotated service")
					err = clientset.CoreV1().Services(v1Svc.ObjectMeta.Namespace).Delete(context.Background(), v1Svc.ObjectMeta.Name, metav1.DeleteOptions{})
					Expect(err).NotTo(HaveOccurred())

					By("assert that ip, port is removed from ipset when service is deleted")
					assertIPPortInIPSet(TPROXYServicesIPSetV4, clusterIP, servicePort, felixes, false)

					// In this second stage we create a service resource that does not have annotation at creation
					// and repeat similar process as above again.

					// this case ensures that the process of annotating is repeatable and certain IPSet callbacks are
					// not called ex. IPSetAdded (Felix fails if a IPSetAdded call to already existing ipset is made)
					By("creating unannotated service, to verify the ipset created callbacks")
					v1Svc.ObjectMeta.Annotations = map[string]string{}
					unannotatedSvc := createService(v1Svc, clientset)

					By("asserting that ip, port exists for EnabledAllServices and doesn't exist for others case when l7 annotation not present")
					assertIPPortInIPSet(TPROXYServicesIPSetV4, clusterIP, servicePort, felixes, TPROXYMode == "EnabledAllServices")

					By("updating the service to have l7 annotation ")
					unannotatedSvc.ObjectMeta.Annotations = map[string]string{l7LoggingAnnotation: "true"}
					_, err = clientset.CoreV1().Services(unannotatedSvc.ObjectMeta.Namespace).Update(context.Background(), unannotatedSvc, metav1.UpdateOptions{})
					Expect(err).NotTo(HaveOccurred())

					By("asserting that ipaddress, port of service propagated to ipset")
					assertIPPortInIPSet(TPROXYServicesIPSetV4, clusterIP, servicePort, felixes, true)

					By("deleting the annotated service")
					err = clientset.CoreV1().Services(v1Svc.ObjectMeta.Namespace).Delete(context.Background(), v1Svc.ObjectMeta.Name, metav1.DeleteOptions{})
					Expect(err).NotTo(HaveOccurred())

					By("asserting that ipaddress, port of service removed from ipset")
					assertIPPortInIPSet(TPROXYServicesIPSetV4, clusterIP, servicePort, felixes, false)

				})

			})

		})
}
