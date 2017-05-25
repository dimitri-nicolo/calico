// Copyright (c) 2017 Tigera, Inc. All rights reserved.

package commands

import (
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/projectcalico/libcalico-go/lib/backend"
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

func EvalPolicySelectors(configFile, policyID string) (err error) {

	apiConfig, err := LoadClientConfig(configFile)
	if err != nil {
		log.Fatal("Failed loading client config")
		os.Exit(1)
	}
	bclient, err := backend.NewClient(*apiConfig)
	if err != nil {
		log.Fatal("Failed to create client")
		os.Exit(1)
	}

	kv, err := bclient.Get(model.PolicyKey{Name: policyID, Tier: "default"})
	if err != nil {
		log.Fatal("Failed to get policy")
		os.Exit(1)
	}
	log.Debugf("Policy: %#v", kv)

	policy := kv.Value.(*model.Policy)

	cbs := NewEvalCmd(configFile)
	cbs.AddSelector("applicable endpoints", policy.Selector)
	for name, ruleSet := range map[string][]model.Rule{
		"inbound":  policy.InboundRules,
		"outbound": policy.OutboundRules,
	} {
		for ii, rule := range ruleSet {
			if rule.SrcSelector != "" {
				cbs.AddSelector(fmt.Sprintf("%v %v SrcSelector", name, ii), rule.SrcSelector)
			}
			if rule.DstSelector != "" {
				cbs.AddSelector(fmt.Sprintf("%v %v DstSelector", name, ii), rule.DstSelector)
			}
			if rule.NotSrcSelector != "" {
				cbs.AddSelector(fmt.Sprintf("%v %v NotSrcSelector", name, ii), rule.NotSrcSelector)
			}
			if rule.NotDstSelector != "" {
				cbs.AddSelector(fmt.Sprintf("%v %v NotDstSelector", name, ii), rule.NotDstSelector)
			}
		}
	}

	noopFilter := func(update api.Update) (filterOut bool) {
		return false
	}
	cbs.Start(noopFilter)

	matches := cbs.GetMatches()
	fmt.Printf("Endpoints matching policy %v:\n", policyID)
	for endpoint := range matches {
		fmt.Printf("  %v\n", endpointName(endpoint))
	}
	return
}
