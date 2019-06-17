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

	"github.com/projectcalico/app-policy/policystore"
	"github.com/projectcalico/app-policy/statscache"

	"github.com/envoyproxy/data-plane-api/envoy/api/v2/core"
	authz "github.com/envoyproxy/data-plane-api/envoy/service/auth/v2"
	"github.com/gogo/googleapis/google/rpc"
	log "github.com/sirupsen/logrus"
)

type authServer struct {
	stores  <-chan *policystore.PolicyStore
	dpStats chan<- statscache.DPStats
	Store   *policystore.PolicyStore
}

// NewServer creates a new authServer and returns a pointer to it.
func NewServer(ctx context.Context, stores <-chan *policystore.PolicyStore, dpStats chan<- statscache.DPStats) *authServer {
	s := &authServer{stores: stores, dpStats: dpStats}
	go s.updateStores(ctx)
	return s
}

// Check applies the currently loaded policy to a network request and renders a policy decision.
func (as *authServer) Check(ctx context.Context, req *authz.CheckRequest) (*authz.CheckResponse, error) {
	log.WithFields(log.Fields{
		"context":         ctx,
		"Req.Method":      req.GetAttributes().GetRequest().GetHttp().GetMethod(),
		"Req.Path":        req.GetAttributes().GetRequest().GetHttp().GetPath(),
		"Req.Protocol":    req.GetAttributes().GetRequest().GetHttp().GetProtocol(),
		"Req.Source":      req.GetAttributes().GetSource(),
		"Req.Destination": req.GetAttributes().GetDestination(),
	}).Debug("Check start")
	resp := authz.CheckResponse{Status: &rpc.Status{Code: INTERNAL}}
	var st rpc.Status
	var statsEnabledForAllowed, statsEnabledForDenied bool

	// Ensure that we only access as.Store once per Check call. The authServer can be updated to point to a different
	// store asynchronously with this call, so we use a local variable to reference the PolicyStore for the duration of
	// this call for consistency.
	store := as.Store
	if store == nil {
		log.Warn("Check request before synchronized to Policy, failing.")
		resp.Status.Code = UNAVAILABLE
		return &resp, nil
	}
	store.Read(func(ps *policystore.PolicyStore) {
		st = checkStore(ps, req)
		statsEnabledForAllowed = ps.DataplaneStatsEnabledForAllowed
		statsEnabledForDenied = ps.DataplaneStatsEnabledForDenied
	})

	// If we are reporting stats for allowed and response is OK, or we are reporting stats for denied and
	// the response is not OK then report the stats.
	if (statsEnabledForAllowed && st.Code == OK) || (statsEnabledForDenied && st.Code != OK) {
		as.reportStats(ctx, st, req)
	}

	resp.Status = &st
	log.WithFields(log.Fields{
		"Req.Method":      req.GetAttributes().GetRequest().GetHttp().GetMethod(),
		"Req.Path":        req.GetAttributes().GetRequest().GetHttp().GetPath(),
		"Req.Protocol":    req.GetAttributes().GetRequest().GetHttp().GetProtocol(),
		"Req.Source":      req.GetAttributes().GetSource(),
		"Req.Destination": req.GetAttributes().GetDestination(),
		"Response":        resp,
	}).Debug("Check complete")
	return &resp, nil
}

// reportStats creates a statistics for this request and reports it to the client.
func (as *authServer) reportStats(ctx context.Context, st rpc.Status, req *authz.CheckRequest) {
	if req.GetAttributes().GetDestination().GetAddress().GetSocketAddress().GetProtocol() != core.TCP {
		log.Debug("No statistics to report for non-TCP request")
		return
	}
	if req.GetAttributes().GetRequest().GetHttp() == nil {
		log.Debug("No statistics to report for non-HTTP request")
		return
	}

	dpStats := statscache.DPStats{
		Tuple: statscache.Tuple{
			SrcIp:    req.GetAttributes().GetSource().GetAddress().GetSocketAddress().GetAddress(),
			DstIp:    req.GetAttributes().GetDestination().GetAddress().GetSocketAddress().GetAddress(),
			SrcPort:  int32(req.GetAttributes().GetSource().GetAddress().GetSocketAddress().GetPortValue()),
			DstPort:  int32(req.GetAttributes().GetDestination().GetAddress().GetSocketAddress().GetPortValue()),
			Protocol: "TCP",
		},
	}

	if st.Code == OK {
		dpStats.Values.HTTPRequestsAllowed = 1
	} else {
		dpStats.Values.HTTPRequestsDenied = 1
	}

	select {
	case as.dpStats <- dpStats:
	case <-ctx.Done():
	}
}

// updateStores pulls PolicyStores off the channel and assigns them.
func (as *authServer) updateStores(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case as.Store = <-as.stores:
			// Variable assignment is atomic, so this is threadsafe as long as each check call accesses authServer.Store
			// only once.
			log.Info("Switching to new in-sync policy store.")
			continue
		}
	}
}
