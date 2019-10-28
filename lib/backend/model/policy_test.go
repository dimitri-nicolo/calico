// Copyright (c) 2017,2019 Tigera, Inc. All rights reserved.

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

package model_test

import (
	"github.com/projectcalico/libcalico-go/lib/backend/model"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Policy functions", func() {
	It("Policy should stringify correctly", func() {
		order := 10.5
		p := model.Policy{
			Order:          &order,
			InboundRules:   []model.Rule{model.Rule{Action: "Deny"}},
			OutboundRules:  []model.Rule{model.Rule{Action: "Allow"}},
			Selector:       "apples=='oranges'",
			DoNotTrack:     false,
			PreDNAT:        true,
			ApplyOnForward: true,
			Types:          []string{"Ingress", "Egress"},
		}
		Expect(p.String()).To(Equal(`order:10.5,selector:"apples=='oranges'",inbound:Deny,outbound:Allow,untracked:false,pre_dnat:true,apply_on_forward:true,types:Ingress;Egress`))
	})

	It("Policy should identify as staged by name", func() {
		Expect(model.PolicyIsStaged("staged:policy1")).To(BeTrue())
		Expect(model.PolicyIsStaged("policy1")).To(BeFalse())
	})

	It("Staged policy name should be less than non-staged equivalent", func() {
		Expect(model.PolicyNameLessThan("tier1.policy0", "tier1.policy1")).To(BeTrue())
		Expect(model.PolicyNameLessThan("tier1.policy1", "tier1.policy0")).To(BeFalse())
		Expect(model.PolicyNameLessThan("staged:knp.default.policy1", "knp.default.policy1")).To(BeTrue())
		Expect(model.PolicyNameLessThan("knp.default.policy1", "staged:knp.default.policy1")).To(BeFalse())
		Expect(model.PolicyNameLessThan("staged:ns1/tier2.policy0", "ns1/tier2.policy1")).To(BeTrue())
		Expect(model.PolicyNameLessThan("ns1/tier2.policy1", "ns1/staged:tier2.policy0")).To(BeFalse())
		Expect(model.PolicyNameLessThan("ns1/staged:tier2.policy0", "ns1/staged:tier2.policy1")).To(BeTrue())
		Expect(model.PolicyNameLessThan("ns1/staged:tier2.policy1", "ns1/staged:tier2.policy0")).To(BeFalse())
	})
})
