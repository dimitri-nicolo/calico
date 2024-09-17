// Copyright (c) 2018-2021 Tigera, Inc. All rights reserved.

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
	"errors"
	"fmt"
	"strings"

	"github.com/projectcalico/calico/app-policy/policystore"
	"github.com/projectcalico/calico/app-policy/types"
	"github.com/projectcalico/calico/felix/ip"
	"github.com/projectcalico/calico/felix/proto"
	"github.com/projectcalico/calico/felix/tproxydefs"
	"github.com/projectcalico/calico/libcalico-go/lib/logutils"

	authz "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	log "github.com/sirupsen/logrus"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/genproto/googleapis/rpc/status"
)

var OK = int32(code.Code_OK)
var PERMISSION_DENIED = int32(code.Code_PERMISSION_DENIED)
var UNAVAILABLE = int32(code.Code_UNAVAILABLE)
var INVALID_ARGUMENT = int32(code.Code_INVALID_ARGUMENT)
var INTERNAL = int32(code.Code_INTERNAL)
var UNKNOWN = int32(code.Code_UNKNOWN)

// Action is an enumeration of actions a policy rule can take if it is matched.
type Action int

const (
	ALLOW Action = iota
	DENY
	LOG
	PASS
	NO_MATCH // Indicates policy did not match request. Cannot be assigned to rule.
)

var (
	rlog = logutils.NewRateLimitedLogger()
)

func LookupEndpointsFromRequest(store *policystore.PolicyStore, req *authz.CheckRequest, sidecar *proto.WorkloadEndpoint) (source, destination []*proto.WorkloadEndpoint, err error) {
	log.Debugf("extracting endpoints from request %s", req.String())

	// Extract source and destination IP addresses if possible:
	requestAttributes := req.GetAttributes()
	if requestAttributes == nil {
		err = errors.New("cannot process specified request data")
		return
	}

	// map destination first
	if sidecar != nil {
		destination = []*proto.WorkloadEndpoint{sidecar}
	} else if addr, port, ok := addrPortFromPeer(requestAttributes.Destination); ok {
		log.Debugf("found destination address we would like to match: [%v:%v]", addr, port)
		destinationIp, err := ip.ParseCIDROrIP(addr)
		if err != nil {
			rlog.Warnf("cannot process addr %v: %v", addr, err)
		} else {
			log.Debug("trying to match destination: ", destinationIp)
			destination = ipToEndpoints(store, destinationIp.Addr())
		}
	}

	// map source next
	if addr, port, ok := addrPortFromPeer(requestAttributes.Source); ok {
		log.Debugf("found source address we would like to match: [%v:%v]", addr, port)
		sourceIp, err := ip.ParseCIDROrIP(addr)
		if err != nil {
			rlog.Warnf("cannot process addr %v: %v", addr, err)
		} else {
			log.Debug("trying to match source: ", sourceIp)
			source = ipToEndpoints(store, sourceIp.Addr())
		}
	}

	return
}

func LookupEndpointKeysFromSrcDst(store *policystore.PolicyStore, src, dst string) (source, destination []proto.WorkloadEndpointID, err error) {
	if store == nil {
		// can't lookup anything without a store
		return source, destination, types.ErrNoStore{}
	}

	// map destination first
	destinationIp, err := ip.ParseCIDROrIP(dst)
	if err != nil {
		rlog.Warnf("cannot process addr %s: %v", dst, err)
	} else {
		log.Debug("trying to match destination: ", destinationIp)
		destination = ipToEndpointKeys(store, destinationIp.Addr())
	}

	// map source next
	sourceIp, err := ip.ParseCIDROrIP(src)
	if err != nil {
		rlog.Warnf("cannot process addr %s: %v", src, err)
	} else {
		log.Debug("trying to match source: ", sourceIp)
		source = ipToEndpointKeys(store, sourceIp.Addr())
	}

	return
}

func addrPortFromPeer(peer *authz.AttributeContext_Peer) (addr string, port uint32, ok bool) {
	if peer == nil {
		return
	}

	addr = peer.GetAddress().GetSocketAddress().GetAddress()
	port = peer.GetAddress().GetSocketAddress().GetPortValue()

	return addr, port, true
}

