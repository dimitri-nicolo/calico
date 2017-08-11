// Copyright (c) 2016 Tigera, Inc. All rights reserved.

package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/projectcalico/felix/calc"
	"github.com/projectcalico/felix/dispatcher"
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	log "github.com/sirupsen/logrus"
)

// MATT: How to make includeRules actually work?
// Will need to be able to call eval selector and get the output programmatically (not printed).
// Do that for each rule in each policy (globally, not just selected).
// Actually I want to be able to do eval selector with many selectors and few EPs.
// Basically I want to be able to control the EP filter used by eval selector.
func DescribeEndpointOrHost(configFile, endpointSubstring, hostname string, hideSelectors bool, hideRuleMatches bool) (err error) {
	disp := dispatcher.NewDispatcher()
	cbs := &describeCmd{
		hideSelectors:     hideSelectors,
		includeRules:      !hideRuleMatches,
		endpointSubstring: endpointSubstring,
		hostname:          hostname,
		dispatcher:        disp,
		done:              make(chan bool),
		epIDToPolIDs:      make(map[interface{}]map[model.PolicyKey]bool),
		epIDToProfileIDs:  make(map[interface{}][]string),
		policySorter:      calc.NewPolicySorter(),
		evalCmd:           nil,
	}
	nrs := &noopRuleScanner{}
	arc := calc.NewActiveRulesCalculator()
	arc.PolicyMatchListener = cbs
	arc.RuleScanner = nrs
	cbs.activeRulesCalculator = arc

	// MATT This approach won't be suitable for not-yet-configured endpoints.
	//      To support them, we'd need to be able to build a fake endpoint kv
	//      for them from the yaml for that endpoint.
	filterUpdate := func(update api.Update) (filterOut bool) {
		if update.Value == nil {
			// MATT: Why is this so much lower priority than checkValid?
			log.Infof("Skipping bad update: %v", update.Key)
			return true
		}
		switch key := update.Key.(type) {
		case model.HostEndpointKey:
			if hostname != "" && key.Hostname != hostname {
				return true
			}
			if !strings.Contains(endpointName(key), endpointSubstring) {
				return true
			}
			ep := update.Value.(*model.HostEndpoint)
			cbs.epIDToProfileIDs[key] = ep.ProfileIDs
		case model.WorkloadEndpointKey:
			if hostname != "" && key.Hostname != hostname {
				return true
			}
			if !strings.Contains(endpointName(key), endpointSubstring) {
				return true
			}
			ep := update.Value.(*model.WorkloadEndpoint)
			cbs.epIDToProfileIDs[key] = ep.ProfileIDs
		}
		// Insert an empty map so we'll list this endpoint even if
		// no policies match it.
		log.Infof("Found active endpoint %#v", update.Key)
		cbs.epIDToPolIDs[update.Key] = make(map[model.PolicyKey]bool, 0)
		arc.OnUpdate(update)
		return false
	}

	// MATT TODO: Compare this to the Felix ValidationFilter.  How is this deficient?
	checkValid := func(update api.Update) (filterOut bool) {
		if update.Value == nil {
			fmt.Printf("WARNING: failed to parse value of key %v; "+
				"ignoring.\n\n", update)
			return true
		}
		return false
	}

	// MATT: It's very opaque why some of these need to be checked,
	//       and some can just be passed straight to the arc/sorter.
	// LOL I'm an idiot.
	disp.Register(model.WorkloadEndpointKey{}, checkValid)
	disp.Register(model.HostEndpointKey{}, checkValid)
	disp.Register(model.PolicyKey{}, checkValid)
	disp.Register(model.TierKey{}, checkValid)
	disp.Register(model.ProfileLabelsKey{}, checkValid)
	disp.Register(model.ProfileRulesKey{}, checkValid)

	if cbs.includeRules {
		// MATT: Would be nice to have a single dispatcher: wouldn't need to worry about
		// the two not working on the same data and giving weird results.
		cbs.evalCmd = NewEvalCmd(configFile)
		cbs.evalCmd.showSelectors = !hideSelectors
		polRules := func(update api.Update) (filterOut bool) {
			// Go through the rules, and generate a selector for each.
			cbs.evalCmd.AddPolicyRuleSelectors(
				update.Value.(*model.Policy),
				"Policy \""+update.Key.(model.PolicyKey).Name+"\" ",
			)
			return false
		}
		disp.Register(model.PolicyKey{}, polRules)
		// TODO: Do a profile version?
	}

	disp.Register(model.WorkloadEndpointKey{}, filterUpdate)
	disp.Register(model.HostEndpointKey{}, filterUpdate)
	disp.Register(model.PolicyKey{}, arc.OnUpdate)
	disp.Register(model.PolicyKey{}, cbs.policySorter.OnUpdate)
	disp.Register(model.TierKey{}, cbs.policySorter.OnUpdate)
	disp.Register(model.ProfileLabelsKey{}, arc.OnUpdate)
	disp.Register(model.ProfileRulesKey{}, arc.OnUpdate)

	bclient := GetClient(configFile)
	syncer := bclient.Syncer(cbs)
	syncer.Start()

	// The describeCmd will notify us once it's in sync and has finished outputting.
	<-cbs.done
	return
}

