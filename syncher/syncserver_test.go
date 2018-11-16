// Copyright (c) 2018 Tigera, Inc. All rights reserved.

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
package syncher

import (
	"context"
	"errors"
	"io/ioutil"
	"net"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/projectcalico/app-policy/policystore"
	"github.com/projectcalico/app-policy/proto"
	"github.com/projectcalico/app-policy/statscache"
	"github.com/projectcalico/app-policy/uds"

	envoyapi "github.com/envoyproxy/data-plane-api/envoy/api/v2/core"
	"google.golang.org/grpc"
)

const addr1Ip = "3.4.6.8"
const addr2Ip = "23.8.58.1"
const addr3Ip = "2.2.2.2"

var addr1 = &envoyapi.Address{
	Address: &envoyapi.Address_SocketAddress{SocketAddress: &envoyapi.SocketAddress{
		Address:  addr1Ip,
		Protocol: envoyapi.TCP,
		PortSpecifier: &envoyapi.SocketAddress_PortValue{
			PortValue: 5429,
		},
	}},
}
var addr2 = &envoyapi.Address{
	Address: &envoyapi.Address_SocketAddress{SocketAddress: &envoyapi.SocketAddress{
		Address:  addr2Ip,
		Protocol: envoyapi.TCP,
		PortSpecifier: &envoyapi.SocketAddress_PortValue{
			PortValue: 6632,
		},
	}},
}
var addr3 = &envoyapi.Address{
	Address: &envoyapi.Address_SocketAddress{SocketAddress: &envoyapi.SocketAddress{
		Address:  addr3Ip,
		Protocol: envoyapi.TCP,
		PortSpecifier: &envoyapi.SocketAddress_PortValue{
			PortValue: 2222,
		},
	}},
}
var profile1 = &proto.Profile{
	InboundRules: []*proto.Rule{
		{
			Action:      "allow",
			SrcIpSetIds: []string{"ipset1", "ipset6"},
		},
	},
}
var profile2 = &proto.Profile{
	OutboundRules: []*proto.Rule{
		{
			Action:      "allow",
			DstIpSetIds: []string{"ipset1", "ipset6"},
		},
	},
}
var policy1 = &proto.Policy{
	InboundRules: []*proto.Rule{
		{
			Action:      "allow",
			SrcIpSetIds: []string{"ipset1", "ipset6"},
		},
	},
}
var policy2 = &proto.Policy{
	OutboundRules: []*proto.Rule{
		{
			Action:      "allow",
			DstIpSetIds: []string{"ipset1", "ipset6"},
		},
	},
}
var endpoint1 = &proto.WorkloadEndpoint{
	Name:       "wep",
	ProfileIds: []string{"profile1", "profile2"},
}
var serviceAccount1 = &proto.ServiceAccountUpdate{
	Id:     &proto.ServiceAccountID{Name: "serviceAccount1", Namespace: "test"},
	Labels: map[string]string{"k1": "v1", "k2": "v2"},
}
var namespace1 = &proto.NamespaceUpdate{
	Id:     &proto.NamespaceID{Name: "namespace1"},
	Labels: map[string]string{"k1": "v1", "k2": "v2"},
}

// IPSetUpdate with a new ID
func TestIPSetUpdateNew(t *testing.T) {
	RegisterTestingT(t)

	id := "test_id"
	store := policystore.NewPolicyStore()
	update := &proto.IPSetUpdate{
		Id:   id,
		Type: proto.IPSetUpdate_IP,
		Members: []string{
			addr1Ip,
			addr2Ip,
		},
	}
	processIPSetUpdate(store, update)
	ipset := store.IPSetByID[id]
	Expect(ipset).ToNot(BeNil())
	Expect(ipset.ContainsAddress(addr1)).To(BeTrue())
	Expect(ipset.ContainsAddress(addr2)).To(BeTrue())
}

// IPSetUpdate with existing ID
func TestIPSetUpdateExists(t *testing.T) {
	RegisterTestingT(t)

	id := "test_id"
	store := policystore.NewPolicyStore()
	ipset := policystore.NewIPSet(proto.IPSetUpdate_IP)
	store.IPSetByID[id] = ipset
	ipset.AddString(addr1Ip)
	ipset.AddString(addr3Ip)

	update := &proto.IPSetUpdate{
		Id:   id,
		Type: proto.IPSetUpdate_IP,
		Members: []string{
			addr1Ip,
			addr2Ip,
		},
	}
	processIPSetUpdate(store, update)
	ipset = store.IPSetByID[id]

	// The update should replace existing set, so we don't expect 2.2.2.2 (addr3) to still be
	Expect(ipset.ContainsAddress(addr1)).To(BeTrue())
	Expect(ipset.ContainsAddress(addr2)).To(BeTrue())
	Expect(ipset.ContainsAddress(addr3)).To(BeFalse())
}

// processUpdate handles IPSetUpdate without a crash.
func TestIPSetUpdateDispatch(t *testing.T) {
	RegisterTestingT(t)

	id := "test_id"
	store := policystore.NewPolicyStore()
	inSync := make(chan struct{})
	update := &proto.ToDataplane{
		Payload: &proto.ToDataplane_IpsetUpdate{IpsetUpdate: &proto.IPSetUpdate{
			Id:   id,
			Type: proto.IPSetUpdate_IP,
			Members: []string{
				addr1Ip,
				addr2Ip,
			}}}}
	Expect(func() { processUpdate(store, inSync, update) }).ToNot(Panic())
}

