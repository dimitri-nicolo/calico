// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package ipsec_test

import (
	"errors"
	"fmt"
	"net"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/projectcalico/felix/ip"
	. "github.com/projectcalico/felix/ipsec"
	"github.com/projectcalico/felix/testutils"
	cnet "github.com/projectcalico/libcalico-go/lib/net"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

var (
	zeroIP   = net.ParseIP("0.0.0.0").To4()
	zeroCIDR = cnet.MustParseCIDR("0.0.0.0/0").IPNet

	hostIP1        = net.ParseIP("10.0.1.1").To4()
	felixHostIP1   = ip.FromNetIP(hostIP1).(ip.V4Addr)
	felixHostCIDR1 = felixHostIP1.AsCIDR().(ip.V4CIDR)
	hostCIDR1      = felixHostCIDR1.ToIPNet()

	workloadIP1        = net.ParseIP("10.0.1.2").To4()
	felixWorkloadIP1   = ip.FromNetIP(workloadIP1).(ip.V4Addr)
	felixWorkloadCIDR1 = felixWorkloadIP1.AsCIDR().(ip.V4CIDR)
	workloadCIDR1      = felixWorkloadCIDR1.ToIPNet()

	hostIP2        = net.ParseIP("10.0.2.1").To4()
	felixHostIP2   = ip.FromNetIP(hostIP2).(ip.V4Addr)
	felixHostCIDR2 = felixHostIP2.AsCIDR().(ip.V4CIDR)
	hostCIDR2      = felixHostCIDR2.ToIPNet()

	// A Calico to-remote-host policy.
	// Calico version: selector and rule.
	caliSel1 = PolicySelector{
		TrafficDst: felixHostCIDR1,
		Dir:        netlink.XFRM_DIR_OUT,
		Mark:       0x10,
		MarkMask:   0xf0,
	}
	caliPol1 = PolicyRule{
		Action:    netlink.XFRM_POLICY_ALLOW,
		TunnelSrc: felixHostIP1,
		TunnelDst: felixHostIP2,
	}
	// Netlink version.
	caliXFRMPolicy1 = netlink.XfrmPolicy{
		Dst: &hostCIDR1,
		Dir: netlink.XFRM_DIR_OUT,
		Tmpls: []netlink.XfrmPolicyTmpl{
			{
				Src:   hostIP1,
				Dst:   hostIP2,
				Reqid: ReqID,
				Mode:  netlink.XFRM_MODE_TUNNEL,
				Proto: netlink.XFRM_PROTO_ESP,
			},
		},
		Mark: &netlink.XfrmMark{
			Value: 0x10,
			Mask:  0xf0,
		},
	}
	// Variant of the above
	caliPol1b = PolicyRule{
		Action: netlink.XFRM_POLICY_BLOCK,
	}
	// Netlink version.
	caliXFRMPolicy1b = netlink.XfrmPolicy{
		Dst:    &hostCIDR1,
		Dir:    netlink.XFRM_DIR_OUT,
		Action: netlink.XFRM_POLICY_BLOCK,
		Tmpls: []netlink.XfrmPolicyTmpl{
			{
				Src:   zeroIP,
				Dst:   zeroIP,
				Reqid: ReqID,
				Mode:  netlink.XFRM_MODE_TUNNEL,
				Proto: netlink.XFRM_PROTO_ESP,
			},
		},
		Mark: &netlink.XfrmMark{
			Value: 0x10,
			Mask:  0xf0,
		},
	}

	// A Calico from-remote-workload policy.
	// Calico version: selector and rule.
	caliSel2 = PolicySelector{
		TrafficSrc: felixWorkloadCIDR1,
		Dir:        netlink.XFRM_DIR_FWD,
	}
	caliPol2 = PolicyRule{
		TunnelSrc: felixHostIP1,
		TunnelDst: felixHostIP2,
	}
	// Netlink version.
	caliXFRMPolicy2 = netlink.XfrmPolicy{
		Src: &workloadCIDR1,
		Dir: netlink.XFRM_DIR_FWD,
		Tmpls: []netlink.XfrmPolicyTmpl{
			{
				Src:   hostIP1,
				Dst:   hostIP2,
				Reqid: ReqID,
				Mode:  netlink.XFRM_MODE_TUNNEL,
				Proto: netlink.XFRM_PROTO_ESP,
			},
		},
	}
	// Netlink version with a zero CIDR instead of a nil.
	caliXFRMPolicy2b = netlink.XfrmPolicy{
		Src: &workloadCIDR1,
		Dst: &zeroCIDR,
		Dir: netlink.XFRM_DIR_FWD,
		Tmpls: []netlink.XfrmPolicyTmpl{
			{
				Src:   hostIP1,
				Dst:   hostIP2,
				Reqid: ReqID,
				Mode:  netlink.XFRM_MODE_TUNNEL,
				Proto: netlink.XFRM_PROTO_ESP,
			},
		},
	}

	// Looks like one of ours but wrong req ID
	nonCaliPolicy1 = netlink.XfrmPolicy{
		Src: &hostCIDR1,
		Dst: &hostCIDR2,
		Dir: netlink.XFRM_DIR_OUT,
		Tmpls: []netlink.XfrmPolicyTmpl{
			{
				Reqid: 101,
			},
		},
	}

	// No templates.
	nonCaliPolicy2 = netlink.XfrmPolicy{
		Src: &hostCIDR1,
		Dst: &hostCIDR2,
		Dir: netlink.XFRM_DIR_FWD,
	}
)

var _ = Describe("IpsecPolicyTable", func() {
	var mockDataplane *mockIPSecDataplane
	var polTable *PolicyTable

	BeforeEach(func() {
		mockDataplane = newMockIPSecDataplane()
		polTable = NewPolicyTableWithShims(ReqID, mockDataplane.newNetlinkHandle, mockDataplane.sleep)
	})

	It("should be constructable", func() {
		_ = NewPolicyTable(50) // For coverage's sake.
	})

	Context("with empty dataplane, no pending updates", func() {
		It("should resync and make no updates", func() {
			polTable.Apply()
			Expect(mockDataplane.ActivePolicies).To(BeEmpty())
			Expect(mockDataplane.NumListCalls).To(Equal(1))
		})
		It("should only resync once", func() {
			polTable.Apply()
			polTable.Apply()
			Expect(mockDataplane.NumListCalls).To(Equal(1))
		})
		It("should apply an update at the right time", func() {
			polTable.SetRule(caliSel1, &caliPol1)
			Expect(mockDataplane.NumUpdateCalls).To(Equal(0))
			polTable.Apply()
			Expect(mockDataplane.NumUpdateCalls).To(Equal(1))
			Expect(mockDataplane.ActivePolicies).To(ConsistOf(caliXFRMPolicy1))
		})
		It("should apply a pair of updates at the right time", func() {
			polTable.SetRule(caliSel1, &caliPol1)
			polTable.SetRule(caliSel2, &caliPol2)
			Expect(mockDataplane.NumUpdateCalls).To(Equal(0))
			polTable.Apply()
			Expect(mockDataplane.NumUpdateCalls).To(Equal(2))
			Expect(mockDataplane.ActivePolicies).To(ConsistOf(caliXFRMPolicy1, caliXFRMPolicy2))
		})
		It("should only apply an update once", func() {
			polTable.SetRule(caliSel1, &caliPol1)
			polTable.Apply()
			polTable.SetRule(caliSel1, &caliPol1)
			polTable.Apply()
			Expect(mockDataplane.NumUpdateCalls).To(Equal(1))
			Expect(mockDataplane.ActivePolicies).To(ConsistOf(caliXFRMPolicy1))
		})
		It("should squash a delete before apply", func() {
			polTable.SetRule(caliSel1, &caliPol1)
			polTable.DeleteRule(caliSel1)
			polTable.Apply()
			Expect(mockDataplane.NumUpdateCalls).To(Equal(0))
			Expect(mockDataplane.NumDeleteCalls).To(Equal(0))
			Expect(mockDataplane.ActivePolicies).To(BeEmpty())
		})
		It("should delete a policy at right time", func() {
			polTable.SetRule(caliSel1, &caliPol1)
			polTable.Apply()
			polTable.DeleteRule(caliSel1)
			Expect(mockDataplane.NumDeleteCalls).To(Equal(0))
			polTable.Apply()
			Expect(mockDataplane.NumUpdateCalls).To(Equal(1))
			Expect(mockDataplane.NumDeleteCalls).To(Equal(1))
			Expect(mockDataplane.ActivePolicies).To(BeEmpty())
		})
		It("should delete only once", func() {
			polTable.SetRule(caliSel1, &caliPol1)
			polTable.Apply()
			polTable.DeleteRule(caliSel1)
			polTable.Apply()
			polTable.Apply()
			Expect(mockDataplane.NumDeleteCalls).To(Equal(1))
			Expect(mockDataplane.ActivePolicies).To(BeEmpty())
		})
		AfterEach(func() {
			// None of these tests should trigger backoff.
			Expect(mockDataplane.cumulativeSleep).To(BeZero())
		})
	})

	for _, failureType := range []string{"newNetlinkHandle", "XfrmPolicyList", "XfrmPolicyUpdate", "XfrmPolicyDel"} {
		failureType := failureType // Create a fresh copy for each loop.
		Describe("with some "+failureType+" errors queued up", func() {
			BeforeEach(func() {
				mockDataplane.Errors.QueueError(failureType)
				mockDataplane.Errors.QueueError(failureType)
			})

			It("should succeed in applying an update and deletion (retrying if needed)", func() {
				polTable.SetRule(caliSel1, &caliPol1)
				polTable.Apply()
				Expect(mockDataplane.ActivePolicies).To(ConsistOf(caliXFRMPolicy1))
				polTable.DeleteRule(caliSel1)
				polTable.Apply()
				Expect(mockDataplane.ActivePolicies).To(BeEmpty())
				// At least one of the operations should have backed off.
				Expect(mockDataplane.cumulativeSleep).To(BeNumerically(">=", time.Millisecond))
			})
		})
		Describe("with persistent "+failureType+" errors queued up", func() {
			BeforeEach(func() {
				// Make sure there's a rule to delete.
				polTable.SetRule(caliSel1, &caliPol1)
				polTable.Apply()

				for i := 0; i < 20; i++ {
					mockDataplane.Errors.QueueError(failureType)
				}
				if failureType == "newNetlinkHandle" {
					// queue up a single failure of an update to trigger a netlink reconnect
					mockDataplane.Errors.QueueError("XfrmPolicyUpdate")
				}
				if failureType == "XfrmPolicyList" {
					// Make sure the apply() does a list operation.
					polTable.QueueResync()
				}
			})

			It("should give up", func() {
				polTable.DeleteRule(caliSel1)
				polTable.SetRule(caliSel2, &caliPol2)
				Expect(polTable.Apply).To(Panic())
				// At least one of the operations should have backed off.
				Expect(mockDataplane.cumulativeSleep).To(BeNumerically(">=", time.Millisecond))
			})
		})
	}

	Context("with a calico policy using a zero-CIDR in the dataplane", func() {
		// Check that a zero CIDR and a nil CIDR are treated equally by the resync.

		BeforeEach(func() {
			mockDataplane.addPolicy(&caliXFRMPolicy2b)
		})

		It("with no pending update, it should remove it", func() {
			polTable.Apply()
			Expect(mockDataplane.ActivePolicies).To(BeEmpty())
		})

		It("with a pending update, it should squash the update", func() {
			polTable.SetRule(caliSel2, &caliPol2)
			polTable.Apply()
			Expect(mockDataplane.ActivePolicies).To(ConsistOf(caliXFRMPolicy2b))
			Expect(mockDataplane.NumDeleteCalls).To(BeZero())
		})

		It("it should handle a delete", func() {
			polTable.SetRule(caliSel2, &caliPol2)
			polTable.Apply()
			polTable.DeleteRule(caliSel2)
			polTable.Apply()
			Expect(mockDataplane.ActivePolicies).To(BeEmpty())
		})
	})

	Context("with some non-calico policy and an unexpected calico policy in the dataplane", func() {
		BeforeEach(func() {
			mockDataplane.addPolicy(&nonCaliPolicy1)
			mockDataplane.addPolicy(&nonCaliPolicy2)
			mockDataplane.addPolicy(&caliXFRMPolicy1)
		})
		It("should clean up only the calico policy", func() {
			polTable.Apply()
			Expect(mockDataplane.ActivePolicies).To(ConsistOf(nonCaliPolicy1, nonCaliPolicy2))
			Expect(mockDataplane.NumListCalls).To(Equal(1))
		})
		It("should only resync once", func() {
			polTable.Apply()
			polTable.Apply()
			Expect(mockDataplane.NumListCalls).To(Equal(1))
		})

		Context("with pending policy matching the dataplane policy", func() {
			BeforeEach(func() {
				polTable.SetRule(caliSel1, &caliPol1)
			})

			It("should leave all policies untouched", func() {
				polTable.Apply()
				Expect(mockDataplane.ActivePolicies).To(ConsistOf(nonCaliPolicy1, nonCaliPolicy2, caliXFRMPolicy1))
				Expect(mockDataplane.NumListCalls).To(Equal(1))
				Expect(mockDataplane.NumUpdateCalls).To(Equal(0))
			})
		})
		Context("with cached policy matching the dataplane policy", func() {
			BeforeEach(func() {
				polTable.SetRule(caliSel1, &caliPol1)
				polTable.Apply()
			})

			It("resync should leave all policies untouched", func() {
				polTable.QueueResync()
				polTable.Apply()
				Expect(mockDataplane.ActivePolicies).To(ConsistOf(nonCaliPolicy1, nonCaliPolicy2, caliXFRMPolicy1))
				Expect(mockDataplane.NumListCalls).To(Equal(2))
				Expect(mockDataplane.NumUpdateCalls).To(Equal(0))
			})
		})
		Context("with pending policy not quite matching the dataplane policy", func() {
			BeforeEach(func() {
				polTable.SetRule(caliSel1, &caliPol1b)
			})

			It("should update the policy", func() {
				polTable.Apply()
				Expect(mockDataplane.ActivePolicies).To(ConsistOf(nonCaliPolicy1, nonCaliPolicy2, caliXFRMPolicy1b))
				Expect(mockDataplane.NumListCalls).To(Equal(1))
				Expect(mockDataplane.NumUpdateCalls).To(Equal(1))
			})
		})
		Context("with pending policy not quite matching the dataplane policy (after first apply)", func() {
			BeforeEach(func() {
				polTable.Apply()
				polTable.SetRule(caliSel1, &caliPol1b)
			})

			It("should update the policy", func() {
				polTable.Apply()
				Expect(mockDataplane.ActivePolicies).To(ConsistOf(nonCaliPolicy1, nonCaliPolicy2, caliXFRMPolicy1b))
				Expect(mockDataplane.NumListCalls).To(Equal(1))
				Expect(mockDataplane.NumUpdateCalls).To(Equal(1))
			})
		})
		Context("with an mismatched policy in the cache", func() {
			BeforeEach(func() {
				polTable.SetRule(caliSel1, &caliPol1)
				polTable.Apply()
				// Simulate another process changing the policy.
				mockDataplane.XfrmPolicyUpdate(&caliXFRMPolicy1b)
			})

			It("should fix the policy", func() {
				polTable.QueueResync()
				polTable.Apply()
				Expect(mockDataplane.ActivePolicies).To(ConsistOf(nonCaliPolicy1, nonCaliPolicy2, caliXFRMPolicy1))
				Expect(mockDataplane.NumListCalls).To(Equal(2))
				Expect(mockDataplane.NumUpdateCalls).To(Equal(2))
			})
		})
		Context("with a policy missing from the dataplane", func() {
			BeforeEach(func() {
				polTable.SetRule(caliSel1, &caliPol1)
				polTable.Apply()
				// Simulate another process deleting the policy.
				mockDataplane.XfrmPolicyDel(&caliXFRMPolicy1)
				Expect(mockDataplane.ActivePolicies).To(ConsistOf(nonCaliPolicy1, nonCaliPolicy2))
			})

			It("should fix the policy", func() {
				polTable.QueueResync()
				polTable.Apply()
				Expect(mockDataplane.ActivePolicies).To(ConsistOf(nonCaliPolicy1, nonCaliPolicy2, caliXFRMPolicy1))
				Expect(mockDataplane.NumListCalls).To(Equal(2))
				Expect(mockDataplane.NumUpdateCalls).To(Equal(1))
			})

			Context("and a pending deletion", func() {
				BeforeEach(func() {
					polTable.DeleteRule(caliSel1)
				})

				It("resync should squash the delete", func() {
					polTable.QueueResync()
					polTable.Apply()
					Expect(mockDataplane.ActivePolicies).To(ConsistOf(nonCaliPolicy1, nonCaliPolicy2))
					Expect(mockDataplane.NumListCalls).To(Equal(2))
					Expect(mockDataplane.NumUpdateCalls).To(Equal(0))
					Expect(mockDataplane.NumDeleteCalls).To(Equal(1)) // Our deletion
				})
			})
			Context("and a pending update", func() {
				BeforeEach(func() {
					polTable.SetRule(caliSel1, &caliPol1b)
				})

				It("resync should squash the delete", func() {
					polTable.QueueResync()
					polTable.Apply()
					Expect(mockDataplane.ActivePolicies).To(ConsistOf(nonCaliPolicy1, nonCaliPolicy2, caliXFRMPolicy1b))
					Expect(mockDataplane.NumListCalls).To(Equal(2))
					Expect(mockDataplane.NumUpdateCalls).To(Equal(1))
					Expect(mockDataplane.NumDeleteCalls).To(Equal(1)) // Our deletion
				})
			})
		})
	})

	AfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			// Useful on failure.
			polTable.DumpStateToLog()
		}
	})
})

