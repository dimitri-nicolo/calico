// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package checker

import (
	"errors"
	"fmt"

	authz "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"

	"google.golang.org/genproto/googleapis/rpc/status"

	"github.com/projectcalico/calico/app-policy/policystore"
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
		// let checkRequest decide if it's a known source to let it proceed to dest checker; or
		// if it's a known dest to actually run policy
		st := checkRequest(ps, req)
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
		st := checkStore(ps, ps.Endpoint, req)
		resp.Status = &st
		return resp, nil
	}
}
