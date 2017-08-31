// Copyright (c) 2017 Tigera, Inc. All rights reserved.

package commands

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	log "github.com/sirupsen/logrus"
)

const APPLICABLE_ENDPOINTS = "applicable endpoints"

func EvalPolicySelectors(configFile, policyName string, hideSelectors, hideRuleMatches bool, outputFormat string) (err error) {

	bclient := GetClient(configFile)

	// Get all appropriately named policies from any tier.
	kvs, err := bclient.List(model.PolicyListOptions{Name: policyName, Tier: ""})
	if err != nil {
		log.Fatal("Failed to get policy")
		os.Exit(1)
	}

	for _, kv := range kvs {
		log.Debugf("Policy: %#v", kv)
		policy := kv.Value.(*model.Policy)

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
	tierPrefix := ""
	if kv.Key.(model.PolicyKey).Tier != "default" {
		tierPrefix = "Tier \"" + kv.Key.(model.PolicyKey).Tier + "\" "
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
			}
		}
	}

	if !hideRuleMatches {
		for _, name := range names {
			wepp := NewWorkloadEndpointPrintFromNameString(name)
			if wepp == nil {
				continue
			}

			sort.Strings(matches[name])
			for _, sel := range matches[name] {
				if !strings.HasPrefix(sel, APPLICABLE_ENDPOINTS) {
					wepp.Rules = append(wepp.Rules, NewRulePrintFromSelectorString(sel))
				}
			}
			output.MatchingEndpoints = append(output.MatchingEndpoints, wepp)
		}
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
	tierPrefix := ""
	if kv.Key.(model.PolicyKey).Tier != "default" {
		tierPrefix = "Tier \"" + kv.Key.(model.PolicyKey).Tier + "\" "
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
