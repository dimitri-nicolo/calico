// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.

package intdataplane

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"time"

	"golang.org/x/sys/unix"

	"github.com/golang-collections/collections/stack"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/felix/ip"
	"github.com/projectcalico/calico/felix/logutils"
	"github.com/projectcalico/calico/felix/proto"
	"github.com/projectcalico/calico/felix/routerule"
	"github.com/projectcalico/calico/felix/routetable"
	"github.com/projectcalico/calico/felix/rules"
	"github.com/projectcalico/calico/libcalico-go/lib/health"
	"github.com/projectcalico/calico/libcalico-go/lib/set"

	"github.com/vishvananda/netlink"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
)

var _ = Describe("EgressIPManager", func() {
	var manager *egressIPManager
	var dpConfig Config
	var rr *mockRouteRules
	var mainTable *mockRouteTable
	var rrFactory *mockRouteRulesFactory
	var rtFactory *mockRouteTableFactory
	var podStatusCallback *mockEgressPodStatusCallback
	var healthAgg *health.HealthAggregator

	BeforeEach(func() {
		rrFactory = &mockRouteRulesFactory{routeRules: nil}

		mainTable = &mockRouteTable{
			index:           0,
			currentRoutes:   map[string][]routetable.Target{},
			currentL2Routes: map[string][]routetable.L2Target{},
		}
		rtFactory = &mockRouteTableFactory{count: 0, tables: make(map[int]*mockRouteTable)}

		// Ten free tables to use.
		tableIndexSet := set.New()
		tableIndexStack := stack.New()
		for i := 10; i > 0; i-- {
			tableIndexStack.Push(i)
			tableIndexSet.Add(i)
		}

		dpConfig = Config{
			RulesConfig: rules.Config{
				IptablesMarkEgress: 0x200,
				EgressIPVXLANVNI:   2,
				EgressIPVXLANPort:  4790,
			},
			EgressIPRoutingRulePriority: 100,
			FelixHostname:               "host0",
		}

		podStatusCallback = &mockEgressPodStatusCallback{state: []statusCallbackEntry{}}
		healthAgg = health.NewHealthAggregator()

		manager = newEgressIPManagerWithShims(
			mainTable,
			rrFactory,
			rtFactory,
			tableIndexSet,
			tableIndexStack,
			"egress.calico",
			dpConfig,
			&mockVXLANDataplane{
				links: []netlink.Link{&mockLink{attrs: netlink.LinkAttrs{Name: "egress.calico"}}},
			},
			logutils.NewSummarizer("test loop"),
			func(ifName string) error { return nil },
			podStatusCallback.statusCallback,
			healthAgg,
			rand.NewSource(1), // Seed with 1 to get predictable tests every time.
		)

		Expect(healthAgg.Summary().Ready).To(BeFalse())

		err := manager.CompleteDeferredWork()
		Expect(err).ToNot(HaveOccurred())
		Expect(healthAgg.Summary().Ready).To(BeTrue())

		// No routeRules should be created.
		Expect(manager.routeRules).To(BeNil())

		manager.OnUpdate(&proto.HostMetadataUpdate{
			Hostname: "host0",
			Ipv4Addr: "172.0.0.2", // mockVXLANDataplane use interface address 172.0.0.2
		})
		manager.lock.Lock()
		nodeIP := manager.nodeIP
		manager.lock.Unlock()
		Expect(nodeIP).To(Equal(net.ParseIP("172.0.0.2")))
		err = manager.configureVXLANDevice(50)
		Expect(err).NotTo(HaveOccurred())
		Expect(manager.vxlanDeviceLinkIndex).To(Equal(6))
	})

	expectIPSetMembers := func(id string, members []gateway) {
		var matchers []types.GomegaMatcher
		for _, m := range members {
			matchers = append(matchers, ipSetMemberEquals(m))
		}
		Expect(manager.ipSetIDToGateways[id]).To(ContainElements(matchers))
	}

	expectNoRulesAndTable := func(srcIPs []string, table int) {
		var activeRules []netlink.Rule
		for _, r := range rr.GetAllActiveRules() {
			activeRules = append(activeRules, *r.NetLinkRule())
		}
		for _, srcIP := range srcIPs {
			Expect(rr.hasRule(100, srcIP, 0x200, table)).To(BeFalse(), "Expect rule with srcIP: %s, and table: %d, to not exist. Active rules = %v", srcIP, table, activeRules)
		}
		rtFactory.Table(table).checkRoutes(routetable.InterfaceNone, nil)
		rtFactory.Table(table).checkRoutes("egress.calico", nil)
	}

	expectRulesAndTable := func(srcIPs []string, table int, hopIPs []string) {
		var activeRules []netlink.Rule
		for _, r := range rr.GetAllActiveRules() {
			activeRules = append(activeRules, *r.NetLinkRule())
		}
		for _, srcIP := range srcIPs {
			Expect(rr.hasRule(100, srcIP, 0x200, table)).To(BeTrue(), "Expect rule with srcIP: %s, and table: %d, to exist. Active rules = %v", srcIP, table, activeRules)
		}

		var targets []routetable.Target
		if len(hopIPs) == 0 {
			targets = []routetable.Target{{
				Type: routetable.TargetTypeUnreachable,
				CIDR: defaultCidr,
			}}
			rtFactory.Table(table).checkRoutes(routetable.InterfaceNone, targets)
		} else if len(hopIPs) == 1 {
			targets = []routetable.Target{{
				Type: routetable.TargetTypeVXLAN,
				CIDR: defaultCidr,
				GW:   ip.FromString(hopIPs[0]),
			}}
			rtFactory.Table(table).checkRoutes("egress.calico", targets)
		} else {
			targets = []routetable.Target{{
				Type:      routetable.TargetTypeVXLAN,
				CIDR:      defaultCidr,
				MultiPath: multiPath(hopIPs, manager.vxlanDeviceLinkIndex),
			}}
			rtFactory.Table(table).checkRoutes(routetable.InterfaceNone, targets)
		}
	}

	Describe("with multiple ipsets and endpoints update", func() {
		var ips0, ips1 []string
		var zeroTime, nowTime, thirtySecsAgo, inThirtySecsTime, inSixtySecsTime time.Time
		BeforeEach(func() {
			zeroTime = time.Time{}
			nowTime = time.Now()
			thirtySecsAgo = nowTime.Add(time.Second * -30)
			inThirtySecsTime = nowTime.Add(time.Second * 30)
			inSixtySecsTime = nowTime.Add(time.Second * 60)

			ips0 = []string{
				formatActiveEgressMemberStr("10.0.0.1"),
				formatActiveEgressMemberStr("10.0.0.2"),
				formatActiveEgressMemberStr("10.0.0.3"),
			}
			ips1 = []string{
				formatActiveEgressMemberStr("10.0.1.1"),
				formatActiveEgressMemberStr("10.0.1.2"),
				formatActiveEgressMemberStr("10.0.1.3"),
			}

			manager.OnUpdate(&proto.IPSetUpdate{
				Id:      "set0",
				Members: ips0,
				Type:    proto.IPSetUpdate_EGRESS_IP,
			})
			manager.OnUpdate(&proto.IPSetUpdate{
				Id:      "set1",
				Members: ips1,
				Type:    proto.IPSetUpdate_EGRESS_IP,
			})
			manager.OnUpdate(&proto.IPSetUpdate{
				Id:      "nonEgressIPSet",
				Members: []string{"10.0.100.1", "10.0.100.2"},
				Type:    proto.IPSetUpdate_IP,
			})

			expectIPSetMembers("set0", []gateway{
				{
					cidr:                "10.0.0.1",
					maintenanceStarted:  zeroTime,
					maintenanceFinished: zeroTime,
				},
				{
					cidr:                "10.0.0.2",
					maintenanceStarted:  zeroTime,
					maintenanceFinished: zeroTime,
				},
				{
					cidr:                "10.0.0.3",
					maintenanceStarted:  zeroTime,
					maintenanceFinished: zeroTime,
				},
			})
			expectIPSetMembers("set1", []gateway{
				{
					cidr:                "10.0.1.1",
					maintenanceStarted:  zeroTime,
					maintenanceFinished: zeroTime,
				},
				{
					cidr:                "10.0.1.2",
					maintenanceStarted:  zeroTime,
					maintenanceFinished: zeroTime,
				},
				{
					cidr:                "10.0.1.3",
					maintenanceStarted:  zeroTime,
					maintenanceFinished: zeroTime,
				},
			})
			Expect(manager.ipSetIDToGateways["nonEgressSet"]).To(BeNil())

			err := manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())

			manager.OnUpdate(dummyWorkloadEndpointUpdate(0, "set0", []string{"10.0.240.0/32"}, 0))
			manager.OnUpdate(dummyWorkloadEndpointUpdate(1, "set0", []string{"10.0.241.0/32"}, 0))
			manager.OnUpdate(dummyWorkloadEndpointUpdate(2, "set0", []string{"10.0.242.0/32"}, 0))
			manager.OnUpdate(dummyWorkloadEndpointUpdate(3, "set1", []string{"10.0.243.0/32"}, 2))
			manager.OnUpdate(dummyWorkloadEndpointUpdate(4, "set1", []string{"10.0.244.0/32"}, 2))
			manager.OnUpdate(dummyWorkloadEndpointUpdate(5, "set1", []string{"10.0.245.0/32"}, 2))

			err = manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())

			// routeRules should be created.
			Expect(manager.routeRules).NotTo(BeNil())
			rr = rrFactory.Rules()

			expectRulesAndTable([]string{"10.0.240.0/32"}, 1, []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"})
			rtFactory.Table(1).checkL2Routes(routetable.InterfaceNone, nil)
			rtFactory.Table(1).checkL2Routes("egress.calico", nil)

			expectRulesAndTable([]string{"10.0.241.0/32"}, 2, []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"})
			rtFactory.Table(1).checkL2Routes(routetable.InterfaceNone, nil)
			rtFactory.Table(1).checkL2Routes("egress.calico", nil)

			expectRulesAndTable([]string{"10.0.242.0/32"}, 3, []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"})
			rtFactory.Table(1).checkL2Routes(routetable.InterfaceNone, nil)
			rtFactory.Table(1).checkL2Routes("egress.calico", nil)

			expectRulesAndTable([]string{"10.0.243.0/32"}, 4, []string{"10.0.1.2", "10.0.1.3"})
			rtFactory.Table(2).checkL2Routes(routetable.InterfaceNone, nil)
			rtFactory.Table(2).checkL2Routes("egress.calico", nil)

			expectRulesAndTable([]string{"10.0.244.0/32"}, 5, []string{"10.0.1.1", "10.0.1.3"})
			rtFactory.Table(2).checkL2Routes(routetable.InterfaceNone, nil)
			rtFactory.Table(2).checkL2Routes("egress.calico", nil)

			expectRulesAndTable([]string{"10.0.245.0/32"}, 6, []string{"10.0.1.1", "10.0.1.2"})
			rtFactory.Table(2).checkL2Routes(routetable.InterfaceNone, nil)
			rtFactory.Table(2).checkL2Routes("egress.calico", nil)

			mainTable.checkRoutes(routetable.InterfaceNone, nil)
			mainTable.checkRoutes("egress.calico", nil)
			mainTable.checkL2Routes("egress.calico", []routetable.L2Target{
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x01},
					GW:      ip.FromString("10.0.0.1"),
					IP:      ip.FromString("10.0.0.1"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x02},
					GW:      ip.FromString("10.0.0.2"),
					IP:      ip.FromString("10.0.0.2"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x03},
					GW:      ip.FromString("10.0.0.3"),
					IP:      ip.FromString("10.0.0.3"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x01, 0x01},
					GW:      ip.FromString("10.0.1.1"),
					IP:      ip.FromString("10.0.1.1"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x01, 0x02},
					GW:      ip.FromString("10.0.1.2"),
					IP:      ip.FromString("10.0.1.2"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x01, 0x03},
					GW:      ip.FromString("10.0.1.3"),
					IP:      ip.FromString("10.0.1.3"),
				},
			})
		})

		It("should support delta update", func() {
			manager.OnUpdate(&proto.IPSetDeltaUpdate{
				Id:             "set1",
				AddedMembers:   []string{formatActiveEgressMemberStr("10.0.1.4"), formatActiveEgressMemberStr("10.0.1.5")},
				RemovedMembers: []string{formatActiveEgressMemberStr("10.0.1.1")},
			})

			err := manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())

			// Changes to an IPSet should have no impact on existing workload tables, only on new workloads.
			expectRulesAndTable([]string{"10.0.243.0/32"}, 4, []string{"10.0.1.2", "10.0.1.3"})
			expectRulesAndTable([]string{"10.0.244.0/32"}, 5, []string{"10.0.1.4", "10.0.1.5"})
			expectRulesAndTable([]string{"10.0.245.0/32"}, 6, []string{"10.0.1.3", "10.0.1.4"})
			mainTable.checkL2Routes("egress.calico", []routetable.L2Target{
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x01},
					GW:      ip.FromString("10.0.0.1"),
					IP:      ip.FromString("10.0.0.1"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x02},
					GW:      ip.FromString("10.0.0.2"),
					IP:      ip.FromString("10.0.0.2"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x03},
					GW:      ip.FromString("10.0.0.3"),
					IP:      ip.FromString("10.0.0.3"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x01, 0x02},
					GW:      ip.FromString("10.0.1.2"),
					IP:      ip.FromString("10.0.1.2"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x01, 0x03},
					GW:      ip.FromString("10.0.1.3"),
					IP:      ip.FromString("10.0.1.3"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x01, 0x04},
					GW:      ip.FromString("10.0.1.4"),
					IP:      ip.FromString("10.0.1.4"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x01, 0x05},
					GW:      ip.FromString("10.0.1.5"),
					IP:      ip.FromString("10.0.1.5"),
				},
			})
		})

		It("should release table correctly", func() {
			manager.OnUpdate(&proto.WorkloadEndpointRemove{
				Id: &proto.WorkloadEndpointID{
					OrchestratorId: "k8s",
					WorkloadId:     "default/pod-1",
					EndpointId:     "endpoint-id-1",
				},
			})

			err := manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())

			Expect(manager.tableIndexStack.Peek()).To(Equal(2))
			Expect(manager.tableIndexStack.Len()).To(Equal(5))
			expectNoRulesAndTable([]string{"10.0.241.0/32"}, 2)
			mainTable.checkL2Routes("egress.calico", []routetable.L2Target{
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x01},
					GW:      ip.FromString("10.0.0.1"),
					IP:      ip.FromString("10.0.0.1"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x02},
					GW:      ip.FromString("10.0.0.2"),
					IP:      ip.FromString("10.0.0.2"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x03},
					GW:      ip.FromString("10.0.0.3"),
					IP:      ip.FromString("10.0.0.3"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x01, 0x01},
					GW:      ip.FromString("10.0.1.1"),
					IP:      ip.FromString("10.0.1.1"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x01, 0x02},
					GW:      ip.FromString("10.0.1.2"),
					IP:      ip.FromString("10.0.1.2"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x01, 0x03},
					GW:      ip.FromString("10.0.1.3"),
					IP:      ip.FromString("10.0.1.3"),
				},
			})

			// Send same workload endpoint remove
			manager.OnUpdate(&proto.WorkloadEndpointRemove{
				Id: &proto.WorkloadEndpointID{
					OrchestratorId: "k8s",
					WorkloadId:     "default/pod-1",
					EndpointId:     "endpoint-id-1",
				},
			})

			err = manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())

			Expect(manager.tableIndexStack.Peek()).To(Equal(2))
			Expect(manager.tableIndexStack.Len()).To(Equal(5))
			rtFactory.Table(2).checkRoutes("egress.calico", nil)
		})

		It("should report unhealthy if run out of table index", func() {
			for i := 2; i < 10; i++ {
				manager.OnUpdate(dummyWorkloadEndpointUpdate(i, "set0", []string{fmt.Sprintf("10.0.24%d.0/32", i)}, 0))
			}

			err := manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			Expect(manager.tableIndexStack.Len()).To(Equal(0))

			breakingWorkloadUpdate := dummyWorkloadEndpointUpdate(11, "set0", []string{"10.0.250.0/32"}, 0)
			manager.OnUpdate(breakingWorkloadUpdate)

			err = manager.CompleteDeferredWork()
			Expect(err).NotTo(HaveOccurred()) // the manager will not report an error to the dataplane but should report unhealthy
			Expect(healthAgg.Summary().Ready).To(BeFalse())
			Expect(manager.pendingWorkloadUpdates).To(HaveKey(*breakingWorkloadUpdate.Id))

			// resolve the issue
			resolvingUpdate := proto.WorkloadEndpointRemove{
				Id: &proto.WorkloadEndpointID{
					OrchestratorId: "k8s",
					WorkloadId:     "default/pod-9",
					EndpointId:     "endpoint-id-9",
				},
			}
			manager.OnUpdate(&resolvingUpdate)
			err = manager.CompleteDeferredWork()
			Expect(err).NotTo(HaveOccurred())
			Expect(healthAgg.Summary().Ready).To(BeTrue())
			Expect(manager.dirtyEgressIPSet).NotTo(ContainElement(*breakingWorkloadUpdate.Id))
		})

		It("should use same table if endpoint has second ip address", func() {
			manager.OnUpdate(dummyWorkloadEndpointUpdate(6, "set0", []string{"10.0.246.0/32", "10.1.246.0/32"}, 0))
			err := manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())

			expectRulesAndTable([]string{"10.0.246.0/32", "10.1.246.0/32"}, 7, []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"})
			mainTable.checkL2Routes("egress.calico", []routetable.L2Target{
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x01},
					GW:      ip.FromString("10.0.0.1"),
					IP:      ip.FromString("10.0.0.1"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x02},
					GW:      ip.FromString("10.0.0.2"),
					IP:      ip.FromString("10.0.0.2"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x03},
					GW:      ip.FromString("10.0.0.3"),
					IP:      ip.FromString("10.0.0.3"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x01, 0x01},
					GW:      ip.FromString("10.0.1.1"),
					IP:      ip.FromString("10.0.1.1"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x01, 0x02},
					GW:      ip.FromString("10.0.1.2"),
					IP:      ip.FromString("10.0.1.2"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x01, 0x03},
					GW:      ip.FromString("10.0.1.3"),
					IP:      ip.FromString("10.0.1.3"),
				},
			})
		})

		It("should set unreachable route if egress ipset has all members removed", func() {
			manager.OnUpdate(&proto.IPSetDeltaUpdate{
				Id:           "set1",
				AddedMembers: []string{},
				RemovedMembers: []string{
					formatActiveEgressMemberStr("10.0.1.1"),
					formatActiveEgressMemberStr("10.0.1.2"),
					formatActiveEgressMemberStr("10.0.1.3"),
				},
			})
			manager.OnUpdate(dummyWorkloadEndpointUpdate(2, "set1", []string{"10.0.242.0/32"}, 0))

			err := manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			expectRulesAndTable([]string{"10.0.242.0/32"}, 3, []string{})
			mainTable.checkL2Routes("egress.calico", []routetable.L2Target{
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x01},
					GW:      ip.FromString("10.0.0.1"),
					IP:      ip.FromString("10.0.0.1"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x02},
					GW:      ip.FromString("10.0.0.2"),
					IP:      ip.FromString("10.0.0.2"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x03},
					GW:      ip.FromString("10.0.0.3"),
					IP:      ip.FromString("10.0.0.3"),
				},
			})
		})

		It("should remove routes and tables for old workload", func() {
			manager.OnUpdate(&proto.WorkloadEndpointUpdate{
				Id: &proto.WorkloadEndpointID{
					OrchestratorId: "k8s",
					WorkloadId:     "default/pod-0",
					EndpointId:     "endpoint-id-0",
				},
				Endpoint: nil,
			})
			err := manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())

			expectNoRulesAndTable([]string{"10.0.240.0/32"}, 1)
		})

		It("should set recreate rule and table for workload if egress ipset changed", func() {
			// pod-0 use table 1 at start.
			expectRulesAndTable([]string{"10.0.240.0/32"}, 1, []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"})
			Expect(rr.hasRule(100, "10.0.240.0/32", 0x200, 1)).To(BeTrue())
			Expect(rr.hasRule(100, "10.0.240.0/32", 0x200, 2)).To(BeFalse())

			// Update pod-0 to use ipset set1.
			manager.OnUpdate(dummyWorkloadEndpointUpdate(0, "set1", []string{"10.0.240.0/32"}, 0))

			err := manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())

			// pod-0 use table 2 as the result.
			expectRulesAndTable([]string{"10.0.240.0/32"}, 1, []string{"10.0.1.1", "10.0.1.2", "10.0.1.3"})
		})

		It("should wait for ipset update", func() {
			id0 := proto.WorkloadEndpointID{
				OrchestratorId: "k8s",
				WorkloadId:     "default/pod-0",
				EndpointId:     "endpoint-id-0",
			}

			endpoint0 := &proto.WorkloadEndpoint{
				State:         "active",
				Mac:           "01:02:03:04:05:06",
				Name:          "cali12345-0",
				ProfileIds:    []string{},
				Tiers:         []*proto.TierInfo{},
				Ipv4Nets:      []string{"10.0.240.0/32"},
				Ipv6Nets:      []string{"2001:db8:2::2/128"},
				EgressIpSetId: "setx",
			}
			// Update pod-0 to use ipset setx.
			manager.OnUpdate(&proto.WorkloadEndpointUpdate{
				Id:       &id0,
				Endpoint: endpoint0,
			})

			err := manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			expectRulesAndTable([]string{"10.0.240.0/32"}, 1, []string{})

			manager.OnUpdate(&proto.IPSetUpdate{
				Id: "setx",
				Members: []string{
					formatActiveEgressMemberStr("10.0.10.1"),
					formatActiveEgressMemberStr("10.0.10.2"),
					formatActiveEgressMemberStr("10.0.10.3"),
				},
				Type: proto.IPSetUpdate_EGRESS_IP,
			})
			err = manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())

			// pod-0 use table 1 as the result.
			expectRulesAndTable([]string{"10.0.240.0/32"}, 1, []string{"10.0.10.1", "10.0.10.2", "10.0.10.3"})
			mainTable.checkL2Routes("egress.calico", []routetable.L2Target{
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x01},
					GW:      ip.FromString("10.0.0.1"),
					IP:      ip.FromString("10.0.0.1"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x02},
					GW:      ip.FromString("10.0.0.2"),
					IP:      ip.FromString("10.0.0.2"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x03},
					GW:      ip.FromString("10.0.0.3"),
					IP:      ip.FromString("10.0.0.3"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x01, 0x01},
					GW:      ip.FromString("10.0.1.1"),
					IP:      ip.FromString("10.0.1.1"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x01, 0x02},
					GW:      ip.FromString("10.0.1.2"),
					IP:      ip.FromString("10.0.1.2"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x01, 0x03},
					GW:      ip.FromString("10.0.1.3"),
					IP:      ip.FromString("10.0.1.3"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x0a, 0x01},
					GW:      ip.FromString("10.0.10.1"),
					IP:      ip.FromString("10.0.10.1"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x0a, 0x02},
					GW:      ip.FromString("10.0.10.2"),
					IP:      ip.FromString("10.0.10.2"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x0a, 0x03},
					GW:      ip.FromString("10.0.10.3"),
					IP:      ip.FromString("10.0.10.3"),
				},
			})
		})

		It("should leave terminating egw pod in existing tables, but not use it for new tables", func() {
			manager.OnUpdate(&proto.IPSetDeltaUpdate{
				Id:             "set0",
				AddedMembers:   []string{formatTerminatingEgressMemberStr("10.0.0.1", nowTime, inSixtySecsTime)},
				RemovedMembers: []string{formatActiveEgressMemberStr("10.0.0.1")},
			})

			err := manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			expectRulesAndTable([]string{"10.0.240.0/32"}, 1, []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"})
			expectRulesAndTable([]string{"10.0.241.0/32"}, 2, []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"})
			expectRulesAndTable([]string{"10.0.242.0/32"}, 3, []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"})
			mainTable.checkL2Routes("egress.calico", []routetable.L2Target{
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x01},
					GW:      ip.FromString("10.0.0.1"),
					IP:      ip.FromString("10.0.0.1"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x02},
					GW:      ip.FromString("10.0.0.2"),
					IP:      ip.FromString("10.0.0.2"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x03},
					GW:      ip.FromString("10.0.0.3"),
					IP:      ip.FromString("10.0.0.3"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x01, 0x01},
					GW:      ip.FromString("10.0.1.1"),
					IP:      ip.FromString("10.0.1.1"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x01, 0x02},
					GW:      ip.FromString("10.0.1.2"),
					IP:      ip.FromString("10.0.1.2"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x01, 0x03},
					GW:      ip.FromString("10.0.1.3"),
					IP:      ip.FromString("10.0.1.3"),
				},
			})
			podStatusCallback.checkState([]statusCallbackEntry{
				{
					namespace: "default",

					name:                "host0-k8s-pod--0-endpoint--id--0",
					ip:                  "10.0.0.1",
					maintenanceStarted:  nowTime,
					maintenanceFinished: inSixtySecsTime,
				},
				{
					namespace:           "default",
					name:                "host0-k8s-pod--1-endpoint--id--1",
					ip:                  "10.0.0.1",
					maintenanceStarted:  nowTime,
					maintenanceFinished: inSixtySecsTime,
				},
				{
					namespace:           "default",
					name:                "host0-k8s-pod--2-endpoint--id--2",
					ip:                  "10.0.0.1",
					maintenanceStarted:  nowTime,
					maintenanceFinished: inSixtySecsTime,
				},
			})

			// Create new endpoint6. It has specified 3 next hops, but only 2 are currently available.
			manager.OnUpdate(dummyWorkloadEndpointUpdate(6, "set0", []string{"10.0.246.0/32"}, 3))
			// Create new endpoint7. It has specified 0 next hops, and so will be allocated all available hops.
			manager.OnUpdate(dummyWorkloadEndpointUpdate(7, "set0", []string{"10.0.247.0/32"}, 0))

			err = manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			expectRulesAndTable([]string{"10.0.246.0/32"}, 7, []string{"10.0.0.2", "10.0.0.3"})
			expectRulesAndTable([]string{"10.0.247.0/32"}, 8, []string{"10.0.0.2", "10.0.0.3"})

			manager.OnUpdate(&proto.IPSetDeltaUpdate{
				Id:             "set0",
				AddedMembers:   []string{formatTerminatingEgressMemberStr("10.0.0.4", zeroTime, zeroTime)},
				RemovedMembers: []string{formatActiveEgressMemberStr("10.0.0.1")},
			})

			// Create new endpoint8. It has specified 3 next hops, which are currently available.
			manager.OnUpdate(dummyWorkloadEndpointUpdate(8, "set0", []string{"10.0.248.0/32"}, 3))
			// Create new endpoint9. It has specified 0 next hops, and so will be allocated all available hops.
			manager.OnUpdate(dummyWorkloadEndpointUpdate(9, "set0", []string{"10.0.249.0/32"}, 0))

			err = manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			expectRulesAndTable([]string{"10.0.248.0/32"}, 9, []string{"10.0.0.2", "10.0.0.3", "10.0.0.4"})
			expectRulesAndTable([]string{"10.0.249.0/32"}, 10, []string{"10.0.0.2", "10.0.0.3", "10.0.0.4"})
		})

		It("should not notify when maintenance window is unchanged", func() {
			manager.OnUpdate(&proto.IPSetDeltaUpdate{
				Id:             "set0",
				AddedMembers:   []string{formatTerminatingEgressMemberStr("10.0.0.1", nowTime, inSixtySecsTime)},
				RemovedMembers: []string{formatActiveEgressMemberStr("10.0.0.1")},
			})

			err := manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			podStatusCallback.checkState([]statusCallbackEntry{
				{
					namespace:           "default",
					name:                "host0-k8s-pod--0-endpoint--id--0",
					ip:                  "10.0.0.1",
					maintenanceStarted:  nowTime,
					maintenanceFinished: inSixtySecsTime,
				},
				{
					namespace:           "default",
					name:                "host0-k8s-pod--1-endpoint--id--1",
					ip:                  "10.0.0.1",
					maintenanceStarted:  nowTime,
					maintenanceFinished: inSixtySecsTime,
				},
				{
					namespace:           "default",
					name:                "host0-k8s-pod--2-endpoint--id--2",
					ip:                  "10.0.0.1",
					maintenanceStarted:  nowTime,
					maintenanceFinished: inSixtySecsTime,
				},
			})

			podStatusCallback.clearState()
			err = manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			podStatusCallback.checkState([]statusCallbackEntry{})
		})

		It("should correctly calculate maintenance window for multiple terminating gateway pods", func() {
			manager.OnUpdate(&proto.IPSetDeltaUpdate{
				Id: "set0",
				AddedMembers: []string{
					formatTerminatingEgressMemberStr("10.0.0.1", thirtySecsAgo, inThirtySecsTime),
					formatTerminatingEgressMemberStr("10.0.0.2", nowTime, inSixtySecsTime),
				},
				RemovedMembers: []string{
					formatActiveEgressMemberStr("10.0.0.1"),
					formatActiveEgressMemberStr("10.0.0.2"),
				},
			})

			err := manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			podStatusCallback.checkState([]statusCallbackEntry{
				{
					namespace:           "default",
					name:                "host0-k8s-pod--0-endpoint--id--0",
					ip:                  "10.0.0.2",
					maintenanceStarted:  nowTime,
					maintenanceFinished: inSixtySecsTime,
				},
				{
					namespace:           "default",
					name:                "host0-k8s-pod--1-endpoint--id--1",
					ip:                  "10.0.0.2",
					maintenanceStarted:  nowTime,
					maintenanceFinished: inSixtySecsTime,
				},
				{
					namespace:           "default",
					name:                "host0-k8s-pod--2-endpoint--id--2",
					ip:                  "10.0.0.2",
					maintenanceStarted:  nowTime,
					maintenanceFinished: inSixtySecsTime,
				},
			})

			podStatusCallback.clearState()
			err = manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			podStatusCallback.checkState([]statusCallbackEntry{})
		})

		It("should correctly calculate maintenance window for multiple active and terminating egw pods", func() {
			// egw 10.0.1.1 is terminating
			manager.OnUpdate(&proto.IPSetDeltaUpdate{
				Id:             "set1",
				AddedMembers:   []string{formatTerminatingEgressMemberStr("10.0.1.1", nowTime, inSixtySecsTime)},
				RemovedMembers: []string{formatActiveEgressMemberStr("10.0.1.1")},
			})
			err := manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			podStatusCallback.checkState([]statusCallbackEntry{
				{
					namespace:           "default",
					name:                "host0-k8s-pod--4-endpoint--id--4",
					ip:                  "10.0.1.1",
					maintenanceStarted:  nowTime,
					maintenanceFinished: inSixtySecsTime,
				},
				{
					namespace:           "default",
					name:                "host0-k8s-pod--5-endpoint--id--5",
					ip:                  "10.0.1.1",
					maintenanceStarted:  nowTime,
					maintenanceFinished: inSixtySecsTime,
				},
			})

			// egw 10.0.1.4 is created to replace egw 10.0.1.1
			manager.OnUpdate(&proto.IPSetDeltaUpdate{
				Id: "set1",
				AddedMembers: []string{
					formatActiveEgressMemberStr("10.0.1.4"),
				},
			})
			err = manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())

			// egw 10.0.1.2 is terminating
			manager.OnUpdate(&proto.IPSetDeltaUpdate{
				Id:             "set1",
				AddedMembers:   []string{formatTerminatingEgressMemberStr("10.0.1.2", thirtySecsAgo, inThirtySecsTime)},
				RemovedMembers: []string{formatActiveEgressMemberStr("10.0.1.2")},
			})
			err = manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			podStatusCallback.checkState([]statusCallbackEntry{
				{
					namespace:           "default",
					name:                "host0-k8s-pod--3-endpoint--id--3",
					ip:                  "10.0.1.2",
					maintenanceStarted:  thirtySecsAgo,
					maintenanceFinished: inThirtySecsTime,
				},
				{
					namespace:           "default",
					name:                "host0-k8s-pod--4-endpoint--id--4",
					ip:                  "10.0.1.1",
					maintenanceStarted:  nowTime,
					maintenanceFinished: inSixtySecsTime,
				},
				{
					namespace:           "default",
					name:                "host0-k8s-pod--5-endpoint--id--5",
					ip:                  "10.0.1.1",
					maintenanceStarted:  nowTime,
					maintenanceFinished: inSixtySecsTime,
				},
			})

			// egw 10.0.0.5 is created to replace egw 10.0.0.2
			manager.OnUpdate(&proto.IPSetDeltaUpdate{
				Id: "set0",
				AddedMembers: []string{
					formatActiveEgressMemberStr("10.0.0.5"),
				},
			})
			err = manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())

			// egw 10.0.0.1 has terminated
			manager.OnUpdate(&proto.IPSetDeltaUpdate{
				Id:           "set0",
				AddedMembers: []string{},
				RemovedMembers: []string{
					formatActiveEgressMemberStr("10.0.0.1"),
				},
			})
			err = manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())

			// egw 10.0.0.2 has terminated
			manager.OnUpdate(&proto.IPSetDeltaUpdate{
				Id:           "set0",
				AddedMembers: []string{},
				RemovedMembers: []string{
					formatActiveEgressMemberStr("10.0.0.2"),
				},
			})
			err = manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			podStatusCallback.checkState([]statusCallbackEntry{
				{
					namespace:           "default",
					name:                "host0-k8s-pod--3-endpoint--id--3",
					ip:                  "10.0.1.2",
					maintenanceStarted:  thirtySecsAgo,
					maintenanceFinished: inThirtySecsTime,
				},
				{
					namespace:           "default",
					name:                "host0-k8s-pod--4-endpoint--id--4",
					ip:                  "10.0.1.1",
					maintenanceStarted:  nowTime,
					maintenanceFinished: inSixtySecsTime,
				},
				{
					namespace:           "default",
					name:                "host0-k8s-pod--5-endpoint--id--5",
					ip:                  "10.0.1.1",
					maintenanceStarted:  nowTime,
					maintenanceFinished: inSixtySecsTime,
				},
			})
		})

		It("should be tolerant of missing timestamp", func() {
			manager.OnUpdate(&proto.IPSetDeltaUpdate{
				Id:             "set1",
				AddedMembers:   []string{formatTerminatingEgressMemberStr("10.0.3.0", nowTime, inSixtySecsTime), "10.0.3.1"},
				RemovedMembers: []string{"10.0.1.1"},
			})
			manager.OnUpdate(dummyWorkloadEndpointUpdate(2, "set1", []string{"10.0.242.0"}, 0))

			err := manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			expectRulesAndTable([]string{"10.0.242.0/32"}, 3, []string{"10.0.1.2", "10.0.1.3", "10.0.3.1"})
			mainTable.checkL2Routes("egress.calico", []routetable.L2Target{
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x01},
					GW:      ip.FromString("10.0.0.1"),
					IP:      ip.FromString("10.0.0.1"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x02},
					GW:      ip.FromString("10.0.0.2"),
					IP:      ip.FromString("10.0.0.2"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x03},
					GW:      ip.FromString("10.0.0.3"),
					IP:      ip.FromString("10.0.0.3"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x01, 0x02},
					GW:      ip.FromString("10.0.1.2"),
					IP:      ip.FromString("10.0.1.2"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x01, 0x03},
					GW:      ip.FromString("10.0.1.3"),
					IP:      ip.FromString("10.0.1.3"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x03, 0x00},
					GW:      ip.FromString("10.0.3.0"),
					IP:      ip.FromString("10.0.3.0"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x03, 0x01},
					GW:      ip.FromString("10.0.3.1"),
					IP:      ip.FromString("10.0.3.1"),
				},
			})
		})
	})

	Describe("with a single ipset and endpoint updates", func() {
		var zeroTime, nowTime, inSixtySecsTime time.Time

		var ips0 []string

		BeforeEach(func() {
			zeroTime = time.Time{}
			nowTime = time.Now()
			inSixtySecsTime = nowTime.Add(time.Second * 60)

			ips0 = []string{
				formatActiveEgressMemberStr("10.0.0.1"),
				formatActiveEgressMemberStr("10.0.0.2"),
				formatActiveEgressMemberStr("10.0.0.3"),
			}

			manager.OnUpdate(&proto.IPSetUpdate{
				Id:      "set0",
				Members: ips0,
				Type:    proto.IPSetUpdate_EGRESS_IP,
			})

			expectIPSetMembers("set0", []gateway{
				{
					cidr:                "10.0.0.1",
					maintenanceStarted:  zeroTime,
					maintenanceFinished: zeroTime,
				},
				{
					cidr:                "10.0.0.2",
					maintenanceStarted:  zeroTime,
					maintenanceFinished: zeroTime,
				},
				{
					cidr:                "10.0.0.3",
					maintenanceStarted:  zeroTime,
					maintenanceFinished: zeroTime,
				},
			})

			err := manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())

			// routeRules should be created.
			Expect(manager.routeRules).NotTo(BeNil())
			rr = rrFactory.Rules()

			mainTable.checkRoutes(routetable.InterfaceNone, nil)
			mainTable.checkRoutes("egress.calico", nil)
			mainTable.checkL2Routes("egress.calico", []routetable.L2Target{
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x01},
					GW:      ip.FromString("10.0.0.1"),
					IP:      ip.FromString("10.0.0.1"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x02},
					GW:      ip.FromString("10.0.0.2"),
					IP:      ip.FromString("10.0.0.2"),
				},
				{
					VTEPMAC: []byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x03},
					GW:      ip.FromString("10.0.0.3"),
					IP:      ip.FromString("10.0.0.3"),
				},
			})
		})

		It("should allocate a new rule and table with three hops when maxNextHops is zero", func() {
			manager.OnUpdate(dummyWorkloadEndpointUpdate(0, "set0", []string{"10.0.240.0/32"}, 0))

			err := manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			expectRulesAndTable([]string{"10.0.240.0/32"}, 1, []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"})
		})

		It("should allocate a new rule and table with one hop when maxNextHops is one", func() {
			manager.OnUpdate(dummyWorkloadEndpointUpdate(0, "set0", []string{"10.0.240.0/32"}, 1))

			err := manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			expectRulesAndTable([]string{"10.0.240.0/32"}, 1, []string{"10.0.0.1"})
		})

		It("should allocate a new rule and table with two hops when maxNextHops is two", func() {
			manager.OnUpdate(dummyWorkloadEndpointUpdate(0, "set0", []string{"10.0.240.0/32"}, 2))

			err := manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			expectRulesAndTable([]string{"10.0.240.0/32"}, 1, []string{"10.0.0.1", "10.0.0.3"})
		})

		It("should allocate a new rule and table with three hops when maxNextHops is three", func() {
			manager.OnUpdate(dummyWorkloadEndpointUpdate(0, "set0", []string{"10.0.240.0/32"}, 3))

			err := manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			expectRulesAndTable([]string{"10.0.240.0/32"}, 1, []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"})
		})

		It("should allocate rules and tables with an even distribution of one hop starting at random index when maxNextHops is one", func() {
			manager.OnUpdate(dummyWorkloadEndpointUpdate(0, "set0", []string{"10.0.240.0/32"}, 1))
			manager.OnUpdate(dummyWorkloadEndpointUpdate(1, "set0", []string{"10.0.240.1/32"}, 1))
			manager.OnUpdate(dummyWorkloadEndpointUpdate(2, "set0", []string{"10.0.240.2/32"}, 1))
			manager.OnUpdate(dummyWorkloadEndpointUpdate(3, "set0", []string{"10.0.240.3/32"}, 1))
			manager.OnUpdate(dummyWorkloadEndpointUpdate(4, "set0", []string{"10.0.240.4/32"}, 1))
			manager.OnUpdate(dummyWorkloadEndpointUpdate(5, "set0", []string{"10.0.240.5/32"}, 1))

			err := manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			expectRulesAndTable([]string{"10.0.240.0/32"}, 1, []string{"10.0.0.1"})
			expectRulesAndTable([]string{"10.0.240.1/32"}, 2, []string{"10.0.0.2"})
			expectRulesAndTable([]string{"10.0.240.2/32"}, 3, []string{"10.0.0.3"})
			expectRulesAndTable([]string{"10.0.240.3/32"}, 4, []string{"10.0.0.1"})
			expectRulesAndTable([]string{"10.0.240.4/32"}, 5, []string{"10.0.0.3"})
			expectRulesAndTable([]string{"10.0.240.5/32"}, 6, []string{"10.0.0.2"})
		})

		It("should allocate rules and tables with an even distribution of two hops starting at random index when maxNextHops is two", func() {
			manager.OnUpdate(dummyWorkloadEndpointUpdate(0, "set0", []string{"10.0.240.0/32"}, 2))
			manager.OnUpdate(dummyWorkloadEndpointUpdate(1, "set0", []string{"10.0.240.1/32"}, 2))
			manager.OnUpdate(dummyWorkloadEndpointUpdate(2, "set0", []string{"10.0.240.2/32"}, 2))
			manager.OnUpdate(dummyWorkloadEndpointUpdate(3, "set0", []string{"10.0.240.3/32"}, 2))
			manager.OnUpdate(dummyWorkloadEndpointUpdate(4, "set0", []string{"10.0.240.4/32"}, 2))
			manager.OnUpdate(dummyWorkloadEndpointUpdate(5, "set0", []string{"10.0.240.5/32"}, 2))

			err := manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			expectRulesAndTable([]string{"10.0.240.0/32"}, 1, []string{"10.0.0.1", "10.0.0.3"})
			expectRulesAndTable([]string{"10.0.240.1/32"}, 2, []string{"10.0.0.1", "10.0.0.2"})
			expectRulesAndTable([]string{"10.0.240.2/32"}, 3, []string{"10.0.0.2", "10.0.0.3"})
			expectRulesAndTable([]string{"10.0.240.3/32"}, 4, []string{"10.0.0.1", "10.0.0.3"})
			expectRulesAndTable([]string{"10.0.240.4/32"}, 5, []string{"10.0.0.2", "10.0.0.3"})
			expectRulesAndTable([]string{"10.0.240.5/32"}, 6, []string{"10.0.0.1", "10.0.0.2"})
		})

		It("should allocate per-deployment route tables excluding any terminating gateway hops", func() {
			ips0 = []string{
				formatActiveEgressMemberStr("10.0.0.1"),
				formatTerminatingEgressMemberStr("10.0.0.2", nowTime, inSixtySecsTime),
				formatActiveEgressMemberStr("10.0.0.3"),
			}

			manager.OnUpdate(&proto.IPSetUpdate{
				Id:      "set0",
				Members: ips0,
				Type:    proto.IPSetUpdate_EGRESS_IP,
			})

			manager.OnUpdate(dummyWorkloadEndpointUpdate(0, "set0", []string{"10.0.240.0/32"}, 1))
			manager.OnUpdate(dummyWorkloadEndpointUpdate(1, "set0", []string{"10.0.240.1/32"}, 1))
			manager.OnUpdate(dummyWorkloadEndpointUpdate(2, "set0", []string{"10.0.240.2/32"}, 1))
			manager.OnUpdate(dummyWorkloadEndpointUpdate(3, "set0", []string{"10.0.240.3/32"}, 1))
			manager.OnUpdate(dummyWorkloadEndpointUpdate(4, "set0", []string{"10.0.240.4/32"}, 1))
			manager.OnUpdate(dummyWorkloadEndpointUpdate(5, "set0", []string{"10.0.240.5/32"}, 1))

			err := manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			expectRulesAndTable([]string{"10.0.240.0/32"}, 1, []string{"10.0.0.1"})
			expectRulesAndTable([]string{"10.0.240.1/32"}, 2, []string{"10.0.0.3"})
			expectRulesAndTable([]string{"10.0.240.2/32"}, 3, []string{"10.0.0.1"})
			expectRulesAndTable([]string{"10.0.240.3/32"}, 4, []string{"10.0.0.3"})
			expectRulesAndTable([]string{"10.0.240.4/32"}, 5, []string{"10.0.0.1"})
			expectRulesAndTable([]string{"10.0.240.5/32"}, 6, []string{"10.0.0.3"})
		})
	})
})

