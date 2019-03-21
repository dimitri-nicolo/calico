// Copyright (c) 2018 Tigera, Inc. All rights reserved.
package cache

import (
	"github.com/projectcalico/libcalico-go/lib/apis/v3"
	bapi "github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/tigera/compliance/pkg/querycache/api"
	"github.com/tigera/compliance/pkg/querycache/dispatcherv1v3"
)

// NetworkSetCache implements the cache interface for the GlobalNetworkSet resource type.
// This interface consists of both the query and the event update interface.
type NetworkSetCache interface {
	GetNetworkSet(model.Key) api.Resource
	RegisterWithDispatcher(dispatcher dispatcherv1v3.Interface)
}

// NewNetworkSetCache creates a new instance of a NetworkSetCache.
func NewNetworkSetCache() NetworkSetCache {
	return &networksetCache{
		networksets: make(map[model.Key]api.Resource),
	}
}

// networksetCache implements NetworkSetCache.
type networksetCache struct {
	// The networksets keyed off the resource key.
	networksets map[model.Key]api.Resource
}

func (c *networksetCache) GetNetworkSet(key model.Key) api.Resource {
	if netset := c.getNetworkSet(key); netset != nil {
		return netset
	}
	return nil
}

func (c *networksetCache) getNetworkSet(key model.Key) api.Resource {
	return c.networksets[key]
}

func (c *networksetCache) onUpdate(update dispatcherv1v3.Update) {
	uv3 := update.UpdateV3
	switch uv3.UpdateType {
	case bapi.UpdateTypeKVNew:
		c.networksets[uv3.Key] = uv3.Value.(api.Resource)
	case bapi.UpdateTypeKVUpdated:
		c.networksets[uv3.Key] = uv3.Value.(api.Resource)
	case bapi.UpdateTypeKVDeleted:
		delete(c.networksets, uv3.Key)
	}
}

func (c *networksetCache) RegisterWithDispatcher(dispatcher dispatcherv1v3.Interface) {
	dispatcher.RegisterHandler(v3.KindGlobalNetworkSet, c.onUpdate)
}
