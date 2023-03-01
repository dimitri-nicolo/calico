// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package middleware

import "context"

type key int

const (
	clusterIDKey key = iota
	tenantIDKey
)

// ClusterIDFromContext retrieves the cluster id from the context
func ClusterIDFromContext(ctx context.Context) string {
	v := ctx.Value(clusterIDKey)
	if v == nil {
		return ""
	}
	return v.(string)
}

// WithClusterID sets the x-cluster-id identifier on the context of a request
func WithClusterID(ctx context.Context, clusterID string) context.Context {
	return context.WithValue(ctx, clusterIDKey, clusterID)
}

// TenantIDFromContext retrieves the tenant id from the context
func TenantIDFromContext(ctx context.Context) string {
	v := ctx.Value(tenantIDKey)
	if v == nil {
		return ""
	}
	return v.(string)
}

// WithTenantID sets the x-tenant-id identifier on the context of a request
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantIDKey, tenantID)
}
