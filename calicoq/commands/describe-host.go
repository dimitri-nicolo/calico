// Copyright (c) 2016 Tigera, Inc. All rights reserved.

package commands

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/tigera/libcalico-go/etcd-driver/etcd"
	"github.com/tigera/libcalico-go/etcd-driver/ipsets"
	"github.com/tigera/libcalico-go/etcd-driver/store"
	"github.com/tigera/libcalico-go/lib/backend"
	"sort"
)

func DescribeHost(hostname string, hideSelectors bool) (err error) {
	disp := store.NewDispatcher()
	cbs := &describeCmd{
		hideSelectors:    hideSelectors,
		dispatcher:       disp,
		done:             make(chan bool),
		epIDToPolIDs:     make(map[interface{}]map[backend.PolicyKey]bool),
		epIDToProfileIDs: make(map[interface{}][]string),
		policySorter:     NewPolicySorter(),
	}
	arc := ipsets.NewActiveRulesCalculator(nil, nil, cbs)
	cbs.activeRulesCalculator = arc

	filterUpdate := func(update *store.ParsedUpdate) {
		if update.Value == nil {
			glog.V(1).Infof("Skipping bad update: %v %v", update.Key, update.ParseErr)
			return
		}
		switch key := update.Key.(type) {
		case backend.HostEndpointKey:
			if key.Hostname != hostname {
				return
			}
			ep := update.Value.(*backend.HostEndpoint)
			cbs.epIDToProfileIDs[key] = ep.ProfileIDs
		case backend.WorkloadEndpointKey:
			if key.Hostname != hostname {
				return
			}
			ep := update.Value.(*backend.WorkloadEndpoint)
			cbs.epIDToProfileIDs[key] = ep.ProfileIDs
		}
		// Insert an empty map so we'll list this endpoint even if
		// no policies match it.
		glog.V(2).Infof("Found active endpoint %#v", update.Key)
		cbs.epIDToPolIDs[update.Key] = make(map[backend.PolicyKey]bool, 0)
		arc.OnUpdate(update)
	}

	checkValid := func(update *store.ParsedUpdate) {
		if update.Value == nil {
			fmt.Printf("WARNING: failed to parse value of key %v; "+
				"ignoring.\n  Parse error: %v\n\n", update.RawUpdate.Key, update.ParseErr)
		}
	}

	disp.Register(backend.WorkloadEndpointKey{}, checkValid)
	disp.Register(backend.HostEndpointKey{}, checkValid)
	disp.Register(backend.PolicyKey{}, checkValid)
	disp.Register(backend.TierKey{}, checkValid)
	disp.Register(backend.ProfileLabelsKey{}, checkValid)
	disp.Register(backend.ProfileRulesKey{}, checkValid)

	disp.Register(backend.WorkloadEndpointKey{}, filterUpdate)
	disp.Register(backend.HostEndpointKey{}, filterUpdate)
	disp.Register(backend.PolicyKey{}, arc.OnUpdate)
	disp.Register(backend.PolicyKey{}, cbs.policySorter.OnUpdate)
	disp.Register(backend.TierKey{}, cbs.policySorter.OnUpdate)
	disp.Register(backend.ProfileLabelsKey{}, arc.OnUpdate)
	disp.Register(backend.ProfileRulesKey{}, arc.OnUpdate)

	config := &store.DriverConfiguration{
		OneShot: true,
	}
	datastore, err := etcd.New(cbs, config)
	datastore.Start()

	<-cbs.done
	return
}

type describeCmd struct {
	// Config.
	hideSelectors bool

	// ActiveRulesCalculator matches policies/profiles against local
	// endpoints and notifies the ActiveSelectorCalculator when
	// their rules become active/inactive.
	activeRulesCalculator *ipsets.ActiveRulesCalculator
	dispatcher            *store.Dispatcher
	epIDToPolIDs          map[interface{}]map[backend.PolicyKey]bool
	epIDToProfileIDs      map[interface{}][]string
	policySorter          *PolicySorter

	done chan bool
}

