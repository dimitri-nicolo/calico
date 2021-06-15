// +build !windows

// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package intdataplane

import (
	"github.com/projectcalico/felix/ipsets"
)

type tproxyManager struct {
	ipSetsV4 *ipsets.IPSets
	ipSetsV6 *ipsets.IPSets
}

func newTproxyManager(maxsize int, ipSetsV4, ipSetsV6 *ipsets.IPSets) *tproxyManager {
	if ipSetsV4 != nil {
		ipSetsV4.AddOrReplaceIPSet(
			ipsets.IPSetMetadata{SetID: "tproxy-services", Type: ipsets.IPSetTypeHashIPPort, MaxSize: maxsize},
			[]string{},
		)
	}

	if ipSetsV6 != nil {
		ipSetsV6.AddOrReplaceIPSet(
			ipsets.IPSetMetadata{SetID: "tproxy-services", Type: ipsets.IPSetTypeHashIPPort, MaxSize: maxsize},
			[]string{},
		)
	}

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
