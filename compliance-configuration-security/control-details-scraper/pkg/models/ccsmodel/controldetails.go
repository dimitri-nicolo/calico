// Copyright (c) 2024 Tigera, Inc. All rights reserved.

package ccsmodel

// TODO move this in the ccs api package
type CCSControlDetails struct {

	// ControlDetailsID is a unique identifier for the control details. We could use a format that has the
	// controlID as a prefix and then appends a suffix like “details” to it, i.e. C-XXXX-details where
	// XXXX is a number. https://hub.armosec.io/docs/controls
	ControlDetailsID string `json:"controlDetailsID"`

	// Name is the name of the control given by kubescape. It’s of the format <C-XXXX - Text>. Ex. “C-0009 - Resource limits”
	Name string `json:"name"`

	// Description is a section that outlines the exact details of the problem this control is intended to detect.
	Description string `json:"description"`

	// Category is the category of this control (as assigned by kubescape).
	Category string `json:"category"`

	// SubCategory is the sub category (if any) of this control (as assigned by kubescape).
	SubCategory string `json:"subCategory"`

	// Remediation is a section that outlines steps for how to fix the underlying problem this control tests for.
	Remediation string `json:"remediation"`

	// Frameworks is a section that outlines the in-built frameworks this control is associated with.
	Frameworks []string `json:"framework"`

	// Prerequisites is a section that outlines the prerequisites requirements before this control can be executed.
	Prerequisites *Prerequisite `json:"prerequisites"`

	// Severity is a section that outlines the Severity rating of this control.
	Severity float64 `json:"severity"`

	// FrameworkOverride section stores the framework specific override for the control.
	FrameworkOverride []*FrameworkOverride `json:"frameworkOverrides"`

	// RelatedResources is a section that outlines the prerequisites requirements before this control can be executed.
	RelatedResources []string `json:"relatedResources"`

	// TestCriteria is a section that outlines what exactly this control tests for.
	TestCriteria string `json:"testCriteria"`

	// ManualCheck is a section that outlines steps for how to perform the control tests manually.
	ManualCheck string `json:"manualCheck"`

	// Configuration is a section that outlines any customizations (if any) this control supports.
	Configuration []*ControlConfigInput `json:"configuration"`

	// Example is a section that outlines examples of how a configuration would trigger a failure for this control.
	// Example string `json:"example"`
}

type Prerequisite struct {
	CloudProviders []string `json:"cloudProviders"`
}

type ControlConfigInput struct {
	Path        string `json:"path"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// FrameworkOverride holds the overrides for each framework.
type FrameworkOverride struct {
	FrameworkName   string   `json:"frameworkName"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	LongDescription string   `json:"long_description"`
	Remediation     string   `json:"remediation"`
	References      []string `json:"references"`
	ManualCheck     string   `json:"manualCheck"`
	ImpactStatement string   `json:"impactStatement"`
	DefaultValue    string   `json:"defaultValue"`
}
