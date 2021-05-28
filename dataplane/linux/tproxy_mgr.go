// +build !windows

// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.
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
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/config"
	"github.com/projectcalico/felix/ipsets"
)

type tproxyManager struct {
	ipSetsV4 *ipsets.IPSets
	ipSetsV6 *ipsets.IPSets
}

func newTproxyManager(ipSetsV4, ipSetsV6 *ipsets.IPSets, dests []config.ServerPort) *tproxyManager {
	maxsize := 1000
	svcs := []string{}
	nps := []string{}

	for _, serverPort := range dests {
		if serverPort.IP == "0.0.0.0" {
			nps = append(nps, fmt.Sprintf("%v", serverPort.Port))
		} else {
			svcs = append(svcs, fmt.Sprintf("%v,tcp:%v", serverPort.IP, serverPort.Port))
		}
	}

	log.WithField("services", svcs).Info("tproxyManager")
	log.WithField("nodeports", nps).Info("tproxyManager")

	ipSetsV4.AddOrReplaceIPSet(
		ipsets.IPSetMetadata{SetID: "tproxy-services", Type: ipsets.IPSetTypeHashIPPort, MaxSize: maxsize},
		svcs,
	)
	ipSetsV4.AddOrReplaceIPSet(
		ipsets.IPSetMetadata{SetID: "tproxy-nodeports", Type: ipsets.IPSetTypeBitmapPort, RangeMax: 0xffff},
		nps,
	)
	ipSetsV6.AddOrReplaceIPSet(
		ipsets.IPSetMetadata{SetID: "tproxy-services", Type: ipsets.IPSetTypeHashIPPort, MaxSize: maxsize},
		[]string{},
	)
	/*
		ipSetsV6.AddOrReplaceIPSet(
			ipsets.IPSetMetadata{SetID: "tproxy-nodeports", Type: ipsets.IPSetTypeBitmapPort, RangeMax: 0xffff},
			[]string{},
		)
	*/

	return &tproxyManager{
		ipSetsV4: ipSetsV4,
		ipSetsV6: ipSetsV6,
	}
}

func (m *tproxyManager) OnUpdate(msg interface{}) {
}

func (m *tproxyManager) CompleteDeferredWork() error {
	return nil
}
