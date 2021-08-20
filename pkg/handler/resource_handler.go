// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package handler

import (
	"context"
	"fmt"

	"github.com/tigera/deep-packet-inspection/pkg/config"

	"github.com/projectcalico/libcalico-go/lib/set"

	"github.com/tigera/deep-packet-inspection/pkg/exec"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"strings"

	log "github.com/sirupsen/logrus"
	cache2 "github.com/tigera/deep-packet-inspection/pkg/cache"
	"github.com/tigera/deep-packet-inspection/pkg/processor"

	bapi "github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/selector"
)

const (
	keyPrefixDPI = "DeepPacketInspection"
)

type Handler interface {
	// Close closes all the internal goroutines and processes.
	Close()
	// OnUpdate handles changes to the resources passed in the CacheRequest array.
	OnUpdate(context.Context, []CacheRequest)
}

// CacheRequest is used to send resource updates from syncer.
type CacheRequest struct {
	UpdateType bapi.UpdateType
	KVPair     model.KVPair
}

// resourceHandler is an implementation of Handler interface.
type resourceHandler struct {
	// cache stores all the labels and selectors and also the current mapping between the them.
	cache cache2.SelectorAndLabelCache

	// nodeName has the current node name.
	nodeName string

	// wepKeyToIface maps WEP key to interface, this is used in combination with wepKeyToProcessors to restart
	// affected snort processes if interface changes.
	wepKeyToIface map[interface{}]string

	// wepKeyToProcessors maps WEP key to set of Processor as each WEP can map multiple DPI selectors, it is used in
	// combination with wepKeyToIface to restart affected snort processes if interface changes.
	wepKeyToProcessors map[interface{}]set.Set

	// dpiKeyToProcessor maps DPI key to its Processor.
	dpiKeyToProcessor map[interface{}]processor.Processor

	// dirtyItems is updated when resourceHandler receives updates and reset after those updates are processed.
	dirtyItems []dirtyItem

	calicoClient clientv3.Interface
	processor    newProcessor
	cfg          *config.Config
}

type newProcessor func(ctx context.Context, calicoClient clientv3.Interface, dpiKey interface{}, nodeName string,
	snortExecFn exec.Snort, snortAlertFileBasePath string, snortAlertFileSize int, snortCommunityRulesFile string) processor.Processor

type requestType int

const (
	labelOrSelectorMatchStarted requestType = iota
	labelOrSelectorMatchStopped
	ifaceUpdated
	ifaceDeleted
)

type dirtyItem struct {
	wepKey      interface{}
	dpiKey      interface{}
	ifaceName   string
	requestType requestType
}

func NewResourceController(calicoClient clientv3.Interface, nodeName string, cfg *config.Config, p newProcessor) Handler {
	hndler := &resourceHandler{
		wepKeyToIface:      make(map[interface{}]string),
		dpiKeyToProcessor:  make(map[interface{}]processor.Processor),
		wepKeyToProcessors: make(map[interface{}]set.Set),
		nodeName:           nodeName,
		calicoClient:       calicoClient,
		processor:          p,
		cfg:                cfg,
	}
	hndler.cache = cache2.NewSelectorAndLabelCache(hndler.onMatchStarted, hndler.onMatchStopped)
	return hndler
}

func (h *resourceHandler) OnUpdate(ctx context.Context, cacheRequests []CacheRequest) {
	for _, c := range cacheRequests {
		if wepKey, ok := c.KVPair.Key.(model.WorkloadEndpointKey); ok {
			switch c.UpdateType {
			case bapi.UpdateTypeKVNew, bapi.UpdateTypeKVUpdated:
				ep := c.KVPair.Value.(*model.WorkloadEndpoint)
				// If WEP interface has changed, add that to the dirtyItems list first before calling cache.UpdateLabels
				// this ensure new snort process are started using the correct WEP interface.
				if h.wepInterfaceUpdate(wepKey, ep.Name) {
					h.dirtyItems = append(h.dirtyItems, dirtyItem{
						wepKey:      wepKey,
						ifaceName:   ep.Name,
						requestType: ifaceUpdated,
					})
				}
				h.cache.UpdateLabels(c.KVPair.Key, ep.Labels)
			case bapi.UpdateTypeKVDeleted:
				// Call cache.DeleteLabel before adding to dirtyItems list, this ensures all
				// related the snort processes are stopped before deleting the WEP interface from wepKeyToIface.
				h.cache.DeleteLabel(c.KVPair.Key)
				h.dirtyItems = append(h.dirtyItems, dirtyItem{
					wepKey:      wepKey,
					requestType: ifaceDeleted,
				})
			default:
				log.Warn("Unknown update type for WorkloadEndpoint")
			}
		} else if k, ok := c.KVPair.Key.(model.Key); ok && strings.HasPrefix(k.String(), keyPrefixDPI) {
			switch c.UpdateType {
			case bapi.UpdateTypeKVNew, bapi.UpdateTypeKVUpdated:
				if dpi, ok := c.KVPair.Value.(*v3.DeepPacketInspection); ok {
					// Include namespace selector to the input selector
					var updatedSelector = fmt.Sprintf("(%s) && (%s == '%s')", dpi.Spec.Selector, v3.LabelNamespace, dpi.Namespace)
					sel, err := selector.Parse(updatedSelector)
					if err != nil {
						// This panic is only triggered due to programming error, the original selector in DPI resource
						// is validated by the apiserver during create/update operation, failure to parse updated selector
						// must be due to programming error when appending namespace selector.
						log.WithError(err).Panic("Failed to parse selector")
					}
					h.cache.UpdateSelector(c.KVPair.Key, sel)
				}
			case bapi.UpdateTypeKVDeleted:
				h.cache.DeleteSelector(c.KVPair.Key)
			default:
				log.Warn("Unknown update type for DeepPacketInspection")
			}
		} else {
			log.Warnf("Unknown object %#v", c)
		}
	}

	h.flushUpdates(ctx)
}

