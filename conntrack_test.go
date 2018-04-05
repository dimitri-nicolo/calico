package nfnetlink_test

import (
	"github.com/tigera/nfnetlink"
	"github.com/tigera/nfnetlink/nfnl"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Conntrack Entry DNAT", func() {
	var cte nfnetlink.CtEntry
	var original_dnat, reply nfnetlink.CtTuple

	BeforeEach(func() {
		original_dnat = nfnetlink.CtTuple{
			Src:        [16]byte{1, 1, 1, 1},
			Dst:        [16]byte{3, 3, 3, 3},
			L3ProtoNum: 2048,
			ProtoNum:   6,
			L4Src: nfnetlink.CtL4Src{
				Port: 12345,
			},
			L4Dst: nfnetlink.CtL4Dst{
				Port: 80,
			},
		}
		reply = nfnetlink.CtTuple{
			Src:        [16]byte{2, 2, 2, 2},
			Dst:        [16]byte{1, 1, 1, 1},
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
			OriginalTuple: original_dnat,
			ReplyTuple:    reply,
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
		It("should return tuple after parsing DNAT info", func() {
			t, _ := cte.OriginalTupleWithoutDNAT()
			Expect(t.Src).To(Equal(reply.Dst))
			Expect(t.Dst).To(Equal(reply.Src))
			Expect(t.L3ProtoNum).To(Equal(original_dnat.L3ProtoNum))
			Expect(t.ProtoNum).To(Equal(original_dnat.ProtoNum))
			Expect(t.L4Src).To(Equal(original_dnat.L4Src))
			Expect(t.L4Dst).To(Equal(original_dnat.L4Dst))
		})

	})
})
