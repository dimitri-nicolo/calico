// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package capture

import (
	"context"
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"k8s.io/utils/strings"

	"github.com/projectcalico/calico/felix/jitter"
	"github.com/projectcalico/calico/felix/proto"
	client "github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	cerrors "github.com/projectcalico/calico/libcalico-go/lib/errors"
	"github.com/projectcalico/calico/libcalico-go/lib/options"
)

// StatusWriter updates PacketCapture with a PacketCaptureStatus. It receives update from the data plane
// and updates the status of the resource
type StatusWriter struct {
	hostname             string
	packetCaptureDir     string
	calicoClient         client.PacketCaptureInterface
	updatesFromDataPlane chan *proto.PacketCaptureStatusUpdate
	pendingUpdates       map[*proto.PacketCaptureID]*proto.PacketCaptureStatusUpdate
	tick                 time.Duration
	done                 chan struct{}
}

// NewStatusWriter creates a *StatusWriter for the current node that has hostname as an identifier with packetCaptureDir
// as the packet capture files location. It updates the status for PacketCapture that match workload endpoints on this
// node. The status updates are received via updatesFromDataPlane channel from the data plane. Any interaction with the
// the PacketCapture Custom Resource will be done via calicoClient and its interface that it provides for PacketCapture.
// On failure to either get or update the PacketCapture resource, it will retry the operation at a duration specified by
// tick.
func NewStatusWriter(hostname, packetCaptureDir string, calicoClient client.PacketCaptureInterface,
	updatesFromDataPlane chan *proto.PacketCaptureStatusUpdate, tick time.Duration) *StatusWriter {
	return &StatusWriter{
		hostname:             hostname,
		packetCaptureDir:     packetCaptureDir,
		calicoClient:         calicoClient,
		updatesFromDataPlane: updatesFromDataPlane,
		pendingUpdates:       make(map[*proto.PacketCaptureID]*proto.PacketCaptureStatusUpdate),
		tick:                 tick,
		done:                 make(chan struct{}),
	}
}

func (sw *StatusWriter) Start() {
	go sw.handleStatusUpdate()
}

func (sw *StatusWriter) Stop() {
	sw.done <- struct{}{}
}

func (sw *StatusWriter) handleStatusUpdate() {
	var update *proto.PacketCaptureStatusUpdate
	var ticker *jitter.Ticker
	var retryC <-chan time.Time
	defer close(sw.done)

	for {
		// Block until we either get an update or it's time to tick a failed update.
		select {
		case <-sw.done:
			if ticker != nil {
				ticker.Stop()
			}
			close(sw.updatesFromDataPlane)
			return
		case update = <-sw.updatesFromDataPlane:
			log.WithField("CAPTURE", update.Id).Debugf("PacketCapture status update from dataplane driver: %#v", update.CaptureFiles)
			sw.pendingUpdates[update.Id] = update
		case <-retryC:
			log.WithField("CAPTURE", update.Id).Info("Retrying failed PacketCapture status update")
		}

		for id, update := range sw.pendingUpdates {
			// Try and reconcile the packet capture status data that have been recently received
			err := sw.reconcileStatusUpdate(update.Id.GetName(), update.Id.GetNamespace(), update.GetCaptureFiles(), convert(update.GetState()))
			if err == nil {
				delete(sw.pendingUpdates, id)
			} else if _, ok := err.(cerrors.ErrorResourceDoesNotExist); ok {
				// Drop pending updates for resources that no longer exist
				delete(sw.pendingUpdates, id)
			} else {
				// Start the retry mechanism in case the status updates fails
				if ticker == nil {
					// reconciling between a duration of a tick and two ticks seconds.
					ticker = jitter.NewTicker(sw.tick, sw.tick)
					retryC = ticker.C
				}
			}
		}

		// Cancel the retry mechanism if we no longer have any updates that need to reconciled
		if len(sw.pendingUpdates) == 0 && ticker != nil {
			ticker.Stop()
			ticker = nil
			retryC = nil
		}

	}
}

func convert(state proto.PacketCaptureStatusUpdate_PacketCaptureState) v3.PacketCaptureState {
	switch state {
	case proto.PacketCaptureStatusUpdate_FINISHED:
		return v3.PacketCaptureStateFinished
	case proto.PacketCaptureStatusUpdate_CAPTURING:
		return v3.PacketCaptureStateCapturing
	case proto.PacketCaptureStatusUpdate_SCHEDULED:
		return v3.PacketCaptureStateScheduled
	case proto.PacketCaptureStatusUpdate_ERROR:
		return v3.PacketCaptureStateError
	case proto.PacketCaptureStatusUpdate_WAITING_FOR_TRAFFIC:
		return v3.PacketCaptureStateWaitingForTraffic
	}

	return v3.PacketCaptureStateError
}

func (sw *StatusWriter) reconcileStatusUpdate(captureName, captureNamespace string, fileNames []string, state v3.PacketCaptureState) error {
	var captureID = strings.JoinQualifiedName(captureNamespace, captureName)
	// Read PacketCapture status resource from datastore and compare it with the fileNames from the dataplane.
	ctx, cancel := context.WithTimeout(context.Background(), sw.tick)
	packetCapture, err := sw.calicoClient.Get(ctx, captureNamespace, captureName, options.GetOptions{})
	cancel()
	if err != nil {
		log.WithField("CAPTURE", captureID).WithError(err).Error("Failed to read PacketCapture resource")
		return err
	}

	// Get last files from status and find the index of the previous status update for this node
	var lastFiles []string
	var lastState *v3.PacketCaptureState
	var index = -1
	for i, f := range packetCapture.Status.Files {
		if f.Node == sw.hostname {
			lastFiles = f.FileNames
			lastState = f.State
			index = i
		}
	}

	// Check if the files needs to be updated.
	if !reflect.DeepEqual(lastFiles, fileNames) || !reflect.DeepEqual(lastState, &state) {
		updateCtx, cancel := context.WithTimeout(context.Background(), sw.tick)
		var updatedPacketCapture = packetCapture.DeepCopy()
		if index == -1 {
			// No update for this node was previously written
			updatedPacketCapture.Status.Files = append(updatedPacketCapture.Status.Files, v3.PacketCaptureFile{
				Node:      sw.hostname,
				Directory: sw.packetCaptureDir,
				FileNames: fileNames,
				State:     &state,
			})
		} else {
			// Override the status update as the files have changed
			updatedPacketCapture.Status.Files[index].FileNames = fileNames
			updatedPacketCapture.Status.Files[index].State = &state
		}

		log.WithField("CAPTURE", captureID).Debugf("Updating status for node %s with %v", sw.hostname, fileNames)
		_, err := sw.calicoClient.Update(updateCtx, updatedPacketCapture, options.SetOptions{})
		cancel()
		if err != nil {
			// tick in some time.
			log.WithField("CAPTURE", captureID).WithError(err).Info("Failed updating node resource")
			return err
		}
		log.WithField("CAPTURE", captureID).Debugf("Updated PacketCapture status from %v to %v", lastFiles, fileNames)
	}

	return nil
}
