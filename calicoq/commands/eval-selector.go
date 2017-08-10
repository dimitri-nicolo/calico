// Copyright (c) 2016 Tigera, Inc. All rights reserved.

package commands

import (
	"fmt"
	"os"

	"github.com/projectcalico/felix/dispatcher"
	"github.com/projectcalico/felix/labelindex"
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/selector"
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
	default:
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

// Restructuring like this should also be useful for a permanently running webserver variant too.
func NewEvalCmd(configFile string) (cbs *EvalCmd) {
	disp := dispatcher.NewDispatcher()
	cbs = &EvalCmd{
		configFile: configFile,
		dispatcher: disp,
		done:       make(chan bool),
		matches:    make(map[interface{}][]string),
	}
	cbs.index = labelindex.NewInheritIndex(cbs.onMatchStarted, cbs.onMatchStopped)
	return cbs
}

func (cbs *EvalCmd) AddSelector(selectorName string, selectorExpression string) {
	if selectorExpression == "" {
		return
	}
	parsedSel, err := selector.Parse(selectorExpression)
	if err != nil {
		fmt.Printf("Invalid selector: %#v. %v.\n", selectorExpression, err)
		os.Exit(1)
	}

	if cbs.showSelectors {
		selectorName = fmt.Sprintf("%v; selector \"%v\"", selectorName, selectorExpression)
	}

	cbs.index.UpdateSelector(selectorName, parsedSel)
}

func (cbs *EvalCmd) AddPolicyRuleSelectors(policy *model.Policy, prefix string) {
	for direction, ruleSet := range map[string][]model.Rule{
		"inbound":  policy.InboundRules,
		"outbound": policy.OutboundRules,
	} {
		for i, rule := range ruleSet {
			cbs.AddSelector(fmt.Sprintf("%v%v rule %v source match", prefix, direction, i+1), rule.SrcSelector)
			cbs.AddSelector(fmt.Sprintf("%v%v rule %v destination match", prefix, direction, i+1), rule.DstSelector)
			cbs.AddSelector(fmt.Sprintf("%v%v rule %v !source match", prefix, direction, i+1), rule.NotSrcSelector)
			cbs.AddSelector(fmt.Sprintf("%v%v rule %v !destination match", prefix, direction, i+1), rule.NotDstSelector)
		}
	}
}

// Call this once you've AddSelector'ed the selectors you want to add.
// We'll always do checkValid, but allow insertion of an additional EP filter.
func (cbs *EvalCmd) Start(endpointFilter dispatcher.UpdateHandler) {
	checkValid := func(update api.Update) (filterOut bool) {
		if update.Value == nil {
			fmt.Printf("WARNING: failed to parse value of key %v; "+
				"ignoring.\n\n\n", update)
			return true
		}
		return false
	}

	cbs.dispatcher.Register(model.WorkloadEndpointKey{}, checkValid)
	cbs.dispatcher.Register(model.HostEndpointKey{}, checkValid)

	cbs.dispatcher.Register(model.WorkloadEndpointKey{}, endpointFilter)
	cbs.dispatcher.Register(model.HostEndpointKey{}, endpointFilter)

	cbs.dispatcher.Register(model.WorkloadEndpointKey{}, cbs.OnUpdate)
	cbs.dispatcher.Register(model.HostEndpointKey{}, cbs.OnUpdate)
	cbs.dispatcher.Register(model.ProfileLabelsKey{}, cbs.OnUpdate)

	bclient := GetClient(cbs.configFile)
	syncer := bclient.Syncer(cbs)
	syncer.Start()
}

// For the final version of this we probably will want to be able to call AddSelector()
// while everything's running and join this function into that so you make the call and
// then a little bit later it returns the results.
// Ideally it probably wouldn't even have an AddSelector(), and this could just be accomplished
// with some custom filter functions on a single dispatcher.
// However solving that now would be more effort, so for this version everything must be
// added before Start()ing.
// Returns a map from endpoint key (model.Host/WorkloadEndpointKey) to a list of strings containing the
// names of the selectors that matched them.
func (cbs *EvalCmd) GetMatches() map[interface{}][]string {
	<-cbs.done
	return cbs.matches
}

func endpointName(key interface{}) string {
	var epName string
	switch epID := key.(type) {
	case model.WorkloadEndpointKey:
		epName = fmt.Sprintf("Workload endpoint %v/%v/%v/%v", epID.Hostname, epID.OrchestratorID, epID.WorkloadID, epID.EndpointID)
	case model.HostEndpointKey:
		epName = fmt.Sprintf("Host endpoint %v/%v", epID.Hostname, epID.EndpointID)
	}
	return epName
}

type EvalCmd struct {
	showSelectors bool
	configFile    string
	dispatcher    *dispatcher.Dispatcher
	index         *labelindex.InheritIndex
	matches       map[interface{}][]string

	done chan bool
}

func (cbs *EvalCmd) OnConfigLoaded(globalConfig map[string]string,
	hostConfig map[string]string) {
	// Ignore for now
}

func (cbs *EvalCmd) OnStatusUpdated(status api.SyncStatus) {
	if status == api.InSync {
		log.Info("Datamodel in sync, we're done.")
		cbs.done <- true
	}
}

func (cbs *EvalCmd) OnKeysUpdated(updates []api.Update) {
	log.Info("Update: ", updates)
	for _, update := range updates {
		// Also removed empty key handling: don't understand it.
		cbs.dispatcher.OnUpdate(update)
	}
}

func (cbs *EvalCmd) OnUpdate(update api.Update) (filterOut bool) {
	if update.Value == nil {
		return true
	}
	switch k := update.Key.(type) {
	case model.WorkloadEndpointKey:
		v := update.Value.(*model.WorkloadEndpoint)
		cbs.index.UpdateLabels(update.Key, v.Labels, v.ProfileIDs)
	case model.HostEndpointKey:
		v := update.Value.(*model.HostEndpoint)
		cbs.index.UpdateLabels(update.Key, v.Labels, v.ProfileIDs)
	case model.ProfileLabelsKey:
		v := update.Value.(map[string]string)
		cbs.index.UpdateParentLabels(k.Name, v)
	default:
		log.Errorf("Unexpected update type: %#v", update)
		return true
	}
	return false
}

func (cbs *EvalCmd) OnUpdates(updates []api.Update) {
	log.Info("Update: ", updates)
	for _, update := range updates {
		// MATT: Removed some handling of empty key: don't understand how it can happen.
		cbs.dispatcher.OnUpdate(update)
	}
}

func (cbs *EvalCmd) onMatchStarted(selId, epId interface{}) {
	if pols, ok := cbs.matches[epId]; ok {
		cbs.matches[epId] = append(pols, selId.(string))
	} else {
		cbs.matches[epId] = []string{selId.(string)}
	}
}

func (cbs *EvalCmd) onMatchStopped(selId, epId interface{}) {
	log.Errorf("Unexpected match stopped event: %v, %v", selId, epId)
}