func multiPath(ips []string, vxlanDeviceIdx int) []routetable.NextHop {
	var multipath []routetable.NextHop
	for _, e := range ips {
		multipath = append(multipath, routetable.NextHop{
			Gw:        ip.FromString(e),
			LinkIndex: vxlanDeviceIdx,
		})
	}
	return multipath
}

func dummyWorkloadEndpointID(podNum int) proto.WorkloadEndpointID {
	return proto.WorkloadEndpointID{
		OrchestratorId: "k8s",
		WorkloadId:     fmt.Sprintf("default/pod-%d", podNum),
		EndpointId:     fmt.Sprintf("endpoint-id-%d", podNum),
	}
}

func dummyWorkloadEndpointUpdate(podNum int, ipSetId string, cidrs []string, nextHops int) *proto.WorkloadEndpointUpdate {
	return &proto.WorkloadEndpointUpdate{
		Id: &proto.WorkloadEndpointID{
			OrchestratorId: "k8s",
			WorkloadId:     fmt.Sprintf("default/pod-%d", podNum),
			EndpointId:     fmt.Sprintf("endpoint-id-%d", podNum),
		},
		Endpoint: &proto.WorkloadEndpoint{
			State:             "active",
			Mac:               "01:02:03:04:05:06",
			Name:              fmt.Sprintf("cali12345-%d", podNum),
			ProfileIds:        []string{},
			Tiers:             []*proto.TierInfo{},
			Ipv4Nets:          cidrs,
			Ipv6Nets:          []string{"2001:db8:2::2/128"},
			EgressIpSetId:     ipSetId,
			EgressMaxNextHops: int32(nextHops),
		},
	}
}

