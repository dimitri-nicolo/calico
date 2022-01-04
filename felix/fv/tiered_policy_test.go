//go:build fvtests
// +build fvtests

// Copyright (c) 2018-2021 Tigera, Inc. All rights reserved.

package fv_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/projectcalico/calico/felix/fv/connectivity"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"os"

	api "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/felix/fv/infrastructure"
	"github.com/projectcalico/calico/felix/fv/utils"
	"github.com/projectcalico/calico/felix/fv/workload"
	"github.com/projectcalico/calico/libcalico-go/lib/apiconfig"
	client "github.com/projectcalico/calico/libcalico-go/lib/clientv3"
)

/*
var (
	float1_0 = float64(1.0)
	float2_0 = float64(2.0)
	float3_0 = float64(3.0)
)*/

// Felix1             Felix2
//  EP1-1 <-+-------> EP2-1
//          \-------> EP2-2
//           `------> EP2-3
//
//       ^           ^-- Apply test policies here (for ingress and egress)
//       `-------------- Allow all policy
//
// Egress Policies (dest ep1-1)
//   Tier1             |   Tier2             | Default         | Profile
//   np1-1 (P2-1,D2-2) |  snp2-1 (A2-1)      | sknp3.1 (N2-1)  | (default A)
//                     |  gnp2-2 (D2-3)      |  -> sknp3.9     |
//
// Ingress Policies (source ep1-1)
//
//   Tier1             |   Tier2             | Default         | Profile
//   np1-1 (A2-1,P2-2) | sgnp2-2 (N2-3)      |  snp3.2 (A2-2)  | (default A)
//                     |  snp2-3 (A2-2,D2-3) |   np3.3 (A2-2)  |
//                     |   np2-4 (D2-3)      |  snp3.4 (A2-2)  |
//
// A=allow; D=deny; N=no-match

