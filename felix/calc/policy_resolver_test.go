// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package calc

import (
	"reflect"
	"testing"

	"github.com/projectcalico/calico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
)

func TestPolicyResolver_OnUpdate_HandleEgressIPSetID(t *testing.T) {
	uut := NewPolicyResolver()
	cbs := &prCallbacks{}
	uut.RegisterCallback(cbs)

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
	cbs.ExpectEndpointTierUpdate(t, we1Key, "")

	uut.OnEndpointEgressDataUpdate(we1Key, epEgressData{ipSetID: "e:abcdef"})

	// Expect OnEndpointTierUpdate with that egress IP set ID.
	cbs.ExpectEndpointTierUpdate(t, we1Key, "e:abcdef")

	uut.OnUpdate(api.Update{
		KVPair: model.KVPair{
			Key: we1Key,
			Value: &model.WorkloadEndpoint{
				Name: "we1",
			},
		},
		UpdateType: api.UpdateTypeKVUpdated,
	})
	cbs.ExpectEndpointTierUpdate(t, we1Key, "e:abcdef")

	uut.OnEndpointEgressDataUpdate(we1Key, epEgressData{})
	cbs.ExpectEndpointTierUpdate(t, we1Key, "")
}

func (tc *prCallbacks) ExpectEndpointTierUpdate(t *testing.T, key model.WorkloadEndpointKey, egressIPSetID string) {
	if len(tc.keys) < 1 {
		t.Error("Expected at least 1 key")
	}

	if len(tc.egressIPSetIDs) < 1 {
		t.Error("Expected at least 1 egressIPSetID")
	}

	if !reflect.DeepEqual(tc.keys[0].(model.WorkloadEndpointKey), key) {
		t.Errorf("Keys - Expected %v \n but got \n %v", key, tc.keys[0].(model.WorkloadEndpointKey))
	}

	if !reflect.DeepEqual(tc.egressIPSetIDs[0], egressIPSetID) {
		t.Errorf("EgressIPSetIDs - Expected %v \n but got \n %v", egressIPSetID, tc.egressIPSetIDs[0])
	}

	tc.keys = tc.keys[1:]
	tc.egressIPSetIDs = tc.egressIPSetIDs[1:]
}

func (tc *prCallbacks) ExpectNoMoreCallbacks(t *testing.T) {
	if len(tc.keys) != 0 {
		t.Error("Expected length of keys to be 0")
	}

	if len(tc.egressIPSetIDs) != 0 {
		t.Error("Expected length of egressIPSetIDs to be 0")
	}
}

type prCallbacks struct {
	keys           []model.Key
	egressIPSetIDs []string
}

func (tc *prCallbacks) OnEndpointTierUpdate(endpointKey model.Key, _ interface{}, egressData EndpointEgressData, _ []TierInfo) {
	tc.keys = append(tc.keys, endpointKey)
	tc.egressIPSetIDs = append(tc.egressIPSetIDs, egressData.EgressIPSetID)
}

func TestPolicyResolver_OnPolicyMatch(t *testing.T) {
	pr := NewPolicyResolver()

	polKey := model.PolicyKey{
		Name: "test-policy",
	}

	pol := model.Policy{}

	endpointKey := model.WorkloadEndpointKey{
		Hostname: "test-workload-ep",
	}

	pr.allPolicies[polKey] = &pol
	pr.OnPolicyMatch(polKey, endpointKey)

	if !pr.policyIDToEndpointIDs.ContainsKey(polKey) {
		t.Error("Adding new policy - expected PolicyIDToEndpointIDs to contain new policy but it does not")
	}
	if !pr.endpointIDToPolicyIDs.ContainsKey(endpointKey) {
		t.Error("Adding new policy - expected EndpointIDToPolicyIDs to contain endpoint but it does not")
	}
	if !pr.dirtyEndpoints.Contains(endpointKey) {
		t.Error("Adding new policy - expected DirtyEndpoints to contain endpoint for policy but it does not")
	}

	pr.OnPolicyMatch(polKey, endpointKey)
}

func TestPolicyResolver_OnPolicyMatchStopped(t *testing.T) {
	pr := NewPolicyResolver()

	polKey := model.PolicyKey{
		Name: "test-policy",
	}

	pol := model.Policy{}

	endpointKey := model.WorkloadEndpointKey{
		Hostname: "test-workload-ep",
	}

	pr.policyIDToEndpointIDs.Put(polKey, endpointKey)
	pr.endpointIDToPolicyIDs.Put(endpointKey, polKey)
	pr.policySorter.UpdatePolicy(polKey, &pol)
	pr.OnPolicyMatchStopped(polKey, endpointKey)

	if pr.policyIDToEndpointIDs.ContainsKey(polKey) {
		t.Error("Deleting existing policy - expected PolicyIDToEndpointIDs not to contain policy but it does")
	}
	if pr.endpointIDToPolicyIDs.ContainsKey(endpointKey) {
		t.Error("Deleting existing policy - expected EndpointIDToPolicyIDs not to contain endpoint but it does")
	}
	if !pr.dirtyEndpoints.Contains(endpointKey) {
		t.Error("Deleting existing policy - expected DirtyEndpoints to contain endpoint but it does not")
	}

	pr.OnPolicyMatchStopped(polKey, endpointKey)
}

func TestPolicyResolver_OnUpdate_Basic(t *testing.T) {
	pr := NewPolicyResolver()

	polKey := model.PolicyKey{
		Name: "test-policy",
	}

	policy := model.Policy{}

	kvp := model.KVPair{
		Key:   polKey,
		Value: &policy,
	}
	update := api.Update{}
	update.Key = kvp.Key
	update.Value = kvp.Value

	pr.OnUpdate(update)

	if _, found := pr.allPolicies[polKey]; !found {
		t.Error("Adding new inactive policy - expected policy to be in AllPolicies but it is not")
	}

	update.Value = nil

	pr.OnUpdate(update)

	if _, found := pr.allPolicies[polKey]; found {
		t.Error("Deleting inactive policy - expected AllPolicies not to contain policy but it does")
	}
}
