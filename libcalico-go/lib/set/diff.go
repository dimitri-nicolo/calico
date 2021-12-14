// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package set

// IterDifferences iterates through the set of items that are in A but not in B, and the set that are in B but not in A.
func IterDifferences(a, b Set, aNotB, bNotA func(interface{}) error) {
	a.Iter(func(item interface{}) error {
		if !b.Contains(item) {
			return aNotB(item)
		}
		return nil
	})
	b.Iter(func(item interface{}) error {
		if !a.Contains(item) {
			return bNotA(item)
		}
		return nil
	})
}
