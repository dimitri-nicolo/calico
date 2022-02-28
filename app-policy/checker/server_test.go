// Copyright (c) 2018-2022 Tigera, Inc. All rights reserved.

package checker

import (
	"context"
	"testing"

	"github.com/projectcalico/calico/app-policy/waf"

	authz "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	. "github.com/onsi/gomega"
	"google.golang.org/genproto/googleapis/rpc/status"

	"github.com/projectcalico/calico/app-policy/policystore"
	"github.com/projectcalico/calico/app-policy/proto"
	"github.com/projectcalico/calico/app-policy/statscache"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
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
	Eventually(chk).Should(Equal(&authz.CheckResponse{Status: &status.Status{Code: OK}}))
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
						Protocol:      core.SocketAddress_TCP,
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
						Protocol:      core.SocketAddress_TCP,
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

	Eventually(chk).Should(Equal(&authz.CheckResponse{Status: &status.Status{Code: OK}}))
	Consistently(dpStats, "200ms", "50ms").ShouldNot(Receive())

	// Enable stats, re-run the request and this time check we do get stats updates.
	store.DataplaneStatsEnabledForAllowed = true
	chk = func() *authz.CheckResponse {
		rsp, err := uut.Check(ctx, req)
		Expect(err).ToNot(HaveOccurred())
		return rsp
	}
	Eventually(chk).Should(Equal(&authz.CheckResponse{Status: &status.Status{Code: OK}}))
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
						Protocol:      core.SocketAddress_TCP,
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
						Protocol:      core.SocketAddress_TCP,
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
	Eventually(chk).Should(Equal(&authz.CheckResponse{Status: &status.Status{Code: PERMISSION_DENIED}}))
	Consistently(dpStats, "200ms", "50ms").ShouldNot(Receive())

	// Enable stats, re-run the request and this time check we do get stats updates.
	store.DataplaneStatsEnabledForDenied = true
	chk = func() *authz.CheckResponse {
		rsp, err := uut.Check(ctx, req)
		Expect(err).ToNot(HaveOccurred())
		return rsp
	}
	Eventually(chk).Should(Equal(&authz.CheckResponse{Status: &status.Status{Code: PERMISSION_DENIED}}))
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

func TestWAFProcessHttpRequestHTTPGetAllowed(t *testing.T) {
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

	req := &authz.CheckRequest{Attributes: &authz.AttributeContext{
		Source: &authz.AttributeContext_Peer{
			Address: &core.Address{
				Address: &core.Address_SocketAddress{
					SocketAddress: &core.SocketAddress{
						Address:       "192.168.202.9",
						PortSpecifier: &core.SocketAddress_PortValue{PortValue: 41938},
					},
				},
			},
		},
		Destination: &authz.AttributeContext_Peer{
			Address: &core.Address{
				Address: &core.Address_SocketAddress{
					SocketAddress: &core.SocketAddress{
						Address:       "192.168.120.76",
						PortSpecifier: &core.SocketAddress_PortValue{PortValue: 80},
					},
				},
			},
		},
		Request: &authz.AttributeContext_Request{
			Http: &authz.AttributeContext_HttpRequest{
				Method:   "GET",
				Path:     "/",
				Host:     "echo-a",
				Protocol: "HTTP/1.1",
			},
		},
	}}

	_ = waf.CheckRulesSetExists(waf.TestCoreRulesetDirectory)

	waf.InitializeModSecurity()
	filenames := waf.GetRulesSetFilenames()
	_ = waf.LoadModSecurityCoreRuleSet(filenames)

	resp, err := uut.Check(ctx, req)
	Expect(err).ToNot(HaveOccurred())
	Expect(resp.GetStatus().GetCode()).To(Equal(OK))
}

func TestWAFProcessHttpRequestHTTPGetDeniedSQLInjection(t *testing.T) {
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

	req := &authz.CheckRequest{Attributes: &authz.AttributeContext{
		Source: &authz.AttributeContext_Peer{
			Address: &core.Address{
				Address: &core.Address_SocketAddress{
					SocketAddress: &core.SocketAddress{
						Address:       "192.168.202.9",
						PortSpecifier: &core.SocketAddress_PortValue{PortValue: 41938},
					},
				},
			},
		},
		Destination: &authz.AttributeContext_Peer{
			Address: &core.Address{
				Address: &core.Address_SocketAddress{
					SocketAddress: &core.SocketAddress{
						Address:       "192.168.120.76",
						PortSpecifier: &core.SocketAddress_PortValue{PortValue: 80},
					},
				},
			},
		},
		Request: &authz.AttributeContext_Request{
			Http: &authz.AttributeContext_HttpRequest{
				Method:   "GET",
				Path:     "/test/artists.php?artist=0+div+1+union%23foo*%2F*bar%0D%0Aselect%23foo%0D%0A1%2C2%2Ccurrent_user",
				Host:     "echo-a",
				Protocol: "HTTP/1.1",
			},
		},
	}}

	_ = waf.CheckRulesSetExists(waf.TestCoreRulesetDenyDirectory)

	waf.InitializeModSecurity()
	filenames := waf.GetRulesSetFilenames()
	_ = waf.LoadModSecurityCoreRuleSet(filenames)

	resp, err := uut.Check(ctx, req)
	Expect(err).To(HaveOccurred())
	Expect(resp.GetStatus().GetCode()).To(Equal(PERMISSION_DENIED))
}

func TestWAFProcessHttpRequestHTTPPostDeniedCrossSiteScript(t *testing.T) {
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

	req := &authz.CheckRequest{Attributes: &authz.AttributeContext{
		Source: &authz.AttributeContext_Peer{
			Address: &core.Address{
				Address: &core.Address_SocketAddress{
					SocketAddress: &core.SocketAddress{
						Address:       "192.168.202.9",
						PortSpecifier: &core.SocketAddress_PortValue{PortValue: 41938},
					},
				},
			},
		},
		Destination: &authz.AttributeContext_Peer{
			Address: &core.Address{
				Address: &core.Address_SocketAddress{
					SocketAddress: &core.SocketAddress{
						Address:       "192.168.120.76",
						PortSpecifier: &core.SocketAddress_PortValue{PortValue: 80},
					},
				},
			},
		},
		Request: &authz.AttributeContext_Request{
			Http: &authz.AttributeContext_HttpRequest{
				Method:   "POST",
				Path:     "/",
				Host:     "echo-a",
				Protocol: "HTTP/1.1",
				Body:     "<script>alert(1)</script>",
				Headers: map[string]string{
					"Content-Type": "application/x-www-form-urlencoded",
				},
			},
		},
	}}

	_ = waf.CheckRulesSetExists(waf.TestCustomRulesetDirectory)

	waf.InitializeModSecurity()
	filenames := waf.GetRulesSetFilenames()
	_ = waf.LoadModSecurityCoreRuleSet(filenames)

	resp, err := uut.Check(ctx, req)
	Expect(err).To(HaveOccurred())
	Expect(resp.GetStatus().GetCode()).To(Equal(PERMISSION_DENIED))
}