type mockIPSecDataplane struct {
	Errors testutils.ErrorProducer

	netlinkHandleOpen bool
	cumulativeSleep   time.Duration

	NumListCalls, NumUpdateCalls, NumDeleteCalls int

	ActivePolicies []netlink.XfrmPolicy
}

func newMockIPSecDataplane() *mockIPSecDataplane {
	return &mockIPSecDataplane{Errors: testutils.NewErrorQueue()}
}

func (m *mockIPSecDataplane) addPolicy(pol *netlink.XfrmPolicy) {
	m.ActivePolicies = append(m.ActivePolicies, *pol)
}

func (m *mockIPSecDataplane) newNetlinkHandle() (NetlinkXFRMIface, error) {
	err := m.Errors.NextError("newNetlinkHandle")
	if err != nil {
		return nil, err
	}
	if m.netlinkHandleOpen {
		Fail("New netlink handle opened without closing previous one")
	}
	m.netlinkHandleOpen = true
	return m, nil
}

func (m *mockIPSecDataplane) sleep(duration time.Duration) {
	m.cumulativeSleep += duration
}

// NetlinkXFRMIface methods

func (m *mockIPSecDataplane) XfrmPolicyList(family int) ([]netlink.XfrmPolicy, error) {
	m.NumListCalls++
	Expect(family).To(Equal(netlink.FAMILY_V4))
	err := m.Errors.NextError("XfrmPolicyList")
	if err != nil {
		return nil, err
	}
	var result []netlink.XfrmPolicy
	for _, x := range m.ActivePolicies {
		result = append(result, x)
	}
	return result, nil
}

