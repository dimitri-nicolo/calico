// Copyright (c) 2017 Tigera, Inc. All rights reserved.

package commands

import (
	"fmt"
	"strconv"
	"strings"

	// TODO: Check glide for these and if they need to be private
	"github.com/projectcalico/go-json/json"
	"github.com/projectcalico/go-yaml-wrapper"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	log "github.com/sirupsen/logrus"
)

func printYAML(outputs []OutputList) error {
	if output, err := yaml.Marshal(outputs); err != nil {
		return err
	} else {
		fmt.Printf("%s", string(output))
	}
	return nil
}

func printJSON(outputs []OutputList) error {
	if output, err := json.MarshalIndent(outputs, "", "  "); err != nil {
		return err
	} else {
		fmt.Printf("%s\n", string(output))
	}
	return nil
}

type OutputList struct {
	Description       string                   `json:"description"`
	Endpoints         []*WorkloadEndpointPrint `json:"endpoints,omitempty"`
	ApplyToEndpoints  []*WorkloadEndpointPrint `json:"policy_applies_to_endpoints,omitempty"`
	MatchingEndpoints []*WorkloadEndpointPrint `json:"endpoints_match_policy,omitempty"`
}

type WorkloadEndpointPrint struct {
	// Identifiers    EndpointIdentifiers `json:"identifiers"`
	Node           string         `json:"node,omitempty"`
	Orchestrator   string         `json:"orchestrator,omitempty"`
	Workload       string         `json:"workload,omitempty"`
	Name           string         `json:"name,omitempty"`
	UntrackedTiers []TierPrint    `json:"untracked_tiers,omitempty"`
	Tiers          []TierPrint    `json:"policy_tiers,omitempty"`
	Policies       []PolicyPrint  `json:"policies,omitempty"`
	Profiles       []ProfilePrint `json:"profiles,omitempty"`
	Rules          []RulePrint    `json:"rule-matches,omitempty"`
	Selector       string         `json:"selector,omitempty"`
}

type EndpointIdentifiers struct {
	Node         string `json:"node,omitempty"`
	Orchestrator string `json:"orchestrator,omitempty"`
	Workload     string `json:"workload,omitempty"`
	Name         string `json:"name,omitempty"`
}

func NewWorkloadEndpointPrintFromEndpointDatum(epd endpointDatum) *WorkloadEndpointPrint {
	return NewWorkloadEndpointPrintFromKey(epd.epID)
}

func NewWorkloadEndpointPrintFromKey(key interface{}) *WorkloadEndpointPrint {
	/*
		idents := EndpointIdentifiers{}
		switch epID := key.(type) {
		case model.WorkloadEndpointKey:
			idents.Node = epID.Hostname
			idents.Orchestrator = epID.OrchestratorID
			idents.Workload = epID.WorkloadID
			idents.Name = epID.EndpointID
		case model.HostEndpointKey:
			idents.Name = epID.EndpointID
		}
		return &WorkloadEndpointPrint{
			Identifiers: idents,
		}
	*/
	wepp := &WorkloadEndpointPrint{}
	switch epID := key.(type) {
	case model.WorkloadEndpointKey:
		wepp.Node = epID.Hostname
		wepp.Orchestrator = epID.OrchestratorID
		wepp.Workload = epID.WorkloadID
		wepp.Name = epID.EndpointID
	case model.HostEndpointKey:
		wepp.Name = epID.EndpointID
	}
	return wepp
}

