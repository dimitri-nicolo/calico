// Copyright (c) 2018 Tigera, Inc. All rights reserved.
package windataplane

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/projectcalico/felix/dataplane/windows/ipsets"
	"github.com/projectcalico/felix/proto"
	"github.com/projectcalico/libcalico-go/lib/set"
)

func TestIPSetsManager(t *testing.T) {
	RegisterTestingT(t)

	ipSetsConfigV4 := ipsets.NewIPVersionConfig(
		ipsets.IPFamilyV4,
	)

	ipSetsV4 := ipsets.NewIPSets(ipSetsConfigV4)

	ipsetsMgr := newIPSetsManager(ipSetsV4)
	//update ipset
	ipsetsMgr.OnUpdate(&proto.IPSetUpdate{
		Id:      "id1",
		Members: []string{"10.0.0.1", "10.0.0.2"},
	})

	Expect(ipSetsV4.GetIPSetMembers("id1")).To(HaveLen(2))
	Expect((set.FromArray(ipSetsV4.GetIPSetMembers("id1")))).To(Equal(set.From("10.0.0.1", "10.0.0.2")))

	//update ipset with delta by removing and adding at the same time
	ipsetsMgr.OnUpdate(&proto.IPSetDeltaUpdate{
		Id:             "id1",
		AddedMembers:   []string{"10.0.0.3", "10.0.0.4"},
		RemovedMembers: []string{"10.0.0.1"},
	})

	Expect(ipSetsV4.GetIPSetMembers("id1")).To(HaveLen(3))
	Expect((set.FromArray(ipSetsV4.GetIPSetMembers("id1")))).To(Equal(set.From("10.0.0.2", "10.0.0.3", "10.0.0.4")))

	//remove ipsets
	ipsetsMgr.OnUpdate(&proto.IPSetRemove{
		Id: "id1",
	})

	Expect(ipSetsV4.GetIPSetMembers("id1")).To(BeNil())

	//update ipsets again here
	ipsetsMgr.OnUpdate(&proto.IPSetUpdate{
		Id:      "id1",
		Members: []string{"10.0.0.2", "10.0.0.3"},
	})

	Expect(ipSetsV4.GetIPSetMembers("id1")).To(HaveLen(2))
	Expect((set.FromArray(ipSetsV4.GetIPSetMembers("id1")))).To(Equal(set.From("10.0.0.2", "10.0.0.3")))

}
