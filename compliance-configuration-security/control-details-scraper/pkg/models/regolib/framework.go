// Copyright (c) 2024 Tigera, Inc. All rights reserved.

package regolib

// Root struct to represent the entire JSON object
type Framework struct {
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Attributes     Attribute              `json:"attributes"`
	ScanningScope  FrameworkScanningScope `json:"scanningScope"`
	TypeTags       []string               `json:"typeTags"`
	ActiveControls []ActiveControl        `json:"activeControls"`
	ControlsNames  []string               `json:"controlsNames"`
}

// Attribute struct to capture the attributes object
type Attribute struct {
	Builtin bool `json:"builtin"`
}

// ScanningScope struct to capture the scanningScope object
type FrameworkScanningScope struct {
	Matches []string `json:"matches"`
}

// ActiveControl struct to capture each activeControl in the array
type ActiveControl struct {
	ControlID string `json:"controlID"`
	Patch     Patch  `json:"patch"`
}

// Patch struct to capture the nested patch object inside each activeControl
type Patch struct {
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	LongDescription string   `json:"long_description"`
	Remediation     string   `json:"remediation"`
	References      []string `json:"references"`
	ManualTest      string   `json:"manual_test"`
	ImpactStatement string   `json:"impact_statement"`
	DefaultValue    string   `json:"default_value"`
}