// IPSetDeltaUpdate with existing ID.
func TestIPSetDeltaUpdateExists(t *testing.T) {
	RegisterTestingT(t)

	id := "test_id"
	store := policystore.NewPolicyStore()
	ipset := policystore.NewIPSet(proto.IPSetUpdate_IP)
	store.IPSetByID[id] = ipset
	ipset.AddString(addr1Ip)
	ipset.AddString(addr3Ip)

	update := &proto.IPSetDeltaUpdate{
		Id: id,
		AddedMembers: []string{
			addr2Ip,
		},
		RemovedMembers: []string{addr3Ip},
	}
	processIPSetDeltaUpdate(store, update)
	ipset = store.IPSetByID[id] // don't assume set pointer doesn't change

	Expect(ipset.ContainsAddress(addr1)).To(BeTrue())
	Expect(ipset.ContainsAddress(addr2)).To(BeTrue())
	Expect(ipset.ContainsAddress(addr3)).To(BeFalse())
}

// IPSetDeltaUpdate with an unknown ID results in a panic.
func TestIPSetDeltaUpdateNonExist(t *testing.T) {
	RegisterTestingT(t)

	id := "test_id"
	store := policystore.NewPolicyStore()

	update := &proto.IPSetDeltaUpdate{
		Id: id,
		AddedMembers: []string{
			addr2Ip,
		},
		RemovedMembers: []string{addr3Ip},
	}
	Expect(func() { processIPSetDeltaUpdate(store, update) }).To(Panic())
}

// processUpdate handles a valid IPSetDeltaUpdate without a panic
func TestIPSetDeltaUpdateDispatch(t *testing.T) {
	RegisterTestingT(t)

	id := "test_id"
	store := policystore.NewPolicyStore()
	ipset := policystore.NewIPSet(proto.IPSetUpdate_IP)
	store.IPSetByID[id] = ipset
	inSync := make(chan struct{})

	update := &proto.ToDataplane{Payload: &proto.ToDataplane_IpsetDeltaUpdate{
		IpsetDeltaUpdate: &proto.IPSetDeltaUpdate{
			Id: id,
			AddedMembers: []string{
				addr2Ip,
			},
			RemovedMembers: []string{addr3Ip},
		},
	}}
	Expect(func() { processUpdate(store, inSync, update) }).ToNot(Panic())
}

// IPSetRemove with an existing ID.
func TestIPSetRemoveExist(t *testing.T) {
	RegisterTestingT(t)

	id := "test_id"
	store := policystore.NewPolicyStore()
	ipset := policystore.NewIPSet(proto.IPSetUpdate_IP)
	store.IPSetByID[id] = ipset

	update := &proto.IPSetRemove{Id: id}
	processIPSetRemove(store, update)
	Expect(store.IPSetByID[id]).To(BeNil())
}

// IPSetRemove with an unknown ID is handled
func TestIPSetRemoveNonExist(t *testing.T) {
	RegisterTestingT(t)

	id := "test_id"
	store := policystore.NewPolicyStore()

	update := &proto.IPSetRemove{Id: id}
	processIPSetRemove(store, update)
	Expect(store.IPSetByID[id]).To(BeNil())
}

// processUpdate with IPSetRemove
func TestIPSetRemoveDispatch(t *testing.T) {
	RegisterTestingT(t)

	id := "test_id"
	store := policystore.NewPolicyStore()
	ipset := policystore.NewIPSet(proto.IPSetUpdate_IP)
	store.IPSetByID[id] = ipset
	inSync := make(chan struct{})

	update := &proto.ToDataplane{Payload: &proto.ToDataplane_IpsetRemove{
		IpsetRemove: &proto.IPSetRemove{Id: id},
	}}
	Expect(func() { processUpdate(store, inSync, update) }).ToNot(Panic())
}

// ActiveProfileUpdate with a new id
func TestActiveProfileUpdateNonExist(t *testing.T) {
	RegisterTestingT(t)

	id := proto.ProfileID{Name: "test_id"}
	store := policystore.NewPolicyStore()

	update := &proto.ActiveProfileUpdate{
		Id:      &id,
		Profile: profile1,
	}
	processActiveProfileUpdate(store, update)
	Expect(store.ProfileByID[id]).To(BeIdenticalTo(profile1))
}

// ActiveProfileUpdate with an existing ID
func TestActiveProfileUpdateExist(t *testing.T) {
	RegisterTestingT(t)

	id := proto.ProfileID{Name: "test_id"}
	store := policystore.NewPolicyStore()
	store.ProfileByID[id] = profile2

	update := &proto.ActiveProfileUpdate{
		Id:      &id,
		Profile: profile1,
	}
	processActiveProfileUpdate(store, update)
	Expect(store.ProfileByID[id]).To(BeIdenticalTo(profile1))
}

// ActiveProfileUpdate without an ID results in panic
func TestActiveProfileUpdateNilId(t *testing.T) {
	RegisterTestingT(t)

	store := policystore.NewPolicyStore()

	update := &proto.ActiveProfileUpdate{
		Profile: profile1,
	}
	Expect(func() { processActiveProfileUpdate(store, update) }).To(Panic())
}

// processUpdate with ActiveProfileUpdate
func TestActiveProfileUpdateDispatch(t *testing.T) {
	RegisterTestingT(t)

	id := proto.ProfileID{Name: "test_id"}
	store := policystore.NewPolicyStore()
	inSync := make(chan struct{})

	update := &proto.ToDataplane{Payload: &proto.ToDataplane_ActiveProfileUpdate{
		ActiveProfileUpdate: &proto.ActiveProfileUpdate{
			Id:      &id,
			Profile: profile1,
		},
	}}
	Expect(func() { processUpdate(store, inSync, update) }).ToNot(Panic())
}

// ActiveProfileRemove with an unkown id is handled without panic.
func TestActiveProfileRemoveNonExist(t *testing.T) {
	RegisterTestingT(t)

	id := proto.ProfileID{Name: "test_id"}
	store := policystore.NewPolicyStore()

	update := &proto.ActiveProfileRemove{Id: &id}
	processActiveProfileRemove(store, update)
	Expect(store.ProfileByID[id]).To(BeNil())
}

