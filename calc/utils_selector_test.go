// Copyright (c) 2016-2020 Tigera, Inc. All rights reserved.
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

package calc_test

import (
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/hash"
	"github.com/projectcalico/libcalico-go/lib/selector"
)

func selectorID(selStr string) string {
	sel, err := selector.Parse(selStr)
	if err != nil {
		log.Panicf("Failed to parse %v: %v", selStr, err)
	}
	return sel.UniqueID()
}

func domainSelectorID(selStr string, domains []string) string {
	if len(selStr) != 0 {
		selID := selectorID(selStr)
		return "d" + selID[1:]
	}
	if len(domains) == 0 {
		log.Panic("either selector or domains must be specified")
	}
	return (hash.MakeUniqueID("d", strings.Join(domains, "#")))
}

func namedPortID(selector, protocol, portName string) string {
	selID := selectorID(selector)
	idToHash := selID + "," + protocol + "," + portName
	return hash.MakeUniqueID("n", idToHash)
}

func egressSelectorID(selStr string) string {
	selID := selectorID(selStr)
	return "e" + selID[1:]
}
