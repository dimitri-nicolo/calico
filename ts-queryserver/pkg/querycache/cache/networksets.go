// Copyright (c) 2018-2020 Tigera, Inc. All rights reserved.
package cache

import (
	log "github.com/sirupsen/logrus"
	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	bapi "github.com/projectcalico/calico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/calico/libcalico-go/lib/set"
	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/api"
	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/dispatcherv1v3"
)

// NetworkSetsCache implements the cache interface for the NetworkSet resource types.
// This interface consists of both the query and the event update interface.
type NetworkSetsCache interface {
	GetNetworkSet(model.Key) api.Resource
	GetNetworkSets(keys set.Set[model.Key]) []api.Resource
	RegisterWithDispatcher(dispatcher dispatcherv1v3.Interface)
}

// NewNetworkSetsCache creates a new instance of a NetworkSetsCache.
func NewNetworkSetsCache() NetworkSetsCache {
	return &networksetsCache{
		globalNetworkSets:      newNetworkSetCache(),
		networkSetsByNamespace: make(map[string]*networksetCache),
	}
}

type networksetsCache struct {
	globalNetworkSets      *networksetCache
	networkSetsByNamespace map[string]*networksetCache
}

// networksetCache implements NetworkSetCache.
type networksetCache struct {
	// The networksets keyed off the resource key.
	networksets map[model.Key]api.Resource
}

func newNetworkSetCache() *networksetCache {
	return &networksetCache{
		networksets: make(map[model.Key]api.Resource),
	}
}

func (c *networksetsCache) GetNetworkSet(key model.Key) api.Resource {
	if netset := c.getNetworkSet(key); netset != nil {
		return netset
	}
	return nil
}

// func GetNetworkSets gets list of networkset keys and returns corresponding (global)networksets resources.
// if the input keys is empty or nil, it will return all networksets and global networksets
func (c *networksetsCache) GetNetworkSets(keys set.Set[model.Key]) []api.Resource {
	if keys == nil || keys.Len() == 0 {
		return c.getAllNetworkSets()
	} else {
		networksets := make([]api.Resource, 0, keys.Len())
		for _, key := range keys.Slice() {
			networksets = append(networksets, c.getNetworkSet(key))
		}
		return networksets
	}
}

func (c *networksetsCache) getNetworkSet(key model.Key) api.Resource {
	nc := c.getNetworkSetCache(key, false)
	if nc == nil {
		return nil
	}
	return nc.networksets[key]
}

func (c *networksetsCache) getNetworkSetCache(nsKey model.Key, create bool) *networksetCache {
	if rKey, ok := nsKey.(model.ResourceKey); ok {
		switch rKey.Kind {
		case apiv3.KindGlobalNetworkSet:
			return c.globalNetworkSets
		case apiv3.KindNetworkSet:
			networkSets := c.networkSetsByNamespace[rKey.Namespace]
			if networkSets == nil && create {
				networkSets = newNetworkSetCache()
				c.networkSetsByNamespace[rKey.Namespace] = networkSets
			}
			return networkSets
		}
	}
	log.WithField("key", nsKey).Error("Unexpected resource in event type, expecting a v3 network set type")
	return nil
}

func (c *networksetsCache) getAllNetworkSets() []api.Resource {
	allNetworkSets := []api.Resource{}

	for _, nc := range c.networkSetsByNamespace {
		for _, ns := range nc.networksets {
			allNetworkSets = append(allNetworkSets, ns)
		}
	}

	for _, ns := range c.globalNetworkSets.networksets {
		allNetworkSets = append(allNetworkSets, ns)
	}

	return allNetworkSets
}

func (c *networksetsCache) onUpdate(update dispatcherv1v3.Update) {
	uv3 := update.UpdateV3
	nc := c.getNetworkSetCache(uv3.Key, true)
	if nc == nil {
		return
	}
	switch uv3.UpdateType {
	case bapi.UpdateTypeKVNew:
		nc.networksets[uv3.Key] = uv3.Value.(api.Resource)
	case bapi.UpdateTypeKVUpdated:
		nc.networksets[uv3.Key] = uv3.Value.(api.Resource)
	case bapi.UpdateTypeKVDeleted:
		delete(nc.networksets, uv3.Key)
	}
}

func (c *networksetsCache) RegisterWithDispatcher(dispatcher dispatcherv1v3.Interface) {
	dispatcher.RegisterHandler(apiv3.KindGlobalNetworkSet, c.onUpdate)
	dispatcher.RegisterHandler(apiv3.KindNetworkSet, c.onUpdate)
}