// ActiveProfileRemove with existing id
func TestActiveProfileRemoveExist(t *testing.T) {
	RegisterTestingT(t)

	id := proto.ProfileID{Name: "test_id"}
	store := policystore.NewPolicyStore()
	store.ProfileByID[id] = profile1

	update := &proto.ActiveProfileRemove{Id: &id}
	processActiveProfileRemove(store, update)
	Expect(store.ProfileByID[id]).To(BeNil())
}

// ActiveProfileRemove without an ID results in panic.
func TestActiveProfileRemoveNilId(t *testing.T) {
	RegisterTestingT(t)

	store := policystore.NewPolicyStore()

	update := &proto.ActiveProfileRemove{}
	Expect(func() { processActiveProfileRemove(store, update) }).To(Panic())
}

// processUpdate handles ActiveProfileRemove
func TestActiveProfileRemoveDispatch(t *testing.T) {
	RegisterTestingT(t)

	id := proto.ProfileID{Name: "test_id"}
	store := policystore.NewPolicyStore()
	inSync := make(chan struct{})

	update := &proto.ToDataplane{Payload: &proto.ToDataplane_ActiveProfileRemove{
		ActiveProfileRemove: &proto.ActiveProfileRemove{Id: &id},
	}}
	Expect(func() { processUpdate(store, inSync, update) }).ToNot(Panic())
}

// ActivePolicyUpdate for a new id
func TestActivePolicyUpdateNonExist(t *testing.T) {
	RegisterTestingT(t)

	id := proto.PolicyID{Tier: "test_tier", Name: "test_id"}
	store := policystore.NewPolicyStore()

	update := &proto.ActivePolicyUpdate{
		Id:     &id,
		Policy: policy1,
	}
	processActivePolicyUpdate(store, update)
	Expect(store.PolicyByID[id]).To(BeIdenticalTo(policy1))
}

// ActivePolicyUpdate for an existing id
func TestActivePolicyUpdateExist(t *testing.T) {
	RegisterTestingT(t)

	id := proto.PolicyID{Tier: "test_tier", Name: "test_id"}
	store := policystore.NewPolicyStore()
	store.PolicyByID[id] = policy2

	update := &proto.ActivePolicyUpdate{
		Id:     &id,
		Policy: policy1,
	}
	processActivePolicyUpdate(store, update)
	Expect(store.PolicyByID[id]).To(BeIdenticalTo(policy1))
}

// ActivePolicyUpdate without an id causes a panic
func TestActivePolicyUpdateNilId(t *testing.T) {
	RegisterTestingT(t)

	store := policystore.NewPolicyStore()

	update := &proto.ActivePolicyUpdate{
		Policy: policy1,
	}
	Expect(func() { processActivePolicyUpdate(store, update) }).To(Panic())
}

// processUpdate handles ActivePolicyDispatch
func TestActivePolicyUpdateDispatch(t *testing.T) {
	RegisterTestingT(t)

	id := proto.PolicyID{Tier: "test_tier", Name: "test_id"}
	store := policystore.NewPolicyStore()
	inSync := make(chan struct{})

	update := &proto.ToDataplane{Payload: &proto.ToDataplane_ActivePolicyUpdate{
		ActivePolicyUpdate: &proto.ActivePolicyUpdate{
			Id:     &id,
			Policy: policy1,
		},
	}}
	Expect(func() { processUpdate(store, inSync, update) }).ToNot(Panic())
}

// ActivePolicyRemove with unknown id is handled
func TestActivePolicyRemoveNonExist(t *testing.T) {
	RegisterTestingT(t)

	id := proto.PolicyID{Tier: "test_tier", Name: "test_id"}
	store := policystore.NewPolicyStore()

	update := &proto.ActivePolicyRemove{Id: &id}
	processActivePolicyRemove(store, update)
	Expect(store.PolicyByID[id]).To(BeNil())
}

// ActivePolicyRemove with existing id
func TestActivePolicyRemoveExist(t *testing.T) {
	RegisterTestingT(t)

	id := proto.PolicyID{Tier: "test_tier", Name: "test_id"}
	store := policystore.NewPolicyStore()
	store.PolicyByID[id] = policy1

	update := &proto.ActivePolicyRemove{Id: &id}
	processActivePolicyRemove(store, update)
	Expect(store.PolicyByID[id]).To(BeNil())
}

// ActivePolicyRemove without an id causes a panic
func TestActivePolicyRemoveNilId(t *testing.T) {
	RegisterTestingT(t)

	store := policystore.NewPolicyStore()

	update := &proto.ActivePolicyRemove{}
	Expect(func() { processActivePolicyRemove(store, update) }).To(Panic())
}

// processUpdate handles ActivePolicyRemove
func TestActivePolicyRemoveDispatch(t *testing.T) {
	RegisterTestingT(t)

	id := proto.PolicyID{Tier: "test_tier", Name: "test_id"}
	store := policystore.NewPolicyStore()
	inSync := make(chan struct{})

	update := &proto.ToDataplane{Payload: &proto.ToDataplane_ActivePolicyRemove{
		ActivePolicyRemove: &proto.ActivePolicyRemove{Id: &id},
	}}
	Expect(func() { processUpdate(store, inSync, update) }).ToNot(Panic())
}

// WorkloadEndpointUpdate sets the endpoint
func TestWorkloadEndpointUpdate(t *testing.T) {
	RegisterTestingT(t)

	store := policystore.NewPolicyStore()

	update := &proto.WorkloadEndpointUpdate{Endpoint: endpoint1}
	processWorkloadEndpointUpdate(store, update)
	Expect(store.Endpoint).To(BeIdenticalTo(endpoint1))
}

