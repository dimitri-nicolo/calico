// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package intdataplane

import (
	"net"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"

	"github.com/projectcalico/felix/testutils"

	"github.com/projectcalico/felix/aws"
	"github.com/projectcalico/felix/ifacemonitor"
	"github.com/projectcalico/felix/ip"
	"github.com/projectcalico/felix/logutils"
	"github.com/projectcalico/felix/proto"
	"github.com/projectcalico/felix/routerule"
	"github.com/projectcalico/felix/routetable"
	"github.com/projectcalico/libcalico-go/lib/set"
)

var (
	awsRTIndexes          = []int{10, 11, 12, 13, 250, 251}
	awsTestNetlinkTimeout = 23 * time.Second
)
var _ = Describe("awsIPManager tests", func() {
	const (
		nodeName = "test-node"
	)
	var (
		m               *awsIPManager
		fakes           *awsIPMgrFakes
		primaryLink     *fakeLink
		primaryMACStr   = "12:34:56:78:90:12"
		primaryMAC      net.HardwareAddr
		secondaryMACStr = "12:34:56:78:90:22"
		secondaryMAC    net.HardwareAddr

		egressGWIP   = "100.64.2.5"
		egressGWCIDR = "100.64.2.5/32"
	)

	BeforeEach(func() {
		fakes = newAWSMgrFakes()
		primaryLink = newFakeLink()
		var err error
		primaryMAC, err = net.ParseMAC(primaryMACStr)
		Expect(err).NotTo(HaveOccurred())
		secondaryMAC, err = net.ParseMAC(secondaryMACStr)
		Expect(err).NotTo(HaveOccurred())
		primaryLink.attrs.HardwareAddr = primaryMAC
		primaryLink.attrs.MTU = 9001
		fakes.Links = []netlink.Link{
			primaryLink,
		}
		opRecorder := logutils.NewSummarizer("aws-ip-mgr-test")

		m = NewAWSIPManager(
			awsRTIndexes,
			Config{
				AWSSecondaryIPRoutingRulePriority: 105,
				NetlinkTimeout:                    awsTestNetlinkTimeout,
			},
			opRecorder,
			fakes,
			OptNetlinkOverride(fakes),
			OptRouteTableOverride(fakes.NewRouteTable),
			OptRouteRulesOverride(fakes.NewRouteRules),
		)
	})

	It("should send empty snapshot if there are no datastore resources", func() {
		Expect(m.CompleteDeferredWork()).NotTo(HaveOccurred())

		Expect(fakes.DatastoreState).To(Equal(&aws.DatastoreState{
			LocalAWSRoutesByDst:       map[ip.CIDR]*proto.RouteUpdate{},
			LocalRouteDestsBySubnetID: map[string]set.Set{},
			PoolIDsBySubnetID:         map[string]set.Set{},
		}))
	})

	It("should not fail on an interface update before an AWS update", func() {
		m.OnUpdate(&ifaceUpdate{
			Name:  "eth1",
			Index: 123,
			State: ifacemonitor.StateDown,
		})

		Expect(m.CompleteDeferredWork()).NotTo(HaveOccurred())
	})

	It("should do nothing if dataplane is empty and no AWS interfaces are needed", func() {
		Expect(m.CompleteDeferredWork()).NotTo(HaveOccurred())

		Expect(fakes.DatastoreState).To(Equal(&aws.DatastoreState{
			LocalAWSRoutesByDst:       map[ip.CIDR]*proto.RouteUpdate{},
			LocalRouteDestsBySubnetID: map[string]set.Set{},
			PoolIDsBySubnetID:         map[string]set.Set{},
		}))

		m.OnSecondaryIfaceStateUpdate(&aws.IfaceState{})

		Expect(m.CompleteDeferredWork()).NotTo(HaveOccurred())
	})

	Context("with pools, a local AWS pod and a non-AWS pod", func() {
		var workloadRoute *proto.RouteUpdate

		BeforeEach(func() {
			// Send in non-AWS pools and an AWS pool for hosts and a pool for workloads.
			m.OnUpdate(&proto.IPAMPoolUpdate{
				Id: "non-aws-pool-masq",
				Pool: &proto.IPAMPool{
					Cidr:       "192.168.0.0/16",
					Masquerade: true,
				},
			})
			m.OnUpdate(&proto.IPAMPoolUpdate{
				Id: "non-aws-pool-non-masq",
				Pool: &proto.IPAMPool{
					Cidr: "192.168.0.0/16",
				},
			})
			m.OnUpdate(&proto.RouteUpdate{
				Type:        proto.RouteType_CIDR_INFO,
				IpPoolType:  proto.IPPoolType_IPIP,
				Dst:         "192.168.0.0/16",
				NatOutgoing: true,
			})
			m.OnUpdate(&proto.IPAMPoolUpdate{
				Id: "hosts-pool",
				Pool: &proto.IPAMPool{
					Cidr:        "100.64.1.0/24",
					Masquerade:  false,
					AwsSubnetId: "subnet-123456789012345657",
				},
			})
			m.OnUpdate(&proto.RouteUpdate{
				Type:          proto.RouteType_CIDR_INFO,
				IpPoolType:    proto.IPPoolType_NO_ENCAP,
				Dst:           "100.64.1.0/24",
				NatOutgoing:   false,
				LocalWorkload: false,
				AwsSubnetId:   "subnet-123456789012345657",
			})
			m.OnUpdate(&proto.IPAMPoolUpdate{
				Id: "workloads-pool",
				Pool: &proto.IPAMPool{
					Cidr:        "100.64.2.0/24",
					Masquerade:  false,
					AwsSubnetId: "subnet-123456789012345657",
				},
			})
			m.OnUpdate(&proto.RouteUpdate{
				Type:          proto.RouteType_CIDR_INFO,
				IpPoolType:    proto.IPPoolType_VXLAN,
				Dst:           "100.64.2.0/24",
				NatOutgoing:   false,
				LocalWorkload: false,
				AwsSubnetId:   "subnet-123456789012345657",
			})

			// Local AWS workload.
			workloadRoute = &proto.RouteUpdate{
				Type:          proto.RouteType_LOCAL_WORKLOAD,
				IpPoolType:    proto.IPPoolType_VXLAN,
				Dst:           egressGWCIDR,
				NatOutgoing:   false,
				LocalWorkload: true, // This means "really a workload, not just a local IPAM block"
				AwsSubnetId:   "subnet-123456789012345657",
			}
			m.OnUpdate(workloadRoute)

			// Local non-AWS workload (should be ignored).
			nonAWSWorkloadRoute := &proto.RouteUpdate{
				Type:          proto.RouteType_LOCAL_WORKLOAD,
				IpPoolType:    proto.IPPoolType_VXLAN,
				Dst:           "192.168.1.2/32",
				NatOutgoing:   true,
				LocalWorkload: true, // This means "really a workload, not just a local IPAM block"
			}
			m.OnUpdate(nonAWSWorkloadRoute)

			Expect(m.CompleteDeferredWork()).NotTo(HaveOccurred())
		})

		It("should send the right snapshot", func() {
			// Should send the expected snapshot, ignoring non-AWS pools and workloads.
			workloadCIDR := ip.MustParseCIDROrIP(workloadRoute.Dst)
			Expect(fakes.DatastoreState).To(Equal(&aws.DatastoreState{
				LocalAWSRoutesByDst: map[ip.CIDR]*proto.RouteUpdate{
					workloadCIDR: workloadRoute,
				},
				LocalRouteDestsBySubnetID: map[string]set.Set{
					"subnet-123456789012345657": set.From(workloadCIDR),
				},
				PoolIDsBySubnetID: map[string]set.Set{
					"subnet-123456789012345657": set.From("hosts-pool", "workloads-pool"),
				},
			}))
		})

		It("should handle a change of subnet", func() {
			m.OnUpdate(&proto.IPAMPoolUpdate{
				Id: "workloads-pool",
				Pool: &proto.IPAMPool{
					Cidr:        "100.64.2.0/24",
					Masquerade:  false,
					AwsSubnetId: "subnet-000002",
				},
			})
			m.OnUpdate(&proto.RouteUpdate{
				Type:          proto.RouteType_CIDR_INFO,
				IpPoolType:    proto.IPPoolType_VXLAN,
				Dst:           "100.64.2.0/24",
				NatOutgoing:   false,
				LocalWorkload: false,
				AwsSubnetId:   "subnet-000002",
			})
			workloadRoute = &proto.RouteUpdate{
				Type:          proto.RouteType_LOCAL_WORKLOAD,
				IpPoolType:    proto.IPPoolType_VXLAN,
				Dst:           egressGWCIDR,
				NatOutgoing:   false,
				LocalWorkload: true, // This means "really a workload, not just a local IPAM block"
				AwsSubnetId:   "subnet-000002",
			}
			m.OnUpdate(workloadRoute)

			Expect(m.CompleteDeferredWork()).NotTo(HaveOccurred())

			// Should send the expected snapshot with updated subnets.
			workloadCIDR := ip.MustParseCIDROrIP(workloadRoute.Dst)
			Expect(fakes.DatastoreState).To(Equal(&aws.DatastoreState{
				LocalAWSRoutesByDst: map[ip.CIDR]*proto.RouteUpdate{
					workloadCIDR: workloadRoute,
				},
				LocalRouteDestsBySubnetID: map[string]set.Set{
					"subnet-000002": set.From(workloadCIDR),
				},
				PoolIDsBySubnetID: map[string]set.Set{
					"subnet-123456789012345657": set.From("hosts-pool"),
					"subnet-000002":             set.From("workloads-pool"),
				},
			}))
		})

		It("should handle a pool deletion", func() {
			m.OnUpdate(&proto.IPAMPoolRemove{
				Id: "workloads-pool",
			})
			m.OnUpdate(&proto.RouteRemove{
				Dst: "100.64.2.0/24",
			})
			workloadRoute = &proto.RouteUpdate{
				Type:          proto.RouteType_LOCAL_WORKLOAD,
				IpPoolType:    proto.IPPoolType_VXLAN,
				Dst:           egressGWCIDR,
				NatOutgoing:   false,
				LocalWorkload: true, // This means "really a workload, not just a local IPAM block"
			}
			m.OnUpdate(workloadRoute)

			// Should send the expected snapshot
			Expect(m.CompleteDeferredWork()).NotTo(HaveOccurred())
			Expect(fakes.DatastoreState).To(Equal(&aws.DatastoreState{
				LocalAWSRoutesByDst:       map[ip.CIDR]*proto.RouteUpdate{},
				LocalRouteDestsBySubnetID: map[string]set.Set{},
				PoolIDsBySubnetID: map[string]set.Set{
					"subnet-123456789012345657": set.From("hosts-pool"),
				},
			}))

			// Delete the other pool, should clean up.
			m.OnUpdate(&proto.IPAMPoolRemove{
				Id: "hosts-pool",
			})

			// Should send the expected snapshot
			Expect(m.CompleteDeferredWork()).NotTo(HaveOccurred())
			Expect(fakes.DatastoreState).To(Equal(&aws.DatastoreState{
				LocalAWSRoutesByDst:       map[ip.CIDR]*proto.RouteUpdate{},
				LocalRouteDestsBySubnetID: map[string]set.Set{},
				PoolIDsBySubnetID:         map[string]set.Set{},
			}))
		})

		Context("after responding with expected AWS state", func() {
			var secondaryLink *fakeLink
			BeforeEach(func() {
				// Pretend the background thread attached a new ENI.
				m.OnSecondaryIfaceStateUpdate(&aws.IfaceState{
					PrimaryNICMAC: primaryMACStr,
					SecondaryNICsByMAC: map[string]aws.Iface{
						secondaryMACStr: {
							ID:              "eni-0001",
							MAC:             secondaryMAC,
							PrimaryIPv4Addr: ip.FromString("100.64.0.5"),
							SecondaryIPv4Addrs: []ip.Addr{
								ip.FromString(egressGWIP),
							},
						},
					},
					SubnetCIDR:  ip.MustParseCIDROrIP("100.64.0.0/16"),
					GatewayAddr: ip.FromString("100.64.0.1"),
				})
				secondaryLink = newFakeLink()
			})

			expectSecondaryLinkConfigured := func() {
				// Only the primary IP gets added.
				secondaryIfacePriIP, err := netlink.ParseAddr("100.64.0.5/16")
				secondaryIfacePriIP.Scope = int(netlink.SCOPE_LINK)
				Expect(err).NotTo(HaveOccurred())
				Expect(secondaryLink.addrs).To(ConsistOf(*secondaryIfacePriIP))
				Expect(secondaryLink.attrs.OperState).To(Equal(netlink.LinkOperState(netlink.OperUp)))

				Expect(fakes.RouteTables).To(HaveLen(1))
				var rtID int
				for _, rt := range fakes.RouteTables {
					gwAddrAsCIDR := ip.MustParseCIDROrIP("100.64.0.1/32")
					Expect(rt.Routes["eth1"]).To(ConsistOf(
						routetable.Target{
							Type: routetable.TargetTypeGlobalUnicast,
							CIDR: gwAddrAsCIDR,
						},
						routetable.Target{
							Type: routetable.TargetTypeGlobalUnicast,
							CIDR: ip.MustParseCIDROrIP("0.0.0.0/0"),
							GW:   gwAddrAsCIDR.Addr(),
						},
					), "Expected gateway and default routes.")
					Expect(rt.Routes[routetable.InterfaceNone]).To(ConsistOf(
						routetable.Target{
							Type: routetable.TargetTypeThrow,
							CIDR: ip.MustParseCIDROrIP("192.168.0.0/16"),
						},
					), "Expected a 'throw' route for the non-AWS IP pool.")
					rtID = rt.Index()
				}

				// Rule for each egress gateway workload.
				Expect(fakes.Rules.Rules).To(ConsistOf(routerule.
					NewRule(4, 105).
					MatchSrcAddress(ip.MustParseCIDROrIP(egressGWIP).ToIPNet()).
					GoToTable(rtID)))
			}

			It("with the interface present, it should network the interface", func() {
				secondaryLink.attrs = netlink.LinkAttrs{
					Name:         "eth1",
					HardwareAddr: secondaryMAC,
				}
				fakes.Links = append(fakes.Links, secondaryLink)

				// CompleteDeferredWork should configure the interface.
				Expect(m.CompleteDeferredWork()).NotTo(HaveOccurred())

				// IP should be added.
				expectSecondaryLinkConfigured()
			})

			errEntry := func(name string) table.TableEntry {
				return table.Entry(name, name)
			}
			table.DescribeTable("with queued error",
				func(name string) {
					fakes.Errors.QueueError(name)

					secondaryLink.attrs = netlink.LinkAttrs{
						Name:         "eth1",
						HardwareAddr: secondaryMAC,
					}

					// Add a bonus address so that AddrDel will be called.
					extraNLAddr, err := netlink.ParseAddr("1.2.3.4/32")
					Expect(err).NotTo(HaveOccurred())
					secondaryLink.addrs = append(secondaryLink.addrs, *extraNLAddr)

					fakes.Links = append(fakes.Links, secondaryLink)

					// CompleteDeferredWork should fail once, then succeed.
					Expect(m.CompleteDeferredWork()).To(HaveOccurred())
					Expect(m.CompleteDeferredWork()).NotTo(HaveOccurred())

					// IP should be added.
					expectSecondaryLinkConfigured()

					fakes.Errors.ExpectAllErrorsConsumed()
				},
				errEntry("LinkList"),
				errEntry("LinkSetMTU"),
				errEntry("LinkSetUp"),
				errEntry("AddrList"),
				errEntry("AddrAdd"),
				errEntry("AddrDel"),
				errEntry("ParseAddr"),
			)

			It("with the interface missing, it should handle the interface showing up.", func() {
				// Should do nothing to start with.
				Expect(m.CompleteDeferredWork()).NotTo(HaveOccurred())

				// Interface shows up.
				secondaryLink.attrs = netlink.LinkAttrs{
					Name:         "eth1",
					HardwareAddr: secondaryMAC,
				}
				fakes.Links = append(fakes.Links, secondaryLink)

				// CompleteDeferredWork should not do anything yet.
				Expect(m.CompleteDeferredWork()).NotTo(HaveOccurred())
				Expect(secondaryLink.addrs).To(BeEmpty())

				// But sending in an interface update should trigger it.
				m.OnUpdate(&ifaceUpdate{
					Name:  "eth1",
					Index: 123,
					State: ifacemonitor.StateDown,
				})

				// CompleteDeferredWork should then configure the interface.
				Expect(m.CompleteDeferredWork()).NotTo(HaveOccurred())

				// IP should be added.
				expectSecondaryLinkConfigured()
			})

			It("should handle an interface flap.", func() {
				// Should do nothing to start with.
				Expect(m.CompleteDeferredWork()).NotTo(HaveOccurred())

				// Interface shows up.
				secondaryLink.attrs = netlink.LinkAttrs{
					Name:         "eth1",
					HardwareAddr: secondaryMAC,
				}
				fakes.Links = append(fakes.Links, secondaryLink)
				m.OnUpdate(&ifaceUpdate{
					Name:  "eth1",
					Index: 123,
					State: ifacemonitor.StateDown,
				})

				// CompleteDeferredWork should then configure the interface.
				Expect(m.CompleteDeferredWork()).NotTo(HaveOccurred())
				expectSecondaryLinkConfigured()

				// Interface re-added with new index
				secondaryLink.attrs = netlink.LinkAttrs{
					Name:         "eth1",
					HardwareAddr: secondaryMAC,
				}
				secondaryLink.addrs = nil
				m.OnUpdate(&ifaceUpdate{
					Name:  "eth1",
					Index: 124,
					State: ifacemonitor.StateDown,
				})

				// CompleteDeferredWork should then configure the interface.
				Expect(m.CompleteDeferredWork()).NotTo(HaveOccurred())
				expectSecondaryLinkConfigured()

				// Resulting signal of the interface going up shouldn't cause a resync.
				m.OnUpdate(&ifaceUpdate{
					Name:  "eth1",
					Index: 124,
					State: ifacemonitor.StateUp,
				})
				Expect(m.dataplaneResyncNeeded).To(BeFalse())

				// Resulting signal of the interface going up shouldn't cause a resync but if it goes down
				// then we do care.
				m.OnUpdate(&ifaceUpdate{
					Name:  "eth1",
					Index: 124,
					State: ifacemonitor.StateDown,
				})
				Expect(m.dataplaneResyncNeeded).To(BeTrue())
			})

			It("should handle an interface IP added.", func() {
				// Interface shows up.
				secondaryLink.attrs = netlink.LinkAttrs{
					Name:         "eth1",
					HardwareAddr: secondaryMAC,
				}
				fakes.Links = append(fakes.Links, secondaryLink)
				m.OnUpdate(&ifaceUpdate{
					Name:  "eth1",
					Index: 123,
					State: ifacemonitor.StateDown,
				})

				// CompleteDeferredWork should then configure the interface.
				Expect(m.CompleteDeferredWork()).NotTo(HaveOccurred())
				expectSecondaryLinkConfigured()

				// New IP added to interface and signalled.
				extraNLAddr, err := netlink.ParseAddr("1.2.3.4/32")
				Expect(err).NotTo(HaveOccurred())
				secondaryLink.addrs = append(secondaryLink.addrs, *extraNLAddr)
				m.OnUpdate(&ifaceAddrsUpdate{
					Name: "eth1",
					Addrs: set.From(
						"daed:beef::",
						"1.2.3.4",
						"100.64.0.5",
					),
				})

				// CompleteDeferredWork should clean up the incorrect IP.
				Expect(m.CompleteDeferredWork()).NotTo(HaveOccurred())
				expectSecondaryLinkConfigured()
			})

			It("should handle an interface IP removed.", func() {
				// Interface shows up.
				secondaryLink.attrs = netlink.LinkAttrs{
					Name:         "eth1",
					HardwareAddr: secondaryMAC,
				}
				fakes.Links = append(fakes.Links, secondaryLink)
				m.OnUpdate(&ifaceUpdate{
					Name:  "eth1",
					Index: 123,
					State: ifacemonitor.StateDown,
				})

				// CompleteDeferredWork should then configure the interface.
				Expect(m.CompleteDeferredWork()).NotTo(HaveOccurred())
				expectSecondaryLinkConfigured()

				// IP deleted.
				secondaryLink.addrs = nil
				m.OnUpdate(&ifaceAddrsUpdate{
					Name: "eth1",
					Addrs: set.From(
						"daed:beef::", // IPv6 ignored.
					),
				})

				// CompleteDeferredWork should add the correct IP.
				Expect(m.CompleteDeferredWork()).NotTo(HaveOccurred())
				expectSecondaryLinkConfigured()

				// Finally signal the correct state.
				m.OnUpdate(&ifaceAddrsUpdate{
					Name: "eth1",
					Addrs: set.From(
						"daed:beef::",
						"100.64.0.5",
					),
				})

				// Should spot it's correct and not schedule an update.
				Expect(m.dataplaneResyncNeeded).To(BeFalse())

				Expect(m.CompleteDeferredWork()).NotTo(HaveOccurred())
				expectSecondaryLinkConfigured()
			})

			It("should handle an interface delete/re-add.", func() {
				// Should do nothing to start with.
				Expect(m.CompleteDeferredWork()).NotTo(HaveOccurred())

				// Interface shows up.
				secondaryLink.attrs = netlink.LinkAttrs{
					Name:         "eth1",
					HardwareAddr: secondaryMAC,
				}
				fakes.Links = append(fakes.Links, secondaryLink)
				m.OnUpdate(&ifaceUpdate{
					Name:  "eth1",
					Index: 123,
					State: ifacemonitor.StateDown,
				})

				// CompleteDeferredWork should then configure the interface.
				Expect(m.CompleteDeferredWork()).NotTo(HaveOccurred())
				expectSecondaryLinkConfigured()

				// Interface re-added with new index
				secondaryLink.attrs = netlink.LinkAttrs{
					Name:         "eth1",
					HardwareAddr: secondaryMAC,
				}
				secondaryLink.addrs = nil

				// Delete.
				m.OnUpdate(&ifaceUpdate{
					Name:  "eth1",
					Index: 123,
					State: ifacemonitor.StateUnknown,
				})
				Expect(m.CompleteDeferredWork()).NotTo(HaveOccurred())

				// Re-add
				m.OnUpdate(&ifaceUpdate{
					Name:  "eth1",
					Index: 124,
					State: ifacemonitor.StateDown,
				})

				// CompleteDeferredWork should then configure the interface.
				Expect(m.CompleteDeferredWork()).NotTo(HaveOccurred())
				expectSecondaryLinkConfigured()
			})

			It("should provide the route tables.", func() {
				secondaryLink.attrs = netlink.LinkAttrs{
					Name:         "eth1",
					HardwareAddr: secondaryMAC,
				}
				fakes.Links = append(fakes.Links, secondaryLink)
				Expect(m.CompleteDeferredWork()).NotTo(HaveOccurred())

				Expect(m.GetRouteTableSyncers()).To(HaveLen(1))
				Expect(m.GetRouteRules()).To(ConsistOf(fakes.Rules))
			})

			It("should handle AWS interface going away.", func() {
				secondaryLink.attrs = netlink.LinkAttrs{
					Name:         "eth1",
					HardwareAddr: secondaryMAC,
				}
				fakes.Links = append(fakes.Links, secondaryLink)
				Expect(m.CompleteDeferredWork()).NotTo(HaveOccurred())

				m.OnSecondaryIfaceStateUpdate(&aws.IfaceState{
					PrimaryNICMAC:      primaryMACStr,
					SecondaryNICsByMAC: map[string]aws.Iface{},
					SubnetCIDR:         ip.MustParseCIDROrIP("100.64.0.0/16"),
					GatewayAddr:        ip.FromString("100.64.0.1"),
				})

				Expect(m.CompleteDeferredWork()).NotTo(HaveOccurred())

				// Routing table should be flushed.
				Expect(fakes.RouteTables).To(HaveLen(1))
				for _, rt := range fakes.RouteTables {
					Expect(rt.Routes["eth1"]).To(BeEmpty())
					Expect(rt.Routes[routetable.InterfaceNone]).To(BeEmpty())
				}
				Expect(fakes.Rules.Rules).To(BeEmpty())
			})

			It("should remove an extra IP on the ENI", func() {
				secondaryLink.attrs = netlink.LinkAttrs{
					Name:         "eth1",
					HardwareAddr: secondaryMAC,
				}
				bogusAddr, err := netlink.ParseAddr("1.2.3.4/24")
				secondaryLink.addrs = append(secondaryLink.addrs, *bogusAddr)
				Expect(err).NotTo(HaveOccurred())
				fakes.Links = append(fakes.Links, secondaryLink)

				Expect(m.CompleteDeferredWork()).NotTo(HaveOccurred())
				expectSecondaryLinkConfigured()
			})
		})
	})
})

