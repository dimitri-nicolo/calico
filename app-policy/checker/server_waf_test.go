//go:build cgo
// +build cgo

package checker

import (
	"bytes"
	"context"
	"testing"

	. "github.com/onsi/gomega"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	authz "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"

	"github.com/projectcalico/calico/app-policy/policystore"
	"github.com/projectcalico/calico/app-policy/statscache"
	"github.com/projectcalico/calico/app-policy/waf"
	"github.com/projectcalico/calico/felix/proto"
)

func TestWAFProcessHttpRequestHTTPGetAllowed(t *testing.T) {
	RegisterTestingT(t)
	waf.InitializeLogging()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dpStats := statscache.New()
	psm := policystore.NewPolicyStoreManager()
	uut := NewServer(ctx, psm, dpStats, WithRegisteredCheckProvider(NewWAFCheckProvider("per-pod-policies")))

	psm.Write(func(s *policystore.PolicyStore) {
		s.Endpoint = &proto.WorkloadEndpoint{
			ProfileIds: []string{"default"},
		}
		s.ProfileByID[proto.ProfileID{Name: "default"}] = &proto.Profile{
			InboundRules: []*proto.Rule{{Action: "Allow"}},
		}
	})

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

func TestWAFProcessHttpRequestPassThroughSQLInjection(t *testing.T) {
	RegisterTestingT(t)

	// Logging
	waf.Logger = nil
	memoryLog := bytes.Buffer{}
	waf.InitializeLogging(&memoryLog)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dpStats := statscache.New()
	psm := policystore.NewPolicyStoreManager()
	uut := NewServer(ctx, psm, dpStats, WithRegisteredCheckProvider(NewWAFCheckProvider("per-pod-policies")))

	psm.Write(func(s *policystore.PolicyStore) {
		s.Endpoint = &proto.WorkloadEndpoint{
			ProfileIds: []string{"default"},
		}
		s.ProfileByID[proto.ProfileID{Name: "default"}] = &proto.Profile{
			InboundRules: []*proto.Rule{{Action: "Allow"}},
		}
	})

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

	_ = waf.CheckRulesSetExists(waf.TestCoreRulesetPassDirectory)

	waf.InitializeModSecurity()
	filenames := waf.GetRulesSetFilenames()
	_ = waf.LoadModSecurityCoreRuleSet(filenames)

	resp, err := uut.Check(ctx, req)
	Expect(err).ToNot(HaveOccurred())
	Expect(resp.GetStatus().GetCode()).To(Equal(OK))
	Expect(memoryLog.String()).To(ContainSubstring(
		"\"message\":\"SQL Injection Attack Detected via libinjection\""))
}

func TestWAFProcessHttpRequestHTTPPostDeniedCrossSiteScript(t *testing.T) {
	RegisterTestingT(t)
	waf.InitializeLogging()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dpStats := statscache.New()
	psm := policystore.NewPolicyStoreManager()
	uut := NewServer(ctx, psm, dpStats, WithRegisteredCheckProvider(NewWAFCheckProvider("per-pod-policies")))

	psm.Write(func(s *policystore.PolicyStore) {
		s.Endpoint = &proto.WorkloadEndpoint{
			ProfileIds: []string{"default"},
		}
		s.ProfileByID[proto.ProfileID{Name: "default"}] = &proto.Profile{
			InboundRules: []*proto.Rule{{Action: "Allow"}},
		}
	})

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
	Expect(err).ToNot(HaveOccurred())
	Expect(resp.GetStatus().GetCode()).To(Equal(PERMISSION_DENIED))
}