// processUpdate handles WorkloadEndpointUpdate
func TestWorkloadEndpointUpdateDispatch(t *testing.T) {
	RegisterTestingT(t)

	store := policystore.NewPolicyStore()
	inSync := make(chan struct{})

	update := &proto.ToDataplane{Payload: &proto.ToDataplane_WorkloadEndpointUpdate{
		WorkloadEndpointUpdate: &proto.WorkloadEndpointUpdate{Endpoint: endpoint1},
	}}
	Expect(func() { processUpdate(store, inSync, update) }).ToNot(Panic())
}

// WorkloadEndpointRemove removes the endpoint
func TestWorkloadEndpointRemove(t *testing.T) {
	RegisterTestingT(t)

	store := policystore.NewPolicyStore()
	store.Endpoint = endpoint1

	update := &proto.WorkloadEndpointRemove{}
	processWorkloadEndpointRemove(store, update)
	Expect(store.Endpoint).To(BeNil())
}

// processUpdate handles WorkloadEndpointRemove
func TestWorkloadEndpointRemoveDispatch(t *testing.T) {
	RegisterTestingT(t)

	store := policystore.NewPolicyStore()
	store.Endpoint = endpoint1
	inSync := make(chan struct{})

	update := &proto.ToDataplane{Payload: &proto.ToDataplane_WorkloadEndpointRemove{
		WorkloadEndpointRemove: &proto.WorkloadEndpointRemove{},
	}}
	Expect(func() { processUpdate(store, inSync, update) }).ToNot(Panic())
}

func TestServiceAccountUpdateDispatch(t *testing.T) {
	RegisterTestingT(t)
	store := policystore.NewPolicyStore()
	inSync := make(chan struct{})

	update := &proto.ToDataplane{Payload: &proto.ToDataplane_ServiceAccountUpdate{serviceAccount1}}

	Expect(func() { processUpdate(store, inSync, update) }).ToNot(Panic())
	Expect(store.ServiceAccountByID).To(Equal(map[proto.ServiceAccountID]*proto.ServiceAccountUpdate{
		*serviceAccount1.Id: serviceAccount1,
	}))
}

func TestServiceAccountUpdateNilId(t *testing.T) {
	RegisterTestingT(t)
	store := policystore.NewPolicyStore()

	Expect(func() { processServiceAccountUpdate(store, &proto.ServiceAccountUpdate{}) }).To(Panic())
}

func TestServiceAccountRemoveDispatch(t *testing.T) {
	RegisterTestingT(t)
	store := policystore.NewPolicyStore()
	store.ServiceAccountByID[*serviceAccount1.Id] = serviceAccount1
	inSync := make(chan struct{})

	remove := &proto.ToDataplane{Payload: &proto.ToDataplane_ServiceAccountRemove{
		ServiceAccountRemove: &proto.ServiceAccountRemove{Id: serviceAccount1.Id}}}
	Expect(func() { processUpdate(store, inSync, remove) }).ToNot(Panic())
	Expect(store.ServiceAccountByID).To(Equal(map[proto.ServiceAccountID]*proto.ServiceAccountUpdate{}))
}

func TestServiceAccountRemoveNilId(t *testing.T) {
	RegisterTestingT(t)
	store := policystore.NewPolicyStore()

	Expect(func() { processServiceAccountRemove(store, &proto.ServiceAccountRemove{}) }).To(Panic())
}

func TestNamespaceUpdateDispatch(t *testing.T) {
	RegisterTestingT(t)
	store := policystore.NewPolicyStore()
	inSync := make(chan struct{})

	update := &proto.ToDataplane{Payload: &proto.ToDataplane_NamespaceUpdate{namespace1}}
	Expect(func() { processUpdate(store, inSync, update) }).ToNot(Panic())
	Expect(store.NamespaceByID).To(Equal(map[proto.NamespaceID]*proto.NamespaceUpdate{
		*namespace1.Id: namespace1,
	}))
}

func TestNamespaceUpdateNilId(t *testing.T) {
	RegisterTestingT(t)
	store := policystore.NewPolicyStore()

	Expect(func() { processNamespaceUpdate(store, &proto.NamespaceUpdate{}) }).To(Panic())
}

func TestNamespaceRemoveDispatch(t *testing.T) {
	RegisterTestingT(t)
	store := policystore.NewPolicyStore()
	store.NamespaceByID[*namespace1.Id] = namespace1
	inSync := make(chan struct{})

	remove := &proto.ToDataplane{Payload: &proto.ToDataplane_NamespaceRemove{
		NamespaceRemove: &proto.NamespaceRemove{Id: namespace1.Id}}}
	Expect(func() { processUpdate(store, inSync, remove) }).ToNot(Panic())
	Expect(store.NamespaceByID).To(Equal(map[proto.NamespaceID]*proto.NamespaceUpdate{}))
}

func TestNamespaceRemoveNilId(t *testing.T) {
	RegisterTestingT(t)
	store := policystore.NewPolicyStore()

	Expect(func() { processNamespaceRemove(store, &proto.NamespaceRemove{}) }).To(Panic())
}

// processUpdate handles InSync
func TestInSyncDispatch(t *testing.T) {
	RegisterTestingT(t)

	store := policystore.NewPolicyStore()
	inSync := make(chan struct{})
	update := &proto.ToDataplane{Payload: &proto.ToDataplane_InSync{}}
	Expect(func() { processUpdate(store, inSync, update) }).ToNot(Panic())
	Expect(inSync).To(BeClosed())
}

// processUpdate for an unhandled Payload causes a panic
func TestProcessUpdateUnknown(t *testing.T) {
	RegisterTestingT(t)

	store := policystore.NewPolicyStore()
	inSync := make(chan struct{})
	update := &proto.ToDataplane{Payload: &proto.ToDataplane_ConfigUpdate{}}
	Expect(func() { processUpdate(store, inSync, update) }).To(Panic())
}