func ipToEndpoints(store *policystore.PolicyStore, addr ip.Addr) []*proto.WorkloadEndpoint {
	return store.IPToIndexes.Get(addr)
}

func ipToEndpointKeys(store *policystore.PolicyStore, addr ip.Addr) []proto.WorkloadEndpointID {
	return store.IPToIndexes.Keys(addr)
}

func checkRequest(store *policystore.PolicyStore, req *authz.CheckRequest, sidecar *proto.WorkloadEndpoint) status.Status {
	src, dst, err := LookupEndpointsFromRequest(store, req, sidecar)
	if err != nil {
		return status.Status{Code: INTERNAL, Message: fmt.Sprintf("endpoint lookup error: %v", err)}
	}
	log.Debugf("Found endpoints from request [src: %v, dst: %v]", src, dst)

	if len(dst) > 0 {
		if sidecar == nil {
			alpIPset, ok := store.IPSetByID[tproxydefs.ApplicationLayerPolicyIPSet]
			if !ok {
				return status.Status{Code: UNKNOWN, Message: "cannot process ALP yet"}
			}

			reqAddr := req.Attributes.Destination.Address
			if !alpIPset.ContainsAddress(reqAddr) {
				return status.Status{Code: UNKNOWN, Message: "ALP not enabled for this request destination"}
			}
		}
		// Destination is local workload, apply its ingress policy.
		// possible there's multiple weps for an ip.
		// let's run through all of them and apply its ingress policy
		for _, ds := range dst {
			if s := checkStore(store, ds, req); s.Code != OK {
				// stop looping on first non-OK status
				return status.Status{
					Code:    s.Code,
					Message: s.Message,
					Details: s.Details,
				}
			}
		}
		// all local destinations aren't getting denied by policy
		// let traffic through
		return status.Status{Code: OK}
	}

	if len(src) > 0 {
		// Source is local but destination is not.  We assume that the traffic reached Envoy as
		// a false positive; for example, a workload connecting out to an L7-annotated service.
		// Let it through; it should be handled by the remote Envoy/Dikastes.

		// NB: in the future we can process egress rules here e.g.
		/*
			if src != nil { // originating node: so process src traffic
				return checkStore(store, src, req) // TODO need flag to reverse policy logic
			}
		*/

		// possible future iteration: apply src egress policy
		// return checkStore(store, src, req, withEgressProcessing{})
		log.Debugf("allowing traffic to continue to its destination hop/next processing leg. (req: %s)", req.String())

		return status.Status{Code: OK, Message: fmt.Sprintf("request %s passing through", req.String())}
	}

	// Don't know source or dest.  Why was this packet sent to us?
	// Assume that we're out of sync and reject it.
	log.Debug("encountered invalid ext_authz request case")
	return status.Status{Code: UNKNOWN} // return unknown so that next check provider can continue processing
}

// checkStore applies the tiered policy plus any config based corrections and returns OK if the check passes or
// PERMISSION_DENIED if the check fails.
func checkStore(store *policystore.PolicyStore, ep *proto.WorkloadEndpoint, req *authz.CheckRequest) (s status.Status) {
	// Check using the configured policy
	s = checkTiers(store, ep, req)

	// If the result from the policy check will result in a drop, check if we are overriding the drop
	// action, and if so modify the result.
	if s.Code != OK {
		switch store.DropActionOverride {
		case policystore.DROP, policystore.LOG_AND_DROP:
			// Leave action unchanged, packet will be dropped.
		case policystore.ACCEPT, policystore.LOG_AND_ACCEPT:
			// Convert action that would result in a drop into an accept.
			rlog.Info("Invoking DropActionOverride: Converting drop action to allow")
			s.Code = OK
		}
	}
	return
}

