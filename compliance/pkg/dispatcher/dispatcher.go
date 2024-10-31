// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package dispatcher

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/calico/compliance/pkg/syncer"
)

const (
	numUpdatesPerPerfLog = 500
)

//TODO(rlb): Thinking that we might want to use resource type rather than the TypeMeta for doing
// fanout.

type DispatcherOnStatusUpdate func(syncer.StatusUpdate)
type DispatcherOnUpdate func(syncer.Update)

// Dispatcher provides methods to send status and updates, and for handlers to register for status and update
// notifications.
//
// Note: Unlike the syncer which can handle multiple updates in a single request the dispatcher only handles
// single updates at a time - this greatly simplifies the filtering processing. The upshot of this is that any
// correlation between a single set of resources in a syncer request needs to be handled at the syncer update
// stage.
type Dispatcher interface {
	OnStatusUpdate(status syncer.StatusUpdate)
	OnUpdate(update syncer.Update)
	RegisterOnStatusUpdateHandler(callback DispatcherOnStatusUpdate)
	RegisterOnUpdateHandler(kind metav1.TypeMeta, updateTypes syncer.UpdateType, callback DispatcherOnUpdate)
}

func NewDispatcher(name string) Dispatcher {
	return &dispatcher{
		resourceTypes:   map[metav1.TypeMeta]*resourceType{},
		outputPerfStats: outputPerfStats(),
		startSync:       time.Now(),
		clog:            log.WithField("name", name),
	}
}

type dispatcher struct {
	resourceTypes           map[metav1.TypeMeta]*resourceType
	onStatusUpdateCallbacks []DispatcherOnStatusUpdate

	// Performance statistics tracking
	outputPerfStats bool
	startSync       time.Time
	updateIdx       int
	updateTime      time.Duration
	memstats        runtime.MemStats

	clog *log.Entry
}

type resourceType struct {
	registrations []onUpdateRegistration
}

type onUpdateRegistration struct {
	types    syncer.UpdateType
	callback DispatcherOnUpdate
}

func (d *dispatcher) RegisterOnStatusUpdateHandler(callback DispatcherOnStatusUpdate) {
	d.onStatusUpdateCallbacks = append(d.onStatusUpdateCallbacks, callback)
}

func (d *dispatcher) RegisterOnUpdateHandler(kind metav1.TypeMeta, updateTypes syncer.UpdateType, callback DispatcherOnUpdate) {
	rt, ok := d.resourceTypes[kind]
	if !ok {
		// Initialise the registration. This will be used to convert the updates.
		rt = &resourceType{}
		d.resourceTypes[kind] = rt
	}
	rt.registrations = append(rt.registrations, onUpdateRegistration{updateTypes, callback})
}

// OnUpdate callback distributes the update across the set of components that registered for this resource type and
// update type. It is guaranteed not to be called at the same time as OnStatusUpdate.
func (d *dispatcher) OnUpdate(update syncer.Update) {
	registration, ok := d.resourceTypes[update.ResourceID.TypeMeta]
	if !ok {
		d.clog.WithField("resourceTypes", d.resourceTypes).Debugf("Update for unregistered resource type: %s", update.ResourceID)
		return
	}

	var startTime time.Time
	if d.outputPerfStats {
		startTime = time.Now()
	}

	// Invoke each callback in the registered order with the update provided the callback requires the current
	// update type.
	for _, reg := range registration.registrations {
		if update.Type&reg.types != 0 {
			reg.callback(update)
		}
	}

	if d.outputPerfStats {
		d.updateTime += time.Since(startTime)
		if d.updateIdx%numUpdatesPerPerfLog == 0 {
			runtime.ReadMemStats(&d.memstats)
			duration := d.updateTime
			if d.updateIdx != 0 {
				duration = duration / time.Duration(numUpdatesPerPerfLog)
			}
			fmt.Printf("NumUpdate = %v\tAvgUpdateDuration = %v\t"+
				"HeapAlloc = %v\tSys = %v\tNumGC = %v\nNumGoRoutines = %v\n",
				d.updateIdx, duration, d.memstats.HeapAlloc/1024,
				d.memstats.Sys/1024, d.memstats.NumGC, runtime.NumGoroutine(),
			)
			d.updateTime = 0
		}
		d.updateIdx++
	}
}

// OnStatusUpdate is a callback to indicate a change in synchronization state.
func (d *dispatcher) OnStatusUpdate(status syncer.StatusUpdate) {
	log.WithField("status", status).Debug("OnStatusUpdate")
	for _, cb := range d.onStatusUpdateCallbacks {
		cb(status)
	}
	if d.outputPerfStats {
		fmt.Printf("on status update (%s) after %v since startup\n", status, time.Since(d.startSync))
	}
}

// outputPerfStats returns true if the environment option indicates we should output to stout performance statistics.
func outputPerfStats() bool {
	rc, _ := strconv.ParseBool(os.Getenv("OUTPUT_PERF_STATS"))
	return rc
}