// TODO Test multiple routing tables
// TODO Test churn reuses table IDs

func newAWSMgrFakes() *awsIPMgrFakes {
	errorProd := testutils.NewErrorProducer()
	return &awsIPMgrFakes{
		RouteTables: map[int]*fakeRouteTable{},
		Errors:      errorProd,
	}
}

type awsIPMgrFakes struct {
	DatastoreState *aws.DatastoreState

	Links []netlink.Link

	RouteTables map[int]*fakeRouteTable
	Rules       *fakeRouteRules
	Errors      testutils.ErrorProducer
}

func (f *awsIPMgrFakes) ParseAddr(s string) (*netlink.Addr, error) {
	if err := f.Errors.NextErrorByCaller(); err != nil {
		return nil, err
	}
	return netlink.ParseAddr(s)
}

func (f *awsIPMgrFakes) OnDatastoreUpdate(ds aws.DatastoreState) {
	f.DatastoreState = &ds
}

func (f *awsIPMgrFakes) LinkSetMTU(iface netlink.Link, mtu int) error {
	if err := f.Errors.NextErrorByCaller(); err != nil {
		return err
	}
	iface.(*fakeLink).attrs.MTU = mtu
	return nil
}

func (f *awsIPMgrFakes) LinkSetUp(iface netlink.Link) error {
	if err := f.Errors.NextErrorByCaller(); err != nil {
		return err
	}
	iface.(*fakeLink).attrs.OperState = netlink.OperUp
	return nil
}

