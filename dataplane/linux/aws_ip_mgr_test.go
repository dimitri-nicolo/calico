// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package intdataplane

import (
	"net"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"

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
		fakes           *fakeAWSIPMgrFakes
		primaryLink     *fakeLink
		primaryMACStr   = "12:34:56:78:90:12"
		primaryMAC      net.HardwareAddr
		secondaryMACStr = "12:34:56:78:90:22"
		secondaryMAC    net.HardwareAddr
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
				AWSSecondaryIPRoutingRulePriority: 101,
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
		err := m.CompleteDeferredWork()
		Expect(err).NotTo(HaveOccurred())

		Expect(fakes.DatastoreState).To(Equal(&aws.DatastoreState{
			LocalAWSRoutesByDst:       map[ip.CIDR]*proto.RouteUpdate{},
			LocalRouteDestsBySubnetID: map[string]set.Set{},
			PoolIDsBySubnetID:         map[string]set.Set{},
		}))
	})

	It("should do nothing if dataplane is empty and no AWS interfaces are needed", func() {
		err := m.CompleteDeferredWork()
		Expect(err).NotTo(HaveOccurred())

		Expect(fakes.DatastoreState).To(Equal(&aws.DatastoreState{
			LocalAWSRoutesByDst:       map[ip.CIDR]*proto.RouteUpdate{},
			LocalRouteDestsBySubnetID: map[string]set.Set{},
			PoolIDsBySubnetID:         map[string]set.Set{},
		}))

		m.OnSecondaryIfaceStateUpdate(&aws.IfaceState{})

		err = m.CompleteDeferredWork()
		Expect(err).NotTo(HaveOccurred())
	})

	Context("with pools, a local AWS pod and a non-AWS pod", func() {
		var workloadRoute *proto.RouteUpdate

		BeforeEach(func() {
			// Send in a non-AWS pool and an AWS pool for hosts and a pool for workloads.
			m.OnUpdate(&proto.IPAMPoolUpdate{
				Id: "non-aws-pool",
				Pool: &proto.IPAMPool{
					Cidr:       "192.168.0.0/16",
					Masquerade: false,
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
				Dst:           "100.64.2.5/32",
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

			err := m.CompleteDeferredWork()
			Expect(err).NotTo(HaveOccurred())
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

		Context("after responding with expected AWS state", func() {
			BeforeEach(func() {
				// Pretend the background thread attached a new ENI.
				m.OnSecondaryIfaceStateUpdate(&aws.IfaceState{
					PrimaryNICMAC: primaryMACStr,
					SecondaryNICsByMAC: map[string]aws.Iface{
						secondaryMACStr: {
							ID:              "eni-0001",
							MAC:             secondaryMAC,
							PrimaryIPv4Addr: ip.FromString("100.64.0.1"),
							SecondaryIPv4Addrs: []ip.Addr{
								ip.FromString("100.64.2.5"),
							},
						},
					},
					SubnetCIDR:  ip.MustParseCIDROrIP("100.64.0.0/16"),
					GatewayAddr: ip.FromString("100.64.0.1"),
				})
			})

			It("with the interface present, it should network the interface", func() {
				secondaryLink := newFakeLink()
				secondaryLink.attrs = netlink.LinkAttrs{
					Name:         "eth1",
					HardwareAddr: secondaryMAC,
				}
				fakes.Links = append(fakes.Links, secondaryLink)

				// CompleteDeferredWork should configure the interface.
				err := m.CompleteDeferredWork()
				Expect(err).NotTo(HaveOccurred())

				// Only the primary IP gets added.
				secondaryIfacePriIP, err := netlink.ParseAddr("100.64.0.1/16")
				secondaryIfacePriIP.Scope = int(netlink.SCOPE_LINK)
				Expect(err).NotTo(HaveOccurred())
				Expect(secondaryLink.addrs).To(ConsistOf(*secondaryIfacePriIP))
			})

			It("with the interface missing, it should handle the interface showing up.", func() {
				// Should do nothing.
				err := m.CompleteDeferredWork()
				Expect(err).NotTo(HaveOccurred())

				// Interface shows up.
				secondaryLink := newFakeLink()
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

				// CompleteDeferredWork should configure the interface.
				err = m.CompleteDeferredWork()
				Expect(err).NotTo(HaveOccurred())

				// Only the primary IP gets added.
				secondaryIfacePriIP, err := netlink.ParseAddr("100.64.0.1/16")
				secondaryIfacePriIP.Scope = int(netlink.SCOPE_LINK)
				Expect(err).NotTo(HaveOccurred())
				Expect(secondaryLink.addrs).To(ConsistOf(*secondaryIfacePriIP))
			})
		})
	})
})

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

func newAWSMgrFakes() *fakeAWSIPMgrFakes {
	return &fakeAWSIPMgrFakes{
		RouteTables: map[int]*fakeRouteTable{},
	}
}

type fakeAWSIPMgrFakes struct {
	DatastoreState *aws.DatastoreState

	Links []netlink.Link

	RouteTables map[int]*fakeRouteTable
	Rules       *fakeRouteRules
}

func (f *fakeAWSIPMgrFakes) OnDatastoreUpdate(ds aws.DatastoreState) {
	f.DatastoreState = &ds
}

func (f *fakeAWSIPMgrFakes) LinkSetMTU(iface netlink.Link, mtu int) error {
	iface.(*fakeLink).attrs.MTU = mtu
	return nil
}

func (f *fakeAWSIPMgrFakes) LinkSetUp(iface netlink.Link) error {
	iface.(*fakeLink).attrs.OperState = netlink.OperUp
	return nil
}

func (f *fakeAWSIPMgrFakes) AddrList(iface netlink.Link, v int) ([]netlink.Addr, error) {
	return iface.(*fakeLink).addrs, nil
}

func (f *fakeAWSIPMgrFakes) AddrDel(iface netlink.Link, n *netlink.Addr) error {
	panic("implement me")
}

func (f *fakeAWSIPMgrFakes) AddrAdd(iface netlink.Link, addr *netlink.Addr) error {
	iface.(*fakeLink).addrs = append(iface.(*fakeLink).addrs, *addr)
	return nil
}

func (f *fakeAWSIPMgrFakes) LinkList() ([]netlink.Link, error) {
	return f.Links, nil
}

func (f *fakeAWSIPMgrFakes) NewRouteTable(
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
	}
	f.RouteTables[index] = rt
	return rt
}

func (f *fakeAWSIPMgrFakes) NewRouteRules(
	ipVersion int,
	priority int,
	tableIndexSet set.Set,
	updateFunc routerule.RulesMatchFunc,
	removeFunc routerule.RulesMatchFunc,
	netlinkTimeout time.Duration,
	newNetlinkHandle func() (routerule.HandleIface, error),
	opRecorder logutils.OpRecorder) (routeRules, error) {

	Expect(ipVersion).To(Equal(4))
	Expect(priority).To(Equal(101))
	Expect(tableIndexSet).To(Equal(set.FromArray(awsRTIndexes)))
	Expect(updateFunc).ToNot(BeNil())
	Expect(removeFunc).ToNot(BeNil())
	Expect(opRecorder).ToNot(BeNil())
	Expect(netlinkTimeout).To(Equal(awsTestNetlinkTimeout))
	h, err := newNetlinkHandle()
	Expect(err).NotTo(HaveOccurred())
	Expect(h).ToNot(BeNil())
	Expect(f.Rules).To(BeNil())

	f.Rules = &fakeRouteRules{}

	return f.Rules, nil
}

type fakeRouteTable struct {
	Regexes  []string
	Timeout  time.Duration
	Protocol netlink.RouteProtocol
	index    int
	Routes   map[string][]routetable.Target
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

func (f *fakeRouteTable) RouteRemove(ifaceName string, cidr ip.CIDR) {
	panic("implement me")
}

func (f *fakeRouteTable) SetL2Routes(ifaceName string, targets []routetable.L2Target) {
	panic("implement me")
}

func (f *fakeRouteTable) QueueResyncIface(ifaceName string) {
	panic("implement me")
}

type fakeRouteRules struct {
}

func (f *fakeRouteRules) SetRule(rule *routerule.Rule) {

}

func (f *fakeRouteRules) RemoveRule(rule *routerule.Rule) {

}

func (f *fakeRouteRules) QueueResync() {

}

func (f *fakeRouteRules) Apply() error {
	return nil
}
