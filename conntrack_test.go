package nfnetlink_test

import (
	"net"

	"github.com/tigera/nfnetlink"
	"github.com/tigera/nfnetlink/nfnl"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Conntrack Entry DNAT", func() {
	var cte nfnetlink.CtEntry
	var orig_dnat, repl nfnetlink.CtTuple

	BeforeEach(func() {
		orig_dnat = nfnetlink.CtTuple{
			Src:        net.ParseIP("1.1.1.1"),
			Dst:        net.ParseIP("3.3.3.3"),
			L3ProtoNum: 2048,
			ProtoNum:   6,
			L4Src: nfnetlink.CtL4Src{
				Port: 12345,
			},
			L4Dst: nfnetlink.CtL4Dst{
				Port: 80,
			},
		}
		repl = nfnetlink.CtTuple{
			Src:        net.ParseIP("2.2.2.2"),
			Dst:        net.ParseIP("1.1.1.1"),
			L3ProtoNum: 2048,
			ProtoNum:   6,
			L4Src: nfnetlink.CtL4Src{
				Port: 80,
			},
			L4Dst: nfnetlink.CtL4Dst{
				Port: 12345,
			},
		}
		cte = nfnetlink.CtEntry{
			OrigTuples: []nfnetlink.CtTuple{orig_dnat},
			ReplTuples: []nfnetlink.CtTuple{repl},
		}
	})
	Describe("Check DNAT", func() {
		BeforeEach(func() {
			cte.Status = cte.Status | nfnl.IPS_DST_NAT
		})
		It("should return true for DNAT check", func() {
			Expect(cte.IsDNAT()).To(Equal(true))
		})
		It("should return true for NAT check", func() {
			Expect(cte.IsNAT()).To(Equal(true))
		})
		It("should return false for SNAT check", func() {
			Expect(cte.IsSNAT()).To(Equal(false))
		})
		It("should return orig tuple", func() {
			Expect(cte.OrigTuple()).To(Equal(orig_dnat))
		})
		It("should return repl tuple", func() {
			Expect(cte.ReplTuple()).To(Equal(repl))
		})
		It("should return tuple after parsing DNAT info", func() {
			t, _ := cte.OrigTupleWithoutDNAT()
			Expect(t.Src).To(Equal(repl.Dst))
			Expect(t.Dst).To(Equal(repl.Src))
			Expect(t.L3ProtoNum).To(Equal(orig_dnat.L3ProtoNum))
			Expect(t.ProtoNum).To(Equal(orig_dnat.ProtoNum))
			Expect(t.L4Src).To(Equal(orig_dnat.L4Src))
			Expect(t.L4Dst).To(Equal(orig_dnat.L4Dst))
		})

	})
})
