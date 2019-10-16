// Copyright (c) 2016-2017,2019 Tigera, Inc. All rights reserved.
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

package calc

import (
	"fmt"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/set"
)

type PolicySorter struct {
	tiers map[string]*tierInfo
}

func NewPolicySorter() *PolicySorter {
	return &PolicySorter{
		tiers: make(map[string]*tierInfo),
	}
}

func policyTypesEqual(pol1, pol2 *model.Policy) bool {
	if pol1.Types == nil {
		return pol2.Types == nil
	}
	if pol2.Types == nil {
		return false
	}
	types1 := set.FromArray(pol1.Types)
	types2 := set.FromArray(pol2.Types)
	return types1.Equals(types2)
}

func (poc *PolicySorter) OnUpdate(update api.Update) (dirty bool) {
	switch key := update.Key.(type) {
	case model.TierKey:
		tierName := key.Name
		logCxt := log.WithField("tierName", tierName)
		tierInfo := poc.tiers[tierName]
		if update.Value != nil {
			newTier := update.Value.(*model.Tier)
			logCxt.WithField("order", newTier.Order).Debug("Tier update")
			if tierInfo == nil {
				tierInfo = NewTierInfo(key.Name)
				poc.tiers[tierName] = tierInfo
				dirty = true
			}
			if tierInfo.Order != newTier.Order {
				tierInfo.Order = newTier.Order
				dirty = true
			}
			tierInfo.Valid = true
		} else {
			// Deletion.
			if tierInfo != nil {
				tierInfo.Valid = false
				if len(tierInfo.Policies) == 0 {
					delete(poc.tiers, tierName)
				}
				dirty = true
			}
		}
	case model.PolicyKey:
		tierInfo := poc.tiers[key.Tier]
		var oldPolicy *model.Policy
		if tierInfo != nil {
			oldPolicy = tierInfo.Policies[key]
		}
		if update.Value != nil {
			newPolicy := update.Value.(*model.Policy)
			if tierInfo == nil {
				tierInfo = NewTierInfo(key.Tier)
				poc.tiers[key.Tier] = tierInfo
			}
			if oldPolicy == nil ||
				oldPolicy.Order != newPolicy.Order ||
				oldPolicy.DoNotTrack != newPolicy.DoNotTrack ||
				oldPolicy.PreDNAT != newPolicy.PreDNAT ||
				oldPolicy.ApplyOnForward != newPolicy.ApplyOnForward ||
				!policyTypesEqual(oldPolicy, newPolicy) {
				dirty = true
			}
			tierInfo.Policies[key] = newPolicy
		} else {
			if oldPolicy != nil {
				delete(tierInfo.Policies, key)
				dirty = true
			}
		}
	}
	return
}

func (poc *PolicySorter) Sorted() []*tierInfo {
	tiers := make([]*tierInfo, 0, len(poc.tiers))
	for _, tier := range poc.tiers {
		tiers = append(tiers, tier)
	}
	tiersByOrder := TierByOrder(tiers)
	log.Debugf("Order before sorting tiers: %v", tiersByOrder)
	sort.Sort(tiersByOrder)
	log.Debugf("Order after sorting tiers: %v", tiersByOrder)
	for _, tierInfo := range poc.tiers {
		tierInfo.OrderedPolicies = make([]PolKV, 0, len(tierInfo.Policies))
		for k, v := range tierInfo.Policies {
			tierInfo.OrderedPolicies = append(tierInfo.OrderedPolicies,
				PolKV{Key: k, Value: v})
		}
		// Note: using explicit Debugf() here rather than WithFields(); we want the []PolKV slice
		// to be stringified with %v rather than %#v (as used by WithField()).
		log.Debugf("Order before sorting: %v", tierInfo.OrderedPolicies)
		sort.Sort(PolicyByOrder(tierInfo.OrderedPolicies))
		log.Debugf("Order after sorting: %v", tierInfo.OrderedPolicies)
	}
	return tiers
}

type TierByOrder []*tierInfo

