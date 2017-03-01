// Copyright (c) 2016 Tigera, Inc. All rights reserved.

package commands

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/tigera/libcalico-go/lib/backend"
	"github.com/tigera/libcalico-go/lib/selector"
	"os"
)

func EvalSelector(sel string) (err error) {
	disp := store.NewDispatcher()
	cbs := &evalCmd{
		dispatcher: disp,
		done:       make(chan bool),
	}
	cbs.index = labels.NewInheritanceIndex(cbs.onMatchStarted, cbs.onMatchStopped)
	parsedSel, err := selector.Parse(sel)
	if err != nil {
		fmt.Printf("Invalid selector: %#v. %v.", sel, err)
		os.Exit(1)
	}

	cbs.index.UpdateSelector("the selector", parsedSel)

	checkValid := func(update *store.ParsedUpdate) {
		if update.Value == nil {
			fmt.Printf("WARNING: failed to parse value of key %v; "+
				"ignoring.\n  Parse error: %v\n\n",
				update.RawUpdate.Key, update.ParseErr)
		}
	}

	disp.Register(backend.WorkloadEndpointKey{}, checkValid)
	disp.Register(backend.HostEndpointKey{}, checkValid)

	disp.Register(backend.WorkloadEndpointKey{}, cbs.OnUpdate)
	disp.Register(backend.HostEndpointKey{}, cbs.OnUpdate)
	disp.Register(backend.ProfileLabelsKey{}, cbs.OnUpdate)

	config := &store.DriverConfiguration{
		OneShot: true,
	}
	datastore, err := etcd.New(cbs, config)
	datastore.Start()

	<-cbs.done
	return
}

func endpointName(key interface{}) string {
	var epName string
	switch epID := key.(type) {
	case backend.WorkloadEndpointKey:
		epName = fmt.Sprintf("Workload endpoint %v/%v/%v", epID.OrchestratorID, epID.WorkloadID, epID.EndpointID)
	case backend.HostEndpointKey:
		epName = fmt.Sprintf("Host endpoint %v", epID.EndpointID)
	}
	return epName
}

type evalCmd struct {
	dispatcher *store.Dispatcher
	index      labels.LabelInheritanceIndex
	matches    []interface{}

	done chan bool
}

func (cbs *evalCmd) OnConfigLoaded(globalConfig map[string]string,
	hostConfig map[string]string) {
	// Ignore for now
}

func (cbs *evalCmd) OnStatusUpdated(status store.DriverStatus) {
	if status == store.InSync {
		glog.V(0).Info("Datamodel in sync, we're done.")
		cbs.done <- true
	}
}

func (cbs *evalCmd) OnKeysUpdated(updates []store.Update) {
	glog.V(3).Info("Update: ", updates)
	for _, update := range updates {
		if len(update.Key) == 0 {
			glog.Fatal("Bug: Key/Value update had empty key")
		}

		cbs.dispatcher.DispatchUpdate(&update)
	}
}

func (cbs *evalCmd) OnUpdate(update *store.ParsedUpdate) {
	if update.Value == nil {
		return
	}
	switch k := update.Key.(type) {
	case backend.WorkloadEndpointKey:
		v := update.Value.(*backend.WorkloadEndpoint)
		cbs.index.UpdateLabels(update.Key, v.Labels, v.ProfileIDs)
	case backend.HostEndpointKey:
		v := update.Value.(*backend.HostEndpoint)
		cbs.index.UpdateLabels(update.Key, v.Labels, v.ProfileIDs)
	case backend.ProfileLabelsKey:
		v := update.Value.(map[string]string)
		cbs.index.UpdateParentLabels(k.Name, v)
	default:
		glog.Errorf("Unexpected update type: %#v", update)
	}
}

func (cbs *evalCmd) onMatchStarted(selId, labelId interface{}) {
	println(endpointName(labelId))
}

func (cbs *evalCmd) onMatchStopped(selId, labelId interface{}) {
	glog.Errorf("Unexpected match stopped event: %v, %v", selId, labelId)
}
