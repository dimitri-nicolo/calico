package checker

import (
	"context"
	"testing"

	authz "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"google.golang.org/genproto/googleapis/rpc/status"

	"github.com/projectcalico/calico/app-policy/policystore"
	"github.com/projectcalico/calico/app-policy/statscache"
)

var _ CheckProvider = (*okProvider)(nil)
var _ CheckProvider = (*denyProvider)(nil)

type okProvider struct{}

func (p *okProvider) Name() string { return "ok-provider" }
func (p *okProvider) Check(*policystore.PolicyStore, *authz.CheckRequest) (*authz.CheckResponse, error) {
	return &authz.CheckResponse{Status: &status.Status{Code: OK}}, nil
}

type denyProvider struct{}

func (p *denyProvider) Name() string { return "deny-provider" }
func (p *denyProvider) Check(*policystore.PolicyStore, *authz.CheckRequest) (*authz.CheckResponse, error) {
	return &authz.CheckResponse{Status: &status.Status{Code: PERMISSION_DENIED}}, nil
}

func TestServerCheckerProvidersNone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	psm := policystore.NewPolicyStoreManager()
	dpStats := statscache.New()
	uut := NewServer(ctx, psm, dpStats) // no providers.. provided

	req := &authz.CheckRequest{}
	resp, err := uut.Check(ctx, req)
	if err != nil {
		t.Error("error must be nil")
	}

	if resp.Status.Code != UNKNOWN {
		t.Error("with no checkproviders it must be unknown")
	}
}

func TestServerCheckerProvidersAllOK(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	psm := policystore.NewPolicyStoreManager()
	dpStats := statscache.New()
	uut := NewServer(ctx, psm, dpStats,
		WithRegisteredCheckProvider(new(okProvider)),
		WithRegisteredCheckProvider(new(okProvider)),
	)

	req := &authz.CheckRequest{}
	resp, err := uut.Check(ctx, req)
	if err != nil {
		t.Error("error must be nil")
	}

	if resp.Status.Code != OK {
		t.Error("with all checkproviders returning ok, it must be ok")
	}
}

func TestServerCheckerProvidersAllDeny(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	psm := policystore.NewPolicyStoreManager()
	dpStats := statscache.New()
	uut := NewServer(ctx, psm, dpStats,
		WithRegisteredCheckProvider(new(denyProvider)),
		WithRegisteredCheckProvider(new(denyProvider)),
	)

	req := &authz.CheckRequest{}
	resp, err := uut.Check(ctx, req)
	if err != nil {
		t.Error("error must be nil")
	}

	if resp.Status.Code == OK {
		t.Error("all checkproviders returns deny so it must not be ok")
	}
}

func TestServerCheckerProvidersFiftyFifty(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	psm := policystore.NewPolicyStoreManager()
	dpStats := statscache.New()
	uut := NewServer(ctx, psm, dpStats,
		WithRegisteredCheckProvider(new(okProvider)),
		WithRegisteredCheckProvider(new(denyProvider)),
	)

	req := &authz.CheckRequest{}
	resp, err := uut.Check(ctx, req)
	if err != nil {
		t.Error("error must be nil")
	}

	if resp.Status.Code == OK {
		t.Error("one of checkproviders is a deny so it must be not ok")
	}
}
