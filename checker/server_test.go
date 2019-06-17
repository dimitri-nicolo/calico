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

package checker

import (
	"context"
	"testing"

	authz "github.com/envoyproxy/data-plane-api/envoy/service/auth/v2"
	"github.com/gogo/googleapis/google/rpc"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/app-policy/policystore"
	"github.com/projectcalico/app-policy/proto"
	"github.com/projectcalico/app-policy/statscache"

	"github.com/envoyproxy/data-plane-api/envoy/api/v2/core"
)

func TestCheckNoStore(t *testing.T) {
	RegisterTestingT(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stores := make(chan *policystore.PolicyStore)
	dpStats := make(chan statscache.DPStats, 10)
	uut := NewServer(ctx, stores, dpStats)

	req := &authz.CheckRequest{}
	resp, err := uut.Check(ctx, req)
	Expect(err).To(BeNil())
	Expect(resp.GetStatus().GetCode()).To(Equal(UNAVAILABLE))
}

func TestCheckStoreNoHTTP(t *testing.T) {
	RegisterTestingT(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stores := make(chan *policystore.PolicyStore)
	dpStats := make(chan statscache.DPStats, 10)
	uut := NewServer(ctx, stores, dpStats)

	store := policystore.NewPolicyStore()
	store.Write(func(s *policystore.PolicyStore) {
		s.Endpoint = &proto.WorkloadEndpoint{
			ProfileIds: []string{"default"},
		}
		s.ProfileByID[proto.ProfileID{Name: "default"}] = &proto.Profile{
			InboundRules: []*proto.Rule{{Action: "Allow"}},
		}
	})
	stores <- store

	// Send in request with no HTTP data. Request should pass, we should have no stats updates for this request.
	req := &authz.CheckRequest{Attributes: &authz.AttributeContext{
		Source: &authz.AttributeContext_Peer{
			Principal: "spiffe://cluster.local/ns/default/sa/steve",
		},
		Destination: &authz.AttributeContext_Peer{
			Principal: "spiffe://cluster.local/ns/default/sa/sammy",
		},
	}}
	chk := func() *authz.CheckResponse {
		rsp, err := uut.Check(ctx, req)
		Expect(err).ToNot(HaveOccurred())
		return rsp
	}
	Eventually(chk).Should(Equal(&authz.CheckResponse{Status: &rpc.Status{Code: OK}}))
	Consistently(dpStats, "200ms", "50ms").ShouldNot(Receive())
}

func TestCheckStoreHTTPAllowed(t *testing.T) {
	RegisterTestingT(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stores := make(chan *policystore.PolicyStore)
	dpStats := make(chan statscache.DPStats, 10)
	uut := NewServer(ctx, stores, dpStats)

	store := policystore.NewPolicyStore()
	store.Write(func(s *policystore.PolicyStore) {
		s.Endpoint = &proto.WorkloadEndpoint{
			ProfileIds: []string{"default"},
		}
		s.ProfileByID[proto.ProfileID{Name: "default"}] = &proto.Profile{
			InboundRules: []*proto.Rule{{Action: "Allow"}},
		}
	})
	stores <- store

	// Send in request with no HTTP data. Request should pass, we should have no stats updates for this request.
	req := &authz.CheckRequest{Attributes: &authz.AttributeContext{
		Source: &authz.AttributeContext_Peer{
			Address: &core.Address{
				Address: &core.Address_SocketAddress{
					SocketAddress: &core.SocketAddress{
						Address:       "1.2.3.4",
						PortSpecifier: &core.SocketAddress_PortValue{PortValue: 1000},
						Protocol:      core.TCP,
					},
				},
			},
			Principal: "spiffe://cluster.local/ns/default/sa/steve",
		},
		Destination: &authz.AttributeContext_Peer{
			Address: &core.Address{
				Address: &core.Address_SocketAddress{
					SocketAddress: &core.SocketAddress{
						Address:       "11.22.33.44",
						PortSpecifier: &core.SocketAddress_PortValue{PortValue: 2000},
						Protocol:      core.TCP,
					},
				},
			},
			Principal: "spiffe://cluster.local/ns/default/sa/sammy",
		},
		Request: &authz.AttributeContext_Request{
			Http: &authz.AttributeContext_HttpRequest{Method: "GET", Path: "/foo"},
		},
	}}

	// Check request is allowed and that we don't get any stats updates (stats are not yet enabled).
	chk := func() *authz.CheckResponse {
		rsp, err := uut.Check(ctx, req)
		Expect(err).ToNot(HaveOccurred())
		return rsp
	}
	Eventually(chk).Should(Equal(&authz.CheckResponse{Status: &rpc.Status{Code: OK}}))
	Consistently(dpStats, "200ms", "50ms").ShouldNot(Receive())

	// Enable stats, re-run the request and this time check we do get stats updates.
	store.DataplaneStatsEnabledForAllowed = true
	chk = func() *authz.CheckResponse {
		rsp, err := uut.Check(ctx, req)
		Expect(err).ToNot(HaveOccurred())
		return rsp
	}
	Eventually(chk).Should(Equal(&authz.CheckResponse{Status: &rpc.Status{Code: OK}}))
	Eventually(dpStats).Should(Receive(Equal(statscache.DPStats{
		Tuple: statscache.Tuple{
			SrcIp:    "1.2.3.4",
			DstIp:    "11.22.33.44",
			SrcPort:  1000,
			DstPort:  2000,
			Protocol: "TCP",
		},
		Values: statscache.Values{
			HTTPRequestsAllowed: 1,
			HTTPRequestsDenied:  0,
		},
	})))
}

func TestCheckStoreHTTPDenied(t *testing.T) {
	RegisterTestingT(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stores := make(chan *policystore.PolicyStore)
	dpStats := make(chan statscache.DPStats, 10)
	uut := NewServer(ctx, stores, dpStats)

	store := policystore.NewPolicyStore()
	store.Write(func(s *policystore.PolicyStore) {
		s.Endpoint = &proto.WorkloadEndpoint{
			ProfileIds: []string{"default"},
		}
		s.ProfileByID[proto.ProfileID{Name: "default"}] = &proto.Profile{
			InboundRules: []*proto.Rule{{Action: "Deny"}},
		}
	})
	stores <- store

	// Send in request with no HTTP data. Request should pass, we should have no stats updates for this request.
	req := &authz.CheckRequest{Attributes: &authz.AttributeContext{
		Source: &authz.AttributeContext_Peer{
			Address: &core.Address{
				Address: &core.Address_SocketAddress{
					SocketAddress: &core.SocketAddress{
						Address:       "1.2.3.4",
						PortSpecifier: &core.SocketAddress_PortValue{PortValue: 1000},
						Protocol:      core.TCP,
					},
				},
			},
			Principal: "spiffe://cluster.local/ns/default/sa/steve",
		},
		Destination: &authz.AttributeContext_Peer{
			Address: &core.Address{
				Address: &core.Address_SocketAddress{
					SocketAddress: &core.SocketAddress{
						Address:       "11.22.33.44",
						PortSpecifier: &core.SocketAddress_PortValue{PortValue: 2000},
						Protocol:      core.TCP,
					},
				},
			},
			Principal: "spiffe://cluster.local/ns/default/sa/sammy",
		},
		Request: &authz.AttributeContext_Request{
			Http: &authz.AttributeContext_HttpRequest{Method: "GET", Path: "/foo"},
		},
	}}

	// Check request is denied and that we don't get any stats updates (stats are not yet enabled).
	chk := func() *authz.CheckResponse {
		rsp, err := uut.Check(ctx, req)
		Expect(err).ToNot(HaveOccurred())
		return rsp
	}
	Eventually(chk).Should(Equal(&authz.CheckResponse{Status: &rpc.Status{Code: PERMISSION_DENIED}}))
	Consistently(dpStats, "200ms", "50ms").ShouldNot(Receive())

	// Enable stats, re-run the request and this time check we do get stats updates.
	store.DataplaneStatsEnabledForDenied = true
	chk = func() *authz.CheckResponse {
		rsp, err := uut.Check(ctx, req)
		Expect(err).ToNot(HaveOccurred())
		return rsp
	}
	Eventually(chk).Should(Equal(&authz.CheckResponse{Status: &rpc.Status{Code: PERMISSION_DENIED}}))
	Eventually(dpStats).Should(Receive(Equal(statscache.DPStats{
		Tuple: statscache.Tuple{
			SrcIp:    "1.2.3.4",
			DstIp:    "11.22.33.44",
			SrcPort:  1000,
			DstPort:  2000,
			Protocol: "TCP",
		},
		Values: statscache.Values{
			HTTPRequestsAllowed: 0,
			HTTPRequestsDenied:  1,
		},
	})))
}