func (f *awsIPMgrFakes) AddrList(iface netlink.Link, family int) ([]netlink.Addr, error) {
	if err := f.Errors.NextErrorByCaller(); err != nil {
		return nil, err
	}
	Expect(family).To(Equal(netlink.FAMILY_V4))
	return iface.(*fakeLink).addrs, nil
}

func (f *awsIPMgrFakes) AddrDel(iface netlink.Link, addr *netlink.Addr) error {
	if err := f.Errors.NextErrorByCaller(); err != nil {
		return err
	}
	link := iface.(*fakeLink)
	newAddrs := link.addrs[:0]
	found := false
	for _, a := range link.addrs {
		if a.Equal(*addr) {
			found = true
			continue
		}
		newAddrs = append(newAddrs, a)
	}
	Expect(found).To(BeTrue(), "Asked to delete non-existent IP")
	link.addrs = newAddrs
	return nil
}

func (f *awsIPMgrFakes) AddrAdd(iface netlink.Link, addr *netlink.Addr) error {
	if err := f.Errors.NextErrorByCaller(); err != nil {
		return err
	}
	link := iface.(*fakeLink)
	link.addrs = append(link.addrs, *addr)
	return nil
}

func (f *awsIPMgrFakes) LinkList() ([]netlink.Link, error) {
	if err := f.Errors.NextErrorByCaller(); err != nil {
		return nil, err
	}
	return f.Links, nil
}