type mockRouteRules struct {
	matchForUpdate routerule.RulesMatchFunc
	matchForRemove routerule.RulesMatchFunc
	activeRules    set.Set
}

func (r *mockRouteRules) GetAllActiveRules() []*routerule.Rule {
	var active []*routerule.Rule
	r.activeRules.Iter(func(item interface{}) error {
		p := item.(*routerule.Rule)
		active = append(active, p)
		return nil
	})

	return active
}

func (r *mockRouteRules) InitFromKernel() {
}

func (r *mockRouteRules) getActiveRule(rule *routerule.Rule, f routerule.RulesMatchFunc) *routerule.Rule {
	var active *routerule.Rule
	r.activeRules.Iter(func(item interface{}) error {
		p := item.(*routerule.Rule)
		if f(p, rule) {
			active = p
			return set.StopIteration
		}
		return nil
	})

	return active
}

func (r *mockRouteRules) SetRule(rule *routerule.Rule) {
	if r.getActiveRule(rule, r.matchForUpdate) == nil {
		rule.LogCxt().Debug("adding rule")
		r.activeRules.Add(rule)
	}
}

func (r *mockRouteRules) RemoveRule(rule *routerule.Rule) {
	if p := r.getActiveRule(rule, r.matchForRemove); p != nil {
		rule.LogCxt().Debug("removing rule")
		r.activeRules.Discard(p)
	}
}

