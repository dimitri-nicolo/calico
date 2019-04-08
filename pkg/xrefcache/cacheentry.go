// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package xrefcache

import (
	"github.com/tigera/compliance/pkg/syncer"
)

// cacheEntryCommon is embedded in each concrete CacheEntry type to provide the updateInProgress identifiers used
// by the cache processing to handle sending of updates only at the end of a syncer update.
type cacheEntryCommon struct {
	updateTypes syncer.UpdateType
	inScope     bool
}

// getUpdateTypes returns the accumulated update types for a resource that is being updated from a syncer update.
func (c *cacheEntryCommon) getUpdateTypes() syncer.UpdateType {
	return c.updateTypes
}

// setUpdateTypes adds the supplied update types to the accumlated set of updates for a resource that is being
// updated from a syncer update.
func (c *cacheEntryCommon) setUpdateTypes(u syncer.UpdateType) {
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

func (c *cacheEntryCommon) getInScopeFlag() syncer.UpdateType {
	if c.inScope {
		return EventInScope
	}
	return 0
}