func (m *mockIPSecDataplane) XfrmPolicyUpdate(updatedPolicy *netlink.XfrmPolicy) error {
	m.NumUpdateCalls++

	err := m.Errors.NextError("XfrmPolicyUpdate")
	if err != nil {
		return err
	}

	Expect(updatedPolicy).NotTo(BeNil())
	if updatedPolicy.Action != netlink.XFRM_POLICY_ALLOW && updatedPolicy.Action != netlink.XFRM_POLICY_BLOCK {
		Fail(fmt.Sprintf("Unexpected action: %v", updatedPolicy.Action))
	}
	Expect(updatedPolicy.Proto).To(BeZero(), "mock dataplane doesn't support this field")
	Expect(updatedPolicy.DstPort).To(BeZero(), "mock dataplane doesn't support this field")
	Expect(updatedPolicy.SrcPort).To(BeZero(), "mock dataplane doesn't support this field")
	Expect(updatedPolicy.Index).To(BeZero(), "mock dataplane doesn't support this field")
	Expect(updatedPolicy.Priority).To(BeZero(), "mock dataplane doesn't support this field")
	Expect(updatedPolicy.Tmpls).To(HaveLen(1))
	Expect(updatedPolicy.Tmpls[0].Reqid).To(Equal(50), "policy was missing req ID")

	for i, x := range m.ActivePolicies {
		if netsEqual(x.Src, updatedPolicy.Src) &&
			netsEqual(x.Dst, updatedPolicy.Dst) &&
			x.Dir == updatedPolicy.Dir &&
			reflect.DeepEqual(x.Mark, updatedPolicy.Mark) {
			m.ActivePolicies[i] = *updatedPolicy
			logrus.Info("Replacing existing policy")
			return nil
		}
	}
	logrus.Info("Adding new policy")
	m.ActivePolicies = append(m.ActivePolicies, *updatedPolicy)
	return nil
}