func (r *mockRouteRules) QueueResync() {}
func (r *mockRouteRules) Apply() error {
	return nil
}

func (r *mockRouteRules) hasRule(priority int, src string, mark int, table int) bool {
	result := false
	r.activeRules.Iter(func(item interface{}) error {
		rule := item.(*routerule.Rule)
		nlRule := rule.NetLinkRule()
		rule.LogCxt().Debug("checking rule")
		if nlRule.Priority == priority &&
			nlRule.Family == unix.AF_INET &&
			nlRule.Src.String() == src &&
			nlRule.Mark == mark &&
			nlRule.Table == table &&
			nlRule.Invert == false {
			result = true
		}
		return nil
	})
	return result
}

type mockRouteTableFactory struct {
	count  int
	tables map[int]*mockRouteTable
}

func (f *mockRouteTableFactory) NewRouteTable(interfacePrefixes []string,
	ipVersion uint8,
	tableIndex int,
	vxlan bool,
	netlinkTimeout time.Duration,
	deviceRouteSourceAddress net.IP,
	deviceRouteProtocol int,
	removeExternalRoutes bool,
	opRecorder logutils.OpRecorder) routeTable {

	table := &mockRouteTable{
		index:           tableIndex,
		currentRoutes:   map[string][]routetable.Target{},
		currentL2Routes: map[string][]routetable.L2Target{},
	}
	f.tables[tableIndex] = table
	f.count += 1

	return table
}

