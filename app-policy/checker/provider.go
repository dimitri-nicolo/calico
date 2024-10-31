// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package checker

import (
	"errors"
	"fmt"

	authz "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	log "github.com/sirupsen/logrus"
	"google.golang.org/genproto/googleapis/rpc/status"

	"github.com/projectcalico/calico/app-policy/policystore"
	"github.com/projectcalico/calico/felix/proto"
)

type CheckProvider interface {
	Name() string
	Check(*policystore.PolicyStore, *authz.CheckRequest) (*authz.CheckResponse, error)
}

type checkFn func(req *authz.CheckRequest, sidecar *proto.WorkloadEndpoint) (*authz.CheckResponse, error)
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

func CheckWorkloadEndpoint(ps *policystore.PolicyStore, req *authz.CheckRequest) (*proto.WorkloadEndpoint, error) {
	if log.IsLevelEnabled(log.DebugLevel) {
		log.WithField("ContextExtensions", req.Attributes.ContextExtensions).Debug("Try to get 'podid' from request")
	}
	podid, ok := req.Attributes.ContextExtensions["podid"]
	if !ok {
		return nil, nil
	}
	wledpId := proto.WorkloadEndpointID{
		OrchestratorId: "k8s",
		WorkloadId:     podid,
		EndpointId:     "eth0",
	}
	wledp, ok := ps.Endpoints[wledpId]
	if !ok || wledp.ApplicationLayer == nil {
		return nil, fmt.Errorf("could not find the workload of id: '%s'", podid)
	}
	if log.IsLevelEnabled(log.DebugLevel) {
		log.WithField("WorkloadEdp", wledp).Debugf("WorkloadEdp found")
	}
	return wledp, nil
}

func (c *ALPCheckProvider) Check(ps *policystore.PolicyStore, req *authz.CheckRequest) (*authz.CheckResponse, error) {
	resp := &authz.CheckResponse{Status: &status.Status{Code: INTERNAL}}
	wledp, err := CheckWorkloadEndpoint(ps, req)
	if err != nil {
		return resp, err
	} else if (wledp == nil && !c.perHostEnabled) || (wledp != nil && wledp.ApplicationLayer.Policy != "Enabled") {
		return &authz.CheckResponse{Status: &status.Status{Code: UNKNOWN}}, nil
	}

	if fn, ok := c.checksBySubscriptionType[c.subscriptionType]; ok {
		return fn(ps)(req, wledp)
	} else {
		err := fmt.Errorf("unknown subscription type: %s", c.subscriptionType)
		return resp, err
	}
}

// default per-host-policy check
func defaultPerHostPolicyCheck(ps *policystore.PolicyStore) checkFn {
	return func(req *authz.CheckRequest, sidecar *proto.WorkloadEndpoint) (*authz.CheckResponse, error) {
		resp := &authz.CheckResponse{Status: &status.Status{Code: UNKNOWN}}
		// let checkRequest decide if it's a known source to let it proceed to dest checker; or
		// if it's a known dest to actually run policy
		st := checkRequest(ps, req, sidecar)
		resp.Status = &st
		return resp, nil
	}
}

// default per-pod-policy check
func defaultPerPodPolicyCheck(ps *policystore.PolicyStore) checkFn {
	return func(req *authz.CheckRequest, sidecar *proto.WorkloadEndpoint) (*authz.CheckResponse, error) {
		resp := &authz.CheckResponse{Status: &status.Status{Code: INTERNAL}}
		if ps.Endpoint == nil {
			return resp, errors.New("endpoint is nil. sync must not have happened yet")
		}
		if sidecar == nil {
			sidecar = ps.Endpoint
		}
		st := checkStore(ps, sidecar, req)
		resp.Status = &st
		return resp, nil
	}
}
