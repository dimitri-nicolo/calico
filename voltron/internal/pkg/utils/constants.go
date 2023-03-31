package utils

const (
	// ClusterHeaderField represents the request header key used to determine which cluster
	// to proxy this request to, or signal which cluster this request originated from.
	ClusterHeaderField = "x-cluster-id"

	// TenantHeaderField represents the request header key used to determine the
	// tenant that this request belongs to.
	TenantHeaderField = "x-tenant-id"
)
