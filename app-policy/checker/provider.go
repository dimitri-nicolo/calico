// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package checker

import (
	"errors"
	"fmt"

	authz "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"google.golang.org/genproto/googleapis/rpc/status"

	"github.com/projectcalico/calico/app-policy/policystore"
	"github.com/projectcalico/calico/felix/ip"
	"github.com/projectcalico/calico/felix/proto"
	"github.com/projectcalico/calico/felix/rules"
	"github.com/projectcalico/calico/felix/types"
)

type CheckProvider interface {
	Name() string
	EnabledForRequest(*policystore.PolicyStore, *authz.CheckRequest) bool
	Check(*policystore.PolicyStore, *authz.CheckRequest) (*authz.CheckResponse, error)
}

type checkFn func(req *authz.CheckRequest) (*authz.CheckResponse, error)
type checkWithStore func(*policystore.PolicyStore) checkFn

type ALPCheckProvider struct {
	subscriptionType         string
	checksBySubscriptionType map[string]checkWithStore
	perHostEnabled           bool
}

type ALPCheckProviderOption func(*ALPCheckProvider)

func WithALPCheckProviderCheckFn(subscriptionType string, fn checkWithStore) ALPCheckProviderOption {
	return func(ap *ALPCheckProvider) {
		ap.checksBySubscriptionType[subscriptionType] = fn
	}
}

func NewALPCheckProvider(subscriptionType string, tproxy bool, opts ...ALPCheckProviderOption) CheckProvider {
	c := &ALPCheckProvider{
		subscriptionType,
		map[string]checkWithStore{
			"per-host-policies": defaultPerHostPolicyCheck,
			"per-pod-policies":  defaultPerPodPolicyCheck,
		},
		tproxy,
	}

	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *ALPCheckProvider) Name() string {
	return "application-layer-policy"
}

func GetSidecar(ps *policystore.PolicyStore, req *authz.CheckRequest) (sidecar *proto.WorkloadEndpoint) {
	var ok bool
	// Try to get sidecar
	if req.Attributes.Destination.Address == nil {
		return
	}
	destIp, err := ip.ParseCIDROrIP(req.Attributes.Destination.Address.GetSocketAddress().Address)
	if err != nil {
		return
	}
	dest := ipToEndpointKeys(ps, destIp.Addr())
	if len(dest) == 0 {
		return
	}
	id := types.ProtoToWorkloadEndpointID(&dest[0])
	if sidecar, ok = ps.Endpoints[id]; ok && sidecar.ApplicationLayer != nil {
		return
	}
	sidecar = nil
	return
}

func (c *ALPCheckProvider) EnabledForRequest(ps *policystore.PolicyStore, req *authz.CheckRequest) bool {
	sidecar := GetSidecar(ps, req)
	return (sidecar == nil && c.perHostEnabled) || (sidecar != nil && sidecar.ApplicationLayer.Policy == "Enabled")
}

func (c *ALPCheckProvider) Check(ps *policystore.PolicyStore, req *authz.CheckRequest) (*authz.CheckResponse, error) {
	if fn, ok := c.checksBySubscriptionType[c.subscriptionType]; ok {
		return fn(ps)(req)
	} else {
		resp := &authz.CheckResponse{Status: &status.Status{Code: INTERNAL}}
		err := fmt.Errorf("unknown subscription type: %s", c.subscriptionType)
		return resp, err
	}
}

// default per-host-policy check
func defaultPerHostPolicyCheck(ps *policystore.PolicyStore) checkFn {
	return func(req *authz.CheckRequest) (*authz.CheckResponse, error) {
		resp := &authz.CheckResponse{Status: &status.Status{Code: UNKNOWN}}
		flow := NewCheckRequestToFlowAdapter(req)
		// let checkRequest decide if it's a known source to let it proceed to dest checker; or
		// if it's a known dest to actually run policy
		st := checkRequest(ps, flow)
		resp.Status = &st
		return resp, nil
	}
}

// default per-pod-policy check
func defaultPerPodPolicyCheck(ps *policystore.PolicyStore) checkFn {
	return func(req *authz.CheckRequest) (*authz.CheckResponse, error) {
		resp := &authz.CheckResponse{Status: &status.Status{Code: INTERNAL}}
		if ps.Endpoint == nil {
			return resp, errors.New("endpoint is nil. sync must not have happened yet")
		}

		flow := NewCheckRequestToFlowAdapter(req)
		st := checkStore(ps, ps.Endpoint, rules.RuleDirIngress, flow)
		resp.Status = &st
		return resp, nil
	}
}
