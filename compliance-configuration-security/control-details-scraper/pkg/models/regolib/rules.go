// Copyright (c) 2024 Tigera, Inc. All rights reserved.

package regolib

import (
	"encoding/json"
	"strings"
)

type Rule struct {
	Name                   string               `json:"name"`
	Attributes             RuleAttribute        `json:"attributes"`
	RuleLanguage           string               `json:"ruleLanguage"`
	Match                  []RuleMatch          `json:"match"`
	DynamicMatch           []RuleMatch          `json:"dynamicMatch"`
	RuleDependencies       []interface{}        `json:"ruleDependencies"` // Empty array suggests a slice of interface{} if type is unknown
	Description            string               `json:"description"`
	Remediation            string               `json:"remediation"`
	RelevantCloudProviders []string             `json:"relevantCloudProviders"`
	RuleQuery              string               `json:"ruleQuery"`
	ControlConfigInputs    []ControlConfigInput `json:"controlConfigInputs"`
}

type RuleAttribute struct {
	HostSensorRule   bool `json:"hostSensorRule"`
	ImageScanRelated bool `json:"imageScanRelated"`
}

type RuleMatch struct {
	ApiGroups   []string `json:"apiGroups"`
	ApiVersions []string `json:"apiVersions"`
	Resources   []string `json:"resources"`
}

type ControlConfigInput struct {
	Path        string `json:"path"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type MaybeBool struct {
	Value bool
}

func (fb *MaybeBool) UnmarshalJSON(data []byte) error {
	// Trim quotes and whitespace, then parse as bool
	trimmedData := strings.Trim(string(data), "\" ")
	err := json.Unmarshal([]byte(trimmedData), &fb.Value)
	if err != nil {
		// Attempt to directly parse the original data as bool
		return json.Unmarshal(data, &fb.Value)
	}
	return nil
}

func (ra *RuleAttribute) UnmarshalJSON(data []byte) error {
	if string(data) == "null" || string(data) == `""` {
		return nil
	}

	type BaseRuleAttribute struct {
		HostSensorRule   MaybeBool `json:"hostSensorRule"`
		ImageScanRelated MaybeBool `json:"imageScanRelated"`
	}
	var aux BaseRuleAttribute

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	ra.HostSensorRule = aux.HostSensorRule.Value

	ra.ImageScanRelated = aux.ImageScanRelated.Value

	return nil
}
