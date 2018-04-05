// +build fvtests

// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package fv_test

import (
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
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

var _ = Context("CNX Metrics, etcd datastore, 2 workloads", func() {

	var (
		etcd   *containers.Container
		felix  *containers.Felix
		client client.Interface
		w      [2]*workload.Workload
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
			log.WithFields(log.Fields{
				"cnx_policy_rule_connections": cprc,
				"cnx_policy_rule_packets":     cprp,
				"cnx_policy_rule_bytes":       cprb,
				"calico_denied_packets":       cdp,
				"calico_denied_bytes":         cdb,
			}).Info("Collected CNX Metrics")
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

	It("should generate connection metrics when full connectivity to and from workload 0", func() {
		time.Sleep(pollingInterval)
		err, stderr := w[0].SendPacketsTo(w[1].IP, 1, 2)
		Expect(err).NotTo(HaveOccurred(), stderr)
		err, stderr = w[1].SendPacketsTo(w[0].IP, 1, 2)
		Expect(err).NotTo(HaveOccurred(), stderr)

		// Pause to allow felix to export metrics.
		time.Sleep(pollingInterval)
		// Local-to-Local traffic causes accounting from both workload perspectives.
		Expect(func() (int, error) {
			return metrics.GetCNXConnectionMetricsIntForPolicy(felix.IP, "profile", "default", "outbound")
		}()).Should(BeNumerically("==", 2))
		Expect(func() (int, error) {
			return metrics.GetCNXPacketMetricsIntForPolicy(felix.IP, "allow", "profile", "default", "outbound", "ingress")
		}()).Should(BeNumerically("==", 2))
		Expect(func() (int, error) {
			return metrics.GetCNXPacketMetricsIntForPolicy(felix.IP, "allow", "profile", "default", "inbound", "ingress")
		}()).Should(BeNumerically("==", 2))
		// We are sending 2 bytes + 8 byte ICMP header + 20 byte IP header, plus Local-to-Local accounting from both workload perspectives.
		Expect(func() (int, error) {
			return metrics.GetCNXByteMetricsIntForPolicy(felix.IP, "allow", "profile", "default", "outbound", "ingress")
		}()).Should(BeNumerically("==", 60))
		Expect(func() (int, error) {
			return metrics.GetCNXByteMetricsIntForPolicy(felix.IP, "allow", "profile", "default", "inbound", "ingress")
		}()).Should(BeNumerically("==", 60))
	})

	Context("should generate denied packet metrics with deny rule", func() {

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
			// Wait a bit for policy to be programmed.
			time.Sleep(pollingInterval)
			err, stderr := w[0].SendPacketsTo(w[1].IP, 1, 2)
			Expect(err).To(HaveOccurred(), stderr)

			// Pause to allow felix to export metrics.
			time.Sleep(pollingInterval)
			Expect(func() (int, error) {
				return metrics.GetCNXConnectionMetricsIntForPolicy(felix.IP, "default", "policy-1", "inbound")
			}()).Should(BeNumerically("==", 0))
			Expect(func() (int, error) {
				return metrics.GetCNXPacketMetricsIntForPolicy(felix.IP, "deny", "default", "policy-1", "inbound", "ingress")
			}()).Should(BeNumerically("==", 1))
			// We are sending 2 bytes + 8 byte ICMP header + 20 byte IP header.
			Expect(func() (int, error) {
				return metrics.GetCNXByteMetricsIntForPolicy(felix.IP, "deny", "default", "policy-1", "inbound", "ingress")
			}()).Should(BeNumerically("==", calculateBytesForPacket("ICMP", 1, 2)))
			Expect(func() (int, error) {
				return metrics.GetCalicoDeniedPacketMetrics(felix.IP, "default", "policy-1")
			}()).Should(BeNumerically("==", 1))
		})
	})
	Context("should generate metrics with egress deny rule", func() {

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
			// Wait a bit for policy to be programmed.
			time.Sleep(pollingInterval)
			err, stderr := w[0].SendPacketsTo(w[1].IP, 1, 2)
			Expect(err).To(HaveOccurred(), stderr)

			// Pause to allow felix to export metrics.
			time.Sleep(pollingInterval)
			Expect(func() (int, error) {
				return metrics.GetCNXConnectionMetricsIntForPolicy(felix.IP, "default", "policy-icmp", "outbound")
			}()).Should(BeNumerically("==", 0))
			Expect(func() (int, error) {
				return metrics.GetCNXPacketMetricsIntForPolicy(felix.IP, "deny", "default", "policy-icmp", "outbound", "egress")
			}()).Should(BeNumerically("==", 1))
			// We are sending 2 bytes + 8 byte ICMP header + 20 byte IP header.
			Expect(func() (int, error) {
				return metrics.GetCNXByteMetricsIntForPolicy(felix.IP, "deny", "default", "policy-icmp", "outbound", "egress")
			}()).Should(BeNumerically("==", calculateBytesForPacket("ICMP", 1, 2)))
			Expect(func() (int, error) {
				return metrics.GetCalicoDeniedPacketMetrics(felix.IP, "default", "policy-icmp")
			}()).Should(BeNumerically("==", 1))
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
			By("Verifying that w0 cannot reach w1")
			// Wait a bit for policy to be programmed.
			time.Sleep(pollingInterval)
			err, stderr := w[0].SendPacketsTo(w[1].IP, 1, 2)
			Expect(err).To(HaveOccurred(), stderr)

			By("Ensuring the stats are accurate")
			// Pause to allow felix to export metrics.
			time.Sleep(pollingInterval)
			Expect(func() (int, error) {
				return metrics.GetCNXConnectionMetricsIntForPolicy(felix.IP, "tier1", "policy-1", "inbound")
			}()).Should(BeNumerically("==", 0))
			Expect(func() (int, error) {
				return metrics.GetCNXPacketMetricsIntForPolicy(felix.IP, "deny", "tier1", "policy-1", "inbound", "ingress")
			}()).Should(BeNumerically("==", 1))
			// We are sending 2 bytes + 8 byte ICMP header + 20 byte IP header.
			Expect(func() (int, error) {
				return metrics.GetCNXByteMetricsIntForPolicy(felix.IP, "deny", "tier1", "policy-1", "inbound", "ingress")
			}()).Should(BeNumerically("==", calculateBytesForPacket("ICMP", 1, 2)))
			Expect(func() (int, error) {
				return metrics.GetCalicoDeniedPacketMetrics(felix.IP, "tier1", "policy-1")
			}()).Should(BeNumerically("==", 1))

			By("Pinging again and verifying that w0 cannot reach w1")
			err, stderr = w[0].SendPacketsTo(w[1].IP, 1, 2)
			Expect(err).To(HaveOccurred(), stderr)

			By("Ensuring the stats are updated")
			time.Sleep(pollingInterval)
			Expect(func() (int, error) {
				return metrics.GetCNXConnectionMetricsIntForPolicy(felix.IP, "tier1", "policy-1", "inbound")
			}()).Should(BeNumerically("==", 0))
			Expect(func() (int, error) {
				return metrics.GetCNXPacketMetricsIntForPolicy(felix.IP, "deny", "tier1", "policy-1", "inbound", "ingress")
			}()).Should(BeNumerically("==", 2))
			// We are sending 2 bytes + 8 byte ICMP header + 20 byte IP header.
			Expect(func() (int, error) {
				return metrics.GetCNXByteMetricsIntForPolicy(felix.IP, "deny", "tier1", "policy-1", "inbound", "ingress")
			}()).Should(BeNumerically("==", calculateBytesForPacket("ICMP", 1, 2)*2))
			Expect(func() (int, error) {
				return metrics.GetCalicoDeniedPacketMetrics(felix.IP, "tier1", "policy-1")
			}()).Should(BeNumerically("==", 2))
		})
	})
	Context("Tests with different packet and byte sizes", func() {
		BeforeEach(func() {
			tier := api.NewTier()
			tier.Name = "tier2"
			o := 10.00
			tier.Spec.Order = &o
			_, err := client.Tiers().Create(utils.Ctx, tier, utils.NoOptions)
			Expect(err).NotTo(HaveOccurred())

			policy := api.NewNetworkPolicy()
			policy.Namespace = "fv"
			policy.Name = "tier2.policy-1"
			policy.Spec.Tier = "tier2"
			policy.Spec.Types = []api.PolicyType{api.PolicyTypeIngress}
			policy.Spec.Ingress = []api.Rule{{Action: api.Deny}}
			policy.Spec.Egress = []api.Rule{{Action: api.Allow}}
			policy.Spec.Selector = w[1].NameSelector()
			_, err = client.NetworkPolicies().Create(utils.Ctx, policy, utils.NoOptions)
			Expect(err).NotTo(HaveOccurred())
		})

		DescribeTable("Deny metrics",
			func(pktCount, pktSize int) {
				// Indicates we need to create the tier.
				By("Verifying that w0 cannot reach w1")
				time.Sleep(pollingInterval)
				err, stderr := w[0].SendPacketsTo(w[1].IP, pktCount, pktSize)
				Expect(err).To(HaveOccurred(), stderr)

				By("Ensuring the stats are accurate")
				// Pause to allow felix to export metrics.
				time.Sleep(pollingInterval)
				Expect(func() (int, error) {
					return metrics.GetCNXConnectionMetricsIntForPolicy(felix.IP, "tier2", "policy-1", "inbound")
				}()).Should(BeNumerically("==", 0))
				Expect(func() (int, error) {
					return metrics.GetCNXPacketMetricsIntForPolicy(felix.IP, "deny", "tier2", "policy-1", "inbound", "ingress")
				}()).Should(BeNumerically("==", pktCount))
				Expect(func() (int, error) {
					return metrics.GetCNXByteMetricsIntForPolicy(felix.IP, "deny", "tier2", "policy-1", "inbound", "ingress")
				}()).Should(BeNumerically("==", calculateBytesForPacket("ICMP", pktCount, pktSize)))
				Expect(func() (int, error) {
					return metrics.GetCalicoDeniedPacketMetrics(felix.IP, "tier2", "policy-1")
				}()).Should(BeNumerically("==", pktCount))
			},
			Entry("Set 1", 1, 2),
			Entry("Set 2", 5, 10),
			Entry("Set 3", 3, 7),
			Entry("Set 4", 5, 5),
		)
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