func (cbs *describeCmd) OnConfigLoaded(globalConfig map[string]string,
	hostConfig map[string]string) {
	// Ignore for now
}

type endpointDatum struct {
	epID   interface{}
	polIDs map[backend.PolicyKey]bool
}

func (epd endpointDatum) EndpointName() string {
	var epName string
	switch epID := epd.epID.(type) {
	case backend.WorkloadEndpointKey:
		epName = fmt.Sprintf("Workload endpoint %v/%v/%v", epID.OrchestratorID, epID.WorkloadID, epID.EndpointID)
	case backend.HostEndpointKey:
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

func (cbs *describeCmd) OnStatusUpdated(status store.DriverStatus) {
	if status == store.InSync {
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

func (cbs *describeCmd) OnKeysUpdated(updates []store.Update) {
	glog.V(3).Info("Update: ", updates)
	for _, update := range updates {
		if len(update.Key) == 0 {
			glog.Fatal("Bug: Key/Value update had empty key")
		}

		cbs.dispatcher.DispatchUpdate(&update)
	}
}

func (cbs *describeCmd) OnPolicyMatch(policyKey backend.PolicyKey, endpointKey interface{}) {
	glog.V(2).Infof("Policy %v/%v now matches %v", policyKey.Tier, policyKey.Name, endpointKey)
	cbs.epIDToPolIDs[endpointKey][policyKey] = true
}

type PolicySorter struct {
	tiers map[string]*TierInfo
}

func NewPolicySorter() *PolicySorter {
	return &PolicySorter{
		tiers: make(map[string]*TierInfo),
	}
}

func (poc *PolicySorter) OnUpdate(update *store.ParsedUpdate) {
	if update.Value == nil {
		glog.Warningf("Ignoring deletion of %v", update.Key)
		return
	}
	switch key := update.Key.(type) {
	case backend.TierKey:
		tier := update.Value.(*backend.Tier)
		tierInfo := poc.tiers[key.Name]
		if tierInfo == nil {
			tierInfo = &TierInfo{Name: key.Name}
			poc.tiers[key.Name] = tierInfo
		}
		tierInfo.Order = tier.Order
		tierInfo.Valid = true
	case backend.PolicyKey:
		policy := update.Value.(*backend.Policy)
		tierInfo := poc.tiers[key.Tier]
		if tierInfo == nil {
			tierInfo = &TierInfo{Name: key.Tier}
			poc.tiers[key.Tier] = tierInfo
		}
		tierInfo.Policies = append(tierInfo.Policies, policy)
	}
}

func (poc *PolicySorter) Sorted() []*TierInfo {
	tiers := make([]*TierInfo, 0, len(poc.tiers))
	for _, tier := range poc.tiers {
		tiers = append(tiers, tier)
	}
	sort.Sort(TierByOrder(tiers))
	for _, tierInfo := range poc.tiers {
		sort.Sort(PolicyByOrder(tierInfo.Policies))
	}
	return tiers
}

type TierByOrder []*TierInfo

func (a TierByOrder) Len() int      { return len(a) }
func (a TierByOrder) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a TierByOrder) Less(i, j int) bool {
	if !a[i].Valid {
		return false
	} else if !a[j].Valid {
		return true
	}
	if a[i].Order == nil {
		return false
	} else if a[j].Order == nil {
		return true
	}
	if *a[i].Order == *a[j].Order {
		return a[i].Name < a[j].Name
	}
	return *a[i].Order < *a[j].Order
}

type PolicyByOrder []*backend.Policy

func (a PolicyByOrder) Len() int      { return len(a) }
func (a PolicyByOrder) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a PolicyByOrder) Less(i, j int) bool {
	if a[i].Order == nil {
		return false
	} else if a[j].Order == nil {
		return true
	}
	if *a[i].Order == *a[j].Order {
		return a[i].Name < a[j].Name
	}
	return *a[i].Order < *a[j].Order
}

type TierInfo struct {
	Name     string
	Valid    bool
	Order    *float32
	Policies []*backend.Policy
}
