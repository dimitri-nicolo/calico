// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package xrefcache_test

import (
	. "github.com/onsi/ginkgo"

	"github.com/tigera/compliance/pkg/syncer"
	"github.com/tigera/compliance/pkg/xrefcache"
)

var _ = Describe("xref cache", func() {
	// Ensure  the client resource list is in-sync with the resource helper.
	It("should support in-sync and complete with no injected configuration", func() {
		cache := xrefcache.NewXrefCache()
		cache.OnStatusUpdate(syncer.StatusUpdate{
			Type: syncer.StatusTypeInSync,
		})
		cache.OnStatusUpdate(syncer.StatusUpdate{
			Type: syncer.StatusTypeComplete,
		})
	})
})
