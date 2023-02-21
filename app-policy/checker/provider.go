// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package checker

import (
	"errors"
	"fmt"

	authz "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"

	"google.golang.org/genproto/googleapis/rpc/status"

	"github.com/projectcalico/calico/app-policy/policystore"
	"github.com/projectcalico/calico/felix/tproxydefs"
)

type CheckProvider interface {
	Name() string
	Check(*policystore.PolicyStore, *authz.CheckRequest) (*authz.CheckResponse, error)
}

type checkFn func(req *authz.CheckRequest) (*authz.CheckResponse, error)
type checkWithStore func(*policystore.PolicyStore) checkFn

type ALPCheckProvider struct {
	subscriptionType         string
	checksBySubscriptionType map[string]checkWithStore
}

type ALPCheckProviderOption func(*ALPCheckProvider)

func WithALPCheckProviderCheckFn(subscriptionType string, fn checkWithStore) ALPCheckProviderOption {
	return func(ap *ALPCheckProvider) {
		ap.checksBySubscriptionType[subscriptionType] = fn
	}
}

func NewALPCheckProvider(subscriptionType string, opts ...ALPCheckProviderOption) CheckProvider {
	c := &ALPCheckProvider{
		subscriptionType,
		map[string]checkWithStore{
			"per-host-policies": defaultPerHostPolicyCheck,
			"per-pod-policies":  defaultPerPodPolicyCheck,
		},
	}

	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *ALPCheckProvider) Name() string {
	return "application-layer-policy"
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
		if alpIPSet, ok := ps.IPSetByID[tproxydefs.ApplicationLayerPolicyIPSet]; ok &&
			// ipset exists.. check if src or dest is in ALP ipset
			(alpIPSet.ContainsAddress(req.Attributes.Source.Address) ||
				alpIPSet.ContainsAddress(req.Attributes.Destination.Address)) {
			// traffic described in request needs to go through an ALP check.
			st := checkRequest(ps, req)
			resp.Status = &st
			return resp, nil
		}

		// traffic described in request doesn't need to go through ALP check; or
		// traffic described in request needs to continue to WAF; or
		// traffic described in request is plaintext;
		// or sent here by mistake.
		//
		// in any case, let it continue to next check
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
		st := checkStore(ps, ps.Endpoint, req)
		resp.Status = &st
		return resp, nil
	}
}
