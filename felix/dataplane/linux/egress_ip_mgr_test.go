// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.

package intdataplane

import (
	"errors"
	"fmt"
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

	BeforeEach(func() {
		rrFactory = &mockRouteRulesFactory{routeRules: nil}

		mainTable = &mockRouteTable{
			index:           0,
			currentRoutes:   map[string][]routetable.Target{},
			currentL2Routes: map[string][]routetable.L2Target{},
		}
		rtFactory = &mockRouteTableFactory{count: 0, tables: make(map[int]*mockRouteTable)}

		// Three free table to use.
		tableIndexSet := set.New()
		tableIndexStack := stack.New()
		for i := 3; i > 0; i-- {
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
		)

		err := manager.CompleteDeferredWork()
		Expect(err).ToNot(HaveOccurred())

		// No routerules should be created.
		Expect(manager.routerules).To(BeNil())

		manager.OnUpdate(&proto.HostMetadataUpdate{
			Hostname: "host0",
			Ipv4Addr: "172.0.0.2", // mockVXLANDataplane use interface address 172.0.0.2
		})
		Expect(manager.NodeIP).To(Equal(net.ParseIP("172.0.0.2")))
		err = manager.configureVXLANDevice(50)
		Expect(err).NotTo(HaveOccurred())
		Expect(manager.vxlanDeviceLinkIndex).To(Equal(6))
	})

	checkSetMember := func(id string, members []ipSetMember) {
		var matchers []types.GomegaMatcher
		for _, m := range members {
			matchers = append(matchers, ipSetMemberEquals(m))
		}
		Expect(manager.activeEgressIPSet[id]).To(ContainElements(matchers))
	}

	multiPath := func(ips []string) []routetable.NextHop {
		multipath := []routetable.NextHop{}

		for _, e := range ips {
			multipath = append(multipath, routetable.NextHop{
				Gw:        ip.FromString(e),
				LinkIndex: manager.vxlanDeviceLinkIndex,
			})
		}
		return multipath
	}

	Describe("with multiple ipsets and endpoints update", func() {
		var ips0, ips1 []string
		var zeroTime, nowTime, inThirtySecsTime, inSixtySecsTime, inNinetySecsTime time.Time
		BeforeEach(func() {
			zeroTime = time.Time{}
			nowTime = time.Now()
			inThirtySecsTime = nowTime.Add(time.Second * 30)
			inSixtySecsTime = nowTime.Add(time.Second * 60)
			inNinetySecsTime = nowTime.Add(time.Second * 90)

			ips0 = []string{formatActiveEgressMemberStr("10.0.0.1"), formatActiveEgressMemberStr("10.0.0.2")}
			ips1 = []string{formatActiveEgressMemberStr("10.0.1.1"), formatActiveEgressMemberStr("10.0.1.2")}

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

			checkSetMember("set0", []ipSetMember{
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
			})
			checkSetMember("set1", []ipSetMember{
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
			})
			Expect(manager.activeEgressIPSet["nonEgressSet"]).To(BeNil())

			err := manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())

			manager.OnUpdate(&proto.WorkloadEndpointUpdate{
				Id: &proto.WorkloadEndpointID{
					OrchestratorId: "k8s",
					WorkloadId:     "default/pod-0",
					EndpointId:     "endpoint-id-0",
				},
				Endpoint: &proto.WorkloadEndpoint{
					State:         "active",
					Mac:           "01:02:03:04:05:06",
					Name:          "cali12345-0",
					ProfileIds:    []string{},
					Tiers:         []*proto.TierInfo{},
					Ipv4Nets:      []string{"10.0.240.0/32"},
					Ipv6Nets:      []string{"2001:db8:2::2/128"},
					EgressIpSetId: "set0",
				},
			})

			manager.OnUpdate(&proto.WorkloadEndpointUpdate{
				Id: &proto.WorkloadEndpointID{
					OrchestratorId: "k8s",
					WorkloadId:     "default/pod-1",
					EndpointId:     "endpoint-id-1",
				},
				Endpoint: &proto.WorkloadEndpoint{
					State:         "active",
					Mac:           "01:02:03:04:05:06",
					Name:          "cali12345-1",
					ProfileIds:    []string{},
					Tiers:         []*proto.TierInfo{},
					Ipv4Nets:      []string{"10.0.241.0/32"},
					Ipv6Nets:      []string{"2001:db8:2::3/128"},
					EgressIpSetId: "set1",
				},
			})

			err = manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())

			// routerules should be created.
			Expect(manager.routerules).NotTo(BeNil())
			rr = rrFactory.Rules()

			Expect(rr.hasRule(100, "10.0.240.0/32", 0x200, 1)).To(BeTrue())
			rtFactory.Table(1).checkRoutes(routetable.InterfaceNone, []routetable.Target{{
				Type:      routetable.TargetTypeVXLAN,
				CIDR:      defaultCidr,
				MultiPath: multiPath([]string{"10.0.0.1", "10.0.0.2"}),
			}})
			rtFactory.Table(1).checkRoutes("egress.calico", nil)
			rtFactory.Table(1).checkL2Routes(routetable.InterfaceNone, nil)
			rtFactory.Table(1).checkL2Routes("egress.calico", nil)

			Expect(rr.hasRule(100, "10.0.241.0/32", 0x200, 2)).To(BeTrue())
			rtFactory.Table(2).checkRoutes(routetable.InterfaceNone, []routetable.Target{{
				Type:      routetable.TargetTypeVXLAN,
				CIDR:      defaultCidr,
				MultiPath: multiPath([]string{"10.0.1.1", "10.0.1.2"}),
			}})
			rtFactory.Table(2).checkL2Routes(routetable.InterfaceNone, nil)
			rtFactory.Table(2).checkL2Routes("egress.calico", nil)

			mainTable.checkRoutes(routetable.InterfaceNone, nil)
			mainTable.checkRoutes("egress.calico", nil)
			mainTable.checkL2Routes("egress.calico", []routetable.L2Target{
				{
					VTEPMAC: net.HardwareAddr([]byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x01}),
					GW:      ip.FromString("10.0.0.1"),
					IP:      ip.FromString("10.0.0.1"),
				},
				{
					VTEPMAC: net.HardwareAddr([]byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x02}),
					GW:      ip.FromString("10.0.0.2"),
					IP:      ip.FromString("10.0.0.2"),
				},
				{
					VTEPMAC: net.HardwareAddr([]byte{0xa2, 0x2a, 0x0a, 0x00, 0x01, 0x01}),
					GW:      ip.FromString("10.0.1.1"),
					IP:      ip.FromString("10.0.1.1"),
				},
				{
					VTEPMAC: net.HardwareAddr([]byte{0xa2, 0x2a, 0x0a, 0x00, 0x01, 0x02}),
					GW:      ip.FromString("10.0.1.2"),
					IP:      ip.FromString("10.0.1.2"),
				},
			})
		})

		It("should support delta update", func() {
			manager.OnUpdate(&proto.IPSetDeltaUpdate{
				Id:             "set1",
				AddedMembers:   []string{formatActiveEgressMemberStr("10.0.3.0"), formatActiveEgressMemberStr("10.0.3.1")},
				RemovedMembers: []string{formatActiveEgressMemberStr("10.0.1.1")},
			})

			err := manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			rtFactory.Table(2).checkRoutes(routetable.InterfaceNone, []routetable.Target{{
				Type:      routetable.TargetTypeVXLAN,
				CIDR:      defaultCidr,
				MultiPath: multiPath([]string{"10.0.1.2", "10.0.3.0", "10.0.3.1"}),
			}})
			mainTable.checkL2Routes("egress.calico", []routetable.L2Target{
				{
					VTEPMAC: net.HardwareAddr([]byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x01}),
					GW:      ip.FromString("10.0.0.1"),
					IP:      ip.FromString("10.0.0.1"),
				},
				{
					VTEPMAC: net.HardwareAddr([]byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x02}),
					GW:      ip.FromString("10.0.0.2"),
					IP:      ip.FromString("10.0.0.2"),
				},
				{
					VTEPMAC: net.HardwareAddr([]byte{0xa2, 0x2a, 0x0a, 0x00, 0x01, 0x02}),
					GW:      ip.FromString("10.0.1.2"),
					IP:      ip.FromString("10.0.1.2"),
				},
				{
					VTEPMAC: net.HardwareAddr([]byte{0xa2, 0x2a, 0x0a, 0x00, 0x03, 0x00}),
					GW:      ip.FromString("10.0.3.0"),
					IP:      ip.FromString("10.0.3.0"),
				},
				{
					VTEPMAC: net.HardwareAddr([]byte{0xa2, 0x2a, 0x0a, 0x00, 0x03, 0x01}),
					GW:      ip.FromString("10.0.3.1"),
					IP:      ip.FromString("10.0.3.1"),
				},
			})
		})

		It("should release table correctly", func() {
			manager.OnUpdate(&proto.IPSetRemove{
				Id: "set1",
			})

			err := manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())

			Expect(manager.tableIndexStack.Peek()).To(Equal(2))
			Expect(manager.tableIndexStack.Len()).To(Equal(2))
			rtFactory.Table(2).checkRoutes(routetable.InterfaceNone, nil)
			rtFactory.Table(2).checkRoutes("egress.calico", nil)
			mainTable.checkL2Routes("egress.calico", []routetable.L2Target{
				{
					VTEPMAC: net.HardwareAddr([]byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x01}),
					GW:      ip.FromString("10.0.0.1"),
					IP:      ip.FromString("10.0.0.1"),
				},
				{
					VTEPMAC: net.HardwareAddr([]byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x02}),
					GW:      ip.FromString("10.0.0.2"),
					IP:      ip.FromString("10.0.0.2"),
				},
			})

			// Send same ipset remove
			manager.OnUpdate(&proto.IPSetRemove{
				Id: "set1",
			})

			err = manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())

			Expect(manager.tableIndexStack.Peek()).To(Equal(2))
			Expect(manager.tableIndexStack.Len()).To(Equal(2))
			rtFactory.Table(2).checkRoutes("egress.calico", nil)
		})

		It("should panic if run out of table index", func() {
			manager.OnUpdate(&proto.IPSetUpdate{
				Id:      "set3",
				Members: ips1,
				Type:    proto.IPSetUpdate_EGRESS_IP,
			})

			err := manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			Expect(manager.tableIndexStack.Len()).To(Equal(0))

			manager.OnUpdate(&proto.IPSetUpdate{
				Id:      "set4",
				Members: ips1,
				Type:    proto.IPSetUpdate_EGRESS_IP,
			})

			Expect(func() {
				_ = manager.CompleteDeferredWork()
			}).To(Panic())
		})

		It("should use same table if endpoints has same egress ipset", func() {
			manager.OnUpdate(&proto.WorkloadEndpointUpdate{
				Id: &proto.WorkloadEndpointID{
					OrchestratorId: "k8s",
					WorkloadId:     "pod-2",
					EndpointId:     "endpoint-id-2",
				},
				Endpoint: &proto.WorkloadEndpoint{
					State:         "active",
					Mac:           "01:02:03:04:05:06",
					Name:          "cali12345-2",
					ProfileIds:    []string{},
					Tiers:         []*proto.TierInfo{},
					Ipv4Nets:      []string{"10.0.242.0/32"},
					Ipv6Nets:      []string{"2001:db8:2::4/128"},
					EgressIpSetId: "set0",
				},
			})
			err := manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())

			Expect(rr.hasRule(100, "10.0.242.0/32", 0x200, 1))
			rtFactory.Table(1).checkRoutes(routetable.InterfaceNone, []routetable.Target{{
				Type:      routetable.TargetTypeVXLAN,
				CIDR:      defaultCidr,
				MultiPath: multiPath([]string{"10.0.0.1", "10.0.0.2"}),
			}})
			mainTable.checkL2Routes("egress.calico", []routetable.L2Target{
				{
					VTEPMAC: net.HardwareAddr([]byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x01}),
					GW:      ip.FromString("10.0.0.1"),
					IP:      ip.FromString("10.0.0.1"),
				},
				{
					VTEPMAC: net.HardwareAddr([]byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x02}),
					GW:      ip.FromString("10.0.0.2"),
					IP:      ip.FromString("10.0.0.2"),
				},
				{
					VTEPMAC: net.HardwareAddr([]byte{0xa2, 0x2a, 0x0a, 0x00, 0x01, 0x01}),
					GW:      ip.FromString("10.0.1.1"),
					IP:      ip.FromString("10.0.1.1"),
				},
				{
					VTEPMAC: net.HardwareAddr([]byte{0xa2, 0x2a, 0x0a, 0x00, 0x01, 0x02}),
					GW:      ip.FromString("10.0.1.2"),
					IP:      ip.FromString("10.0.1.2"),
				},
			})
		})

		It("should set unreachable route if egress ipset has all members removed", func() {
			manager.OnUpdate(&proto.IPSetDeltaUpdate{
				Id:             "set1",
				AddedMembers:   []string{},
				RemovedMembers: []string{formatActiveEgressMemberStr("10.0.1.1"), formatActiveEgressMemberStr("10.0.1.2")},
			})

			err := manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			rtFactory.Table(2).checkRoutes(routetable.InterfaceNone, []routetable.Target{{
				Type: routetable.TargetTypeUnreachable,
				CIDR: defaultCidr,
			}})
			mainTable.checkL2Routes("egress.calico", []routetable.L2Target{
				{
					VTEPMAC: net.HardwareAddr([]byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x01}),
					GW:      ip.FromString("10.0.0.1"),
					IP:      ip.FromString("10.0.0.1"),
				},
				{
					VTEPMAC: net.HardwareAddr([]byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x02}),
					GW:      ip.FromString("10.0.0.2"),
					IP:      ip.FromString("10.0.0.2"),
				},
			})
		})

		It("should remove routes for old workload", func() {
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

			Expect(rr.hasRule(100, "10.0.240.0/32", 0x200, 1)).To(BeFalse())
		})

		It("should set correct route for workload if egress ipset changed", func() {
			// pod-0 use table 1 at start.
			Expect(rr.hasRule(100, "10.0.240.0/32", 0x200, 1)).To(BeTrue())
			Expect(rr.hasRule(100, "10.0.240.0/32", 0x200, 2)).To(BeFalse())

			// Update pod-0 to use ipset set1.
			manager.OnUpdate(&proto.WorkloadEndpointUpdate{
				Id: &proto.WorkloadEndpointID{
					OrchestratorId: "k8s",
					WorkloadId:     "default/pod-0",
					EndpointId:     "endpoint-id-0",
				},
				Endpoint: &proto.WorkloadEndpoint{
					State:         "active",
					Mac:           "01:02:03:04:05:06",
					Name:          "cali12345-0",
					ProfileIds:    []string{},
					Tiers:         []*proto.TierInfo{},
					Ipv4Nets:      []string{"10.0.240.0/32"},
					Ipv6Nets:      []string{"2001:db8:2::2/128"},
					EgressIpSetId: "set1",
				},
			})
			err := manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())

			// pod-0 use table 2 as the result.
			Expect(rr.hasRule(100, "10.0.240.0/32", 0x200, 1)).To(BeFalse())
			Expect(rr.hasRule(100, "10.0.240.0/32", 0x200, 2)).To(BeTrue())
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

			// endpoint0 stay in pendingWlEpUpdates
			for i := 0; i < 3; i++ {
				err := manager.CompleteDeferredWork()
				Expect(err).ToNot(HaveOccurred())
				Expect(manager.pendingWlEpUpdates[id0]).To(Equal(endpoint0))
			}

			manager.OnUpdate(&proto.IPSetUpdate{
				Id:      "setx",
				Members: []string{formatActiveEgressMemberStr("10.0.10.1"), formatActiveEgressMemberStr("10.0.10.2")},
				Type:    proto.IPSetUpdate_EGRESS_IP,
			})
			err := manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())

			// pod-0 use table 3 as the result.
			Expect(rr.hasRule(100, "10.0.240.0/32", 0x200, 1)).To(BeFalse())
			Expect(rr.hasRule(100, "10.0.240.0/32", 0x200, 3)).To(BeTrue())
			rtFactory.Table(3).checkRoutes(routetable.InterfaceNone, []routetable.Target{{
				Type:      routetable.TargetTypeVXLAN,
				CIDR:      defaultCidr,
				MultiPath: multiPath([]string{"10.0.10.1", "10.0.10.2"}),
			}})
			mainTable.checkL2Routes("egress.calico", []routetable.L2Target{
				{
					VTEPMAC: net.HardwareAddr([]byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x01}),
					GW:      ip.FromString("10.0.0.1"),
					IP:      ip.FromString("10.0.0.1"),
				},
				{
					VTEPMAC: net.HardwareAddr([]byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x02}),
					GW:      ip.FromString("10.0.0.2"),
					IP:      ip.FromString("10.0.0.2"),
				},
				{
					VTEPMAC: net.HardwareAddr([]byte{0xa2, 0x2a, 0x0a, 0x00, 0x01, 0x01}),
					GW:      ip.FromString("10.0.1.1"),
					IP:      ip.FromString("10.0.1.1"),
				},
				{
					VTEPMAC: net.HardwareAddr([]byte{0xa2, 0x2a, 0x0a, 0x00, 0x01, 0x02}),
					GW:      ip.FromString("10.0.1.2"),
					IP:      ip.FromString("10.0.1.2"),
				},
				{
					VTEPMAC: net.HardwareAddr([]byte{0xa2, 0x2a, 0x0a, 0x00, 0x0a, 0x01}),
					GW:      ip.FromString("10.0.10.1"),
					IP:      ip.FromString("10.0.10.1"),
				},
				{
					VTEPMAC: net.HardwareAddr([]byte{0xa2, 0x2a, 0x0a, 0x00, 0x0a, 0x02}),
					GW:      ip.FromString("10.0.10.2"),
					IP:      ip.FromString("10.0.10.2"),
				},
			})
		})

		It("should support terminating egw pod", func() {
			manager.OnUpdate(&proto.IPSetDeltaUpdate{
				Id:             "set0",
				AddedMembers:   []string{formatTerminatingEgressMemberStr("10.0.0.1", nowTime, inSixtySecsTime)},
				RemovedMembers: []string{formatActiveEgressMemberStr("10.0.0.1")},
			})

			err := manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			rtFactory.Table(1).checkRoutes(routetable.InterfaceNone, []routetable.Target{{
				Type:      routetable.TargetTypeVXLAN,
				CIDR:      defaultCidr,
				MultiPath: multiPath([]string{"10.0.0.1", "10.0.0.2"}),
			}})
			mainTable.checkL2Routes("egress.calico", []routetable.L2Target{
				{
					VTEPMAC: net.HardwareAddr([]byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x01}),
					GW:      ip.FromString("10.0.0.1"),
					IP:      ip.FromString("10.0.0.1"),
				},
				{
					VTEPMAC: net.HardwareAddr([]byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x02}),
					GW:      ip.FromString("10.0.0.2"),
					IP:      ip.FromString("10.0.0.2"),
				},
				{
					VTEPMAC: net.HardwareAddr([]byte{0xa2, 0x2a, 0x0a, 0x00, 0x01, 0x01}),
					GW:      ip.FromString("10.0.1.1"),
					IP:      ip.FromString("10.0.1.1"),
				},
				{
					VTEPMAC: net.HardwareAddr([]byte{0xa2, 0x2a, 0x0a, 0x00, 0x01, 0x02}),
					GW:      ip.FromString("10.0.1.2"),
					IP:      ip.FromString("10.0.1.2"),
				},
			})
			podStatusCallback.checkState([]statusCallbackEntry{
				{
					namespace:           "default",
					name:                "host0-k8s-pod--0-endpoint--id--0",
					ip:                  "10.0.0.1",
					maintenanceStarted:  nowTime,
					maintenanceFinished: inSixtySecsTime,
				},
			})
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
			})

			podStatusCallback.clearState()
			err = manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			podStatusCallback.checkState([]statusCallbackEntry{})
		})

		It("should correctly calculate maintenance window for multiple terminating egw pods", func() {
			manager.OnUpdate(&proto.IPSetDeltaUpdate{
				Id: "set0",
				AddedMembers: []string{
					formatTerminatingEgressMemberStr("10.0.0.1", nowTime, inSixtySecsTime),
					formatTerminatingEgressMemberStr("10.0.0.2", inThirtySecsTime, inNinetySecsTime),
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
					maintenanceStarted:  inThirtySecsTime,
					maintenanceFinished: inNinetySecsTime,
				},
			})

			podStatusCallback.clearState()
			err = manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			podStatusCallback.checkState([]statusCallbackEntry{})
		})

		It("should correctly calculate maintenance window for multiple active and terminating egw pods", func() {
			// egw 10.0.0.1 is terminating
			manager.OnUpdate(&proto.IPSetDeltaUpdate{
				Id: "set0",
				AddedMembers: []string{
					formatTerminatingEgressMemberStr("10.0.0.1", nowTime, inSixtySecsTime),
				},
				RemovedMembers: []string{
					formatActiveEgressMemberStr("10.0.0.1"),
				},
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
			})

			// egw 10.0.0.3 is created to replace egw 10.0.0.1
			manager.OnUpdate(&proto.IPSetDeltaUpdate{
				Id: "set0",
				AddedMembers: []string{
					formatActiveEgressMemberStr("10.0.0.3"),
				},
			})
			podStatusCallback.clearState()
			err = manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			podStatusCallback.checkState([]statusCallbackEntry{})

			// egw 10.0.0.2 is terminating
			manager.OnUpdate(&proto.IPSetDeltaUpdate{
				Id: "set0",
				AddedMembers: []string{
					formatTerminatingEgressMemberStr("10.0.0.2", inThirtySecsTime, inNinetySecsTime),
				},
				RemovedMembers: []string{
					formatActiveEgressMemberStr("10.0.0.2"),
				},
			})
			podStatusCallback.clearState()
			err = manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			podStatusCallback.checkState([]statusCallbackEntry{
				{
					namespace:           "default",
					name:                "host0-k8s-pod--0-endpoint--id--0",
					ip:                  "10.0.0.2",
					maintenanceStarted:  inThirtySecsTime,
					maintenanceFinished: inNinetySecsTime,
				},
			})

			// egw 10.0.0.4 is created to replace egw 10.0.0.2
			manager.OnUpdate(&proto.IPSetDeltaUpdate{
				Id: "set0",
				AddedMembers: []string{
					formatActiveEgressMemberStr("10.0.0.4"),
				},
			})
			podStatusCallback.clearState()
			err = manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			podStatusCallback.checkState([]statusCallbackEntry{})

			// egw 10.0.0.1 has terminated
			manager.OnUpdate(&proto.IPSetDeltaUpdate{
				Id:           "set0",
				AddedMembers: []string{},
				RemovedMembers: []string{
					formatActiveEgressMemberStr("10.0.0.1"),
				},
			})
			podStatusCallback.clearState()
			err = manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			podStatusCallback.checkState([]statusCallbackEntry{})

			// egw 10.0.0.2 has terminated
			manager.OnUpdate(&proto.IPSetDeltaUpdate{
				Id:           "set0",
				AddedMembers: []string{},
				RemovedMembers: []string{
					formatActiveEgressMemberStr("10.0.0.2"),
				},
			})
			podStatusCallback.clearState()
			err = manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			podStatusCallback.checkState([]statusCallbackEntry{})
		})

		It("should be tolerant of missing timestamp", func() {
			manager.OnUpdate(&proto.IPSetDeltaUpdate{
				Id:             "set1",
				AddedMembers:   []string{formatTerminatingEgressMemberStr("10.0.3.0", nowTime, inSixtySecsTime), "10.0.3.1"},
				RemovedMembers: []string{"10.0.1.1"},
			})

			err := manager.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
			rtFactory.Table(2).checkRoutes(routetable.InterfaceNone, []routetable.Target{{
				Type:      routetable.TargetTypeVXLAN,
				CIDR:      defaultCidr,
				MultiPath: multiPath([]string{"10.0.1.2", "10.0.3.0", "10.0.3.1"}),
			}})
			mainTable.checkL2Routes("egress.calico", []routetable.L2Target{
				{
					VTEPMAC: net.HardwareAddr([]byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x01}),
					GW:      ip.FromString("10.0.0.1"),
					IP:      ip.FromString("10.0.0.1"),
				},
				{
					VTEPMAC: net.HardwareAddr([]byte{0xa2, 0x2a, 0x0a, 0x00, 0x00, 0x02}),
					GW:      ip.FromString("10.0.0.2"),
					IP:      ip.FromString("10.0.0.2"),
				},
				{
					VTEPMAC: net.HardwareAddr([]byte{0xa2, 0x2a, 0x0a, 0x00, 0x01, 0x02}),
					GW:      ip.FromString("10.0.1.2"),
					IP:      ip.FromString("10.0.1.2"),
				},
				{
					VTEPMAC: net.HardwareAddr([]byte{0xa2, 0x2a, 0x0a, 0x00, 0x03, 0x00}),
					GW:      ip.FromString("10.0.3.0"),
					IP:      ip.FromString("10.0.3.0"),
				},
				{
					VTEPMAC: net.HardwareAddr([]byte{0xa2, 0x2a, 0x0a, 0x00, 0x03, 0x01}),
					GW:      ip.FromString("10.0.3.1"),
					IP:      ip.FromString("10.0.3.1"),
				},
			})
		})
	})
})