// These tests include tests of Kubernetes policies as well as other policy types. To ensure we have the correct
// behavior, run using the Kubernetes infrastructure only.
var _ = infrastructure.DatastoreDescribe("connectivity tests with policy tiers _BPF-SAFE_", []apiconfig.DatastoreType{apiconfig.Kubernetes}, func(getInfra infrastructure.InfraFactory) {

	const (
		wepPort = 8055
		svcPort = 8066
	)
	wepPortStr := fmt.Sprintf("%d", wepPort)
	svcPortStr := fmt.Sprintf("%d", svcPort)

	var (
		infra                      infrastructure.DatastoreInfra
		opts                       infrastructure.TopologyOptions
		felixes                    []*infrastructure.Felix
		client                     client.Interface
		ep1_1, ep2_1, ep2_2, ep2_3 *workload.Workload
		cc                         *connectivity.Checker
	)
	clusterIP := "10.101.0.10"

	BeforeEach(func() {
		infra = getInfra()
		opts = infrastructure.DefaultTopologyOptions()
		opts.IPIPEnabled = false

		// Start felix instances.
		felixes, client = infrastructure.StartNNodeTopology(2, opts, infra)

		// Install a default profile that allows all ingress and egress, in the absence of any Policy.
		infra.AddDefaultAllow()

		// Create workload on host 1.
		ep1_1 = workload.Run(felixes[0], "ep1-1", "default", "10.65.0.0", wepPortStr, "tcp")
		ep1_1.ConfigureInInfra(infra)

		ep2_1 = workload.Run(felixes[1], "ep2-1", "default", "10.65.1.0", wepPortStr, "tcp")
		ep2_1.ConfigureInInfra(infra)

		ep2_2 = workload.Run(felixes[1], "ep2-2", "default", "10.65.1.1", wepPortStr, "tcp")
		ep2_2.ConfigureInInfra(infra)

		ep2_3 = workload.Run(felixes[1], "ep2-3", "default", "10.65.1.2", wepPortStr, "tcp")
		ep2_3.ConfigureInInfra(infra)

		// Create tiers tier1 and tier2
		tier := api.NewTier()
		tier.Name = "tier1"
		tier.Spec.Order = &float1_0
		_, err := client.Tiers().Create(utils.Ctx, tier, utils.NoOptions)

		tier = api.NewTier()
		tier.Name = "tier2"
		tier.Spec.Order = &float2_0
		_, err = client.Tiers().Create(utils.Ctx, tier, utils.NoOptions)

		// Allow all traffic to/from ep1-1
		gnp := api.NewGlobalNetworkPolicy()
		gnp.Name = "default.ep1-1-allow-all"
		gnp.Spec.Order = &float1_0
		gnp.Spec.Tier = "default"
		gnp.Spec.Selector = ep1_1.NameSelector()
		gnp.Spec.Types = []api.PolicyType{api.PolicyTypeIngress, api.PolicyTypeEgress}
		gnp.Spec.Egress = []api.Rule{{Action: api.Allow}}
		gnp.Spec.Ingress = []api.Rule{{Action: api.Allow}}
		_, err = client.GlobalNetworkPolicies().Create(utils.Ctx, gnp, utils.NoOptions)
		Expect(err).NotTo(HaveOccurred())

		// np1-1  egress: (P2-1,D2-2) ingress: (A2-1,P2-2)
		np := api.NewNetworkPolicy()
		np.Name = "tier1.np1-1"
		np.Namespace = "default"
		np.Spec.Order = &float1_0
		np.Spec.Tier = "tier1"
		np.Spec.Selector = "name in {'" + ep2_1.Name + "', '" + ep2_2.Name + "'}"
		np.Spec.Types = []api.PolicyType{api.PolicyTypeIngress, api.PolicyTypeEgress}
		np.Spec.Egress = []api.Rule{
			{Action: api.Pass, Source: api.EntityRule{Selector: ep2_1.NameSelector()}},
			{Action: api.Deny, Source: api.EntityRule{Selector: ep2_2.NameSelector()}},
		}
		np.Spec.Ingress = []api.Rule{
			{Action: api.Allow, Destination: api.EntityRule{Selector: ep2_1.NameSelector()}},
			{Action: api.Pass, Destination: api.EntityRule{Selector: ep2_2.NameSelector()}},
		}
		_, err = client.NetworkPolicies().Create(utils.Ctx, np, utils.NoOptions)
		Expect(err).NotTo(HaveOccurred())

		// (s)np2.1 egress: (A2-1)
		snp := api.NewStagedNetworkPolicy()
		snp.Name = "tier2.np2-1"
		snp.Namespace = "default"
		snp.Spec.Order = &float1_0
		snp.Spec.Tier = "tier2"
		snp.Spec.Selector = ep2_1.NameSelector()
		snp.Spec.Types = []api.PolicyType{api.PolicyTypeEgress}
		snp.Spec.Egress = []api.Rule{{Action: api.Allow}}
		_, err = client.StagedNetworkPolicies().Create(utils.Ctx, snp, utils.NoOptions)
		Expect(err).NotTo(HaveOccurred())

		// gnp2-2 egress: (A2-3)
		gnp = api.NewGlobalNetworkPolicy()
		gnp.Name = "tier2.gnp2-2"
		gnp.Spec.Order = &float2_0
		gnp.Spec.Tier = "tier2"
		gnp.Spec.Selector = ep2_3.NameSelector()
		gnp.Spec.Types = []api.PolicyType{api.PolicyTypeEgress}
		gnp.Spec.Egress = []api.Rule{{Action: api.Deny}}
		_, err = client.GlobalNetworkPolicies().Create(utils.Ctx, gnp, utils.NoOptions)
		Expect(err).NotTo(HaveOccurred())

		// (s)gnp2-2 ingress: (N2-3)
		sgnp := api.NewStagedGlobalNetworkPolicy()
		sgnp.Name = "tier2.gnp2-2"
		sgnp.Spec.Order = &float2_0
		sgnp.Spec.Tier = "tier2"
		sgnp.Spec.Selector = ep2_3.NameSelector()
		sgnp.Spec.Types = []api.PolicyType{api.PolicyTypeIngress}
		_, err = client.StagedGlobalNetworkPolicies().Create(utils.Ctx, sgnp, utils.NoOptions)
		Expect(err).NotTo(HaveOccurred())

		// (s)np2-3 ingress: (A2-2, D2-3)
		snp = api.NewStagedNetworkPolicy()
		snp.Name = "tier2.np2-3"
		snp.Namespace = "default"
		snp.Spec.Order = &float3_0
		snp.Spec.Tier = "tier2"
		snp.Spec.Selector = "name in {'" + ep2_2.Name + "', '" + ep2_3.Name + "'}"
		snp.Spec.Types = []api.PolicyType{api.PolicyTypeIngress}
		snp.Spec.Ingress = []api.Rule{
			{Action: api.Allow, Destination: api.EntityRule{Selector: ep2_2.NameSelector()}},
			{Action: api.Deny, Destination: api.EntityRule{Selector: ep2_3.NameSelector()}},
		}
		_, err = client.StagedNetworkPolicies().Create(utils.Ctx, snp, utils.NoOptions)
		Expect(err).NotTo(HaveOccurred())

		// np2-4 ingress: (D2-3)
		np = api.NewNetworkPolicy()
		np.Name = "tier2.np2-4"
		np.Namespace = "default"
		np.Spec.Order = &float3_0
		np.Spec.Tier = "tier2"
		np.Spec.Selector = ep2_3.NameSelector()
		np.Spec.Types = []api.PolicyType{api.PolicyTypeIngress}
		np.Spec.Ingress = []api.Rule{{Action: api.Deny}}
		_, err = client.NetworkPolicies().Create(utils.Ctx, np, utils.NoOptions)
		Expect(err).NotTo(HaveOccurred())

		// (s)knp3.1->sknp3.9 egress: (N2-1)
		for i := 0; i < 9; i++ {
			sknp := api.NewStagedKubernetesNetworkPolicy()
			sknp.Name = fmt.Sprintf("knp3-%d", i+1)
			sknp.Namespace = "default"
			sknp.Spec.PodSelector = metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": ep2_1.Name,
				},
			}
			sknp.Spec.PolicyTypes = []networkingv1.PolicyType{networkingv1.PolicyTypeEgress}
			_, err = client.StagedKubernetesNetworkPolicies().Create(utils.Ctx, sknp, utils.NoOptions)
			Expect(err).NotTo(HaveOccurred())
		}

		// (s)np3.2 ingress: (A2-2)
		snp = api.NewStagedNetworkPolicy()
		snp.Name = "default.np3-2"
		snp.Namespace = "default"
		snp.Spec.Order = &float1_0
		snp.Spec.Tier = "default"
		snp.Spec.Selector = ep2_2.NameSelector()
		snp.Spec.Types = []api.PolicyType{api.PolicyTypeIngress}
		snp.Spec.Ingress = []api.Rule{{Action: api.Allow}}
		_, err = client.StagedNetworkPolicies().Create(utils.Ctx, snp, utils.NoOptions)
		Expect(err).NotTo(HaveOccurred())

		// np3.3 ingress: (A2-2)
		np = api.NewNetworkPolicy()
		np.Name = "default.np3-3"
		np.Namespace = "default"
		np.Spec.Order = &float2_0
		np.Spec.Tier = "default"
		np.Spec.Selector = ep2_2.NameSelector()
		np.Spec.Types = []api.PolicyType{api.PolicyTypeIngress}
		np.Spec.Ingress = []api.Rule{{Action: api.Allow}}
		_, err = client.NetworkPolicies().Create(utils.Ctx, np, utils.NoOptions)
		Expect(err).NotTo(HaveOccurred())

		// (s)np3.4 ingress: (A2-2)
		snp = api.NewStagedNetworkPolicy()
		snp.Name = "default.np3-4"
		snp.Namespace = "default"
		snp.Spec.Order = &float3_0
		snp.Spec.Tier = "default"
		snp.Spec.Selector = ep2_2.NameSelector()
		snp.Spec.Types = []api.PolicyType{api.PolicyTypeIngress}
		snp.Spec.Ingress = []api.Rule{{Action: api.Allow}}
		_, err = client.StagedNetworkPolicies().Create(utils.Ctx, snp, utils.NoOptions)
		Expect(err).NotTo(HaveOccurred())
		// Create a service that maps to ep2_1. Rather than checking connectivity to the endpoint we'll go via
		// the service to test the destination service name handling.
		svcName := "test-service"
		k8sClient := infra.(*infrastructure.K8sDatastoreInfra).K8sClient
		tSvc := k8sService(svcName, clusterIP, ep2_1, svcPort, wepPort, 0, "tcp")
		tSvcNamespace := tSvc.ObjectMeta.Namespace
		_, err = k8sClient.CoreV1().Services(tSvcNamespace).Create(context.Background(), tSvc, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		// Wait for the endpoints to be updated and for the address to be ready.
		Expect(ep2_1.IP).NotTo(Equal(""))
		getEpsFunc := k8sGetEpsForServiceFunc(k8sClient, tSvc)
		epCorrectFn := func() error {
			eps := getEpsFunc()
			if len(eps) != 1 {
				return fmt.Errorf("Wrong number of endpoints: %#v", eps)
			}
			addrs := eps[0].Addresses
			if len(addrs) != 1 {
				return fmt.Errorf("Wrong number of addresses: %#v", eps[0])
			}
			if addrs[0].IP != ep2_1.IP {
				return fmt.Errorf("Unexpected IP: %s != %s", addrs[0].IP, ep2_1.IP)
			}
			ports := eps[0].Ports
			if len(ports) != 1 {
				return fmt.Errorf("Wrong number of ports: %#v", eps[0])
			}
			if ports[0].Port != int32(wepPort) {
				return fmt.Errorf("Wrong port %d != svcPort", ports[0].Port)
			}
			return nil
		}
		Eventually(epCorrectFn, "10s").ShouldNot(HaveOccurred())

		if os.Getenv("FELIX_FV_ENABLE_BPF") != "true" {
			// Wait for felix to see and program some expected nflog entries, and for the cluster IP to appear.
			rulesProgrammed := func() bool {
				out0, err := felixes[0].ExecOutput("iptables-save", "-t", "filter")
				Expect(err).NotTo(HaveOccurred())
				out1, err := felixes[1].ExecOutput("iptables-save", "-t", "filter")
				Expect(err).NotTo(HaveOccurred())
				return strings.Count(out0, "APE0|default.ep1-1-allow-all") > 0 &&
					strings.Count(out1, "APE0|default.ep1-1-allow-all") == 0 &&
					strings.Count(out0, "DPI|default/staged:default.np3-4") == 0 &&
					strings.Count(out1, "DPI|default/staged:default.np3-4") > 0
			}
			Eventually(rulesProgrammed, "10s", "1s").Should(BeTrue(),
				"Expected iptables rules to appear on the correct felix instances")

			// Mimic the kube-proxy service iptable clusterIP rule.
			for _, f := range felixes {
				f.Exec("iptables", "-t", "nat", "-A", "PREROUTING",
					"-p", "tcp",
					"-d", clusterIP,
					"-m", "tcp", "--dport", svcPortStr,
					"-j", "DNAT", "--to-destination",
					ep2_1.IP+":"+wepPortStr)
			}

		}
	})

	It("should test connectivity between workloads with policy tiers", func() {
		cc = &connectivity.Checker{}
		cc.ExpectSome(ep1_1, connectivity.TargetIP(clusterIP), uint16(svcPort)) // allowed by np1-1
		cc.ExpectSome(ep1_1, ep2_2)                                             // allowed by np3-3
		cc.ExpectNone(ep1_1, ep2_3)                                             // denied by np2-4

		cc.ExpectSome(ep2_1, ep1_1) // allowed by profile
		cc.ExpectNone(ep2_2, ep1_1) // denied by np1-1
		cc.ExpectNone(ep2_3, ep1_1) // denied by gnp2-2

		// Do 3 rounds of connectivity checking.
		cc.CheckConnectivity()
	})

	AfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			for _, felix := range felixes {
				felix.Exec("iptables-save", "-c")
				felix.Exec("ipset", "list")
				felix.Exec("ip", "r")
				felix.Exec("ip", "a")
			}
		}

		ep1_1.Stop()
		ep2_1.Stop()
		ep2_2.Stop()
		ep2_3.Stop()
		for _, felix := range felixes {
			felix.Stop()
		}

		if CurrentGinkgoTestDescription().Failed {
			infra.DumpErrorData()
		}
		infra.Stop()
	})
})
