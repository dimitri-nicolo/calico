package checker_test

import (
	"fmt"
	"net/url"

	envoycore "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoyauthz "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/projectcalico/calico/app-policy/checker"
	"github.com/projectcalico/calico/felix/proto"
	"github.com/projectcalico/calico/felix/tproxydefs"
)

func perHostCheckProviderScenarios() []*checkAuthScenario {
	subscriptionType := "per-host-policies"
	inboundRule := &proto.Rule{
		Action: "Allow",
		HttpMatch: &proto.HTTPMatch{
			Methods: []string{"GET"},
			Paths: []*proto.HTTPMatch_PathMatch{
				{PathMatch: &proto.HTTPMatch_PathMatch_Prefix{Prefix: "/public"}},
			},
		},
	}

	basicUpdates := append(
		[]*proto.ToDataplane{
			wepUpdate("pod-1", []string{"10.0.1.1/32"}, []string{"default"}),
			wepUpdate("pod-2", []string{"10.0.2.2/32"}, []string{"default"}),
			ipsetUpdate(tproxydefs.ApplicationLayerPolicyIPSet, []string{"10.0.1.1", "10.0.2.2"}),
		},
		policyAndProfileUpdate("secure", "default", inboundRule)...,
	)

	knownSrcOrDstCheckTest := &checkAuthScenario{
		subscriptionType: subscriptionType,
		comment:          "checker basic tests",
		updates:          basicUpdates,
		cases: []*checkAuthScenarioCases{
			{
				"known source should pass OK",
				newRequest(
					"GET", "/public/assets",
					nil,
					// known source
					newPeer("10.0.1.1", "default", "default"),
					// some random dest
					newPeer("10.52.1.1", "default", "default"),
				),
				checker.OK,
			},
			{
				"known dest should pass OK due to policy",
				newRequest(
					"GET", "/public/assets",
					nil,
					// some random src
					newPeer("10.52.1.1", "default", "default"),
					// known dest
					newPeer("10.0.1.1", "default", "default"),
				),
				checker.OK,
			},
			{
				"known dest should get rejected with PERMISSION_DENIED due to policy",
				newRequest(
					"GET", "/private/data",
					nil,
					// some random src
					newPeer("10.52.1.1", "default", "default"),
					// known dest
					newPeer("10.0.1.1", "default", "default"),
				),
				checker.PERMISSION_DENIED,
			},
			{
				"dest/src not present in IP set should return UNKNOWN",
				newRequest(
					"GET", "/public/assets",
					nil,
					// unknown src
					newPeer("10.42.1.1", "default", "default"),
					// also unknown dst
					newPeer("10.52.1.1", "default", "default"),
				),
				checker.UNKNOWN,
			},
		},
	}

	return []*checkAuthScenario{
		knownSrcOrDstCheckTest,
	}
}

func newRequest(
	method, requestUrl string,
	// todo: add body, rawbody for waf processing tests
	headers map[string]string,
	src, dst *envoyauthz.AttributeContext_Peer,
) *envoyauthz.CheckRequest {

	u, _ := url.Parse(requestUrl)

	return &envoyauthz.CheckRequest{
		Attributes: &envoyauthz.AttributeContext{
			Source:      src,
			Destination: dst,
			Request: &envoyauthz.AttributeContext_Request{
				Time: timestamppb.Now(),
				Http: &envoyauthz.AttributeContext_HttpRequest{
					Scheme: u.Scheme,
					Host:   u.Host,
					Method: method,
					Path:   u.Path,
					Query:  u.RawQuery,
				},
			},
		},
	}
}

func newPeer(address, ns, sa string) *envoyauthz.AttributeContext_Peer {
	return &envoyauthz.AttributeContext_Peer{
		Principal: fmt.Sprintf("spiffe://cluster.local/ns/%s/sa/%s", ns, sa),
		Address: &envoycore.Address{
			Address: &envoycore.Address_SocketAddress{
				SocketAddress: &envoycore.SocketAddress{
					Protocol: envoycore.SocketAddress_TCP,
					Address:  address,
				},
			},
		},
	}
}