func TestSyncRestart(t *testing.T) {
	RegisterTestingT(t)

	server := newTestSyncServer()
	defer server.Shutdown()
	server.Start()

	uut := NewClient(server.GetTarget(), uds.GetDialOptions(), ClientOptions{})
	stores := make(chan *policystore.PolicyStore)
	dpStats := make(chan statscache.DPStats, 10)

	cCtx, cCancel := context.WithCancel(context.Background())
	defer cCancel()
	go uut.Start(cCtx, stores, dpStats)

	server.SendInSync()
	select {
	case <-time.After(1 * time.Second):
		t.Error("Failed to get sync'd PolicyStore")
	case <-stores:
		// pass
	}

	server.Restart()
	select {
	case <-stores:
		t.Error("New PolicyStore should wait for inSync")
	case <-time.After(100 * time.Millisecond):
		// pass
	}

	server.SendInSync()
	select {
	case <-time.After(1 * time.Second):
		t.Error("Failed to get sync'd PolicyStore")
	case <-stores:
		// pass
	}
}

func TestSyncCancelBeforeInSync(t *testing.T) {
	RegisterTestingT(t)

	server := newTestSyncServer()
	defer server.Shutdown()
	server.Start()

	uut := NewClient(server.GetTarget(), uds.GetDialOptions(), ClientOptions{})
	stores := make(chan *policystore.PolicyStore)
	dpStats := make(chan statscache.DPStats, 10)

	cCtx, cCancel := context.WithCancel(context.Background())
	syncDone := make(chan struct{})
	go func() {
		uut.Start(cCtx, stores, dpStats)
		close(syncDone)
	}()

	time.Sleep(10 * time.Millisecond)
	cCancel()
	Eventually(syncDone).Should(BeClosed())
}

func TestSyncCancelAfterInSync(t *testing.T) {
	RegisterTestingT(t)

	server := newTestSyncServer()
	defer server.Shutdown()
	server.Start()

	uut := NewClient(server.GetTarget(), uds.GetDialOptions(), ClientOptions{})
	stores := make(chan *policystore.PolicyStore)
	dpStats := make(chan statscache.DPStats, 10)

	cCtx, cCancel := context.WithCancel(context.Background())
	syncDone := make(chan struct{})
	go func() {
		uut.Start(cCtx, stores, dpStats)
		close(syncDone)
	}()

	server.SendInSync()
	select {
	case <-time.After(1 * time.Second):
		t.Error("Failed to get sync'd PolicyStore")
	case <-stores:
		// pass
	}
	cCancel()
	Eventually(syncDone).Should(BeClosed())
}

func TestSyncServerCancelBeforeInSync(t *testing.T) {
	RegisterTestingT(t)

	server := newTestSyncServer()
	defer server.Shutdown()
	server.Start()

	uut := NewClient(server.GetTarget(), uds.GetDialOptions(), ClientOptions{})
	stores := make(chan *policystore.PolicyStore)
	dpStats := make(chan statscache.DPStats, 10)

	cCtx, cCancel := context.WithCancel(context.Background())
	defer cCancel()

	syncDone := make(chan struct{})
	go func() {
		uut.Start(cCtx, stores, dpStats)
		close(syncDone)
	}()

	server.Shutdown()
	time.Sleep(10 * time.Millisecond)
	cCancel()
	Eventually(syncDone).Should(BeClosed())
}

func TestDPStatsAfterConnection(t *testing.T) {
	RegisterTestingT(t)

	server := newTestSyncServer()
	defer server.Shutdown()
	server.Start()

	uut := NewClient(server.GetTarget(), uds.GetDialOptions(), ClientOptions{StatsFlushInterval: 100 * time.Millisecond})
	stores := make(chan *policystore.PolicyStore)
	dpStats := make(chan statscache.DPStats, 10)

	cCtx, cCancel := context.WithCancel(context.Background())
	defer cCancel()

	syncDone := make(chan struct{})
	go func() {
		uut.Start(cCtx, stores, dpStats)
		close(syncDone)
	}()

	// Wait for in sync, so that we can be sure we've connected.
	server.SendInSync()
	select {
	case <-time.After(1 * time.Second):
		t.Error("Failed to get sync'd PolicyStore")
	case <-stores:
		// pass
	}

	// Send an aggregated DPStats update and check we have the corresponding protobuf stored. Allow enough time
	// for the flush interval and the reconnection time.
	dpStats <- statscache.DPStats{
		Tuple: statscache.Tuple{
			SrcIp:    "1.2.3.4",
			DstIp:    "11.22.33.44",
			SrcPort:  1000,
			DstPort:  2000,
			Protocol: "TCP",
		},
		Values: statscache.Values{
			HTTPRequestsAllowed: 1,
			HTTPRequestsDenied:  3,
		},
	}
	Eventually(server.GetDataplaneStats, "700ms", "50ms").Should(Equal([]*proto.DataplaneStats{
		{
			SrcIp:    "1.2.3.4",
			DstIp:    "11.22.33.44",
			SrcPort:  1000,
			DstPort:  2000,
			Protocol: &proto.Protocol{&proto.Protocol_Name{Name: "TCP"}},
			Stats: []*proto.Statistic{
				{
					Direction:  proto.Statistic_IN,
					Relativity: proto.Statistic_DELTA,
					Kind:       proto.Statistic_HTTP_REQUESTS,
					Action:     proto.Action_ALLOWED,
					Value:      1,
				},
				{
					Direction:  proto.Statistic_IN,
					Relativity: proto.Statistic_DELTA,
					Kind:       proto.Statistic_HTTP_REQUESTS,
					Action:     proto.Action_DENIED,
					Value:      3,
				},
			},
		},
	}))

	cCancel()
	Eventually(syncDone).Should(BeClosed())
}

