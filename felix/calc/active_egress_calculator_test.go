// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.

package calc

import (
	"fmt"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	libapiv3 "github.com/projectcalico/calico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ActiveEgressCalculator", func() {

	var (
		aec *ActiveEgressCalculator
		cbs *testCallbacks
	)

	we1Key := model.WorkloadEndpointKey{WorkloadID: "we1"}
	we1Value1 := &model.WorkloadEndpoint{
		Name:           "we1",
		EgressSelector: "black == 'white'",
	}
	we1Value2 := &model.WorkloadEndpoint{
		Name:           "we1",
		EgressSelector: "black == 'red'",
	}
	we2Key := model.WorkloadEndpointKey{WorkloadID: "we2"}

	BeforeEach(func() {
		aec = NewActiveEgressCalculator("EnabledPerNamespaceOrPerPod")
		cbs = &testCallbacks{}
		aec.OnIPSetActive = cbs.OnIPSetActive
		aec.OnIPSetInactive = cbs.OnIPSetInactive
		aec.OnEndpointEgressDataUpdate = cbs.OnEndpointEgressDataUpdate
	})

	It("generates expected callbacks for a single WorkloadEndpoint", func() {

		By("creating a WorkloadEndpoint with egress selector")
		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key:   we1Key,
				Value: we1Value1,
			},
			UpdateType: api.UpdateTypeKVNew,
		})

		// Expect IPSetActive and EgressIPSetIDUpdate.
		ipSetID1 := cbs.ExpectActive()
		cbs.ExpectEgressUpdate(we1Key, epEgressData{ipSetID: ipSetID1})
		cbs.ExpectNoMoreCallbacks()

		By("changing WorkloadEndpoint's egress selector")
		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key:   we1Key,
				Value: we1Value2,
			},
			UpdateType: api.UpdateTypeKVUpdated,
		})

		// Expect IPSetInactive for old selector.
		cbs.ExpectInactive(ipSetID1)

		// Expect IPSetActive and EgressIPSetIDUpdate with new ID.
		ipSetID2 := cbs.ExpectActive()
		cbs.ExpectEgressUpdate(we1Key, epEgressData{ipSetID: ipSetID2})
		cbs.ExpectNoMoreCallbacks()
		Expect(ipSetID2).NotTo(Equal(ipSetID1))

		By("deleting WorkloadEndpoint")
		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key:   we1Key,
				Value: nil,
			},
			UpdateType: api.UpdateTypeKVUpdated,
		})

		// Expect IPSetInactive for old selector.
		cbs.ExpectInactive(ipSetID2)
		cbs.ExpectEgressUpdate(we1Key, epEgressData{})
		cbs.ExpectNoMoreCallbacks()
	})

	It("generates expected callbacks for two WorkloadEndpoints with same selector", func() {

		By("creating two WorkloadEndpoints with the same egress selector")
		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key:   we1Key,
				Value: we1Value1,
			},
			UpdateType: api.UpdateTypeKVNew,
		})
		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key:   we2Key,
				Value: we1Value1,
			},
			UpdateType: api.UpdateTypeKVNew,
		})

		// Expect 1 IPSetActive and 2 EgressIPSetIDUpdates.
		ipSetID := cbs.ExpectActive()
		cbs.ExpectEgressUpdate(we1Key, epEgressData{ipSetID: ipSetID})
		cbs.ExpectEgressUpdate(we2Key, epEgressData{ipSetID: ipSetID})
		cbs.ExpectNoMoreCallbacks()

		By("deleting WorkloadEndpoint #1")
		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key:   we1Key,
				Value: nil,
			},
			UpdateType: api.UpdateTypeKVUpdated,
		})

		// Expect EgressUpdate for that endpoint.
		cbs.ExpectEgressUpdate(we1Key, epEgressData{})
		cbs.ExpectNoMoreCallbacks()

		By("deleting WorkloadEndpoint #2")
		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key:   we2Key,
				Value: nil,
			},
			UpdateType: api.UpdateTypeKVUpdated,
		})

		// Expect IPSetInactive for old selector.
		cbs.ExpectEgressUpdate(we2Key, epEgressData{})
		cbs.ExpectInactive(ipSetID)
		cbs.ExpectNoMoreCallbacks()
	})

	It("generates expected callbacks for WorkloadEndpoint with profile", func() {

		By("creating WorkloadEndpoint with profile ID but no egress selector")
		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key: we1Key,
				Value: &model.WorkloadEndpoint{
					Name:       "we1",
					ProfileIDs: []string{"webclient"},
				},
			},
			UpdateType: api.UpdateTypeKVNew,
		})

		cbs.ExpectNoMoreCallbacks()

		By("adding Profile with egress selector")
		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key: model.ResourceKey{Kind: v3.KindProfile, Name: "webclient"},
				Value: &v3.Profile{
					Spec: v3.ProfileSpec{
						EgressGateway: &v3.EgressSpec{
							Selector: "server == 'bump'",
						},
					},
				},
			},
			UpdateType: api.UpdateTypeKVNew,
		})

		// Expect IPSetActive and EgressIPSetIDUpdate.
		ipSetID1 := cbs.ExpectActive()
		cbs.ExpectEgressUpdate(we1Key, epEgressData{ipSetID: ipSetID1})
		cbs.ExpectNoMoreCallbacks()

		By("updating Profile with different selector")
		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key: model.ResourceKey{Kind: v3.KindProfile, Name: "webclient"},
				Value: &v3.Profile{
					Spec: v3.ProfileSpec{
						EgressGateway: &v3.EgressSpec{
							Selector: "server == 'wire'",
						},
					},
				},
			},
			UpdateType: api.UpdateTypeKVUpdated,
		})

		// Expect IPSetInactive for old selector.
		cbs.ExpectInactive(ipSetID1)

		// Expect IPSetActive and EgressIPSetIDUpdate with new ID.
		ipSetID2 := cbs.ExpectActive()
		cbs.ExpectEgressUpdate(we1Key, epEgressData{ipSetID: ipSetID2})
		cbs.ExpectNoMoreCallbacks()
		Expect(ipSetID2).NotTo(Equal(ipSetID1))

		By("updating WorkloadEndpoint with its own egress selector")
		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key: we1Key,
				Value: &model.WorkloadEndpoint{
					Name:           "we1",
					ProfileIDs:     []string{"webclient"},
					EgressSelector: "black == 'red'",
				},
			},
			UpdateType: api.UpdateTypeKVUpdated,
		})

		// Expect IPSetInactive for old selector.
		cbs.ExpectInactive(ipSetID2)

		// Expect IPSetActive and EgressIPSetIDUpdate for new WE selector.
		ipSetID3 := cbs.ExpectActive()
		cbs.ExpectEgressUpdate(we1Key, epEgressData{ipSetID: ipSetID3})
		cbs.ExpectNoMoreCallbacks()
		Expect(ipSetID3).NotTo(Equal(ipSetID1))
		Expect(ipSetID3).NotTo(Equal(ipSetID2))

		By("updating WorkloadEndpoint with no egress selector")
		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key: we1Key,
				Value: &model.WorkloadEndpoint{
					Name:       "we1",
					ProfileIDs: []string{"webclient"},
				},
			},
			UpdateType: api.UpdateTypeKVUpdated,
		})

		// Expect IPSetInactive for old (WE) selector.
		cbs.ExpectInactive(ipSetID3)

		// Expect IPSetActive and EgressIPSetIDUpdate for new (profile) selector.
		Expect(cbs.ExpectActive()).To(Equal(ipSetID2))
		cbs.ExpectEgressUpdate(we1Key, epEgressData{ipSetID: ipSetID2})
		cbs.ExpectNoMoreCallbacks()

		By("updating Profile with no egress selector")
		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key: model.ResourceKey{Kind: v3.KindProfile, Name: "webclient"},
				Value: &v3.Profile{
					Spec: v3.ProfileSpec{},
				},
			},
			UpdateType: api.UpdateTypeKVUpdated,
		})

		// Expect IPSetInactive for old (profile) selector.
		cbs.ExpectInactive(ipSetID2)

		// Expect EgressIPSetIDUpdate with IP set ID "".
		cbs.ExpectEgressUpdate(we1Key, epEgressData{})
		cbs.ExpectNoMoreCallbacks()

		By("deleting the WorkloadEndpoint")
		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key:   we1Key,
				Value: nil,
			},
			UpdateType: api.UpdateTypeKVDeleted,
		})
		cbs.ExpectNoMoreCallbacks()

		By("deleting the Profile")
		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key:   model.ResourceKey{Kind: v3.KindProfile, Name: "webclient"},
				Value: nil,
			},
			UpdateType: api.UpdateTypeKVDeleted,
		})
		cbs.ExpectNoMoreCallbacks()
	})

	It("generates expected callbacks for multiple WorkloadEndpoints and Profiles", func() {

		By("creating 5 WEs with profile A, 5 WEs with profile B")
		for _, profile := range []string{"a", "b"} {
			for i := 0; i < 5; i++ {
				name := fmt.Sprintf("we%v-%v", i, profile)
				aec.OnUpdate(api.Update{
					KVPair: model.KVPair{
						Key: model.WorkloadEndpointKey{WorkloadID: name},
						Value: &model.WorkloadEndpoint{
							Name:       name,
							ProfileIDs: []string{profile},
						},
					},
					UpdateType: api.UpdateTypeKVNew,
				})
			}
		}
		cbs.ExpectNoMoreCallbacks()

		By("creating profile A with egress selector")
		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key: model.ResourceKey{Kind: v3.KindProfile, Name: "a"},
				Value: &v3.Profile{
					Spec: v3.ProfileSpec{
						EgressGateway: &v3.EgressSpec{
							Selector: "server == 'a'",
						},
					},
				},
			},
			UpdateType: api.UpdateTypeKVNew,
		})

		// Expect Active for that selector and EgressIPSetIDUpdate for the 5 using WEs.
		ipSetA := cbs.ExpectActive()
		for i := 0; i < 5; i++ {
			name := fmt.Sprintf("we%v-a", i)
			cbs.ExpectEgressUpdate(model.WorkloadEndpointKey{WorkloadID: name}, epEgressData{ipSetID: ipSetA})
		}
		cbs.ExpectNoMoreCallbacks()

		By("changing profile A’s selector")
		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key: model.ResourceKey{Kind: v3.KindProfile, Name: "a"},
				Value: &v3.Profile{
					Spec: v3.ProfileSpec{
						EgressGateway: &v3.EgressSpec{
							Selector: "server == 'aprime'",
						},
					},
				},
			},
			UpdateType: api.UpdateTypeKVUpdated,
		})

		// Expect Inactive for old, Active for new, EgressIPSetIDUpdate for the 5 using WEs.
		cbs.ExpectInactive(ipSetA)
		ipSetAPrime := cbs.ExpectActive()
		for i := 0; i < 5; i++ {
			name := fmt.Sprintf("we%v-a", i)
			cbs.ExpectEgressUpdate(model.WorkloadEndpointKey{WorkloadID: name}, epEgressData{ipSetID: ipSetAPrime})
		}
		cbs.ExpectNoMoreCallbacks()

		By("creating profile B with different egress selector")
		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key: model.ResourceKey{Kind: v3.KindProfile, Name: "b"},
				Value: &v3.Profile{
					Spec: v3.ProfileSpec{
						EgressGateway: &v3.EgressSpec{
							Selector: "server == 'b'",
						},
					},
				},
			},
			UpdateType: api.UpdateTypeKVNew,
		})

		// Expect Active for that selector and EgressIPSetIDUpdate for the 5 using WEs.
		ipSetB := cbs.ExpectActive()
		for i := 0; i < 5; i++ {
			name := fmt.Sprintf("we%v-b", i)
			cbs.ExpectEgressUpdate(model.WorkloadEndpointKey{WorkloadID: name}, epEgressData{ipSetID: ipSetB})
		}
		cbs.ExpectNoMoreCallbacks()

		By("deleting profile A")
		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key:   model.ResourceKey{Kind: v3.KindProfile, Name: "a"},
				Value: nil,
			},
			UpdateType: api.UpdateTypeKVDeleted,
		})

		// Expect Inactive for its selector and EgressIPSetIDUpdate ““ for the 5 using WEs.
		cbs.ExpectInactive(ipSetAPrime)
		for i := 0; i < 5; i++ {
			name := fmt.Sprintf("we%v-a", i)
			cbs.ExpectEgressUpdate(model.WorkloadEndpointKey{WorkloadID: name}, epEgressData{})
		}
		cbs.ExpectNoMoreCallbacks()

		By("deleting the endpoints using profile A")
		for i := 0; i < 5; i++ {
			name := fmt.Sprintf("we%v-a", i)
			aec.OnUpdate(api.Update{
				KVPair: model.KVPair{
					Key:   model.WorkloadEndpointKey{WorkloadID: name},
					Value: nil,
				},
				UpdateType: api.UpdateTypeKVDeleted,
			})
		}
		cbs.ExpectNoMoreCallbacks()

	})

	It("ignores unexpected update", func() {
		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key: model.ResourceKey{Kind: libapiv3.KindNode, Name: "a"},
				Value: &libapiv3.Node{
					Spec: libapiv3.NodeSpec{},
				},
			},
			UpdateType: api.UpdateTypeKVUpdated,
		})
		cbs.ExpectNoMoreCallbacks()

		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key: model.HostConfigKey{
					Hostname: "myhost",
					Name:     "IPv4VXLANTunnelAddr",
				},
				Value: "10.0.0.0",
			},
			UpdateType: api.UpdateTypeKVUpdated,
		})
		cbs.ExpectNoMoreCallbacks()
	})

	It("handles when profile is defined before endpoint", func() {

		By("defining profile A with egress selector")
		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key: model.ResourceKey{Kind: v3.KindProfile, Name: "a"},
				Value: &v3.Profile{
					Spec: v3.ProfileSpec{
						EgressGateway: &v3.EgressSpec{
							Selector: "server == 'a'",
						},
					},
				},
			},
			UpdateType: api.UpdateTypeKVNew,
		})
		cbs.ExpectNoMoreCallbacks()

		By("defining an endpoint that uses that profile")
		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key: model.WorkloadEndpointKey{WorkloadID: "we1"},
				Value: &model.WorkloadEndpoint{
					Name:       "we1",
					ProfileIDs: []string{"a"},
				},
			},
			UpdateType: api.UpdateTypeKVNew,
		})
		ipSetID := cbs.ExpectActive()
		cbs.ExpectEgressUpdate(model.WorkloadEndpointKey{WorkloadID: "we1"}, epEgressData{ipSetID: ipSetID})
		cbs.ExpectNoMoreCallbacks()

		By("updating profile with same selector")
		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key: model.ResourceKey{Kind: v3.KindProfile, Name: "a"},
				Value: &v3.Profile{
					Spec: v3.ProfileSpec{
						EgressGateway: &v3.EgressSpec{
							Selector: "server == 'a'",
						},
						LabelsToApply: map[string]string{"a": "b"},
					},
				},
			},
			UpdateType: api.UpdateTypeKVUpdated,
		})
		cbs.ExpectNoMoreCallbacks()

	})

	It("handles when WorkloadEndpoint and profile both specify selectors", func() {

		By("creating WorkloadEndpoint with profile ID and egress selector")
		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key: we1Key,
				Value: &model.WorkloadEndpoint{
					Name:           "we1",
					ProfileIDs:     []string{"webclient"},
					EgressSelector: "black == 'red'",
				},
			},
			UpdateType: api.UpdateTypeKVNew,
		})

		ipSetWE := cbs.ExpectActive()
		cbs.ExpectEgressUpdate(we1Key, epEgressData{ipSetID: ipSetWE})
		cbs.ExpectNoMoreCallbacks()

		By("adding Profile with egress selector")
		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key: model.ResourceKey{Kind: v3.KindProfile, Name: "webclient"},
				Value: &v3.Profile{
					Spec: v3.ProfileSpec{
						EgressGateway: &v3.EgressSpec{
							Selector: "server == 'bump'",
						},
					},
				},
			},
			UpdateType: api.UpdateTypeKVNew,
		})

		// Expect no change.
		cbs.ExpectNoMoreCallbacks()

		By("updating Profile with different selector")
		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key: model.ResourceKey{Kind: v3.KindProfile, Name: "webclient"},
				Value: &v3.Profile{
					Spec: v3.ProfileSpec{
						EgressGateway: &v3.EgressSpec{
							Selector: "server == 'wire'",
						},
					},
				},
			},
			UpdateType: api.UpdateTypeKVUpdated,
		})

		// Expect no change.
		cbs.ExpectNoMoreCallbacks()

		By("defining 2nd WorkloadEndpoint with no selector and different profile")
		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key: we2Key,
				Value: &model.WorkloadEndpoint{
					Name:       "we2",
					ProfileIDs: []string{"other"},
				},
			},
			UpdateType: api.UpdateTypeKVNew,
		})

		// Expect no callbacks.
		cbs.ExpectNoMoreCallbacks()

		By("changing first profile not to have egress selector")
		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key: model.ResourceKey{Kind: v3.KindProfile, Name: "webclient"},
				Value: &v3.Profile{
					Spec: v3.ProfileSpec{},
				},
			},
			UpdateType: api.UpdateTypeKVUpdated,
		})

		// Expect no change.
		cbs.ExpectNoMoreCallbacks()
	})

	It("generates expected callbacks for two WorkloadEndpoints with different but equivalent selector", func() {

		we1Value := &model.WorkloadEndpoint{
			Name:           "we1",
			EgressSelector: `black == "red"`,
		}
		we2Value := &model.WorkloadEndpoint{
			Name:           "we1",
			EgressSelector: "black == 'red'",
		}

		By("creating two WorkloadEndpoints with similar egress selector")
		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key:   we1Key,
				Value: we1Value,
			},
			UpdateType: api.UpdateTypeKVNew,
		})
		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key:   we2Key,
				Value: we2Value,
			},
			UpdateType: api.UpdateTypeKVNew,
		})

		// Expect 1 IPSetActive and 2 EgressIPSetIDUpdates.
		ipSetID := cbs.ExpectActive()
		cbs.ExpectEgressUpdate(we1Key, epEgressData{ipSetID: ipSetID})
		cbs.ExpectEgressUpdate(we2Key, epEgressData{ipSetID: ipSetID})
		cbs.ExpectNoMoreCallbacks()

		By("deleting WorkloadEndpoint #1")
		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key:   we1Key,
				Value: nil,
			},
			UpdateType: api.UpdateTypeKVUpdated,
		})

		// Expect EgressUpdate for that endpoint.
		cbs.ExpectEgressUpdate(we1Key, epEgressData{})
		cbs.ExpectNoMoreCallbacks()

		By("deleting WorkloadEndpoint #2")
		aec.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key:   we2Key,
				Value: nil,
			},
			UpdateType: api.UpdateTypeKVUpdated,
		})

		// Expect IPSetInactive for old selector.
		cbs.ExpectEgressUpdate(we2Key, epEgressData{})
		cbs.ExpectInactive(ipSetID)
		cbs.ExpectNoMoreCallbacks()
	})
})

