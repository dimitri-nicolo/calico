// Copyright (c) 2016 Tigera, Inc. All rights reserved.

package commands

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/projectcalico/felix/go/felix/calc"
	"github.com/projectcalico/felix/go/felix/dispatcher"
	"github.com/projectcalico/libcalico-go/lib/api"
	"github.com/projectcalico/libcalico-go/lib/backend"
	"github.com/projectcalico/libcalico-go/lib/client"
	"os"
	"sort"
)

func DescribeHost(hostname string, hideSelectors bool) (err error) {
	disp := dispatcher.NewDispatcher()
	cbs := &describeCmd{
		hideSelectors:    hideSelectors,
		dispatcher:       disp,
		done:             make(chan bool),
		epIDToPolIDs:     make(map[interface{}]map[model.PolicyKey]bool),
		epIDToProfileIDs: make(map[interface{}][]string),
		policySorter:     calc.NewPolicySorter(),
	}
	arc := calc.NewActiveRulesCalculator()
	arc.PolicyMatchListener = cbs
	cbs.activeRulesCalculator = arc

	// MATT This approach won't be suitable for not-yet-configured endpoints.
	//      To support them, we'd need to be able to build a fake endpoint kv
	//      for them from the yaml for that endpoint.
	filterUpdate := func(update api.Update) {
		if update.Value == nil {
			// MATT: Why is this so much lower priority than checkValid?
			glog.V(1).Infof("Skipping bad update: %v %v", update.Key, update.ParseErr)
			return
		}
		switch key := update.Key.(type) {
		case model.HostEndpointKey:
			if key.Hostname != hostname {
				return
			}
			ep := update.Value.(*model.HostEndpoint)
			cbs.epIDToProfileIDs[key] = ep.ProfileIDs
		case model.WorkloadEndpointKey:
			if key.Hostname != hostname {
				return
			}
			ep := update.Value.(*model.WorkloadEndpoint)
			cbs.epIDToProfileIDs[key] = ep.ProfileIDs
		}
		// Insert an empty map so we'll list this endpoint even if
		// no policies match it.
		glog.V(2).Infof("Found active endpoint %#v", update.Key)
		cbs.epIDToPolIDs[update.Key] = make(map[model.PolicyKey]bool, 0)
		arc.OnUpdate(update)
	}

	// MATT TODO: Compare this to the Felix ValidationFilter.  How is this deficient?
	checkValid := func(update api.Update) {
		if update.Value == nil {
			fmt.Printf("WARNING: failed to parse value of key %v; "+
				"ignoring.\n  Parse error: %v\n\n", update.RawUpdate.Key, update.ParseErr)
		}
	}

	// MATT: It's very opaque why some of these need to be checked,
	//       and some can just be passed straight to the arc/sorter.
	disp.Register(model.WorkloadEndpointKey{}, checkValid)
	disp.Register(model.HostEndpointKey{}, checkValid)
	disp.Register(model.PolicyKey{}, checkValid)
	disp.Register(model.TierKey{}, checkValid)
	disp.Register(model.ProfileLabelsKey{}, checkValid)
	disp.Register(model.ProfileRulesKey{}, checkValid)

	disp.Register(model.WorkloadEndpointKey{}, filterUpdate)
	disp.Register(model.HostEndpointKey{}, filterUpdate)
	disp.Register(model.PolicyKey{}, arc.OnUpdate)
	disp.Register(model.PolicyKey{}, cbs.policySorter.OnUpdate)
	disp.Register(model.TierKey{}, cbs.policySorter.OnUpdate)
	disp.Register(model.ProfileLabelsKey{}, arc.OnUpdate)
	disp.Register(model.ProfileRulesKey{}, arc.OnUpdate)

	apiConfig, err := client.LoadClientConfig("")
	if err != nil {
		glog.Fatal("Failed loading client config")
		os.Exit(1)
	}
	client, err := client.New(*apiConfig)
	if err != nil {
		glog.Fatal("Failed to create client")
		os.Exit(1)
	}
	syncer, err := client.Syncer(cbs)
	if err != nil {
		glog.Fatal("Failed to create syncer")
		os.Exit(1)
	}
	syncer.Start()

	// The describeCmd will notify us once it's in sync and has finished outputting.
	<-cbs.done
	return
}

type describeCmd struct {
	// Config.
	hideSelectors bool

	// ActiveRulesCalculator matches policies/profiles against local
	// endpoints and notifies the ActiveSelectorCalculator when
	// their rules become active/inactive.
	activeRulesCalculator *calc.ActiveRulesCalculator
	dispatcher            *dispatcher.Dispatcher
	epIDToPolIDs          map[interface{}]map[model.PolicyKey]bool
	epIDToProfileIDs      map[interface{}][]string
	policySorter          *calc.PolicySorter

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
		fmt.Println("Policies that match each endpoint:")
		tiers := cbs.policySorter.Sorted()

		epData := make([]endpointDatum, 0)

		for epID, polIDs := range cbs.epIDToPolIDs {
			epData = append(epData, endpointDatum{epID, polIDs})
		}
		sort.Sort(ByName(epData))
		for _, epDatum := range epData {
			epName := epDatum.EndpointName()
			epID := epDatum.epID
			polIDs := epDatum.polIDs
			glog.V(2).Infof("Looking at endpoint %v with policies %v", epID, polIDs)
			fmt.Printf("\n%v\n", epName)
			fmt.Println("  Policies:")
			for _, tier := range tiers {
				glog.V(2).Infof("Looking at tier %v", tier)
				tierMatches := false
				for _, pol := range tier.Policies {
					glog.V(2).Infof("Looking at policy %v", pol.PolicyKey)
					if polIDs[pol.PolicyKey] {
						if !tierMatches {
							order := "default"
							if !tier.Valid {
								order = "missing"
							}
							tierMatches = true
							if tier.Order != nil {
								order = fmt.Sprint(*tier.Order)
							}
							fmt.Printf("    Tier %#v (order %v):\n", tier.Name, order)
							if !tier.Valid {
								fmt.Printf("    WARNING: tier metadata missing; packets will skip tier\n")
							}
						}
						order := "default"
						if pol.Order != nil {
							order = fmt.Sprint(*pol.Order)
						}
						if cbs.hideSelectors {
							fmt.Printf("      Policy %#v (order %v)\n", pol.Name, order)
						} else {
							fmt.Printf("      Policy %#v (order %v; selector '%v')\n", pol.Name, order, pol.Selector)
						}
					}
				}
			}
			profIDs := cbs.epIDToProfileIDs[epID]
			if len(profIDs) > 0 {
				fmt.Printf("  Profiles:\n")
				for _, profID := range cbs.epIDToProfileIDs[epID] {
					fmt.Printf("    Profile %v\n", profID)
				}
			}
		}
		cbs.done <- true
	}
}

func (cbs *describeCmd) OnUpdates(updates []api.Update) {
	glog.V(3).Info("Update: ", updates)
	for _, update := range updates {
		if len(update.Key) == 0 {
			glog.Fatal("Bug: Key/Value update had empty key")
		}

		cbs.dispatcher.DispatchUpdate(&update)
	}
}

func (cbs *describeCmd) OnPolicyMatch(policyKey model.PolicyKey, endpointKey interface{}) {
	glog.V(2).Infof("Policy %v/%v now matches %v", policyKey.Tier, policyKey.Name, endpointKey)
	cbs.epIDToPolIDs[endpointKey][policyKey] = true
}

type TierInfo struct {
	Name     string
	Valid    bool
	Order    *float32
	Policies []*model.Policy
}
