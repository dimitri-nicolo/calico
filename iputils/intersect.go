// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package iputils

import (
	"github.com/projectcalico/felix/ip"
	"github.com/projectcalico/libcalico-go/lib/set"
)

func IntersectCIDRs(aStrs []string, bStrs []string) (out []string) {
	aCIDRs := parseCIDRs(aStrs)
	bCIDRs := parseCIDRs(bStrs)

	intersection := set.New()

	for _, a := range aCIDRs {
		for _, b := range bCIDRs {
			if a.Prefix() == b.Prefix() {
				// Same length prefix, compare IPs.
				if a.Addr() == b.Addr() {
					intersection.Add(a)
				}
			} else if a.Prefix() < b.Prefix() {
				// See if a contains b.
				aNet := a.ToIPNet()
				if aNet.Contains(b.ToIPNet().IP) {
					// a contains b so intersection is b
					intersection.Add(b)
				}
			} else {
				// See if b contains a.
				bNet := b.ToIPNet()
				if bNet.Contains(a.ToIPNet().IP) {
					// b contains a so intersection is a
					intersection.Add(a)
				}
			}
		}
	}

	intersection.Iter(func(item interface{}) error {
		out = append(out, item.(ip.CIDR).String())
		return set.RemoveItem
	})
	return
}

func parseCIDRs(in []string) (out []ip.CIDR) {
	for _, s := range in {
		out = append(out, ip.MustParseCIDROrIP(s))
	}
	return
}
