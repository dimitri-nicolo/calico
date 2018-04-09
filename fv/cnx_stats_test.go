// +build fvtests

// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package fv_test

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/fv/containers"
	"github.com/projectcalico/felix/fv/metrics"
	"github.com/projectcalico/felix/fv/utils"
	"github.com/projectcalico/felix/fv/workload"
	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/numorstring"
)

// Pause time before felix will generate CNX metrics
var pollingInterval = time.Duration(10) * time.Second

var _ = Context("CNX Metrics, etcd datastore, 4 workloads", func() {

	var (
		etcd   *containers.Container
		felix  *containers.Felix
		client client.Interface
		w      [4]*workload.Workload
	)

	BeforeEach(func() {
		felix, etcd, client = containers.StartSingleNodeEtcdTopology(containers.DefaultTopologyOptions())

		// Default profile that ensures connectivity.
		defaultProfile := api.NewProfile()
		defaultProfile.Name = "default"
		defaultProfile.Spec.LabelsToApply = map[string]string{"default": ""}
		defaultProfile.Spec.Egress = []api.Rule{{Action: api.Allow}}
		defaultProfile.Spec.Ingress = []api.Rule{{
			Action: api.Allow,
			Source: api.EntityRule{Selector: "default == ''"},
		}}
		_, err := client.Profiles().Create(utils.Ctx, defaultProfile, utils.NoOptions)
		Expect(err).NotTo(HaveOccurred())

		// Create two workloads, using the profile created above.
		for ii := range w {
			iiStr := strconv.Itoa(ii)
			w[ii] = workload.Run(felix, "w"+iiStr, "cali1"+iiStr, "10.65.0.1"+iiStr, "8055", "tcp")
			w[ii].Configure(client)
		}
	})

	AfterEach(func() {

		if CurrentGinkgoTestDescription().Failed {
			felix.Exec("iptables-save", "-c")
			felix.Exec("ip", "r")
			cprc, _ := metrics.GetCNXMetrics(felix.IP, "cnx_policy_rule_connections")
			cprp, _ := metrics.GetCNXMetrics(felix.IP, "cnx_policy_rule_packets")
			cprb, _ := metrics.GetCNXMetrics(felix.IP, "cnx_policy_rule_bytes")
			cdp, _ := metrics.GetCNXMetrics(felix.IP, "calico_denied_packets")
			cdb, _ := metrics.GetCNXMetrics(felix.IP, "calico_denied_bytes")
			log.Info("Collected CNX Metrics\n\n" +
				"cnx_policy_rule_connections\n" +
				"===========================\n" +
				strings.Join(cprc, "\n") + "\n\n" +
				"cnx_policy_rule_packets\n" +
				"=======================\n" +
				strings.Join(cprp, "\n") + "\n\n" +
				"cnx_policy_rule_bytes\n" +
				"=====================\n" +
				strings.Join(cprb, "\n") + "\n\n" +
				"calico_denied_packets\n" +
				"=====================\n" +
				strings.Join(cdp, "\n") + "\n\n" +
				"calico_denied_bytes\n" +
				"===================\n" +
				strings.Join(cdb, "\n") + "\n\n",
			)
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

	It("should generate connection metrics for rule matches on a profile", func() {

		var conns, bytes, packets int
		incCounts := func(deltaConn, numPackets, packetSize int) {
			conns += deltaConn
			packets += numPackets
			bytes += calculateBytesForPacket("ICMP", numPackets, packetSize)
		}
		expectCounts := func() {
			// Pause to allow felix to export metrics.
			time.Sleep(pollingInterval)
			// Local-to-Local traffic causes accounting from both workload perspectives.
			Expect(func() (int, error) {
				return metrics.GetCNXConnectionMetricsIntForPolicy(felix.IP, "profile", "default", "outbound")
			}()).Should(BeNumerically("==", conns))
			// ICMP request and responses are the same size.
			Expect(func() (int, error) {
				return metrics.GetCNXPacketMetricsIntForPolicy(felix.IP, "allow", "profile", "default", "outbound", "ingress")
			}()).Should(BeNumerically("==", packets))
			Expect(func() (int, error) {
				return metrics.GetCNXPacketMetricsIntForPolicy(felix.IP, "allow", "profile", "default", "inbound", "ingress")
			}()).Should(BeNumerically("==", packets))
			Expect(func() (int, error) {
				return metrics.GetCNXByteMetricsIntForPolicy(felix.IP, "allow", "profile", "default", "outbound", "ingress")
			}()).Should(BeNumerically("==", bytes))
			Expect(func() (int, error) {
				return metrics.GetCNXByteMetricsIntForPolicy(felix.IP, "allow", "profile", "default", "inbound", "ingress")
			}()).Should(BeNumerically("==", bytes))
		}

		By("Sending pings from w0->w1 and w1->w0 and checking received counts")
		// Wait a bit for policy to be programmed.
		time.Sleep(pollingInterval)
		err, stderr := w[0].SendPacketsTo(w[1].IP, 1, 2)
		Expect(err).NotTo(HaveOccurred(), stderr)
		incCounts(1, 1, 2)
		err, stderr = w[1].SendPacketsTo(w[0].IP, 1, 2)
		Expect(err).NotTo(HaveOccurred(), stderr)
		incCounts(1, 1, 2)
		expectCounts()

		By("Sending pings more pings from w0->w1 and sending pings from w1->w2 and w2->w3 and checking received counts")
		err, stderr = w[0].SendPacketsTo(w[1].IP, 1, 1)
		Expect(err).NotTo(HaveOccurred(), stderr)
		incCounts(1, 1, 1)
		err, stderr = w[1].SendPacketsTo(w[2].IP, 2, 5)
		Expect(err).NotTo(HaveOccurred(), stderr)
		incCounts(1, 2, 5)
		err, stderr = w[2].SendPacketsTo(w[3].IP, 3, 7)
		Expect(err).NotTo(HaveOccurred(), stderr)
		incCounts(1, 3, 7)
		expectCounts()
	})

	Context("should generate connection metrics for rule matches on a policy", func() {

		It("should generate connection metrics when there is full connectivity between workloads, testing ingress rule index", func() {

			By("Creating a policy with multiple ingress rules matching on different workloads")
			policy := api.NewNetworkPolicy()
			policy.Namespace = "fv"
			policy.Name = "default.policy-test-ingress-idx"
			policy.Spec.Tier = "default"
			policy.Spec.Types = []api.PolicyType{api.PolicyTypeIngress, api.PolicyTypeEgress}
			policy.Spec.Ingress = []api.Rule{
				{
					Action:      api.Allow,
					Destination: api.EntityRule{Selector: w[0].NameSelector()},
				},
				{
					Action:      api.Allow,
					Destination: api.EntityRule{Selector: w[1].NameSelector()},
				},
				{
					Action:      api.Allow,
					Destination: api.EntityRule{Selector: w[2].NameSelector()},
				},
				{
					Action:      api.Allow,
					Destination: api.EntityRule{Selector: w[3].NameSelector()},
				},
			}
			policy.Spec.Egress = []api.Rule{{Action: api.Allow}}
			policy.Spec.Selector = "default == ''"
			_, err := client.NetworkPolicies().Create(utils.Ctx, policy, utils.NoOptions)
			Expect(err).NotTo(HaveOccurred())

			// There are 4 ingress rules, only one egress. All match on the single egress rule.
			var igrConns, igrBytes, igrPackets [4]int
			var egrConns, egrBytes, egrPackets int
			incCounts := func(deltaConn, numPackets, packetSize, igrRuleIdx int) {
				egrConns += deltaConn
				egrPackets += numPackets
				egrBytes += calculateBytesForPacket("ICMP", numPackets, packetSize)
				igrConns[igrRuleIdx] += deltaConn
				igrPackets[igrRuleIdx] += numPackets
				igrBytes[igrRuleIdx] += calculateBytesForPacket("ICMP", numPackets, packetSize)
			}
			expectCounts := func() {
				// Pause to allow felix to export metrics.
				time.Sleep(pollingInterval)
				// Local-to-Local traffic causes accounting from both workload perspectives.
				Expect(func() (int, error) {
					return metrics.GetCNXConnectionMetricsIntForPolicy(felix.IP, "default", "policy-test-ingress-idx", "outbound")
				}()).Should(BeNumerically("==", egrConns))
				Expect(func() (int, error) {
					return metrics.GetCNXPacketMetricsIntForPolicy(felix.IP, "allow", "default", "policy-test-ingress-idx", "outbound", "egress")
				}()).Should(BeNumerically("==", egrPackets))
				Expect(func() (int, error) {
					return metrics.GetCNXPacketMetricsIntForPolicy(felix.IP, "allow", "default", "policy-test-ingress-idx", "inbound", "egress")
				}()).Should(BeNumerically("==", egrPackets))
				Expect(func() (int, error) {
					return metrics.GetCNXByteMetricsIntForPolicy(felix.IP, "allow", "default", "policy-test-ingress-idx", "outbound", "egress")
				}()).Should(BeNumerically("==", egrBytes))
				Expect(func() (int, error) {
					return metrics.GetCNXByteMetricsIntForPolicy(felix.IP, "allow", "default", "policy-test-ingress-idx", "inbound", "egress")
				}()).Should(BeNumerically("==", egrBytes))
				for i := range igrConns {
					// Include the ruleIdx in the expect description to assist in debugging.
					ruleIdxString := fmt.Sprintf("RuleIndex=%d", i)
					Expect(func() (int, error) {
						return metrics.GetCNXConnectionMetricsIntForPolicy(felix.IP, "default", "policy-test-ingress-idx", "inbound", i)
					}()).Should(BeNumerically("==", igrConns[i]), ruleIdxString)
					Expect(func() (int, error) {
						return metrics.GetCNXPacketMetricsIntForPolicy(felix.IP, "allow", "default", "policy-test-ingress-idx", "outbound", "ingress", i)
					}()).Should(BeNumerically("==", igrPackets[i]), ruleIdxString)
					Expect(func() (int, error) {
						return metrics.GetCNXPacketMetricsIntForPolicy(felix.IP, "allow", "default", "policy-test-ingress-idx", "inbound", "ingress", i)
					}()).Should(BeNumerically("==", igrPackets[i]), ruleIdxString)
					Expect(func() (int, error) {
						return metrics.GetCNXByteMetricsIntForPolicy(felix.IP, "allow", "default", "policy-test-ingress-idx", "outbound", "ingress", i)
					}()).Should(BeNumerically("==", igrBytes[i]), ruleIdxString)
					Expect(func() (int, error) {
						return metrics.GetCNXByteMetricsIntForPolicy(felix.IP, "allow", "default", "policy-test-ingress-idx", "inbound", "ingress", i)
					}()).Should(BeNumerically("==", igrBytes[i]), ruleIdxString)
				}
			}

			By("Sending pings from w0->w1 and w1->w0 and checking received counts")
			// Wait a bit for policy to be programmed.
			time.Sleep(pollingInterval)
			err, stderr := w[0].SendPacketsTo(w[1].IP, 1, 2)
			Expect(err).NotTo(HaveOccurred(), stderr)
			incCounts(1, 1, 2, 1) // Ingress to w1, so matches on rule index 1
			err, stderr = w[1].SendPacketsTo(w[0].IP, 1, 2)
			Expect(err).NotTo(HaveOccurred(), stderr)
			incCounts(1, 1, 2, 0) // Ingress to w0, so matches on rule index 0
			expectCounts()

			By("Sending pings more pings from w0->w1 and sending pings from w1->w2 and w2->w3 and checking received counts")
			err, stderr = w[0].SendPacketsTo(w[1].IP, 1, 1)
			Expect(err).NotTo(HaveOccurred(), stderr)
			incCounts(1, 1, 1, 1) // Ingress to w1, so matches on rule index 1
			err, stderr = w[1].SendPacketsTo(w[2].IP, 2, 5)
			Expect(err).NotTo(HaveOccurred(), stderr)
			incCounts(1, 2, 5, 2) // Ingress to w2, so matches on rule index 2
			err, stderr = w[2].SendPacketsTo(w[3].IP, 3, 7)
			Expect(err).NotTo(HaveOccurred(), stderr)
			incCounts(1, 3, 7, 3) // Ingress to w3, so matches on rule index 3
			expectCounts()
		})

		It("should generate connection metrics when there is full connectivity between workloads, testing egress rule index", func() {

			By("Creating a policy with multiple egress rules matching on different workloads")
			policy := api.NewNetworkPolicy()
			policy.Namespace = "fv"
			policy.Name = "default.policy-test-egress-idx"
			policy.Spec.Tier = "default"
			policy.Spec.Types = []api.PolicyType{api.PolicyTypeIngress, api.PolicyTypeEgress}
			policy.Spec.Egress = []api.Rule{
				{
					Action:      api.Allow,
					Destination: api.EntityRule{Selector: w[0].NameSelector()},
				},
				{
					Action:      api.Allow,
					Destination: api.EntityRule{Selector: w[1].NameSelector()},
				},
				{
					Action:      api.Allow,
					Destination: api.EntityRule{Selector: w[2].NameSelector()},
				},
				{
					Action:      api.Allow,
					Destination: api.EntityRule{Selector: w[3].NameSelector()},
				},
			}
			policy.Spec.Ingress = []api.Rule{{Action: api.Allow}}
			policy.Spec.Selector = "default == ''"
			_, err := client.NetworkPolicies().Create(utils.Ctx, policy, utils.NoOptions)
			Expect(err).NotTo(HaveOccurred())

			// There are 4 egress rules, only one ingress. All match on the single ingress rule.
			var egrConns, egrBytes, egrPackets [4]int
			var igrConns, igrBytes, igrPackets int
			incCounts := func(deltaConn, numPackets, packetSize, egrRuleIdx int) {
				igrConns += deltaConn
				igrPackets += numPackets
				igrBytes += calculateBytesForPacket("ICMP", numPackets, packetSize)
				egrConns[egrRuleIdx] += deltaConn
				egrPackets[egrRuleIdx] += numPackets
				egrBytes[egrRuleIdx] += calculateBytesForPacket("ICMP", numPackets, packetSize)
			}
			expectCounts := func() {
				// Pause to allow felix to export metrics.
				time.Sleep(pollingInterval)
				// Local-to-Local traffic causes accounting from both workload perspectives.
				Expect(func() (int, error) {
					return metrics.GetCNXConnectionMetricsIntForPolicy(felix.IP, "default", "policy-test-egress-idx", "outbound")
				}()).Should(BeNumerically("==", igrConns))
				Expect(func() (int, error) {
					return metrics.GetCNXPacketMetricsIntForPolicy(felix.IP, "allow", "default", "policy-test-egress-idx", "outbound", "ingress")
				}()).Should(BeNumerically("==", igrPackets))
				Expect(func() (int, error) {
					return metrics.GetCNXPacketMetricsIntForPolicy(felix.IP, "allow", "default", "policy-test-egress-idx", "inbound", "ingress")
				}()).Should(BeNumerically("==", igrPackets))
				Expect(func() (int, error) {
					return metrics.GetCNXByteMetricsIntForPolicy(felix.IP, "allow", "default", "policy-test-egress-idx", "outbound", "ingress")
				}()).Should(BeNumerically("==", igrBytes))
				Expect(func() (int, error) {
					return metrics.GetCNXByteMetricsIntForPolicy(felix.IP, "allow", "default", "policy-test-egress-idx", "inbound", "ingress")
				}()).Should(BeNumerically("==", igrBytes))
				for i := range egrConns {
					// Include the ruleIdx in the expect description to assist in debugging.
					ruleIdxString := fmt.Sprintf("RuleIndex=%d", i)
					Expect(func() (int, error) {
						return metrics.GetCNXConnectionMetricsIntForPolicy(felix.IP, "default", "policy-test-egress-idx", "outbound", i)
					}()).Should(BeNumerically("==", egrConns[i]), ruleIdxString)
					Expect(func() (int, error) {
						return metrics.GetCNXPacketMetricsIntForPolicy(felix.IP, "allow", "default", "policy-test-egress-idx", "outbound", "egress", i)
					}()).Should(BeNumerically("==", egrPackets[i]), ruleIdxString)
					Expect(func() (int, error) {
						return metrics.GetCNXPacketMetricsIntForPolicy(felix.IP, "allow", "default", "policy-test-egress-idx", "inbound", "egress", i)
					}()).Should(BeNumerically("==", egrPackets[i]), ruleIdxString)
					Expect(func() (int, error) {
						return metrics.GetCNXByteMetricsIntForPolicy(felix.IP, "allow", "default", "policy-test-egress-idx", "outbound", "egress", i)
					}()).Should(BeNumerically("==", egrBytes[i]), ruleIdxString)
					Expect(func() (int, error) {
						return metrics.GetCNXByteMetricsIntForPolicy(felix.IP, "allow", "default", "policy-test-egress-idx", "inbound", "egress", i)
					}()).Should(BeNumerically("==", egrBytes[i]), ruleIdxString)
				}
			}

			By("Sending pings from w0->w1 and w1->w0 and checking received counts")
			// Wait a bit for policy to be programmed.
			time.Sleep(pollingInterval)
			err, stderr := w[0].SendPacketsTo(w[1].IP, 1, 2)
			Expect(err).NotTo(HaveOccurred(), stderr)
			incCounts(1, 1, 2, 1) // Egress to w1, so matches on rule index 1
			err, stderr = w[1].SendPacketsTo(w[0].IP, 1, 2)
			Expect(err).NotTo(HaveOccurred(), stderr)
			incCounts(1, 1, 2, 0) // Egress to w0, so matches on rule index 0
			expectCounts()

			By("Sending pings more pings from w0->w1 and sending pings from w1->w2 and w2->w3 and checking received counts")
			err, stderr = w[0].SendPacketsTo(w[1].IP, 1, 1)
			Expect(err).NotTo(HaveOccurred(), stderr)
			incCounts(1, 1, 1, 1) // Egress to w1, so matches on rule index 1
			err, stderr = w[1].SendPacketsTo(w[2].IP, 2, 5)
			Expect(err).NotTo(HaveOccurred(), stderr)
			incCounts(1, 2, 5, 2) // Egress to w2, so matches on rule index 2
			err, stderr = w[2].SendPacketsTo(w[3].IP, 3, 7)
			Expect(err).NotTo(HaveOccurred(), stderr)
			incCounts(1, 3, 7, 3) // Egress to w3, so matches on rule index 3
			expectCounts()
		})
	})

	Context("should generate denied packet metrics with deny rule to w1", func() {

		BeforeEach(func() {
			policy := api.NewNetworkPolicy()
			policy.Namespace = "fv"
			policy.Name = "default.policy-1"
			policy.Spec.Tier = "default"
			policy.Spec.Types = []api.PolicyType{api.PolicyTypeIngress}
			policy.Spec.Ingress = []api.Rule{{Action: api.Deny}}
			policy.Spec.Egress = []api.Rule{{Action: api.Allow}}
			policy.Spec.Selector = w[1].NameSelector()
			_, err := client.NetworkPolicies().Create(utils.Ctx, policy, utils.NoOptions)
			Expect(err).NotTo(HaveOccurred())
		})

		It("w0 cannot connect to w1 and denied packet metrics are generated", func() {
			var bytes, packets int
			incCounts := func(numPackets, packetSize int) {
				packets += numPackets
				bytes += calculateBytesForPacket("ICMP", numPackets, packetSize)
			}
			expectCounts := func() {
				// Pause to allow felix to export metrics.
				time.Sleep(pollingInterval)
				Expect(func() (int, error) {
					return metrics.GetCNXConnectionMetricsIntForPolicy(felix.IP, "default", "policy-1", "inbound")
				}()).Should(BeNumerically("==", 0))
				Expect(func() (int, error) {
					return metrics.GetCNXPacketMetricsIntForPolicy(felix.IP, "deny", "default", "policy-1", "inbound", "ingress")
				}()).Should(BeNumerically("==", packets))
				Expect(func() (int, error) {
					return metrics.GetCNXByteMetricsIntForPolicy(felix.IP, "deny", "default", "policy-1", "inbound", "ingress")
				}()).Should(BeNumerically("==", bytes))
				Expect(func() (int, error) {
					return metrics.GetCalicoDeniedPacketMetrics(felix.IP, "default", "policy-1")
				}()).Should(BeNumerically("==", packets))
			}

			By("Sending pings from w0->w1 and checking denied counts")
			// Wait a bit for policy to be programmed.
			time.Sleep(pollingInterval)
			err, stderr := w[0].SendPacketsTo(w[1].IP, 1, 2)
			Expect(err).To(HaveOccurred(), stderr)
			incCounts(1, 2)
			expectCounts()

			By("Sending pings more pings from w0->w1 and sending pings from w2->w1 and w3->w1 and checking denied counts")
			err, stderr = w[0].SendPacketsTo(w[1].IP, 1, 2)
			Expect(err).To(HaveOccurred(), stderr)
			incCounts(1, 2)
			expectCounts()
			err, stderr = w[2].SendPacketsTo(w[1].IP, 3, 5)
			Expect(err).To(HaveOccurred(), stderr)
			incCounts(3, 5)
			expectCounts()
			err, stderr = w[3].SendPacketsTo(w[1].IP, 2, 3)
			Expect(err).To(HaveOccurred(), stderr)
			incCounts(2, 3)
			expectCounts()
		})
	})

	Context("should generate metrics with egress deny rule from w0", func() {

		BeforeEach(func() {
			proto := numorstring.ProtocolFromString("ICMP")
			policy := api.NewNetworkPolicy()
			policy.Namespace = "fv"
			policy.Name = "default.policy-icmp"
			policy.Spec.Tier = "default"
			policy.Spec.Types = []api.PolicyType{api.PolicyTypeEgress}
			policy.Spec.Ingress = []api.Rule{{Action: api.Deny}}
			policy.Spec.Egress = []api.Rule{{Protocol: &proto, Action: api.Deny}}
			policy.Spec.Selector = w[0].NameSelector()
			_, err := client.NetworkPolicies().Create(utils.Ctx, policy, utils.NoOptions)
			Expect(err).NotTo(HaveOccurred())
		})

		It("w0 cannot connect to w1 and denied packet metrics are generated", func() {
			var bytes, packets int
			incCounts := func(numPackets, packetSize int) {
				packets += numPackets
				bytes += calculateBytesForPacket("ICMP", numPackets, packetSize)
			}
			expectCounts := func() {
				// Pause to allow felix to export metrics.
				time.Sleep(pollingInterval)
				Expect(func() (int, error) {
					return metrics.GetCNXConnectionMetricsIntForPolicy(felix.IP, "default", "policy-icmp", "outbound")
				}()).Should(BeNumerically("==", 0))
				Expect(func() (int, error) {
					return metrics.GetCNXPacketMetricsIntForPolicy(felix.IP, "deny", "default", "policy-icmp", "outbound", "egress")
				}()).Should(BeNumerically("==", packets))
				Expect(func() (int, error) {
					return metrics.GetCNXByteMetricsIntForPolicy(felix.IP, "deny", "default", "policy-icmp", "outbound", "egress")
				}()).Should(BeNumerically("==", bytes))
				Expect(func() (int, error) {
					return metrics.GetCalicoDeniedPacketMetrics(felix.IP, "default", "policy-icmp")
				}()).Should(BeNumerically("==", packets))
			}

			By("Sending pings from w0->w1 and checking denied counts")
			// Wait a bit for policy to be programmed.
			time.Sleep(pollingInterval)
			err, stderr := w[0].SendPacketsTo(w[1].IP, 1, 2)
			Expect(err).To(HaveOccurred(), stderr)
			incCounts(1, 2)
			expectCounts()

			By("Sending pings from w0->w1,w2&w3 and checking denied counts")
			// Wait a bit for policy to be programmed.
			time.Sleep(pollingInterval)
			err, stderr = w[0].SendPacketsTo(w[1].IP, 1, 0)
			Expect(err).To(HaveOccurred(), stderr)
			incCounts(1, 0)
			time.Sleep(pollingInterval)
			err, stderr = w[0].SendPacketsTo(w[2].IP, 3, 4)
			Expect(err).To(HaveOccurred(), stderr)
			incCounts(3, 4)
			time.Sleep(pollingInterval)
			err, stderr = w[0].SendPacketsTo(w[3].IP, 5, 6)
			Expect(err).To(HaveOccurred(), stderr)
			incCounts(5, 6)
			expectCounts()
		})
	})

	Context("should generate denied packet metrics with deny rule in a different tier", func() {

		BeforeEach(func() {
			tier := api.NewTier()
			tier.Name = "tier1"
			o := 10.00
			tier.Spec.Order = &o
			_, err := client.Tiers().Create(utils.Ctx, tier, utils.NoOptions)
			Expect(err).NotTo(HaveOccurred())

			policy := api.NewNetworkPolicy()
			policy.Namespace = "fv"
			policy.Name = "tier1.policy-1"
			policy.Spec.Tier = "tier1"
			policy.Spec.Types = []api.PolicyType{api.PolicyTypeIngress}
			policy.Spec.Ingress = []api.Rule{{Action: api.Deny}}
			policy.Spec.Egress = []api.Rule{{Action: api.Allow}}
			policy.Spec.Selector = w[1].NameSelector()
			_, err = client.NetworkPolicies().Create(utils.Ctx, policy, utils.NoOptions)
			Expect(err).NotTo(HaveOccurred())
		})

		It("w0 cannot connect to w1 and denied packet metrics are generated for tier1", func() {
			var bytes, packets int
			incCounts := func(numPackets, packetSize int) {
				packets += numPackets
				bytes += calculateBytesForPacket("ICMP", numPackets, packetSize)
			}
			expectCounts := func() {
				// Pause to allow felix to export metrics.
				time.Sleep(pollingInterval)
				Expect(func() (int, error) {
					return metrics.GetCNXConnectionMetricsIntForPolicy(felix.IP, "tier1", "policy-1", "inbound")
				}()).Should(BeNumerically("==", 0))
				Expect(func() (int, error) {
					return metrics.GetCNXPacketMetricsIntForPolicy(felix.IP, "deny", "tier1", "policy-1", "inbound", "ingress")
				}()).Should(BeNumerically("==", packets))
				Expect(func() (int, error) {
					return metrics.GetCNXByteMetricsIntForPolicy(felix.IP, "deny", "tier1", "policy-1", "inbound", "ingress")
				}()).Should(BeNumerically("==", bytes))
				Expect(func() (int, error) {
					return metrics.GetCalicoDeniedPacketMetrics(felix.IP, "tier1", "policy-1")
				}()).Should(BeNumerically("==", packets))
			}

			By("Verifying that w0 cannot reach w1")
			// Wait a bit for policy to be programmed.
			time.Sleep(pollingInterval)
			err, stderr := w[0].SendPacketsTo(w[1].IP, 1, 2)
			Expect(err).To(HaveOccurred(), stderr)
			incCounts(1, 2)
			expectCounts()

			By("Pinging again and verifying that w0 cannot reach w1")
			err, stderr = w[0].SendPacketsTo(w[1].IP, 1, 2)
			Expect(err).To(HaveOccurred(), stderr)
			incCounts(1, 2)
			expectCounts()
		})
	})

	Context("Tests with very long tier and policy names", func() {
		longTierName := "this-in-a-very-long-tier-name-012345678900123456789001234567890"
		longPolicyName := "this-is-a-very-long-policy-name-012345678900123456789001234567890"

		BeforeEach(func() {
			tier := api.NewTier()
			tier.Name = longTierName
			o := 10.00
			tier.Spec.Order = &o
			_, err := client.Tiers().Create(utils.Ctx, tier, utils.NoOptions)
			Expect(err).NotTo(HaveOccurred())

			policy := api.NewNetworkPolicy()
			policy.Namespace = "fv"
			policy.Name = longTierName + "." + longPolicyName
			policy.Spec.Tier = longTierName
			policy.Spec.Types = []api.PolicyType{api.PolicyTypeIngress}
			policy.Spec.Ingress = []api.Rule{{Action: api.Deny}}
			policy.Spec.Egress = []api.Rule{{Action: api.Allow}}
			policy.Spec.Selector = w[1].NameSelector()
			_, err = client.NetworkPolicies().Create(utils.Ctx, policy, utils.NoOptions)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Deny metrics for long named tier and policy", func() {
			// Indicates we need to create the tier.
			By("Verifying that w0 cannot reach w1")
			time.Sleep(pollingInterval)
			err, stderr := w[0].SendPacketsTo(w[1].IP, 3, 5)
			Expect(err).To(HaveOccurred(), stderr)

			By("Ensuring the stats are accurate")
			// Pause to allow felix to export metrics.
			time.Sleep(pollingInterval)
			Expect(func() (int, error) {
				return metrics.GetCNXConnectionMetricsIntForPolicy(felix.IP, longTierName, longPolicyName, "inbound")
			}()).Should(BeNumerically("==", 0))
			Expect(func() (int, error) {
				return metrics.GetCNXPacketMetricsIntForPolicy(felix.IP, "deny", longTierName, longPolicyName, "inbound", "ingress")
			}()).Should(BeNumerically("==", 3))
			Expect(func() (int, error) {
				return metrics.GetCNXByteMetricsIntForPolicy(felix.IP, "deny", longTierName, longPolicyName, "inbound", "ingress")
			}()).Should(BeNumerically("==", calculateBytesForPacket("ICMP", 3, 5)))
			Expect(func() (int, error) {
				return metrics.GetCalicoDeniedPacketMetrics(felix.IP, longTierName, longPolicyName)
			}()).Should(BeNumerically("==", 3))
		})
	})
})

func calculateBytesForPacket(proto string, pktCount, packetSize int) int {
	switch proto {
	case "ICMP":
		// 8 byte ICMP header + 20 byte IP header
		return (packetSize + 8 + 20) * pktCount
	default:
		// Not implemented for now.
		return -1
	}
}
