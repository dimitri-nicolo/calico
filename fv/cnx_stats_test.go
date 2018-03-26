// +build fvtests

// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package fv_test

import (
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/fv/containers"
	"github.com/projectcalico/felix/fv/metrics"
	"github.com/projectcalico/felix/fv/utils"
	"github.com/projectcalico/felix/fv/workload"
	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"
)

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
		Expect(w[1]).To(HaveConnectivityTo(w[0]))
		Expect(w[0]).To(HaveConnectivityTo(w[1]))
		Eventually(func() (int, error) {
			return metrics.GetCNXConnectionMetricsIntForPolicy(felix.IP, "profile", "default", "outbound")
		}, "30s", "1s").Should(BeNumerically(">", 0))
		Eventually(func() (int, error) {
			return metrics.GetCNXPacketMetricsIntForPolicy(felix.IP, "allow", "profile", "default", "outbound", "ingress")
		}, "30s", "1s").Should(BeNumerically(">", 0))
		Eventually(func() (int, error) {
			return metrics.GetCNXPacketMetricsIntForPolicy(felix.IP, "allow", "profile", "default", "inbound", "ingress")
		}, "30s", "1s").Should(BeNumerically(">", 0))
		Eventually(func() (int, error) {
			return metrics.GetCNXByteMetricsIntForPolicy(felix.IP, "allow", "profile", "default", "outbound", "ingress")
		}, "30s", "1s").Should(BeNumerically(">", 0))
		Eventually(func() (int, error) {
			return metrics.GetCNXByteMetricsIntForPolicy(felix.IP, "allow", "profile", "default", "inbound", "ingress")
		}, "30s", "1s").Should(BeNumerically(">", 0))
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

		It("w1 cannot connect to w1 and denied packet metrics are generated", func() {
			Eventually(w[0], "10s", "1s").ShouldNot(HaveConnectivityTo(w[1]))
			Expect(w[0]).NotTo(HaveConnectivityTo(w[1]))
			Eventually(func() (int, error) {
				return metrics.GetCNXConnectionMetricsIntForPolicy(felix.IP, "deny", "policy-1", "inbound")
			}, "30s", "1s").Should(BeNumerically("==", 0))
			Eventually(func() (int, error) {
				return metrics.GetCNXPacketMetricsIntForPolicy(felix.IP, "deny", "default", "policy-1", "inbound", "ingress")
			}, "30s", "1s").Should(BeNumerically(">", 0))
			Eventually(func() (int, error) {
				return metrics.GetCNXByteMetricsIntForPolicy(felix.IP, "deny", "default", "policy-1", "inbound", "ingress")
			}, "30s", "1s").Should(BeNumerically(">", 0))
			Eventually(func() (int, error) {
				return metrics.GetCalicoDeniedPacketMetrics(felix.IP, "default", "policy-1")
			}, "30s", "1s").Should(BeNumerically(">", 0))
		})
	})
	Context("should generate metrics with egress deny rule", func() {

		BeforeEach(func() {
			policy := api.NewNetworkPolicy()
			policy.Namespace = "fv"
			policy.Name = "default.policy-1"
			policy.Spec.Tier = "default"
			policy.Spec.Types = []api.PolicyType{api.PolicyTypeEgress}
			policy.Spec.Ingress = []api.Rule{{Action: api.Deny}}
			policy.Spec.Egress = []api.Rule{{Action: api.Deny}}
			policy.Spec.Selector = w[0].NameSelector()
			_, err := client.NetworkPolicies().Create(utils.Ctx, policy, utils.NoOptions)
			Expect(err).NotTo(HaveOccurred())
		})

		It("w0 cannot connect to w1 and denied packet metrics are generated", func() {
			Eventually(w[0], "10s", "1s").ShouldNot(HaveConnectivityTo(w[1]))
			Expect(w[0]).NotTo(HaveConnectivityTo(w[1]))
			Eventually(func() (int, error) {
				return metrics.GetCNXConnectionMetricsIntForPolicy(felix.IP, "deny", "policy-1", "outbound")
			}, "30s", "1s").Should(BeNumerically("==", 0))
			Eventually(func() (int, error) {
				return metrics.GetCNXPacketMetricsIntForPolicy(felix.IP, "deny", "default", "policy-1", "outbound", "egress")
			}, "30s", "1s").Should(BeNumerically(">", 0))
			Eventually(func() (int, error) {
				return metrics.GetCNXByteMetricsIntForPolicy(felix.IP, "deny", "default", "policy-1", "outbound", "egress")
			}, "30s", "1s").Should(BeNumerically(">", 0))
			Eventually(func() (int, error) {
				return metrics.GetCalicoDeniedPacketMetrics(felix.IP, "default", "policy-1")
			}, "30s", "1s").Should(BeNumerically(">", 0))
		})
	})
})
