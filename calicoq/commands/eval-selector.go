// Copyright (c) 2016 Tigera, Inc. All rights reserved.

package commands

import (
	"fmt"

	"github.com/projectcalico/libcalico-go/lib/backend/api"
	log "github.com/sirupsen/logrus"
)

func EvalSelector(configFile, sel string, outputFormat string) (err error) {
	cbs := NewEvalCmd(configFile)
	cbs.AddSelector("the selector", sel)
	noopFilter := func(update api.Update) (filterOut bool) {
		return false
	}
	cbs.Start(noopFilter)

	matches := cbs.GetMatches()

	switch outputFormat {
	case "yaml":
		EvalSelectorPrintYAML(sel, matches)
	case "json":
		EvalSelectorPrintJSON(sel, matches)
	case "ps":
		EvalSelectorPrint(sel, matches)
	}
	return
}

func EvalSelectorPrint(sel string, matches map[interface{}][]string) {
	fmt.Printf("Endpoints matching selector %v:\n", sel)
	for endpoint := range matches {
		fmt.Printf("  %v\n", endpointName(endpoint))
	}
}

func EvalSelectorPrintYAML(sel string, matches map[interface{}][]string) {
	output := EvalSelectorPrintObjects(sel, matches)
	err := printYAML([]OutputList{output})
	if err != nil {
		log.Errorf("Unexpected error printing to YAML: %s", err)
		fmt.Println("Unexpected error printing to YAML")
	}
}

func EvalSelectorPrintJSON(sel string, matches map[interface{}][]string) {
	output := EvalSelectorPrintObjects(sel, matches)
	err := printJSON([]OutputList{output})
	if err != nil {
		log.Errorf("Unexpected error printing to JSON: %s", err)
		fmt.Println("Unexpected error printing to JSON")
	}
}

func EvalSelectorPrintObjects(sel string, matches map[interface{}][]string) OutputList {
	output := OutputList{
		Description: fmt.Sprintf("Endpoints matching selector %v:\n", sel),
	}
	for endpoint := range matches {
		output.Endpoints = append(output.Endpoints, NewWorkloadEndpointPrintFromKey(endpoint))
	}

	return output
}
