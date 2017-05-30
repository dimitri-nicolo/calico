// Copyright (c) 2017 Tigera, Inc. All rights reserved.

package commands

import (
	"fmt"
	"os"
	"sort"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

const APPLICABLE_ENDPOINTS = "applicable endpoints"

func EvalPolicySelectors(configFile, policyID string, hideSelectors, hideRuleMatches bool) (err error) {

	bclient := GetClient(configFile)

	kv, err := bclient.Get(model.PolicyKey{Name: policyID, Tier: "default"})
	if err != nil {
		log.Fatal("Failed to get policy")
		os.Exit(1)
	}
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
	names := []string{}
	for name, _ := range matches {
		names = append(names, name)
	}
	sort.Strings(names)

	fmt.Printf("Policy %v applies to these endpoints:\n", policyID)
	for _, name := range names {
		for _, sel := range matches[name] {
			if strings.HasPrefix(sel, APPLICABLE_ENDPOINTS) {
				fmt.Printf("  %v%v\n", name, strings.TrimPrefix(sel, APPLICABLE_ENDPOINTS))
				break
			}
		}
	}

	if !hideRuleMatches {
		fmt.Printf("\nEndpoints matching policy %v rules:\n", policyID)
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

	return
}