// checkTiers applies the tiered policy in the given store and returns OK if the check passes, or PERMISSION_DENIED if
// the check fails. Note, if no policy matches, the default is PERMISSION_DENIED.
func checkTiers(store *policystore.PolicyStore, ep *proto.WorkloadEndpoint, req *authz.CheckRequest) (s status.Status) {
	s = status.Status{Code: PERMISSION_DENIED}
	// nothing to check. return early
	if ep == nil {
		return
	}
	reqCache, err := NewRequestCache(store, req)
	if err != nil {
		rlog.Errorf("Failed to init requestCache: %v", err)
		return
	}
	defer func() {
		if r := recover(); r != nil {
			// Recover from the panic if we know what it is and we know what to do with it.
			if v, ok := r.(*InvalidDataFromDataPlane); ok {
				log.Debug("encountered InvalidFromDataPlane: ", v.string)
				s = status.Status{Code: INVALID_ARGUMENT}
			} else {
				panic(r)
			}
		}
	}()
	for _, tier := range ep.Tiers {
		log.Debug("Checking policy tier", tier.GetName())
		policies := tier.IngressPolicies
		if len(policies) == 0 {
			// No ingress policy in this tier, move on to next one.
			continue
		} else {
			log.Debug("policies: ", policies)
		}

		action := NO_MATCH
	Policy:
		for i, name := range policies {
			pID := proto.PolicyID{Tier: tier.GetName(), Name: name}
			policy := store.PolicyByID[pID]
			action = checkPolicy(policy, reqCache)
			log.Debugf("Policy checked (ordinal=%d, profileId=%v, action=%v)", i, pID, action)
			switch action {
			case NO_MATCH:
				continue Policy
			// If the Policy matches, end evaluation (skipping profiles, if any)
			case ALLOW:
				s.Code = OK
				return
			case DENY:
				s.Code = PERMISSION_DENIED
				return
			case PASS:
				// Pass means end evaluation of policies and proceed to next tier (or profiles), if any.
				break Policy
			case LOG:
				log.Debug("policy should never return LOG action")
				s.Code = INVALID_ARGUMENT
				return
			}
		}
		// Done evaluating policies in the tier. If no policy rules have matched, there is an implicit default deny
		// at the end of the tier.
		if action == NO_MATCH {
			log.Debug("No policy matched. Tier default DENY applies.")
			s.Code = PERMISSION_DENIED
			return
		}
	}
	// If we reach here, there were either no tiers, or a policy PASSed the request.
	if len(ep.ProfileIds) > 0 {
		for i, name := range ep.ProfileIds {
			pID := proto.ProfileID{Name: name}
			profile := store.ProfileByID[pID]
			action := checkProfile(profile, reqCache)
			log.Debugf("Profile checked (ordinal=%d, profileId=%v, action=%v)", i, pID, action)
			switch action {
			case NO_MATCH:
				continue
			case ALLOW:
				s.Code = OK
				return
			case DENY, PASS:
				s.Code = PERMISSION_DENIED
				return
			case LOG:
				log.Debug("profile should never return LOG action")
				s.Code = INVALID_ARGUMENT
				return
			}
		}
	} else {
		log.Debug("0 active profiles, deny request.")
		s.Code = PERMISSION_DENIED
	}
	return
}

// checkPolicy checks if the policy matches the request data, and returns the action.
func checkPolicy(policy *proto.Policy, req *requestCache) (action Action) {
	if policy == nil {
		return Action(INTERNAL)
	}

	// Note that we support only inbound policy.
	return checkRules(policy.InboundRules, req, policy.Namespace)
}

func checkProfile(profile *proto.Profile, req *requestCache) (action Action) {
	// profiles or profile updates might not be available yet. use internal here
	if profile == nil {
		return Action(INTERNAL)
	}

	return checkRules(profile.InboundRules, req, "")
}

func checkRules(rules []*proto.Rule, req *requestCache, policyNamespace string) (action Action) {
	for _, r := range rules {
		if match(r, req, policyNamespace) {
			log.Debugf("checkRules: Rule matched %v", *r)
			a := actionFromString(r.Action)
			if a != LOG {
				// We don't support actually logging requests, but if we hit a LOG action, we should
				// continue processing rules.
				return a
			}
		}
	}
	return NO_MATCH
}

// actionFromString converts a string action name, like "allow" into an Action.
func actionFromString(s string) Action {
	// Felix currently passes us the v1 resource types where the "pass" action is called "next-tier".
	// Here we support both the v1 and v3 action names.
	m := map[string]Action{
		"allow":     ALLOW,
		"deny":      DENY,
		"pass":      PASS,
		"next-tier": PASS,
		"log":       LOG,
	}
	a, found := m[strings.ToLower(s)]
	if !found {
		log.Errorf("Got bad action %v", s)
		panic(&InvalidDataFromDataPlane{"got bad action"})
	}
	return a
}