func NewWorkloadEndpointPrintFromNameString(name string) *WorkloadEndpointPrint {
	// name is of the form "Workload endpoint <node>/<orchestrator>/<workload>/<name>
	// sel is of the form "applicable endpoints; selector <selector>
	/*
		idents := EndpointIdentifiers{}
		endpointStrings := strings.Split(name, " ")
		if len(endpointStrings) != 3 {
			log.Errorf("Workload name is not in the \"Workload endpoint <node>/<orchestrator>/<workload>/<name>\" format: %s", name)
			return &WorkloadEndpointPrint{}
		}

		endpointIdents := strings.Split(endpointStrings[2], "/")
		idents.Node = endpointIdents[0]
		idents.Orchestrator = endpointIdents[1]
		idents.Workload = endpointIdents[2]
		idents.Name = endpointIdents[3]

		return &WorkloadEndpointPrint{
			Identifiers: idents,
		}
	*/
	wepp := &WorkloadEndpointPrint{}
	endpointStrings := strings.Split(name, " ")
	if len(endpointStrings) != 3 || endpointStrings[0] != "Workload" || endpointStrings[1] != "endpoint" {
		log.Errorf("Workload endpoint name is not in the \"Workload endpoint <node>/<orchestrator>/<workload>/<name>\" format: %s", name)
		return wepp
	}

	endpointIdents := strings.Split(endpointStrings[2], "/")
	if len(endpointIdents) != 4 {
		log.Errorf("Workload endpoint name does not have its identifiers <node>/<orchestrator>/<workload>/<name> separated by \"/\": %s", name)
		return wepp
	}
	wepp.Node = endpointIdents[0]
	wepp.Orchestrator = endpointIdents[1]
	wepp.Workload = endpointIdents[2]
	wepp.Name = endpointIdents[3]

	return wepp
}

type PolicyPrint struct {
	Name      string `json:"name,omitempty"`
	Order     string `json:"order,omitempty"`
	Selector  string `json:"selector,omitempty"`
	TierName  string `json:"tier_name,omitempty"`
	TierOrder string `json:"tier_order,omitempty"`
}

type TierPrint struct {
	Name     string        `json:"name"`
	Order    string        `json:"order"`
	Policies []PolicyPrint `json:"policies"`
}

type ProfilePrint struct {
	Name string `json:"name"`
}

type RulePrint struct {
	PolicyName   string `json:"policy_name,omitempty"`
	TierName     string `json:"tier_name,omitempty"`
	Direction    string `json:"direction"`
	SelectorType string `json:"selector_type"`
	Order        int    `json:"order"`
	Selector     string `json:"selector"`
}

func NewRulePrintFromMatchString(match string) RulePrint {
	// Takes in a policy string formatted by EvalCmd.GetMatches for policies of the format:
	// Policy "<policy name>" <inbound/outbound> rule <rule number> <source/destination> match; selector "<selector>"
	rp := RulePrint{}

	// Split by spaces to extract the information
	info := strings.SplitN(match, " ", 9)

	// TODO: Figure out what the right error handling would be here
	// TODO: Refactor this and its callers eventually to use the PolicyKey objects to get Tier names
	if info[0] != "Policy" || info[3] != "rule" || info[6] != "match;" || info[7] != "selector" {
		log.Errorf("Match string is not in the format: Policy \"policy name>\" <inbound/outbound> rule <rule number> <source/destination> match; selector \"<selector>\": %s", match)
		return rp
	}

	var err error
	rp.PolicyName = info[1][1 : len(info[1])-1]
	rp.Direction = info[2]
	rp.SelectorType = info[5]
	rp.Selector = info[8][1 : len(info[8])-1]
	rp.Order, err = strconv.Atoi(info[4])
	if err != nil {
		log.Errorf("Unable to create Policy Rule from match string: %s", err)
	}

	return rp
}

func NewRulePrintFromSelectorString(selector string) RulePrint {
	// Takes in a policy string formatted by EvalCmd.GetMatches of the format:
	// <direction> rule <rule number> <selector type> match; selector "<selector>"
	rp := RulePrint{}

	// Split by spaces to extract the information
	info := strings.SplitN(selector, " ", 7)

	// TODO: Figure out what the right error handling would be here
	if strings.HasPrefix(selector, APPLICABLE_ENDPOINTS) || len(info) != 7 || info[1] != "rule" || info[4] != "match;" || info[5] != "selector" {
		log.Errorf("Selector string not in the format <direction> rule <rule number> <selector type> match; selector \"<selector>\": %s", selector)
		return rp
	}

	var err error
	rp.Direction = info[0]
	rp.SelectorType = info[3]
	rp.Selector = info[6][1 : len(info[6])-1]
	rp.Order, err = strconv.Atoi(info[2])
	if err != nil {
		log.Errorf("Unable to create Policy Rule from match string: %s", err)
	}

	return rp
}
