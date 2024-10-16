// Copyright (c) 2024 Tigera, Inc. All rights reserved.

package regolib

type Control struct {
	Name            string            `json:"name"`
	Attributes      ControlAttributes `json:"attributes"`
	Description     string            `json:"description"`
	Remediation     string            `json:"remediation"`
	RulesNames      []string          `json:"rulesNames"`
	LongDescription string            `json:"long_description"`
	Test            string            `json:"test"`
	ControlID       string            `json:"controlID"`
	BaseScore       float64           `json:"baseScore"`
	// Example         string               `json:"example"`
	Category      ControlCategory      `json:"category"`
	ScanningScope ControlScanningScope `json:"scanningScope"`
	ManualTest    string               `json:"manual_test"`
}

type ControlAttributes struct {
	ControlTypeTags []string `json:"controlTypeTags"`
	ActionRequired  string   `json:"actionRequired"`
}

type ControlCategory struct {
	Name        string             `json:"name"`
	SubCategory ControlSubCategory `json:"subCategory"`
}

type ControlSubCategory struct {
	Name string `json:"name"`
}

type ControlScanningScope struct {
	Matches []string `json:"matches"`
}