func TestDPStatsBeforeConnectionNoAggregation(t *testing.T) {
	RegisterTestingT(t)

	server := newTestSyncServer()
	defer server.Shutdown()

	uut := NewClient(server.GetTarget(), uds.GetDialOptions(), ClientOptions{StatsFlushInterval: 50 * time.Millisecond})
	stores := make(chan *policystore.PolicyStore)

	dpStats := make(chan statscache.DPStats, 10)

	cCtx, cCancel := context.WithCancel(context.Background())
	defer cCancel()
	syncDone := make(chan struct{})
	go func() {
		uut.Start(cCtx, stores, dpStats)
		close(syncDone)
	}()

	// Before starting the server send in 2*MaxStatsCaches updates, pausing in between to allow each update to be
	// in an aggregated stats flush. This should build up stats sets in the syncserver, and should start aging out
	// the oldest sets, so that only the last MaxStatsCaches remain.
	for i := uint(0); i < 2*MaxStatsCaches; i++ {
		dpStats <- statscache.DPStats{
			Tuple: statscache.Tuple{
				SrcIp:    "1.2.3.4",
				DstIp:    "11.22.33.44",
				SrcPort:  1000,
				DstPort:  2000,
				Protocol: "TCP",
			},
			Values: statscache.Values{
				HTTPRequestsAllowed: 0,
				HTTPRequestsDenied:  1 << i,
			},
		}
		Consistently(server.GetDataplaneStats, "100ms", "10ms").Should(HaveLen(0))
	}

	// Start the server. This should allow the connection to complete, and we should receive the last
	// `MaxStatsCaches` stats as individual unaggregated stats.
	server.Start()

	// Wait for in sync to complete since that guarantees we are connected.
	server.SendInSync()
	select {
	case <-time.After(1 * time.Second):
		t.Error("Failed to get sync'd PolicyStore")
	case <-stores:
		// pass
	}

	expected := []*proto.DataplaneStats{}
	for i := uint(MaxStatsCaches); i < 2*MaxStatsCaches; i++ {
		expected = append(expected, &proto.DataplaneStats{
			SrcIp:    "1.2.3.4",
			DstIp:    "11.22.33.44",
			SrcPort:  1000,
			DstPort:  2000,
			Protocol: &proto.Protocol{&proto.Protocol_Name{Name: "TCP"}},
			Stats: []*proto.Statistic{
				{
					Direction:  proto.Statistic_IN,
					Relativity: proto.Statistic_DELTA,
					Kind:       proto.Statistic_HTTP_REQUESTS,
					Action:     proto.Action_DENIED,
					Value:      1 << i,
				},
			},
		})
	}
	Eventually(server.GetDataplaneStats, "500ms", "50ms").Should(Equal(expected))

	cCancel()
	Eventually(syncDone).Should(BeClosed())
}

func TestDPStatsBeforeConnectionAggregation(t *testing.T) {
	RegisterTestingT(t)

	server := newTestSyncServer()
	defer server.Shutdown()

	uut := NewClient(server.GetTarget(), uds.GetDialOptions(), ClientOptions{StatsFlushInterval: 50 * time.Millisecond})
	stores := make(chan *policystore.PolicyStore)

	// To avoid timing windows where the stats cache flushes, fill the channel before creating the server.
	dpStats := make(chan statscache.DPStats, 4*MaxStatsCaches)
	var totalAllowed int64
	var totalDenied int64
	for i := uint(0); i < 2*MaxStatsCaches; i++ {
		dpStats <- statscache.DPStats{
			Tuple: statscache.Tuple{
				SrcIp:    "1.2.3.4",
				DstIp:    "11.22.33.44",
				SrcPort:  1000,
				DstPort:  2000,
				Protocol: "TCP",
			},
			Values: statscache.Values{
				HTTPRequestsAllowed: 1<<i + 1,
				HTTPRequestsDenied:  0,
			},
		}
		dpStats <- statscache.DPStats{
			Tuple: statscache.Tuple{
				SrcIp:    "1.2.3.4",
				DstIp:    "11.22.33.44",
				SrcPort:  1000,
				DstPort:  2000,
				Protocol: "TCP",
			},
			Values: statscache.Values{
				HTTPRequestsAllowed: 0,
				HTTPRequestsDenied:  1 << i,
			},
		}
		totalAllowed += 1<<i + 1
		totalDenied += 1 << i
		Consistently(server.GetDataplaneStats, "100ms", "10ms").Should(HaveLen(0))
	}

	cCtx, cCancel := context.WithCancel(context.Background())
	defer cCancel()
	syncDone := make(chan struct{})
	go func() {
		uut.Start(cCtx, stores, dpStats)
		close(syncDone)
	}()

	// Start the server. This should allow the connection to complete, and we should receive one aggregated
	// statistic.
	server.Start()

	// Wait for in sync to complete since that guarantees we are connected.
	server.SendInSync()
	select {
	case <-time.After(1 * time.Second):
		t.Error("Failed to get sync'd PolicyStore")
	case <-stores:
		// pass
	}

	expected := []*proto.DataplaneStats{
		{
			SrcIp:    "1.2.3.4",
			DstIp:    "11.22.33.44",
			SrcPort:  1000,
			DstPort:  2000,
			Protocol: &proto.Protocol{&proto.Protocol_Name{Name: "TCP"}},
			Stats: []*proto.Statistic{
				{
					Direction:  proto.Statistic_IN,
					Relativity: proto.Statistic_DELTA,
					Kind:       proto.Statistic_HTTP_REQUESTS,
					Action:     proto.Action_ALLOWED,
					Value:      totalAllowed,
				},
				{
					Direction:  proto.Statistic_IN,
					Relativity: proto.Statistic_DELTA,
					Kind:       proto.Statistic_HTTP_REQUESTS,
					Action:     proto.Action_DENIED,
					Value:      totalDenied,
				},
			},
		},
	}
	Eventually(server.GetDataplaneStats, "500ms", "50ms").Should(Equal(expected))
	Consistently(server.GetDataplaneStats, "500ms", "50ms").Should(Equal(expected))

	cCancel()
	Eventually(syncDone).Should(BeClosed())
}

