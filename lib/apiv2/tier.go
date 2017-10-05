// Copyright (c) 2017 Tigera, Inc. All rights reserved.

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

package apiv2

// TierSpec contains the specification for a security policy tier resource.  A tier contains a set of
// policies that are applied to packets.  Multiple tiers may be created and each tier is applied
// in the order specified in the tier specification.
type TierSpec struct {
	// Order is an optional field that specifies the order in which the tier is applied.
	// Tiers with higher "order" are applied after those with lower order.  If the order
	// is omitted, it may be considered to be "infinite" - i.e. the tier will be applied
	// last.  Tiers with identical order will be applied in alphanumerical order based
	// on the Tier "Name".
	Order *float64 `json:"order,omitempty"`
}