// MATT: Might want to integrate this into the describeCmd and use these callbacks to
//       save-off rules that reference stuff?
type noopRuleScanner struct {
}

func (rs *noopRuleScanner) OnPolicyActive(model.PolicyKey, *model.Policy) {
	return
}

func (rs *noopRuleScanner) OnPolicyInactive(model.PolicyKey) {
	return
}

func (rs *noopRuleScanner) OnProfileActive(model.ProfileRulesKey, *model.ProfileRules) {
	return
}

func (rs *noopRuleScanner) OnProfileInactive(model.ProfileRulesKey) {
	return
}

type describeCmd struct {
	// Config.
	hideSelectors     bool
	includeRules      bool
	endpointSubstring string
	hostname          string

	// ActiveRulesCalculator matches policies/profiles against local
	// endpoints and notifies the ActiveSelectorCalculator when
	// their rules become active/inactive.
	activeRulesCalculator *calc.ActiveRulesCalculator
	dispatcher            *dispatcher.Dispatcher
	epIDToPolIDs          map[interface{}]map[model.PolicyKey]bool
	epIDToProfileIDs      map[interface{}][]string
	policySorter          *calc.PolicySorter

	evalCmd *EvalCmd

	done chan bool
}

func (cbs *describeCmd) OnConfigLoaded(globalConfig map[string]string,
	hostConfig map[string]string) {
	// Ignore for now
}

type endpointDatum struct {
	epID   interface{}
	polIDs map[model.PolicyKey]bool
}

func (epd endpointDatum) EndpointName() string {
	var epName string
	switch epID := epd.epID.(type) {
	case model.WorkloadEndpointKey:
		epName = fmt.Sprintf("Workload endpoint %v/%v/%v", epID.OrchestratorID, epID.WorkloadID, epID.EndpointID)
	case model.HostEndpointKey:
		epName = fmt.Sprintf("Host endpoint %v", epID.EndpointID)
	}
	return epName
}

type ByName []endpointDatum

func (a ByName) Len() int      { return len(a) }
func (a ByName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByName) Less(i, j int) bool {
	return a[i].EndpointName() < a[j].EndpointName()
}