func TestDPStatsReportReturnsError(t *testing.T) {
	RegisterTestingT(t)

	server := newTestSyncServer()
	defer server.Shutdown()

	uut := NewClient(server.GetTarget(), uds.GetDialOptions(), ClientOptions{StatsFlushInterval: 50 * time.Millisecond})
	stores := make(chan *policystore.PolicyStore)

	dpStats := make(chan statscache.DPStats, 4*MaxStatsCaches)

	cCtx, cCancel := context.WithCancel(context.Background())
	defer cCancel()
	syncDone := make(chan struct{})
	go func() {
		uut.Start(cCtx, stores, dpStats)
		close(syncDone)
	}()

	// Start the server. This should allow the connection to complete, and we should receive one aggregated
	// statistic.
	server.Start()

	// Tell the Report fn to return an error and then send in the stats. We should not receive any updates.
	server.SetReportReturnValues(false, errors.New("foobarbaz"))
	dpStats <- statscache.DPStats{
		Tuple: statscache.Tuple{
			SrcIp:    "1.2.3.4",
			DstIp:    "11.22.33.44",
			SrcPort:  1000,
			DstPort:  2000,
			Protocol: "TCP",
		},
		Values: statscache.Values{
			HTTPRequestsAllowed: 15,
			HTTPRequestsDenied:  0,
		},
	}

	expected := []*proto.DataplaneStats{
		{
			SrcIp:    "1.2.3.4",
			DstIp:    "11.22.33.44",
			SrcPort:  1000,
			DstPort:  2000,
			Protocol: &proto.Protocol{&proto.Protocol_Name{Name: "TCP"}},
			Stats: []*proto.Statistic{
				{
					Direction:  proto.Statistic_IN,
					Relativity: proto.Statistic_DELTA,
					Kind:       proto.Statistic_HTTP_REQUESTS,
					Action:     proto.Action_ALLOWED,
					Value:      15,
				},
			},
		},
	}
	Consistently(server.GetDataplaneStats).Should(HaveLen(0))

	// Tell the Report fn to succeed and check we get our stats.
	server.SetReportReturnValues(true, nil)
	Eventually(server.GetDataplaneStats).Should(Equal(expected))

	cCancel()
	Eventually(syncDone).Should(BeClosed())
}

func TestDPStatsReportReturnsUnsuccessful(t *testing.T) {
	RegisterTestingT(t)

	server := newTestSyncServer()
	defer server.Shutdown()

	uut := NewClient(server.GetTarget(), uds.GetDialOptions(), ClientOptions{StatsFlushInterval: 50 * time.Millisecond})
	stores := make(chan *policystore.PolicyStore)

	dpStats := make(chan statscache.DPStats, 4*MaxStatsCaches)

	cCtx, cCancel := context.WithCancel(context.Background())
	defer cCancel()
	syncDone := make(chan struct{})
	go func() {
		uut.Start(cCtx, stores, dpStats)
		close(syncDone)
	}()

	// Start the server. This should allow the connection to complete, and we should receive one aggregated
	// statistic.
	server.Start()

	// Wait for in sync to complete since that guarantees we are connected.
	server.SendInSync()
	select {
	case <-time.After(1 * time.Second):
		t.Error("Failed to get sync'd PolicyStore")
	case <-stores:
		// pass
	}

	// Tell the Report fn to return unsuccessful (which can occur if the remote end is no longer expecting statistics
	// to be sent to it) and then send in the stats. We should not receive any updates.
	server.SetReportReturnValues(false, errors.New("foobarbaz"))
	dpStats <- statscache.DPStats{
		Tuple: statscache.Tuple{
			SrcIp:    "1.2.3.4",
			DstIp:    "11.22.33.44",
			SrcPort:  1000,
			DstPort:  2000,
			Protocol: "TCP",
		},
		Values: statscache.Values{
			HTTPRequestsAllowed: 15,
			HTTPRequestsDenied:  0,
		},
	}
	Consistently(server.GetDataplaneStats).Should(HaveLen(0))

	// Tell the Report fn to succeed and check we still don't get the stats - they should have been dropped at this
	// point.
	server.SetReportReturnValues(true, nil)
	Consistently(server.GetDataplaneStats).Should(HaveLen(0))

	cCancel()
	Eventually(syncDone).Should(BeClosed())
}

