// Copyright (c) 2017-2024 Tigera, Inc. All rights reserved.
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

package intdataplane

import (
	"sort"

	"github.com/davecgh/go-spew/spew"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/felix/iptables"
)

type mockTable struct {
	Table          string
	currentChains  map[string]*iptables.Chain
	expectedChains map[string]*iptables.Chain
	UpdateCalled   bool
}

func newMockTable(table string) *mockTable {
	return &mockTable{
		Table:          table,
		currentChains:  map[string]*iptables.Chain{},
		expectedChains: map[string]*iptables.Chain{},
	}
}

func logChains(message string, chains []*iptables.Chain) {
	if chains == nil {
		log.Debug(message, " with nil chains")
	} else {
		log.Debug(message, spew.Sdump(chains))
	}
}

func (t *mockTable) UpdateChain(chain *iptables.Chain) {
	t.UpdateChains([]*iptables.Chain{chain})
}

func (t *mockTable) UpdateChains(chains []*iptables.Chain) {
	t.UpdateCalled = true
	logChains("UpdateChains", chains)
	for _, chain := range chains {
		t.currentChains[chain.Name] = chain
	}
}

func (t *mockTable) RemoveChains(chains []*iptables.Chain) {
	logChains("RemoveChains", chains)
	for _, chain := range chains {
		t.RemoveChainByName(chain.Name)
		delete(t.currentChains, chain.Name)
	}
}

func (t *mockTable) RemoveChainByName(name string) {
	delete(t.currentChains, name)
}

func (t *mockTable) checkChains(expecteds [][]*iptables.Chain) {
	t.expectedChains = map[string]*iptables.Chain{}
	for _, expected := range expecteds {
		for _, chain := range expected {
			t.expectedChains[chain.Name] = chain
		}
	}
	t.checkChainsSameAsBefore(1)
}

func (t *mockTable) checkChainsSameAsBefore(optionalOffset ...int) {
	log.Debug("Expected chains")
	for _, chain := range t.expectedChains {
		log.WithField("chain", *chain).Debug("")
	}

	var currentChains []iptables.Chain
	for _, c := range t.currentChains {
		currentChains = append(currentChains, *c)
	}
	sort.Slice(currentChains, func(i, j int) bool {
		return currentChains[i].Name < currentChains[j].Name
	})
	var expectedChains []iptables.Chain
	for _, c := range t.expectedChains {
		expectedChains = append(expectedChains, *c)
	}
	sort.Slice(expectedChains, func(i, j int) bool {
		return expectedChains[i].Name < expectedChains[j].Name
	})
	offset := 1
	if len(optionalOffset) > 0 {
		offset += optionalOffset[0]
	}
	format.MaxLength = 1000000
	ExpectWithOffset(offset, currentChains).To(Equal(expectedChains), t.Table+" chains incorrect")
}

func (t *mockTable) getCurrentChainByName(name string) *iptables.Chain {
	return t.currentChains[name]
}
