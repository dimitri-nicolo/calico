// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package xrefcache

import (
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/projectcalico/calico/compliance/pkg/syncer"
)

// cacheEntryCommon should be embedded in each concrete CacheEntry type to provide the various internal handling methods
// for each.
type cacheEntryCommon struct {
	updateTypes syncer.UpdateType
	inScope     bool
	id          v3.ResourceID
}

// getUpdateTypes returns the accumulated update types for a resource that is being updated from a syncer update.
func (c *cacheEntryCommon) getUpdateTypes() syncer.UpdateType {
	return c.updateTypes
}

// addUpdateTypes adds the supplied update types to the accumlated set of updates for a resource that is being
// updated from a syncer update.
func (c *cacheEntryCommon) addUpdateTypes(u syncer.UpdateType) {
	c.updateTypes |= u
}

// resetUpdateTypes is called at the end of the syncer update processing to reset the accumlated set of updates
// for the resource being updated from a syncer update.
func (c *cacheEntryCommon) resetUpdateTypes() {
	c.updateTypes = 0
}

// setInscope marks the cache entry as in-scope according to the registered selectors. Once in-scope the resource
// remains in-scope.
func (c *cacheEntryCommon) setInscope() {
	c.inScope = true
}

// getInScopeFlag returns the inscope flag if this resource is in scope.
func (c *cacheEntryCommon) getInScopeFlag() syncer.UpdateType {
	if c.inScope {
		return EventInScope
	}
	return 0
}

// setDeleted flags this resource as deleted.
func (c *cacheEntryCommon) setDeleted() {
	c.updateTypes |= EventResourceDeleted
}

// isDeleted returns true if this resource was deleted.
func (c *cacheEntryCommon) isDeleted() bool {
	return c.updateTypes&EventResourceDeleted != 0
}

// setResourceID sets the resource ID for this entry.
func (c *cacheEntryCommon) setResourceID(id v3.ResourceID) {
	c.id = id
}

// getResourceID returns the resource ID.
func (c *cacheEntryCommon) getResourceID() v3.ResourceID {
	return c.id
}