// processUpdate handles ConfigUpdate DropActionOverride values.
func TestConfigUpdateDropActionOverride(t *testing.T) {
	RegisterTestingT(t)

	store := policystore.NewPolicyStore()
	inSync := make(chan struct{})

	// Check defaulted value.
	Expect(store.DropActionOverride).To(Equal(policystore.DROP))

	update := &proto.ToDataplane{
		Payload: &proto.ToDataplane_ConfigUpdate{ConfigUpdate: &proto.ConfigUpdate{
			Config: map[string]string{
				"DropActionOverride": "ThisIsABadValue",
			},
		}},
	}
	Expect(func() { processUpdate(store, inSync, update) }).ToNot(Panic())
	Expect(store.DropActionOverride).To(Equal(policystore.DROP))

	update = &proto.ToDataplane{
		Payload: &proto.ToDataplane_ConfigUpdate{ConfigUpdate: &proto.ConfigUpdate{
			Config: map[string]string{
				"DropActionOverride": "ThisIsABadValue",
			},
		}},
	}
	Expect(func() { processUpdate(store, inSync, update) }).ToNot(Panic())
	Expect(store.DropActionOverride).To(Equal(policystore.DROP))

	update = &proto.ToDataplane{
		Payload: &proto.ToDataplane_ConfigUpdate{ConfigUpdate: &proto.ConfigUpdate{
			Config: map[string]string{
				"DropActionOverride": "ACCEPT",
			},
		}},
	}
	Expect(func() { processUpdate(store, inSync, update) }).ToNot(Panic())
	Expect(store.DropActionOverride).To(Equal(policystore.ACCEPT))

	update = &proto.ToDataplane{
		Payload: &proto.ToDataplane_ConfigUpdate{ConfigUpdate: &proto.ConfigUpdate{
			Config: map[string]string{
				"DropActionOverride": "LOGandACCEPT",
			},
		}},
	}
	Expect(func() { processUpdate(store, inSync, update) }).ToNot(Panic())
	Expect(store.DropActionOverride).To(Equal(policystore.LOG_AND_ACCEPT))

	update = &proto.ToDataplane{
		Payload: &proto.ToDataplane_ConfigUpdate{ConfigUpdate: &proto.ConfigUpdate{
			Config: map[string]string{
				"DropActionOverride": "LOGandDROP",
			},
		}},
	}
	Expect(func() { processUpdate(store, inSync, update) }).ToNot(Panic())
	Expect(store.DropActionOverride).To(Equal(policystore.LOG_AND_DROP))
}

// processUpdate handles ConfigUpdate with unknown config.
func TestConfigUpdateUnknownConfig(t *testing.T) {
	RegisterTestingT(t)

	store := policystore.NewPolicyStore()
	inSync := make(chan struct{})

	update := &proto.ToDataplane{
		Payload: &proto.ToDataplane_ConfigUpdate{ConfigUpdate: &proto.ConfigUpdate{
			Config: map[string]string{
				"ThisIsNotValid": "AndAnArbitraryValue",
			},
		}},
	}
	Expect(func() { processUpdate(store, inSync, update) }).ToNot(Panic())
}

type testSyncServer struct {
	cxt        context.Context
	cancel     func()
	updates    chan proto.ToDataplane
	path       string
	gRPCServer *grpc.Server
	listener   net.Listener
	cLock      sync.Mutex
	cancelFns  []func()
	dpStats    []*proto.DataplaneStats
	reportErr  error
	reportRet  bool
}

func newTestSyncServer() *testSyncServer {
	cxt, cancel := context.WithCancel(context.Background())
	socketDir := makeTmpListenerDir()
	socketPath := path.Join(socketDir, ListenerSocket)
	ss := &testSyncServer{
		cxt: cxt, cancel: cancel, updates: make(chan proto.ToDataplane), path: socketPath, gRPCServer: grpc.NewServer(),
		reportRet: true,
	}
	proto.RegisterPolicySyncServer(ss.gRPCServer, ss)
	return ss
}

func (ss *testSyncServer) Shutdown() {
	ss.cancel()
	ss.Stop()
}

func (ss *testSyncServer) Start() {
	ss.listen()
}

func (ss *testSyncServer) Stop() {
	ss.cLock.Lock()
	for _, c := range ss.cancelFns {
		c()
	}
	ss.cancelFns = make([]func(), 0)
	ss.cLock.Unlock()

	err := os.Remove(ss.path)
	if err != nil && !os.IsNotExist(err) {
		// A test may call Stop/Shutdown multiple times. It shouldn't fail if it does.
		Expect(err).ToNot(HaveOccurred())
	}
}

func (ss *testSyncServer) Restart() {
	ss.Stop()
	ss.Start()
}

func (ss *testSyncServer) Sync(_ *proto.SyncRequest, stream proto.PolicySync_SyncServer) error {
	ctx, cancel := context.WithCancel(ss.cxt)
	ss.cLock.Lock()
	ss.cancelFns = append(ss.cancelFns, cancel)
	ss.cLock.Unlock()
	var update proto.ToDataplane
	for {
		select {
		case <-ctx.Done():
			return nil
		case update = <-ss.updates:
			stream.Send(&update)
		}
	}
}

func (ss *testSyncServer) Report(_ context.Context, d *proto.DataplaneStats) (*proto.ReportResult, error) {
	ss.cLock.Lock()
	defer ss.cLock.Unlock()

	if ss.reportErr != nil || !ss.reportRet {
		// Returning error, so don't store the stats.
		return &proto.ReportResult{
			Successful: ss.reportRet,
		}, ss.reportErr
	}

	// Store the stats and return success.
	ss.dpStats = append(ss.dpStats, d)
	return &proto.ReportResult{
		Successful: true,
	}, nil
}

func (ss *testSyncServer) SendInSync() {
	ss.updates <- proto.ToDataplane{Payload: &proto.ToDataplane_InSync{InSync: &proto.InSync{}}}
}

func (ss *testSyncServer) GetTarget() string {
	return ss.path
}

func (ss *testSyncServer) GetDataplaneStats() []*proto.DataplaneStats {
	ss.cLock.Lock()
	defer ss.cLock.Unlock()
	s := make([]*proto.DataplaneStats, len(ss.dpStats))
	copy(s, ss.dpStats)
	return s
}

func (ss *testSyncServer) SetReportReturnValues(ret bool, err error) {
	ss.cLock.Lock()
	defer ss.cLock.Unlock()
	ss.reportRet = ret
	ss.reportErr = err
}

func (ss *testSyncServer) listen() {
	ss.listener = openListener(ss.path)
	go ss.gRPCServer.Serve(ss.listener)
}

const ListenerSocket = "policysync.sock"

func makeTmpListenerDir() string {
	dirPath, err := ioutil.TempDir("/tmp", "felixut")
	Expect(err).ToNot(HaveOccurred())
	return dirPath
}

func openListener(socketPath string) net.Listener {
	lis, err := net.Listen("unix", socketPath)
	Expect(err).ToNot(HaveOccurred())
	return lis
}