func (f *awsIPMgrFakes) NewRouteTable(
	regexes []string,
	version uint8,
	vxlan bool,
	timeout time.Duration,
	deviceRouteSourceAddress net.IP,
	deviceRouteProtocol netlink.RouteProtocol,
	removeExternalRoutes bool,
	index int,
	reporter logutils.OpRecorder) routeTable {

	Expect(version).To(BeNumerically("==", 4))
	Expect(vxlan).To(BeFalse())
	Expect(deviceRouteSourceAddress).To(BeNil())
	Expect(removeExternalRoutes).To(BeTrue())
	Expect(reporter).ToNot(BeNil())

	rt := &fakeRouteTable{
		Regexes:  regexes,
		Timeout:  timeout,
		Protocol: deviceRouteProtocol,
		index:    index,
		Routes:   map[string][]routetable.Target{},
		Errors:   f.Errors,
	}
	f.RouteTables[index] = rt
	return rt
}

func (f *awsIPMgrFakes) NewRouteRules(
	ipVersion int,
	priority int,
	tableIndexSet set.Set,
	updateFunc routerule.RulesMatchFunc,
	removeFunc routerule.RulesMatchFunc,
	netlinkTimeout time.Duration,
	newNetlinkHandle func() (routerule.HandleIface, error),
	opRecorder logutils.OpRecorder) (routeRules, error) {

	if err := f.Errors.NextErrorByCaller(); err != nil {
		return nil, err
	}

	Expect(ipVersion).To(Equal(4))
	Expect(priority).To(Equal(105))
	Expect(tableIndexSet).To(Equal(set.FromArray(awsRTIndexes)))
	Expect(updateFunc).ToNot(BeNil())
	Expect(removeFunc).ToNot(BeNil())
	Expect(opRecorder).ToNot(BeNil())
	Expect(netlinkTimeout).To(Equal(awsTestNetlinkTimeout))
	h, err := newNetlinkHandle()
	Expect(err).NotTo(HaveOccurred())
	Expect(h).ToNot(BeNil())
	Expect(f.Rules).To(BeNil())

	f.Rules = &fakeRouteRules{
		Errors: f.Errors,
	}

	return f.Rules, nil
}

