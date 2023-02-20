package checker_test

import (
	"context"
	"testing"

	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/genproto/googleapis/rpc/status"

	envoyauthz "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"

	"github.com/projectcalico/calico/app-policy/checker"
	"github.com/projectcalico/calico/app-policy/policystore"
	"github.com/projectcalico/calico/app-policy/proto"
	"github.com/projectcalico/calico/app-policy/statscache"
	"github.com/projectcalico/calico/felix/tproxydefs"
)

type checkWAFScenario struct {
	comment     string
	updates     []*proto.ToDataplane
	checkerOpts []checker.WAFCheckProviderOption
	req         *envoyauthz.CheckRequest
	res         int32
}

func TestCheckWAFScenarios(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subscriptionType := "per-host-policies"

T:
	for _, scenario := range []*checkWAFScenario{
		{
			"no ipset members should return UNKNOWN",
			nil,
			[]checker.WAFCheckProviderOption{
				checker.WithWAFCheckProviderCheckFn(func(req *envoyauthz.CheckRequest) (*envoyauthz.CheckResponse, error) {
					return &envoyauthz.CheckResponse{
						Status: &status.Status{Code: checker.OK},
					}, nil
				}),
			},
			newRequest("GET", "/", nil, nil, nil),
			checker.UNKNOWN,
		},
		{
			"if valid ipset member should return OK if check is OK",
			[]*proto.ToDataplane{
				ipsetUpdate(tproxydefs.ServiceIPsIPSet, []string{"10.0.1.1"}),
			},
			[]checker.WAFCheckProviderOption{
				checker.WithWAFCheckProviderCheckFn(func(req *envoyauthz.CheckRequest) (*envoyauthz.CheckResponse, error) {
					return &envoyauthz.CheckResponse{
						Status: &status.Status{Code: checker.OK},
					}, nil
				}),
			},
			newRequest("GET", "/", nil, nil, newPeer("10.0.1.1", "default", "default")),
			checker.OK,
		},
		{
			"if valid ipset member should return PERMISSION_DENIED if check is PERMISSION_DENIED",
			[]*proto.ToDataplane{
				ipsetUpdate(tproxydefs.ServiceIPsIPSet, []string{"10.0.1.1"}),
			},
			[]checker.WAFCheckProviderOption{
				checker.WithWAFCheckProviderCheckFn(func(req *envoyauthz.CheckRequest) (*envoyauthz.CheckResponse, error) {
					return &envoyauthz.CheckResponse{
						Status: &status.Status{Code: checker.PERMISSION_DENIED},
					}, nil
				}),
			},
			newRequest("GET", "/", nil, nil, newPeer("10.0.1.1", "default", "default")),
			checker.PERMISSION_DENIED,
		},
		{
			"if valid ipset member is known as src only, should return UNKNOWN",
			[]*proto.ToDataplane{
				ipsetUpdate(tproxydefs.ServiceIPsIPSet, []string{"10.0.1.1"}),
			},
			[]checker.WAFCheckProviderOption{
				checker.WithWAFCheckProviderCheckFn(func(req *envoyauthz.CheckRequest) (*envoyauthz.CheckResponse, error) {
					return &envoyauthz.CheckResponse{
						Status: &status.Status{Code: checker.OK},
					}, nil
				}),
			},
			newRequest("GET", "/", nil, newPeer("10.0.1.1", "default", "default"), newPeer("10.42.1.1", "default", "default")),
			checker.UNKNOWN,
		},
	} {

		ps := policystore.NewPolicyStoreManager()
		ps.Write(func(ps *policystore.PolicyStore) {
			for _, update := range append(scenario.updates, inSync()) {
				ps.ProcessUpdate(subscriptionType, update)
			}
		})
		ps.OnInSync()
		dpStats := make(chan statscache.DPStats)
		checkServer := checker.NewServer(
			ctx, ps, dpStats,
			checker.WithSubscriptionType(subscriptionType),
			checker.WithRegisteredCheckProvider(
				checker.NewWAFCheckProvider(
					subscriptionType,
					scenario.checkerOpts...,
				),
			),
		)

		res, err := checkServer.Check(ctx, scenario.req)
		if err != nil {
			t.Errorf("waf case %s failed with error: %v", scenario.comment, err)
			continue T
		}
		if res.Status.Code != scenario.res {
			t.Errorf(
				"waf case %s failed: expected %s, got %s",
				scenario.comment,
				code.Code(scenario.res),
				code.Code(res.Status.Code),
			)
		}
	}

}