func (f *mockRouteTableFactory) Table(i int) *mockRouteTable {
	Expect(f.tables[i]).NotTo(BeNil())
	return f.tables[i]
}

type mockRouteRulesFactory struct {
	routeRules *mockRouteRules
}

func (f *mockRouteRulesFactory) NewRouteRules(
	ipVersion int,
	tableIndexSet set.Set,
	updateFunc, removeFunc routerule.RulesMatchFunc,
	netlinkTimeout time.Duration,
	opRecorder logutils.OpRecorder,
) routeRules {
	rr := &mockRouteRules{
		matchForUpdate: routerule.RulesMatchSrcFWMarkTable,
		matchForRemove: routerule.RulesMatchSrcFWMark,
		activeRules:    set.New(),
	}
	f.routeRules = rr
	return rr
}

func (f *mockRouteRulesFactory) Rules() *mockRouteRules {
	return f.routeRules
}

func formatActiveEgressMemberStr(cidr string) string {
	return formatTerminatingEgressMemberStr(cidr, time.Time{}, time.Time{})
}

func formatTerminatingEgressMemberStr(cidr string, start, finish time.Time) string {
	startBytes, err := start.MarshalText()
	Expect(err).NotTo(HaveOccurred())
	finishBytes, err := finish.MarshalText()
	Expect(err).NotTo(HaveOccurred())
	return fmt.Sprintf("%s,%s,%s", cidr, string(startBytes), string(finishBytes))
}

