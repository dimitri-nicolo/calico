// Copyright (c) 2017 Tigera, Inc. All rights reserved.

package commands

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/backend/syncersv1/updateprocessors"
	log "github.com/sirupsen/logrus"
)

const APPLICABLE_ENDPOINTS = "applicable endpoints"

func EvalPolicySelectors(configFile, policyName string, hideSelectors, hideRuleMatches bool, outputFormat string) (err error) {

	bclient := GetClient(configFile)
	ctx := context.Background()

	// Get all appropriately named policies from any tier.
	// kvs, err := bclient.List(ctx, model.PolicyListOptions{Name: policyName, Tier: ""}, "")
	// policyName will be of the form <namespace>/<name>
	var ns string
	parts := strings.SplitN(policyName, "/", 2)
	name := parts[0]
	if len(parts) == 2 {
		ns = parts[0]
		name = parts[1]
	}

	npkvs, err := bclient.List(ctx, model.ResourceListOptions{Name: name, Namespace: ns, Kind: apiv3.KindNetworkPolicy}, "")
	if err != nil {
		log.Fatal("Failed to get network policy")
		os.Exit(1)
	}

	gnpkvs, err := bclient.List(ctx, model.ResourceListOptions{Name: name, Namespace: ns, Kind: apiv3.KindGlobalNetworkPolicy}, "")
	if err != nil {
		log.Fatal("Failed to get global network policy")
		os.Exit(1)
	}

	kvs := append(npkvs.KVPairs, gnpkvs.KVPairs...)

	for _, kv := range kvs {
		log.Debugf("Policy: %#v", kv)
		// Convert the V2 Policy object to a V1 Policy object
		// TODO: Get rid of the conversion method when felix is updated to use the v2 data model
		var policy *model.Policy
		switch kv.Value.(type) {
		case *apiv3.NetworkPolicy:
			policy = convertNetworkPolicyV2ToV1Value(kv.Value.(*apiv3.NetworkPolicy).Spec, ns)
		case *apiv3.GlobalNetworkPolicy:
			policy = convertGlobalPolicyV2ToV1Spec(kv.Value.(*apiv3.GlobalNetworkPolicy).Spec)
		}

		cbs := NewEvalCmd(configFile)
		cbs.showSelectors = !hideSelectors
		cbs.AddSelector(APPLICABLE_ENDPOINTS, policy.Selector)
		if !hideRuleMatches {
			cbs.AddPolicyRuleSelectors(policy, "")
		}

		noopFilter := func(update api.Update) (filterOut bool) {
			return false
		}
		cbs.Start(noopFilter)

		matches := map[string][]string{}
		for endpointID, selectors := range cbs.GetMatches() {
			matches[endpointName(endpointID)] = selectors
		}

		switch outputFormat {
		case "yaml":
			EvalPolicySelectorsPrintYAML(policyName, hideRuleMatches, kv, matches)
		case "json":
			EvalPolicySelectorsPrintJSON(policyName, hideRuleMatches, kv, matches)
		case "ps":
			EvalPolicySelectorsPrint(policyName, hideRuleMatches, kv, matches)
		}
	}

	return
}

func EvalPolicySelectorsPrintYAML(policyName string, hideRuleMatches bool, kv *model.KVPair, matches map[string][]string) {
	output := EvalPolicySelectorsPrintObjects(policyName, hideRuleMatches, kv, matches)
	err := printYAML([]OutputList{output})
	if err != nil {
		log.Errorf("Unexpected error printing to YAML: %s", err)
		fmt.Println("Unexpected error printing to YAML")
	}
}

func EvalPolicySelectorsPrintJSON(policyName string, hideRuleMatches bool, kv *model.KVPair, matches map[string][]string) {
	output := EvalPolicySelectorsPrintObjects(policyName, hideRuleMatches, kv, matches)
	err := printJSON([]OutputList{output})
	if err != nil {
		log.Errorf("Unexpected error printing to JSON: %s", err)
		fmt.Println("Unexpected error printing to JSON")
	}
}

func EvalPolicySelectorsPrintObjects(policyName string, hideRuleMatches bool, kv *model.KVPair, matches map[string][]string) OutputList {
	names := []string{}
	for name, _ := range matches {
		names = append(names, name)
	}
	sort.Strings(names)

	// Display tier when non-default.
	var tier string
	switch kv.Value.(type) {
	case *apiv3.NetworkPolicy:
		tier = kv.Value.(*apiv3.NetworkPolicy).Spec.Tier
	case *apiv3.GlobalNetworkPolicy:
		tier = kv.Value.(*apiv3.GlobalNetworkPolicy).Spec.Tier
	}
	tierPrefix := ""
	if tier != "default" && tier != "" {
		tierPrefix = "Tier \"" + tier + "\" "
	}

	output := OutputList{
		Description: fmt.Sprintf("Endpoints that %sPolicy %s applies to and the endpoints that match the policy", tierPrefix, policyName),
	}

	for _, name := range names {
		wepp := NewWorkloadEndpointPrintFromNameString(name)
		if wepp == nil {
			continue
		}

		for _, sel := range matches[name] {
			if strings.HasPrefix(sel, APPLICABLE_ENDPOINTS) {
				// sel is of the form "applicable endpoints; selector <selector>
				// if the selector is hidden, it will be of the form "applicable endpoints"
				if len(sel) == 4 {
					selector := strings.SplitN(sel, " ", 4)[3]
					wepp.Selector = selector[1 : len(selector)-1]
					output.ApplyToEndpoints = append(output.ApplyToEndpoints, wepp)
					break
				}
			} else if !hideRuleMatches {
				wepp.Rules = append(wepp.Rules, NewRulePrintFromSelectorString(sel))
			}
		}

		output.MatchingEndpoints = append(output.MatchingEndpoints, wepp)
	}

	return output
}

