// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package calc

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
)

var _ = Describe("PolicyResolver", func() {
	var (
		uut *PolicyResolver
		cbs *prCallbacks
	)

	BeforeEach(func() {
		uut = NewPolicyResolver()
		cbs = &prCallbacks{}
		uut.RegisterCallback(cbs)
	})

	It("handles egress IP set ID", func() {

		By("OnUpdate for local endpoint")
		we1Key := model.WorkloadEndpointKey{
			WorkloadID: "we1",
		}
		uut.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key: we1Key,
				Value: &model.WorkloadEndpoint{
					Name: "we1",
				},
			},
			UpdateType: api.UpdateTypeKVNew,
		})
		uut.OnDatamodelStatus(api.InSync)

		// Expect OnEndpointTierUpdate with no egress ID.
		cbs.ExpectEndpointTierUpdate(we1Key, "")

		By("OnEndpointEgressDataUpdate for that endpoint")
		uut.OnEndpointEgressDataUpdate(we1Key, epEgressData{ipSetID: "e:abcdef"})

		// Expect OnEndpointTierUpdate with that egress IP set ID.
		cbs.ExpectEndpointTierUpdate(we1Key, "e:abcdef")

		By("another OnUpdate for same WE")
		uut.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key: we1Key,
				Value: &model.WorkloadEndpoint{
					Name: "we1",
				},
			},
			UpdateType: api.UpdateTypeKVUpdated,
		})
		cbs.ExpectEndpointTierUpdate(we1Key, "e:abcdef")

		By("OnEndpointEgressDataUpdate for that endpoint with no egress IP set")
		uut.OnEndpointEgressDataUpdate(we1Key, epEgressData{})
		cbs.ExpectEndpointTierUpdate(we1Key, "")
	})
})

type prCallbacks struct {
	keys           []model.Key
	egressIPSetIDs []string
}

func (tc *prCallbacks) OnEndpointTierUpdate(endpointKey model.Key, endpoint interface{}, egressData EndpointEgressData, filteredTiers []tierInfo) {
	tc.keys = append(tc.keys, endpointKey)
	tc.egressIPSetIDs = append(tc.egressIPSetIDs, egressData.EgressIPSetID)
}

func (tc *prCallbacks) ExpectEndpointTierUpdate(key model.WorkloadEndpointKey, egressIPSetID string) {
	Expect(len(tc.keys)).To(BeNumerically(">=", 1))
	Expect(len(tc.egressIPSetIDs)).To(BeNumerically(">=", 1))
	Expect(tc.keys[0].(model.WorkloadEndpointKey)).To(Equal(key))
	Expect(tc.egressIPSetIDs[0]).To(Equal(egressIPSetID))
	tc.keys = tc.keys[1:]
	tc.egressIPSetIDs = tc.egressIPSetIDs[1:]
}

func (tc *prCallbacks) ExpectNoMoreCallbacks() {
	Expect(len(tc.keys)).To(BeZero())
	Expect(len(tc.egressIPSetIDs)).To(BeZero())
}