// Close calls close on all the processors running and tracking snort processes.
func (h *resourceHandler) Close() {
	for _, v := range h.dpiKeyToProcessor {
		v.Close()
	}
}

// onMatchStarted is called when there is a new WEP with label that matches the selector in DPI.
// It adds the WEP and DPI key to dirtyItems, to later start snort on the WEP interface.
func (h *resourceHandler) onMatchStarted(dpiKey, wepKey interface{}) {
	log.WithField("DPI", dpiKey).Debugf("Snort match available for WEP %v", wepKey)
	h.dirtyItems = append(h.dirtyItems, dirtyItem{
		wepKey:      wepKey,
		dpiKey:      dpiKey,
		requestType: labelOrSelectorMatchStarted,
	})
}

// onMatchStopped is called when previous WEP with label that matches the selector in DPI is no longer valid.
// It adds the WEP and DPI key to dirtyItems, to later stop snort on the WEP interface.
func (h *resourceHandler) onMatchStopped(dpiKey, wepKey interface{}) {
	log.WithField("DPI", dpiKey).Debugf("Stopping previous match for WEP %v", wepKey)
	h.dirtyItems = append(h.dirtyItems, dirtyItem{
		wepKey:      wepKey,
		dpiKey:      dpiKey,
		requestType: labelOrSelectorMatchStopped,
	})
}

// wepInterfaceUpdate returns true if old WEP interface is different from the new WEP interface passed or
// if it is a WEP not in cache.
func (h *resourceHandler) wepInterfaceUpdate(key model.WorkloadEndpointKey, iface string) bool {
	oldIface, ok := h.wepKeyToIface[key]
	return !ok || (oldIface != iface)
}

// flushUpdates processes all the items in the dirtyItems list.
// If WEP interface is updated or deleted, update the cache that maps WEP key to interface,
// If labels or selectors are updated, either add or remove the WEP interface from the processor
// which in turn starts/stops snort process on that interface.
func (h *resourceHandler) flushUpdates(ctx context.Context) {
	for _, i := range h.dirtyItems {
		switch i.requestType {
		case ifaceUpdated:
			oldIface := h.wepKeyToIface[i.wepKey]
			log.Debugf("Updating the cached WEP interface from %s to %s for WEP %v", oldIface, i.ifaceName, i.wepKey)
			h.wepKeyToIface[i.wepKey] = i.ifaceName
			// stop and remove all old WEP interfaces
			prcs, ok := h.wepKeyToProcessors[i.wepKey]
			if ok {
				prcs.Iter(func(p interface{}) error {
					p.(processor.Processor).Remove(i.wepKey.(model.WorkloadEndpointKey))
					return nil
				})
				// add the updated WEP interfaces
				prcs.Iter(func(p interface{}) error {
					p.(processor.Processor).Add(ctx, i.wepKey.(model.WorkloadEndpointKey), i.ifaceName)
					return nil
				})
			}
		case ifaceDeleted:
			log.Debugf("Deleting the cached WEP interface %s for WEP %v", i.ifaceName, i.wepKey)
			delete(h.wepKeyToIface, i.wepKey)
			delete(h.wepKeyToProcessors, i.wepKey)
		case labelOrSelectorMatchStarted:
			p, ok := h.dpiKeyToProcessor[i.dpiKey]
			if !ok {
				// Start and store new processor if it doesn't exist.
				p = h.processor(ctx, h.calicoClient, i.dpiKey, h.nodeName, exec.NewExec, h.cfg.SnortAlertFileBasePath, h.cfg.SnortAlertFileSize, h.cfg.SnortCommunityRulesFile)

				// Update the mapping of WEP interface to processor and also mapping of DPI key to processor
				v, ok := h.wepKeyToProcessors[i.wepKey]
				if !ok {
					v = set.New()
				}
				v.Add(p)
				h.wepKeyToProcessors[i.wepKey] = v
				h.dpiKeyToProcessor[i.dpiKey] = p
			}

			log.Debugf("Adding WEP interface %s to DPI %v", i.ifaceName, i.dpiKey)
			p.Add(ctx, i.wepKey.(model.WorkloadEndpointKey), h.wepKeyToIface[i.wepKey])
		case labelOrSelectorMatchStopped:
			p, ok := h.dpiKeyToProcessor[i.dpiKey]
			if ok {
				log.Debugf("Removing WEP %v for DPI %v", i.wepKey, i.dpiKey)
				p.Remove(i.wepKey.(model.WorkloadEndpointKey))
				if p.WEPInterfaceCount() == 0 {
					p.Close()
					v, ok := h.wepKeyToProcessors[i.wepKey]
					if ok {
						v.Discard(p)
					}
					h.wepKeyToProcessors[i.wepKey] = v
					delete(h.dpiKeyToProcessor, i.dpiKey)
				}
			}
		}
	}

	// Clear the dirtyItems list after handling all the changes.
	h.dirtyItems = []dirtyItem{}
}