func EvalPolicySelectorsPrint(policyName string, hideRuleMatches bool, kv *model.KVPair, matches map[string][]string) {
	names := []string{}
	for name, _ := range matches {
		names = append(names, name)
	}
	sort.Strings(names)

	// Display tier when non-default.
	var tier string
	switch kv.Value.(type) {
	case *apiv3.NetworkPolicy:
		tier = kv.Value.(*apiv3.NetworkPolicy).Spec.Tier
	case *apiv3.GlobalNetworkPolicy:
		tier = kv.Value.(*apiv3.GlobalNetworkPolicy).Spec.Tier
	}
	tierPrefix := ""
	if tier != "default" && tier != "" {
		tierPrefix = "Tier \"" + tier + "\" "
	}

	fmt.Printf("%vPolicy \"%v\" applies to these endpoints:\n", tierPrefix, policyName)
	for _, name := range names {
		for _, sel := range matches[name] {
			if strings.HasPrefix(sel, APPLICABLE_ENDPOINTS) {
				fmt.Printf("  %v%v\n", name, strings.TrimPrefix(sel, APPLICABLE_ENDPOINTS))
				break
			}
		}
	}

	if !hideRuleMatches {
		fmt.Printf("\nEndpoints matching %vPolicy \"%v\" rules:\n", tierPrefix, policyName)
		for _, name := range names {
			endpointPrefix := fmt.Sprintf("  %v\n", name)
			sort.Strings(matches[name])
			for _, sel := range matches[name] {
				if !strings.HasPrefix(sel, APPLICABLE_ENDPOINTS) {
					fmt.Printf("%v    %v\n", endpointPrefix, sel)
					endpointPrefix = ""
				}
			}
		}
	}
}

// These are slightly modified copies (they do not return an error) of
// the conversion methods in libcalico-go. Copying it here so that we
// do not have more work later to keep libcalico-go-private in sync
// with libcalico-go.
// TODO: Delete this when the Felix syncer uses the v2 model and the
// referencing logic is changed.
func convertGlobalPolicyV2ToV1Spec(spec apiv3.GlobalNetworkPolicySpec) *model.Policy {
	v1value := &model.Policy{
		Order:          spec.Order,
		InboundRules:   updateprocessors.RulesAPIV2ToBackend(spec.Ingress, ""),
		OutboundRules:  updateprocessors.RulesAPIV2ToBackend(spec.Egress, ""),
		Selector:       spec.Selector,
		Types:          policyTypesAPIV2ToBackend(spec.Types),
		DoNotTrack:     spec.DoNotTrack,
		PreDNAT:        spec.PreDNAT,
		ApplyOnForward: spec.ApplyOnForward,
	}

	return v1value
}

func convertNetworkPolicyV2ToV1Value(spec apiv3.NetworkPolicySpec, ns string) *model.Policy {
	// If this policy is namespaced, then add a namespace selector.
	selector := spec.Selector
	if ns != "" {
		nsSelector := fmt.Sprintf("%s == '%s'", apiv3.LabelNamespace, ns)
		if selector == "" {
			selector = nsSelector
		} else {
			selector = fmt.Sprintf("(%s) && %s", selector, nsSelector)
		}
	}

	v1value := &model.Policy{
		Order:          spec.Order,
		InboundRules:   updateprocessors.RulesAPIV2ToBackend(spec.Ingress, ns),
		OutboundRules:  updateprocessors.RulesAPIV2ToBackend(spec.Egress, ns),
		Selector:       selector,
		Types:          policyTypesAPIV2ToBackend(spec.Types),
		ApplyOnForward: true,
	}

	return v1value
}

// Copy of the function in libcalico-go.
// TODO: Remove this when the Felix syncer uses the v2 model and the
// referencing logic is changed
func policyTypesAPIV2ToBackend(ptypes []apiv3.PolicyType) []string {
	var v1ptypes []string
	for _, ptype := range ptypes {
		v1ptypes = append(v1ptypes, strings.ToLower(string(ptype)))
	}
	return v1ptypes
}
