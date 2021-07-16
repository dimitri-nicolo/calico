// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.

package intdataplane

import (
	"regexp"
	"strings"

	"github.com/projectcalico/felix/multidict"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/capture"

	"github.com/projectcalico/felix/ifacemonitor"
	"github.com/projectcalico/felix/proto"
	"github.com/projectcalico/libcalico-go/lib/set"
)

type captureManager struct {
	wlIfacesRegexp        *regexp.Regexp
	pendingIfaceUpdates   map[string]ifacemonitor.State
	activeUpIfaces        set.Set
	pendingWlEpUpdates    map[proto.WorkloadEndpointID]*proto.WorkloadEndpoint
	activeWlEndpoints     map[proto.WorkloadEndpointID]*proto.WorkloadEndpoint
	pendingPacketCaptures map[capture.Key]*proto.WorkloadEndpointID
	ifaceToPacketCapture  multidict.StringToIface
	activePacketCaptures  capture.ActiveCaptures
}

// newCaptureManager buffers capture activation/deactivation commands until interfaces and marked up and running
// packet capture updates are a tuple formed by a capture id and workload endpoint id
// The updates will be buffered until the interfaces that correspond to the workload endpoint are up
func newCaptureManager(captures capture.ActiveCaptures, wlInterfacePrefixes []string) *captureManager {
	captureManager := captureManager{}
	captureManager.wlIfacesRegexp = regexp.MustCompile("^(" + strings.Join(wlInterfacePrefixes, "|") + ").*")
	captureManager.activeUpIfaces = set.New()
	captureManager.pendingIfaceUpdates = make(map[string]ifacemonitor.State)
	captureManager.activeWlEndpoints = make(map[proto.WorkloadEndpointID]*proto.WorkloadEndpoint)
	captureManager.pendingWlEpUpdates = make(map[proto.WorkloadEndpointID]*proto.WorkloadEndpoint)
	captureManager.activePacketCaptures = captures
	captureManager.pendingPacketCaptures = make(map[capture.Key]*proto.WorkloadEndpointID)
	captureManager.ifaceToPacketCapture = multidict.NewStringToIface()

	return &captureManager
}

func (c *captureManager) OnUpdate(protoBufMsg interface{}) {
	log.WithField("msg", protoBufMsg).Debug("Received message")
	switch msg := protoBufMsg.(type) {
	case *proto.WorkloadEndpointUpdate:
		// store workload endpoint id to an workload endpoint
		c.pendingWlEpUpdates[*msg.Id] = msg.Endpoint
	case *proto.WorkloadEndpointRemove:
		// store workload endpoint id to nil
		c.pendingWlEpUpdates[*msg.Id] = nil
	case *proto.PacketCaptureUpdate:
		// store a packet capture id to workload endpoint id
		var key = capture.Key{
			WorkloadEndpointId: msg.Endpoint.WorkloadId,
			CaptureName:        msg.Id.Name,
			Namespace:          msg.Id.Namespace,
		}
		c.pendingPacketCaptures[key] = msg.Endpoint
	case *proto.PacketCaptureRemove:
		var key = capture.Key{
			WorkloadEndpointId: msg.Endpoint.WorkloadId,
			CaptureName:        msg.Id.Name,
			Namespace:          msg.Id.Namespace,
		}
		if val, found := c.pendingPacketCaptures[key]; found && val != nil {
			// delete any pending packet captures starts that have not been issued
			delete(c.pendingPacketCaptures, key)
		} else {
			// store a packet capture id to workload endpoint id
			c.pendingPacketCaptures[key] = nil
		}
	case *ifaceUpdate:
		// store interface name to its state
		c.pendingIfaceUpdates[msg.Name] = msg.State
	}
}

func (c *captureManager) CompleteDeferredWork() error {
	// resolve any interfaces to active interfaces
	// pending interface updates will not be cleared at this
	// stage
	for ifaceName, state := range c.pendingIfaceUpdates {
		if state == ifacemonitor.StateUp && c.wlIfacesRegexp.MatchString(ifaceName) {
			c.activeUpIfaces.Add(ifaceName)
		} else {
			c.activeUpIfaces.Discard(ifaceName)
		}
	}

	// resolve any workload endpoints to active workload endpoints
	for k, v := range c.pendingWlEpUpdates {
		if v != nil {
			c.activeWlEndpoints[k] = v
		} else {
			delete(c.activeWlEndpoints, k)
		}
		delete(c.pendingWlEpUpdates, k)
	}

	// resolve any packet capture to active workload endpoints and active interfaces
	for k, v := range c.pendingPacketCaptures {
		// A pending packet capture buffers any start/stop command until it is matched by
		// an active workload endpoint and active interfaces. Captures will be held between
		// batches until the workload endpoint has been matched and the interface state is
		// marked as UP.
		if v != nil {
			workload, hasAWorkloadEndpoint := c.activeWlEndpoints[*v]
			// We only start a capture if both the conditions below are met
			// Otherwise, a capture will not be marked as active
			if hasAWorkloadEndpoint && c.activeUpIfaces.Contains(workload.Name) {
				err := c.activePacketCaptures.Add(k, workload.Name)
				if err != nil {
					log.WithField("CAPTURE", k.CaptureName).WithError(err).Error("Failed to start capture")
					continue
				}
				c.ifaceToPacketCapture.Put(workload.Name, k)
				// we delete the pending capture because we have an active workload endpoint matching the update event
				delete(c.pendingPacketCaptures, k)
			}
		} else {
			err, iface := c.activePacketCaptures.Remove(k)
			if err != nil {
				log.WithField("CAPTURE", k.CaptureName).WithError(err).Error("Failed to stop capture")
				continue
			}
			c.ifaceToPacketCapture.Discard(iface, k)
			// we delete the pending capture because we processed the removal event (this means that a workload endpoint
			// has cannot be matched against a capture any more - the endpoint was deleted or the labels selector is not
			// being matched)
			delete(c.pendingPacketCaptures, k)
		}
	}

	// We apply again any interface updates; In case an interface went up / down
	// while the capture is still active, we will start/stop the capture gracefully
	for ifaceName, state := range c.pendingIfaceUpdates {
		if c.wlIfacesRegexp.MatchString(ifaceName) {
			switch state {
			case ifacemonitor.StateUp:
				c.ifaceToPacketCapture.Iter(ifaceName, func(value interface{}) {
					// In case the capture was already started, Add will
					// return an error. In case an interface went up after
					// being marked as down and the capture was not deleted,
					// it will start the capture
					var err = c.activePacketCaptures.Add(value.(capture.Key), ifaceName)
					if err != nil && err != capture.ErrDuplicate {
						log.WithField("CAPTURE", value.(capture.Key).CaptureName).WithError(err).Error("Failed to start capture")
					}
				})
			case ifacemonitor.StateDown:
				c.ifaceToPacketCapture.Iter(ifaceName, func(value interface{}) {
					// In case the capture was already stopped, Remove will
					// return an error. In case an interface went down after
					// being marked as up and the capture was not deleted,
					// it will stop the capture
					var err, _ = c.activePacketCaptures.Remove(value.(capture.Key))
					if err != nil && err != capture.ErrNotFound {
						log.WithField("CAPTURE", value.(capture.Key).CaptureName).WithError(err).Error("Failed to stop capture")
					}
				})
			}
			delete(c.pendingIfaceUpdates, ifaceName)
		}
	}

	return nil
}
