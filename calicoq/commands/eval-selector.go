// Copyright (c) 2016 Tigera, Inc. All rights reserved.

package commands

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/projectcalico/felix/go/felix/dispatcher"
	"github.com/projectcalico/felix/go/felix/labelindex"
	"github.com/projectcalico/libcalico-go/lib/backend"
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/client"
	"github.com/projectcalico/libcalico-go/lib/selector"
	"os"
)

func EvalSelector(sel string) (err error) {
	cbs = NewEvalCmd()
	cbs.AddSelector("the selector", sel)
	cbs.Start()

	<-cbs.done
	return
}

// Restructuring like this should also be useful for a permanently running webserver variant too.
func NewEvalCmd() (cbs *evalCmd) {
	disp := dispatcher.NewDispatcher()
	cbs := &evalCmd{
		dispatcher: disp,
		done:       make(chan bool),
	}
	cbs.index = labelindex.NewInheritIndex(cbs.onMatchStarted, cbs.onMatchStopped)
	return &cbs
}

func (cbs *evalCmd) AddSelector(selectorName string, selector string) {
	parsedSel, err := selector.Parse(selector)
	if err != nil {
		fmt.Printf("Invalid selector: %#v. %v.", sel, err)
		os.Exit(1)
	}

	cbs.index.UpdateSelector(selectorName, parsedSel)
}

// Call this once you've AddSelector'ed the selectors you want to add.
func (cbs *evalCmd) Start() {
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

	cbs.dispatcher.Register(model.WorkloadEndpointKey{}, cbs.OnUpdate)
	cbs.dispatcher.Register(model.HostEndpointKey{}, cbs.OnUpdate)
	cbs.dispatcher.Register(model.ProfileLabelsKey{}, cbs.OnUpdate)

	apiConfig, err := client.LoadClientConfig("")
	if err != nil {
		glog.Fatal("Failed loading client config")
		os.Exit(1)
	}
	bclient, err := backend.NewClient(*apiConfig)
	if err != nil {
		glog.Fatal("Failed to create client")
		os.Exit(1)
	}
	syncer := bclient.Syncer(cbs)
	syncer.Start()
}

// For the final version of this we probably will want to be able to call AddSelector()
// while everything's running and join this function into that so you make the call and
// then a little bit later it returns the results.
// However solving that now would be more effort, so for this version everything must be
// added before Start()ing.
func (cbs *evalCmd) GetMatches() map[string]string {
	<-cbs.done
	return cbs.matches
}

func endpointName(key interface{}) string {
	var epName string
	switch epID := key.(type) {
	case model.WorkloadEndpointKey:
		epName = fmt.Sprintf("Workload endpoint %v/%v/%v", epID.OrchestratorID, epID.WorkloadID, epID.EndpointID)
	case model.HostEndpointKey:
		epName = fmt.Sprintf("Host endpoint %v", epID.EndpointID)
	}
	return epName
}

type evalCmd struct {
	dispatcher *dispatcher.Dispatcher
	index      *labelindex.InheritIndex
	matches    []interface{}

	done chan bool
}

func (cbs *evalCmd) OnConfigLoaded(globalConfig map[string]string,
	hostConfig map[string]string) {
	// Ignore for now
}

func (cbs *evalCmd) OnStatusUpdated(status api.SyncStatus) {
	if status == api.InSync {
		glog.V(0).Info("Datamodel in sync, we're done.")
		cbs.done <- true
	}
}

func (cbs *evalCmd) OnKeysUpdated(updates []api.Update) {
	glog.V(3).Info("Update: ", updates)
	for _, update := range updates {
		// Also removed empty key handling: don't understand it.
		cbs.dispatcher.OnUpdate(update)
	}
}

func (cbs *evalCmd) OnUpdate(update api.Update) (filterOut bool) {
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
		glog.Errorf("Unexpected update type: %#v", update)
		return true
	}
	return false
}

func (cbs *evalCmd) OnUpdates(updates []api.Update) {
	glog.V(3).Info("Update: ", updates)
	for _, update := range updates {
		// MATT: Removed some handling of empty key: don't understand how it can happen.
		cbs.dispatcher.OnUpdate(update)
	}
}

func (cbs *evalCmd) onMatchStarted(selId, epId interface{}) {
	fmt.Printf("%v\n", endpointName(epId))
}

func (cbs *evalCmd) onMatchStopped(selId, epId interface{}) {
	glog.Errorf("Unexpected match stopped event: %v, %v", selId, epId)
}