type mockRouteRules struct {
	matchForUpdate routerule.RulesMatchFunc
	matchForRemove routerule.RulesMatchFunc
	activeRules    set.Set
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

func ipSetMemberEquals(expected ipSetMember) types.GomegaMatcher {
	return &ipSetMemberMatcher{expected: expected}
}

type ipSetMemberMatcher struct {
	expected ipSetMember
}

func (m *ipSetMemberMatcher) Match(actual interface{}) (bool, error) {
	member, ok := actual.(ipSetMember)
	if !ok {
		return false, fmt.Errorf("ipSetMemberMatcher must be passed an ipSetMember. Got\n%s", format.Object(actual, 1))
	}
	// Need to compare time.Time using Equal(), since having a nil loc and a UTC loc are equivalent.
	match := m.expected.cidr == member.cidr &&
		m.expected.maintenanceStarted.Equal(member.maintenanceStarted) &&
		m.expected.maintenanceFinished.Equal(member.maintenanceFinished)
	return match, nil

}

func (m *ipSetMemberMatcher) FailureMessage(actual interface{}) string {
	return fmt.Sprintf("Expected %v to match ipSetMember: %v", actual.(ipSetMember), m.expected)
}

func (m *ipSetMemberMatcher) NegatedFailureMessage(actual interface{}) string {
	return fmt.Sprintf("Expected %v to not match ipSetMember: %v", actual.(ipSetMember), m.expected)
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
	Expect(t.state).To(ContainElements(matchers))
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