func (a TierByOrder) Len() int      { return len(a) }
func (a TierByOrder) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a TierByOrder) Less(i, j int) bool {
	if !a[i].Valid {
		return false
	} else if !a[j].Valid {
		return true
	}
	if a[i].Order == nil {
		return false
	} else if a[j].Order == nil {
		return true
	}
	if *a[i].Order == *a[j].Order {
		return a[i].Name < a[j].Name
	}
	return *a[i].Order < *a[j].Order
}
func (a TierByOrder) String() string {
	parts := make([]string, len(a))
	for i, ti := range a {
		order := "default"
		if ti.Order != nil {
			order = fmt.Sprintf("%f", *ti.Order)
		}
		parts[i] = fmt.Sprintf("%s(%s)", ti.Name, order)
	}
	return strings.Join(parts, ", ")
}

// Note: PolKV is really internal to the calc package.  It is named with an initial capital so that
// the test package calc_test can also use it.
type PolKV struct {
	Key   model.PolicyKey
	Value *model.Policy

	// Caches for whether the policy governs ingress and/or egress traffic.
	ingress *bool
	egress  *bool
}

func (p PolKV) String() string {
	orderStr := "nil policy"
	if p.Value != nil {
		if p.Value.Order != nil {
			orderStr = fmt.Sprintf("%v", *p.Value.Order)
		} else {
			orderStr = "default"
		}
	}
	return fmt.Sprintf("%s(%s)", p.Key.Name, orderStr)
}

func (p *PolKV) governsType(wanted string) bool {
	// Back-compatibility: no Types means Ingress and Egress.
	if len(p.Value.Types) == 0 {
		return true
	}
	for _, t := range p.Value.Types {
		if strings.EqualFold(t, wanted) {
			return true
		}
	}
	return false
}

func (p *PolKV) GovernsIngress() bool {
	if p.ingress == nil {
		governsIngress := p.governsType("ingress")
		p.ingress = &governsIngress
	}
	return *p.ingress
}

func (p *PolKV) GovernsEgress() bool {
	if p.egress == nil {
		governsEgress := p.governsType("egress")
		p.egress = &governsEgress
	}
	return *p.egress
}

type PolicyByOrder []PolKV

func (a PolicyByOrder) Len() int      { return len(a) }
func (a PolicyByOrder) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a PolicyByOrder) Less(i, j int) bool {
	bothNil := a[i].Value.Order == nil && a[j].Value.Order == nil
	bothSet := a[i].Value.Order != nil && a[j].Value.Order != nil
	ordersEqual := bothNil || bothSet && (*a[i].Value.Order == *a[j].Value.Order)

	if ordersEqual {
		return model.PolicyNameLessThan(a[i].Key.Name, a[j].Key.Name)
	}

	// nil order maps to "infinity"
	if a[i].Value.Order == nil {
		return false
	} else if a[j].Value.Order == nil {
		return true
	}

	// Otherwise, use numeric comparison.
	return *a[i].Value.Order < *a[j].Value.Order
}

type tierInfo struct {
	Name            string
	Valid           bool
	Order           *float64
	Policies        map[model.PolicyKey]*model.Policy
	OrderedPolicies []PolKV
}

func NewTierInfo(name string) *tierInfo {
	return &tierInfo{
		Name:     name,
		Policies: make(map[model.PolicyKey]*model.Policy),
	}
}

func NewTierInfoSlice() []tierInfo {
	return nil
}

func (t tierInfo) String() string {
	policies := make([]string, len(t.OrderedPolicies))
	for ii, pol := range t.OrderedPolicies {
		polType := "t"
		if pol.Value != nil {
			if pol.Value.DoNotTrack {
				polType = "u"
			} else if pol.Value.PreDNAT {
				polType = "p"
			}

			//Append ApplyOnForward flag.
			if pol.Value.ApplyOnForward {
				polType = polType + "f"
			}
		}
		policies[ii] = fmt.Sprintf("%v(%v)", pol.Key.Name, polType)
	}
	return fmt.Sprintf("%v -> %v", t.Name, policies)
}
