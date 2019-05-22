// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package xrefcache

import (
	"github.com/projectcalico/libcalico-go/lib/apis/v3"

	"github.com/tigera/compliance/pkg/syncer"
)

// TODO(rlb): This is currently an embedded struct in the cache entry data for each sub-cache. This is not necessary.
// This structure should be private to the cross ref cache internals and should contain the converted and augmented
// cache entry rather than the other way around. This should not be passed to the sub-cache at all.

// cacheEntryCommon is embedded in each concrete CacheEntry type to provide the updateInProgress identifiers used
// by the cache processing to handle sending of updates only at the end of a syncer update.
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
