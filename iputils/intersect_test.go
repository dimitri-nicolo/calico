// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package iputils

import (
	"sort"
	"testing"

	. "github.com/onsi/gomega"
)

func TestIntersectCIDRs(t *testing.T) {
	for _, test := range []struct {
		Name        string
		as, bs, exp []string
	}{
		{"zero", []string{"0.0.0.0/0"}, []string{"128.0.0.0/1", "0.0.0.0/1"}, []string{"128.0.0.0/1", "0.0.0.0/1"}},
		{"self", []string{"10.0.0.1"}, []string{"10.0.0.1"}, []string{"10.0.0.1/32"}},
		{"different format", []string{"10.0.0.1/32"}, []string{"10.0.0.1"}, []string{"10.0.0.1/32"}},
		{"smallest wins", []string{"10.0.1.0/24"}, []string{"10.0.1.128/25"}, []string{"10.0.1.128/25"}},
		{"non-match", []string{"10.0.1.0/24"}, []string{"10.0.2.0/24"}, nil},
		{
			"some matches",
			[]string{"10.0.1.0/24", "10.1.0.0/16"},
			[]string{"10.0.2.0/24", "10.0.1.1/32", "10.0.1.2/32", "11.0.1.1/32", "10.0.1.128/26",
				"10.1.2.0/24"},
			[]string{"10.0.1.1/32", "10.0.1.2/32", "10.0.1.128/26", "10.1.2.0/24"},
		},
		{
			"overlap in both directions",
			[]string{"10.0.1.0/24", "10.0.2.1/32"},
			[]string{"10.0.2.0/24", "10.0.1.1/32"},
			[]string{"10.0.1.1/32", "10.0.2.1/32"},
		},
	} {
		test := test
		sort.Strings(test.as)
		sort.Strings(test.bs)
		sort.Strings(test.exp)

		t.Run(test.Name, func(t *testing.T) {
			RegisterTestingT(t)
			cidrs := IntersectCIDRs(test.as, test.bs)
			sort.Strings(cidrs)
			Expect(cidrs).To(Equal(test.exp))

			// Intersection should be commutative.
			t.Run("as and bs reversed", func(t *testing.T) {
				RegisterTestingT(t)
				cidrs := IntersectCIDRs(test.bs, test.as)
				sort.Strings(cidrs)
				Expect(cidrs).To(Equal(test.exp))
			})
		})
	}
}
