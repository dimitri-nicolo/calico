// Copyright 2019 Tigera Inc. All rights reserved.

package filters

import (
	"context"
	"time"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
)

type auditLogCache struct {
	cache      map[string]bool
	auditLogDB db.AuditLog
}

func newAuditLogCache(al db.AuditLog) *auditLogCache {
	return &auditLogCache{
		cache:      make(map[string]bool),
		auditLogDB: al,
	}
}

func (c *auditLogCache) isKeyFiltered(ctx context.Context, k auditKey) (bool, error) {
	keyString := k.String()

	if cached, ok := c.cache[keyString]; ok {
		return cached, nil
	}

	filtered, err := c.auditLogDB.ObjectCreatedBetween(
		ctx, k.kind, k.namespace, k.name,
		k.timestamp.Add(CreatedBefore),
		k.timestamp.Add(time.Second*time.Duration(k.bucketSpan)).Add(CreatedAfter),
	)
	if err != nil {
		return false, err
	}
	if filtered {
		c.cache[keyString] = filtered
		return true, nil
	}

	filtered, err = c.auditLogDB.ObjectDeletedBetween(
		ctx, k.kind, k.namespace, k.name,
		k.timestamp.Add(DeletedBefore),
		k.timestamp.Add(time.Second*time.Duration(k.bucketSpan)).Add(DeletedAfter),
	)
	if err != nil {
		return false, err
	}
	c.cache[keyString] = filtered

	return filtered, nil
}

func (c *auditLogCache) areKeysFiltered(ctx context.Context, keys ...auditKey) (bool, error) {
	for _, k := range keys {
		if filtered, err := c.isKeyFiltered(ctx, k); err != nil {
			return false, err
		} else if filtered {
			return true, nil
		}
	}

	return false, nil
}
