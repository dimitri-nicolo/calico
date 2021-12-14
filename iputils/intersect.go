// Copyright (c) 2018-2020 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package iputils

import (
	"sort"

	"github.com/projectcalico/felix/iptree"
)

func IntersectCIDRs(aStrs []string, bStrs []string) (out []string) {
	aCIDRs := parseCIDRs(aStrs)
	bCIDRs := parseCIDRs(bStrs)

	intersection := iptree.Intersect(aCIDRs, bCIDRs)
	out = intersection.CoveringCIDRStrings()

	// Sort the output for determinism both in testing and in rule generation.
	sort.Strings(out)

	return
}

func parseCIDRs(in []string) (out *iptree.IPTree) {
	out = iptree.New(4)
	for _, s := range in {
		out.AddCIDRString(s)
	}
	return
}