func ipSetMemberEquals(expected gateway) types.GomegaMatcher {
	return &ipSetMemberMatcher{expected: expected}
}

type ipSetMemberMatcher struct {
	expected gateway
}

func (m *ipSetMemberMatcher) Match(actual interface{}) (bool, error) {
	member, ok := actual.(gateway)
	if !ok {
		return false, fmt.Errorf("ipSetMemberMatcher must be passed an gateway. Got\n%s", format.Object(actual, 1))
	}
	// Need to compare time.Time using Equal(), since having a nil loc and a UTC loc are equivalent.
	match := m.expected.cidr == member.cidr &&
		m.expected.maintenanceStarted.Equal(member.maintenanceStarted) &&
		m.expected.maintenanceFinished.Equal(member.maintenanceFinished)
	return match, nil

}

func (m *ipSetMemberMatcher) FailureMessage(actual interface{}) string {
	return fmt.Sprintf("Expected %v to match gateway: %v", actual.(gateway), m.expected)
}

func (m *ipSetMemberMatcher) NegatedFailureMessage(actual interface{}) string {
	return fmt.Sprintf("Expected %v to not match gateway: %v", actual.(gateway), m.expected)
}

type statusCallbackEntry struct {
	namespace           string
	name                string
	ip                  string
	maintenanceStarted  time.Time
	maintenanceFinished time.Time
}