func (m *mockIPSecDataplane) XfrmPolicyDel(policyToDelete *netlink.XfrmPolicy) error {
	m.NumDeleteCalls++
	err := m.Errors.NextError("XfrmPolicyDel")
	if err != nil {
		return err
	}

	var newActivePolicies []netlink.XfrmPolicy
	for _, x := range m.ActivePolicies {
		if netsEqual(x.Src, policyToDelete.Src) &&
			netsEqual(x.Dst, policyToDelete.Dst) &&
			x.Dir == policyToDelete.Dir &&
			reflect.DeepEqual(x.Mark, policyToDelete.Mark) {
			continue
		}
		newActivePolicies = append(newActivePolicies, x)
	}
	if len(newActivePolicies) == len(m.ActivePolicies) {
		return errors.New("policy not found")
	}
	m.ActivePolicies = newActivePolicies
	return nil
}

func (m *mockIPSecDataplane) Delete() {
	if !m.netlinkHandleOpen {
		Fail("Netlink handle closed while not open")
	}
	m.netlinkHandleOpen = false
}

func netsEqual(a, b *net.IPNet) bool {
	if a == nil {
		a = &zeroCIDR
	}
	if b == nil {
		b = &zeroCIDR
	}
	if a == b {
		return true
	}
	return a.String() == b.String()
}
