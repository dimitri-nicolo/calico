// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package calc

import (
	"fmt"
	"reflect"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/labelindex"

	"github.com/projectcalico/felix/dispatcher"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	sel "github.com/projectcalico/libcalico-go/lib/selector"
)

// PacketCaptureCalculator will match local workload endpoints against a packet capture resource
// by matching labels with the packet capture selector
type PacketCaptureCalculator struct {
	// Cache all packet captures
	allPacketCaptures map[model.ResourceKey]*v3.PacketCapture

	// Label index, matching packet capture selectors against local endpoints.
	labelIndex *labelindex.InheritIndex

	// Packet Capture Callback to output the start/stop commands
	packetCaptureCallbacks
}

// NewPacketCaptureCalculator creates a new PacketCalculator with a given set of callback
// The callbacks will be used to inform when a match started/stop for a local endpoint
func NewPacketCaptureCalculator(callbacks packetCaptureCallbacks) *PacketCaptureCalculator {
	pcc := &PacketCaptureCalculator{}
	pcc.allPacketCaptures = make(map[model.ResourceKey]*v3.PacketCapture)
	pcc.labelIndex = labelindex.NewInheritIndex(pcc.onMatchStarted, pcc.onMatchStopped)
	pcc.packetCaptureCallbacks = callbacks
	return pcc
}

func (pcc *PacketCaptureCalculator) onMatchStarted(selID, labelId interface{}) {
	log.WithField("CAPTURE", selID).Infof("Start matching %v to packet capture", labelId)
	pcc.OnPacketCaptureActive(selID.(model.ResourceKey), labelId.(model.WorkloadEndpointKey))
}

func (pcc *PacketCaptureCalculator) onMatchStopped(selID, labelId interface{}) {
	captureKey := selID.(model.ResourceKey)
	log.WithField("CAPTURE", selID).Debugf("Stop matching %v to packet capture", labelId)
	pcc.OnPacketCaptureInactive(captureKey, labelId.(model.WorkloadEndpointKey))
}

func (pcc *PacketCaptureCalculator) RegisterWith(localEndpointDispatcher, allUpdDispatcher *dispatcher.Dispatcher) {
	// It needs local workload endpoints
	localEndpointDispatcher.Register(model.WorkloadEndpointKey{}, pcc.OnUpdate)

	// and profile labels and tags
	allUpdDispatcher.Register(model.ProfileLabelsKey{}, pcc.OnUpdate)
	allUpdDispatcher.Register(model.ProfileTagsKey{}, pcc.OnUpdate)
	// and packet captures.
	allUpdDispatcher.Register(model.ResourceKey{}, pcc.OnUpdate)
}

func (pcc *PacketCaptureCalculator) OnUpdate(update api.Update) (_ bool) {
	switch key := update.Key.(type) {
	case model.WorkloadEndpointKey:
		// updating index labels and matching selectors
		pcc.labelIndex.OnUpdate(update)
	case model.ProfileLabelsKey:
		// updating index labels and matching selectors
		pcc.labelIndex.OnUpdate(update)
	case model.ProfileTagsKey:
		// updating index labels and matching selectors
		pcc.labelIndex.OnUpdate(update)
	case model.ResourceKey:
		switch key.Kind {
		case v3.KindPacketCapture:
			if update.Value != nil {
				old, found := pcc.allPacketCaptures[key]
				if found && reflect.DeepEqual(old, update.Value.(*v3.PacketCapture)) {
					log.WithField("key", update.Key).Debug("No-op policy change; ignoring.")
					return
				}

				pcc.updatePacketCapture(update.Value.(*v3.PacketCapture), key)
			} else {
				pcc.deletePacketCapture(key)
			}
		default:
			// Ignore other kinds of v3 resource.
		}
	default:
		log.Infof("Ignoring unexpected update: %v %#v",
			reflect.TypeOf(update.Key), update)
	}

	return
}

func (pcc *PacketCaptureCalculator) updatePacketCapture(capture *v3.PacketCapture, key model.ResourceKey) {
	sel := pcc.parseSelector(capture)
	// add/update the packet capture value
	pcc.allPacketCaptures[key] = capture
	// update selector index and start matching against workload endpoints
	pcc.labelIndex.UpdateSelector(key, sel)
}

func (pcc *PacketCaptureCalculator) parseSelector(capture *v3.PacketCapture) sel.Selector {
	// update the selector with the namespace selector
	var updatedSelector = fmt.Sprintf("(%s) && (%s == '%s')", capture.Spec.Selector, v3.LabelNamespace, capture.Namespace)
	sel, err := sel.Parse(updatedSelector)
	if err != nil {
		log.WithError(err).Panic("Failed to parse selector")
	}
	return sel
}

func (pcc *PacketCaptureCalculator) deletePacketCapture(key model.ResourceKey) {
	// delete all traces of the packet resource
	delete(pcc.allPacketCaptures, key)
	// delete selector index and stop matching against workload endpoints
	pcc.labelIndex.DeleteSelector(key)
}