type mockEgressPodStatusCallback struct {
	state []statusCallbackEntry
	Fail  bool
}

var (
	statusCallbackFail = errors.New("mock egress pod status callback failure")
)

func (t *mockEgressPodStatusCallback) statusCallback(namespace, name, ip string, maintenanceStarted, maintenanceFinished time.Time) error {
	log.WithFields(log.Fields{
		"namespace":           namespace,
		"name":                name,
		"ip":                  ip,
		"maintenanceStarted":  maintenanceStarted,
		"maintenanceFinished": maintenanceFinished,
	}).Info("mockEgressPodStatusCallback")
	if t.Fail {
		return statusCallbackFail
	}
	t.state = append(t.state, statusCallbackEntry{
		namespace:           namespace,
		name:                name,
		ip:                  ip,
		maintenanceStarted:  maintenanceStarted,
		maintenanceFinished: maintenanceFinished,
	})
	return nil
}

func (t *mockEgressPodStatusCallback) checkState(expected []statusCallbackEntry) {
	var matchers []types.GomegaMatcher
	for _, e := range expected {
		matchers = append(matchers, statusCallbackEntryEquals(e))
	}
	Expect(t.state).To(ConsistOf(matchers))
}

func (t *mockEgressPodStatusCallback) clearState() {
	t.state = nil
}

