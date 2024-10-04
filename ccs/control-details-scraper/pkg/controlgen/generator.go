// Copyright (c) 2024 Tigera, Inc. All rights reserved.

package controlgen

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/projectcalico/calico/ccs/control-details-scraper/pkg/models/ccsmodel"
	"github.com/projectcalico/calico/ccs/control-details-scraper/pkg/models/regolib"
)

type contolGenerator struct {
	regolibPath string
	writeToPath string
}

func NewControlsGenerator(regolibPath string,
	writeToPath string) *contolGenerator {
	return &contolGenerator{
		regolibPath: regolibPath,
		writeToPath: writeToPath,
	}
}

func (r *contolGenerator) Generate() error {
	// Check if the regolibrary directory exists
	if _, err := os.Stat(r.regolibPath); os.IsNotExist(err) {
		return errors.New("regolibrary not found.")
	}

	// Directories to gather control details information from
	var frameworks []*regolib.Framework
	var rules []*regolib.Rule
	var controls []*regolib.Control

	frameworks, err := readRegoLib[regolib.Framework](filepath.Join(r.regolibPath, "./frameworks"), "")
	if err != nil {
		return err
	}

	rules, err = readRegoLib[regolib.Rule](filepath.Join(r.regolibPath, "./rules"), ".metadata.json")
	if err != nil {
		return err
	}

	controls, err = readRegoLib[regolib.Control](filepath.Join(r.regolibPath, "./controls"), "")
	if err != nil {
		return err
	}

	controlDetails := toCCSControlDetails(controls, frameworks, rules)
	if err != nil {
		return err
	}

	err = r.writeToFile(controlDetails)
	if err != nil {
		return err
	}

	return nil
}

func toCCSControlDetails(controls []*regolib.Control, frameworks []*regolib.Framework, rules []*regolib.Rule) []*ccsmodel.CCSControlDetails {
	var controlDetails []*ccsmodel.CCSControlDetails

	var rulesMap = make(map[string]*regolib.Rule)
	for _, rule := range rules {
		rulesMap[rule.Name] = rule
	}

	controlsMap := make(map[string]*ccsmodel.CCSControlDetails, len(controls))
	for _, control := range controls {
		controlDetail := &ccsmodel.CCSControlDetails{
			ControlDetailsID:  control.ControlID,
			Name:              control.Name,
			Description:       control.Description,
			Category:          control.Category.Name,
			SubCategory:       control.Category.SubCategory.Name,
			Remediation:       control.Remediation,
			TestCriteria:      control.Test,
			Severity:          control.BaseScore,
			ManualCheck:       control.ManualTest,
			Frameworks:        []string{},
			FrameworkOverride: []*ccsmodel.FrameworkOverride{},
			// Example:          control.Example,
		}
		controlsMap[control.ControlID] = controlDetail
		for _, rule := range control.RulesNames {
			ruleFound := rulesMap[rule]
			if ruleFound != nil {
				controlDetail.RelatedResources = findRelatedResources(ruleFound)
				controlDetail.Prerequisites = findPrerequisites(ruleFound)
				controlDetail.Configuration = findConfiguration(ruleFound)
			}
		}
		controlDetails = append(controlDetails, controlDetail)
	}

	for _, framework := range frameworks {
		for _, control := range framework.ActiveControls {
			controlFound := controlsMap[control.ControlID]
			if controlFound != nil {
				controlFound.Frameworks = append(controlFound.Frameworks, framework.Name)
				controlFound.FrameworkOverride = append(controlFound.FrameworkOverride, &ccsmodel.FrameworkOverride{
					FrameworkName:   framework.Name,
					Name:            control.Patch.Name,
					Description:     control.Patch.Description,
					LongDescription: control.Patch.LongDescription,
					Remediation:     control.Patch.Remediation,
					References:      control.Patch.References,
					ImpactStatement: control.Patch.ImpactStatement,
					ManualCheck:     control.Patch.ManualTest,
					DefaultValue:    control.Patch.DefaultValue,
				})
			}
		}
	}

	return controlDetails
}

func findRelatedResources(rule *regolib.Rule) []string {
	relatedResources := []string{}
	for _, match := range rule.Match {
		relatedResources = append(relatedResources, match.Resources...)

	}
	return relatedResources
}

func findPrerequisites(rule *regolib.Rule) *ccsmodel.Prerequisite {
	prerequisite := &ccsmodel.Prerequisite{}

	prerequisite.CloudProviders = append(prerequisite.CloudProviders, rule.RelevantCloudProviders...)

	return prerequisite
}

func findConfiguration(rule *regolib.Rule) []*ccsmodel.ControlConfigInput {
	configs := []*ccsmodel.ControlConfigInput{}
	for _, config := range rule.ControlConfigInputs {
		ccsconfig := &ccsmodel.ControlConfigInput{
			Path:        config.Path,
			Name:        config.Name,
			Description: config.Description,
		}
		configs = append(configs, ccsconfig)
	}
	return configs
}

func (r *contolGenerator) writeToFile(controls []*ccsmodel.CCSControlDetails) error {
	controlBytes, err := json.Marshal(controls)
	if err != nil {
		return err
	}

	err = os.WriteFile(r.writeToPath+"controlDetails.json", controlBytes, os.FileMode(0666))
	if err != nil {
		return err
	}

	return nil
}

func readRegoLib[T regolib.Rule | regolib.Framework | regolib.Control](dir string, match string) ([]*T, error) {
	var regolibStructs []*T

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".json" && strings.HasSuffix(filepath.Base(path), match) {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			var regoLibStruct *T
			if err := json.Unmarshal(data, &regoLibStruct); err != nil {
				return err
			}
			regolibStructs = append(regolibStructs, regoLibStruct)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("Error reading files: %s", err.Error())

	}

	return regolibStructs, nil
}
