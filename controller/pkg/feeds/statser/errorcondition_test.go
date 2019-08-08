// Copyright 2019 Tigera Inc. All rights reserved.

package statser

import (
	"errors"
	"testing"

	. "github.com/onsi/gomega"
	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

func TestNewErrorConditions(t *testing.T) {
	g := NewWithT(t)

	ec, err := NewErrorConditions(1)
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(ec).ShouldNot(BeNil())

	ec, err = NewErrorConditions(0)
	g.Expect(err).Should(HaveOccurred())

	ec, err = NewErrorConditions(-1)
	g.Expect(err).Should(HaveOccurred())
}

func TestErrorConditions_Add(t *testing.T) {
	g := NewWithT(t)

	ec, err := NewErrorConditions(3)
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(ec.Errors()).Should(HaveLen(0))

	ec.Add("a", errors.New("t1"))
	g.Expect(ec.Errors()).Should(HaveLen(1))

	// Test duplicate
	ec.Add("a", errors.New("t1"))
	g.Expect(ec.Errors()).Should(HaveLen(1))

	// Test novel error
	ec.Add("a", errors.New("t2"))
	g.Expect(ec.Errors()).Should(HaveLen(2))

	// Test novel type
	ec.Add("b", errors.New("t2"))
	g.Expect(ec.Errors()).Should(HaveLen(3))

	// Test oversize
	ec.Add("c", errors.New("t2"))
	g.Expect(ec.Errors()).Should(ConsistOf([]v3.ErrorCondition{
		{"a", "t2"},
		{"b", "t2"},
		{"c", "t2"},
	}))

	// Test lru
	ec.Add("a", errors.New("t2"))
	ec.Add("d", errors.New("t2"))
	g.Expect(ec.Errors()).Should(ConsistOf([]v3.ErrorCondition{
		{"a", "t2"},
		{"c", "t2"},
		{"d", "t2"},
	}))
}

func TestErrorConditions_Clear(t *testing.T) {
	g := NewWithT(t)

	ec, err := NewErrorConditions(3)
	g.Expect(err).ShouldNot(HaveOccurred())

	ec.Add("a", errors.New("t1"))
	ec.Add("a", errors.New("t2"))
	ec.Add("b", errors.New("t3"))

	// Test clear one
	ec.Clear("b")
	g.Expect(ec.Errors()).Should(ConsistOf([]v3.ErrorCondition{
		{"a", "t1"},
		{"a", "t2"},
	}))

	// Test clear multiple
	ec.Add("b", errors.New("t3"))
	ec.Clear("a")
	g.Expect(ec.Errors()).Should(ConsistOf([]v3.ErrorCondition{
		{"b", "t3"},
	}))

	// Test clear everything
	ec.Clear("b")
	g.Expect(ec.Errors()).Should(HaveLen(0))
}

func TestErrorConditions_Errors(t *testing.T) {
	g := NewWithT(t)

	ec, err := NewErrorConditions(3)
	g.Expect(err).ShouldNot(HaveOccurred())

	ec.Add("a", errors.New("t1"))
	ec.Add("a", errors.New("t2"))
	ec.Add("b", errors.New("t3"))

	g.Expect(ec.Errors()).Should(ConsistOf([]v3.ErrorCondition{
		{"a", "t1"},
		{"a", "t2"},
		{"b", "t3"},
	}))
}

func TestErrorConditions_TypedErrors(t *testing.T) {
	g := NewWithT(t)

	ec, err := NewErrorConditions(3)
	g.Expect(err).ShouldNot(HaveOccurred())

	ec.Add("a", errors.New("t1"))
	ec.Add("a", errors.New("t2"))
	ec.Add("b", errors.New("t3"))

	g.Expect(ec.TypedErrors("a")).Should(ConsistOf([]v3.ErrorCondition{
		{"a", "t1"},
		{"a", "t2"},
	}))

	g.Expect(ec.TypedErrors("b")).Should(ConsistOf([]v3.ErrorCondition{
		{"b", "t3"},
	}))
}