func statusCallbackEntryEquals(expected statusCallbackEntry) types.GomegaMatcher {
	return &statusCallbackEntryMatcher{expected: expected}
}

type statusCallbackEntryMatcher struct {
	expected statusCallbackEntry
}

func (m *statusCallbackEntryMatcher) Match(actual interface{}) (bool, error) {
	e, ok := actual.(statusCallbackEntry)
	if !ok {
		return false, fmt.Errorf("statusCallbackEntryMatcher must be passed a statusCallbackEntry. Got\n%s", format.Object(actual, 1))
	}
	// Need to compare time.Time using Equal(), since having a nil loc and a UTC loc are equivalent.
	match := m.expected.namespace == e.namespace &&
		m.expected.name == e.name &&
		m.expected.ip == e.ip &&
		m.expected.maintenanceStarted.Equal(e.maintenanceStarted) &&
		m.expected.maintenanceFinished.Equal(e.maintenanceFinished)
	return match, nil

}

func (m *statusCallbackEntryMatcher) FailureMessage(actual interface{}) string {
	return fmt.Sprintf("Expected %v to match statusCallbackEntry: %v", actual.(statusCallbackEntry), m.expected)
}

func (m *statusCallbackEntryMatcher) NegatedFailureMessage(actual interface{}) string {
	return fmt.Sprintf("Expected %v to not match statusCallbackEntry: %v", actual.(statusCallbackEntry), m.expected)
}

func expectUsageMapsEqual(actual, expected map[int][]string) {
	for k, v := range expected {
		Expect(actual).To(HaveKeyWithValue(k, v))
	}
}