type testCallbacks struct {
	activeCalls      []*IPSetData
	inactiveCalls    []*IPSetData
	egressUpdateKeys []model.WorkloadEndpointKey
	egressDatas      []epEgressData
}

func (tc *testCallbacks) OnIPSetActive(ipSet *IPSetData) {
	tc.activeCalls = append(tc.activeCalls, ipSet)
}

func (tc *testCallbacks) OnIPSetInactive(ipSet *IPSetData) {
	tc.inactiveCalls = append(tc.inactiveCalls, ipSet)
}

func (tc *testCallbacks) OnEndpointEgressDataUpdate(key model.WorkloadEndpointKey, egressData epEgressData) {
	tc.egressUpdateKeys = append(tc.egressUpdateKeys, key)
	tc.egressDatas = append(tc.egressDatas, egressData)
}

func (tc *testCallbacks) ExpectActive() string {
	ExpectWithOffset(1, len(tc.activeCalls)).To(BeNumerically(">=", 1), "Expected OnIPSetActive call")
	ExpectWithOffset(1, tc.activeCalls[0].IsEgressSelector).To(BeTrue(), "Expected IP set for an egress selector")
	ipSetID := tc.activeCalls[0].cachedUID
	ExpectWithOffset(1, ipSetID).To(HavePrefix("e:"))
	tc.activeCalls = tc.activeCalls[1:]
	return ipSetID
}

