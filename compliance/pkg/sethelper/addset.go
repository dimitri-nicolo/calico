// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package sethelper

import "github.com/projectcalico/calico/libcalico-go/lib/set"

// AddSet adds the contents of set "from" into the set "to".
func AddSet(from, to set.Set) {
	from.Iter(func(item interface{}) error {
		to.Add(item)
		return nil
	})
}
