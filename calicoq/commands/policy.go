// Copyright (c) 2017 Tigera, Inc. All rights reserved.

package commands

import (
	"fmt"
	"os"
	"sort"

	log "github.com/Sirupsen/logrus"
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

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
	cbs.AddSelector("applicable endpoints", policy.Selector)
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

	fmt.Printf("Endpoints matching policy %v:\n", policyID)
	for _, name := range names {
		fmt.Printf("  %v\n", name)
		sort.Strings(matches[name])
		for _, sel := range matches[name] {
			fmt.Printf("    %v\n", sel)
		}
	}
	return
}
