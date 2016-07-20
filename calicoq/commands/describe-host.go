// Copyright (c) 2016 Tigera, Inc. All rights reserved.

package commands

import (
	"github.com/tigera/libcalico-go/etcd-driver/etcd"
	"github.com/tigera/libcalico-go/etcd-driver/store"
	"github.com/tigera/libcalico-go/etcd-driver/ipsets"
	"github.com/tigera/libcalico-go/lib/backend"
	"github.com/golang/glog"
	"fmt"
)

func DescribeHost(hostname string) (err error) {
	disp := store.NewDispatcher()
	cbs := &describeCmd{
		dispatcher: disp,
		done: make(chan bool),
		epIDToPolIDs: make(map[interface{}][]backend.PolicyKey),
	}
	arc := ipsets.NewActiveRulesCalculator(nil, nil, cbs)
	cbs.activeRulesCalculator = arc

	filterUpdate := func(update *store.ParsedUpdate) {
		switch key := update.Key.(type) {
		case backend.HostEndpointKey:
			if key.Hostname != hostname {
				return
			}
		case backend.WorkloadEndpointKey:
			if key.Hostname != hostname {
				return
			}
		}
		// Insert an empty slice so we'll list this endpoint even if
		// no policies match it.
		glog.V(2).Infof("Found active endpoint %#v", update.Key)
		cbs.epIDToPolIDs[update.Key] = make([]backend.PolicyKey, 0)
		arc.OnUpdate(update)
	}

	disp.Register(backend.WorkloadEndpointKey{}, filterUpdate)
	disp.Register(backend.HostEndpointKey{}, filterUpdate)
	disp.Register(backend.PolicyKey{}, arc.OnUpdate)
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
	// ActiveRulesCalculator matches policies/profiles against local
	// endpoints and notifies the ActiveSelectorCalculator when
	// their rules become active/inactive.
	activeRulesCalculator *ipsets.ActiveRulesCalculator
	dispatcher            *store.Dispatcher
	epIDToPolIDs          map[interface{}][]backend.PolicyKey

	done                  chan bool
}

func (cbs *describeCmd) OnConfigLoaded(globalConfig map[string]string,
hostConfig map[string]string) {
	// Ignore for now
}

func (cbs *describeCmd) OnStatusUpdated(status store.DriverStatus) {
	if status == store.InSync {
		fmt.Println("Policies that match each endpoint:")
		for epID, polIDs := range cbs.epIDToPolIDs {
			fmt.Printf("\nEndpoint %v\n", epID)
			for _, polID := range polIDs {
				fmt.Printf("  %v/%v\n", polID.Tier, polID.Name)
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
	cbs.epIDToPolIDs[endpointKey] =
		append(cbs.epIDToPolIDs[policyKey.Tier], policyKey)
}

