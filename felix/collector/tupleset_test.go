// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package collector

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/libcalico-go/lib/set"
)

var _ = Describe("Set", func() {
	var s tupleSet
	t1 := *NewTuple(localIp1, localIp2, proto_tcp, 1, 1)
	t2 := *NewTuple(localIp1, localIp2, proto_tcp, 2, 2)
	t3 := *NewTuple(localIp1, localIp2, proto_tcp, 3, 3)
	BeforeEach(func() {
		s = NewTupleSet()
	})
	It("should be empty", func() {
		Expect(s.Len()).To(BeZero())
	})
	Describe("after adding t1 and t2", func() {
		BeforeEach(func() {
			s.Add(t1)
			s.Add(t2)
			s.Add(t2) // Duplicate should have no effect
		})
		It("should have 2 tuples", func() {
			Expect(s.Len()).Should(Equal(2))
		})
		It("should contain t1", func() {
			Expect(s.Contains(t1)).To(BeTrue())
		})
		It("should contain t2", func() {
			Expect(s.Contains(t2)).To(BeTrue())
		})
		It("should not contain t3", func() {
			Expect(s.Contains(t3)).To(BeFalse())
		})
		Describe("after removing t2", func() {
			BeforeEach(func() {
				s.Discard(t2)
			})
			It("should have 1 tple", func() {
				Expect(s.Len()).Should(Equal(1))
			})
			It("should contain t1", func() {
				Expect(s.Contains(t1)).To(BeTrue())
			})
			It("should not contain t2", func() {
				Expect(s.Contains(t2)).To(BeFalse())
			})
			It("should not contain t3", func() {
				Expect(s.Contains(t3)).To(BeFalse())
			})
		})
	})
})

func BenchmarkSetGeneric(b *testing.B) {
	t := *NewTuple(localIp1, localIp2, proto_tcp, 1000, 1000)
	s := set.New[Tuple]()
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		s.Add(t)
	}
}

func BenchmarkSetTuple(b *testing.B) {
	t := *NewTuple(localIp1, localIp2, proto_tcp, 1000, 1000)
	s := NewTupleSet()
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		s.Add(t)
	}
}