func (tc *testCallbacks) ExpectInactive(id string) {
	ExpectWithOffset(1, len(tc.inactiveCalls)).To(BeNumerically(">=", 1), "Expected OnIPSetInactive call")
	ExpectWithOffset(1, tc.inactiveCalls[0].IsEgressSelector).To(BeTrue(), "Expected IP set for an egress selector")
	ExpectWithOffset(1, tc.inactiveCalls[0].cachedUID).To(Equal(id))
	tc.inactiveCalls = tc.inactiveCalls[1:]
}

func (tc *testCallbacks) ExpectEgressUpdate(key model.WorkloadEndpointKey, egressData epEgressData) {
	ExpectWithOffset(1, tc.egressUpdateKeys).To(ContainElement(key), "Expected OnEndpointEgressDataUpdate call")
	keyPos := -1
	for i, uk := range tc.egressUpdateKeys {
		if uk == key {
			ExpectWithOffset(1, tc.egressDatas[i]).To(Equal(egressData))
			keyPos = i
			break
		}
	}
	Expect(keyPos).NotTo(Equal(-1))
	tc.egressUpdateKeys = append(tc.egressUpdateKeys[:keyPos], tc.egressUpdateKeys[keyPos+1:]...)
	tc.egressDatas = append(tc.egressDatas[:keyPos], tc.egressDatas[keyPos+1:]...)
}

func (tc *testCallbacks) ExpectNoMoreCallbacks() {
	ExpectWithOffset(1, len(tc.activeCalls)).To(BeZero(), "Expected no more OnIPSetActive calls")
	ExpectWithOffset(1, len(tc.inactiveCalls)).To(BeZero(), "Expected no more OnIPSetInactive calls")
	ExpectWithOffset(1, len(tc.egressUpdateKeys)).To(BeZero(), "Expected no more OnEndpointEgressDataUpdate calls")
	ExpectWithOffset(1, len(tc.egressDatas)).To(BeZero(), "Expected no more OnEndpointEgressDataUpdate calls")
}