func (cbs *describeCmd) OnStatusUpdated(status api.SyncStatus) {
	if status == api.InSync {
		var matches map[interface{}][]string
		if cbs.includeRules {
			endpointMatch := func(update api.Update) (filterOut bool) {
				if update.Value == nil {
					// MATT: Why is this so much lower priority than checkValid?
					log.Infof("Skipping bad update: %v", update.Key)
					return true
				}
				switch key := update.Key.(type) {
				case model.HostEndpointKey:
					if cbs.hostname != "" && key.Hostname != cbs.hostname {
						return true
					}
					if !strings.Contains(endpointName(key), cbs.endpointSubstring) {
						return true
					}
				case model.WorkloadEndpointKey:
					if cbs.hostname != "" && key.Hostname != cbs.hostname {
						return true
					}
					if !strings.Contains(endpointName(key), cbs.endpointSubstring) {
						return true
					}
				}
				return false
			}
			cbs.evalCmd.Start(endpointMatch)
			matches = cbs.evalCmd.GetMatches()
		}

		if cbs.hostname != "" {
			fmt.Printf("Policies and profiles for each endpoint on host \"%v\":\n", cbs.hostname)
		} else {
			fmt.Printf("Policies and profiles for endpoints matching \"%v\":\n", cbs.endpointSubstring)
		}
		tiers := cbs.policySorter.Sorted() // MATT: map[model.PolicyKey]*model.Policy
		epData := make([]endpointDatum, 0)

		for epID, polIDs := range cbs.epIDToPolIDs {
			epData = append(epData, endpointDatum{epID, polIDs})
		}
		sort.Sort(ByName(epData))
		for _, epDatum := range epData {
			epName := epDatum.EndpointName()
			epID := epDatum.epID
			polIDs := epDatum.polIDs
			log.Infof("Looking at endpoint %v with policies %v", epID, polIDs)
			fmt.Printf("\n%v\n", epName)
			fmt.Println("  Policies:")
			for _, untracked := range []bool{true, false} {
				suffix := map[bool]string{true: " [untracked]", false: ""}[untracked]
				for _, tier := range tiers {
					log.Infof("Looking at tier %v", tier)
					for _, pol := range tier.OrderedPolicies { // pol is a PolKV
						log.Infof("Looking at policy %v", pol.Key)
						if pol.Value.DoNotTrack != untracked {
							continue
						}
						if polIDs[pol.Key] {
							tierText := ""
							tierOrder := "default"

							if tier.Order != nil {
								tierOrder = fmt.Sprint(*tier.Order)
							}
							if tier.Name != "default" {
								if !tier.Valid {
									fmt.Printf("    WARNING: tier %#v metadata missing; packets will skip tier\n", tier.Name)
									tierOrder = "missing"
								}
								tierText = fmt.Sprintf("Tier %#v (order %v) ", tier.Name, tierOrder)
							}

							polOrder := "default"
							if pol.Value.Order != nil {
								polOrder = fmt.Sprint(*pol.Value.Order)
							}
							if cbs.hideSelectors {
								fmt.Printf("    %sPolicy %#v (order %v)%v\n", tierText, pol.Key.Name, polOrder, suffix)
							} else {
								fmt.Printf("    %sPolicy %#v (order %v; selector \"%v\")%v\n", tierText, pol.Key.Name, polOrder, pol.Value.Selector, suffix)
							}
						}
					}
				}
			}
			profIDs := cbs.epIDToProfileIDs[epID]
			if len(profIDs) > 0 {
				fmt.Printf("  Profiles:\n")
				for _, profID := range cbs.epIDToProfileIDs[epID] {
					fmt.Printf("    Profile \"%v\"\n", profID)
				}
			}

			if cbs.includeRules {
				if policies, ok := matches[epID]; ok {
					fmt.Printf("  Rule matches:\n")
					sort.Strings(policies)
					for _, policy := range policies {
						fmt.Printf("    %v\n", policy)
					}
				}
			}
		}

		cbs.done <- true
	}
}

func (cbs *describeCmd) OnUpdates(updates []api.Update) {
	log.Info("Update: ", updates)
	for _, update := range updates {
		// MATT: Removed some handling of empty key: don't understand how it can happen.
		cbs.dispatcher.OnUpdate(update)
	}
}

func (cbs *describeCmd) OnPolicyMatch(policyKey model.PolicyKey, endpointKey interface{}) {
	log.Infof("Policy %v/%v now matches %v", policyKey.Tier, policyKey.Name, endpointKey)
	cbs.epIDToPolIDs[endpointKey][policyKey] = true
}

func (cbs *describeCmd) OnPolicyMatchStopped(policyKey model.PolicyKey, endpointKey interface{}) {
	// Matt: Maybe we should remove something here, but it's an edge case
	return
}

type TierInfo struct {
	Name     string
	Valid    bool
	Order    *float32
	Policies []*model.Policy
}
