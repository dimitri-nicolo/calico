// +build !windows

// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package intdataplane

import (
	"fmt"

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
	for _, serverPort := range dests {
		svcs = append(svcs, fmt.Sprintf("%v,tcp:%v", serverPort.IP, serverPort.Port))
	}
	ipSetsV4.AddOrReplaceIPSet(
		ipsets.IPSetMetadata{SetID: "tproxy-services", Type: ipsets.IPSetTypeHashIPPort, MaxSize: maxsize},
		svcs,
	)
	ipSetsV6.AddOrReplaceIPSet(
		ipsets.IPSetMetadata{SetID: "tproxy-services", Type: ipsets.IPSetTypeHashIPPort, MaxSize: maxsize},
		[]string{},
	)

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