func newFakeLink() *fakeLink {
	return &fakeLink{}
}

type fakeLink struct {
	attrs netlink.LinkAttrs
	addrs []netlink.Addr
}

func (f *fakeLink) Attrs() *netlink.LinkAttrs {
	return &f.attrs
}

func (f *fakeLink) Type() string {
	return "device"
}

type fakeRouteTable struct {
	Regexes  []string
	Timeout  time.Duration
	Protocol netlink.RouteProtocol
	index    int
	Routes   map[string][]routetable.Target
	Errors   testutils.ErrorProducer
}

func (f *fakeRouteTable) OnIfaceStateChanged(s string, state ifacemonitor.State) {
}

func (f *fakeRouteTable) QueueResync() {
}

func (f *fakeRouteTable) Apply() error {
	return nil
}

func (f *fakeRouteTable) Index() int {
	return f.index
}

func (f *fakeRouteTable) SetRoutes(ifaceName string, targets []routetable.Target) {
	if ifaceName != routetable.InterfaceNone {
		Expect(f.Regexes[0]).To(Or(Equal("^" + ifaceName + "$")))
	}
	f.Routes[ifaceName] = targets
}

func (f *fakeRouteTable) RouteRemove(_ string, _ ip.CIDR) {
	panic("implement me")
}

func (f *fakeRouteTable) SetL2Routes(_ string, _ []routetable.L2Target) {
	panic("implement me")
}

func (f *fakeRouteTable) QueueResyncIface(_ string) {
	panic("implement me")
}

type fakeRouteRules struct {
	Rules  []*routerule.Rule
	Errors testutils.ErrorProducer
}

func (f *fakeRouteRules) SetRule(rule *routerule.Rule) {
	for _, r := range f.Rules {
		// Can't easily inspect the contents of routerule.Rule but comparison is good enough for now.
		if reflect.DeepEqual(r, rule) {
			return
		}
	}
	f.Rules = append(f.Rules, rule)
}

func (f *fakeRouteRules) RemoveRule(rule *routerule.Rule) {
	newRules := f.Rules[:0]
	found := false
	for _, r := range f.Rules {
		if reflect.DeepEqual(r, rule) {
			found = true
			continue
		}
		newRules = append(newRules, r)
	}
	Expect(found).To(BeTrue(), "asked to delete non-existent rule")
	f.Rules = newRules
}

func (f *fakeRouteRules) QueueResync() {

}

func (f *fakeRouteRules) Apply() error {
	return nil
}
