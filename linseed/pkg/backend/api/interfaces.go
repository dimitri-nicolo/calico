// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package api

import (
	"context"

	"k8s.io/apiserver/pkg/apis/audit"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

// FlowBackend defines the interface for interacting with L3 flows
type FlowBackend interface {
	List(context.Context, ClusterInfo, v1.L3FlowParams) (*v1.Results[v1.L3Flow], error)
}

// FlowLogBackend defines the interface for interacting with L3 flow logs
type FlowLogBackend interface {
	// Initialize initializes the backend and must be called prior to using this interface.
	// This should be called exactly once. Multiple calls to this function will have no effect.
	Initialize(context.Context) error

	// Create creates the given L3 logs.
	Create(context.Context, ClusterInfo, []v1.FlowLog) (*v1.BulkResponse, error)
}

// L7FlowBackend defines the interface for interacting with L7 flows.
type L7FlowBackend interface {
	List(context.Context, ClusterInfo, v1.L7FlowParams) ([]v1.L7Flow, error)
}

// L7LogBackend defines the interface for interacting with L7 flow logs.
type L7LogBackend interface {
	// Initialize initializes the backend and must be called prior to using this interface.
	// This should be called exactly once. Multiple calls to this function will have no effect.
	Initialize(context.Context) error

	// Create creates the given L7 logs.
	Create(context.Context, ClusterInfo, []L7Log) error
}

// DNSFlowBackend defines the interface for interacting with DNS flows
type DNSFlowBackend interface {
	List(context.Context, ClusterInfo, v1.DNSFlowParams) ([]v1.DNSFlow, error)
}

// DNSLogBackend defines the interface for interacting with DNS logs
type DNSLogBackend interface {
	// Initialize initializes the backend and must be called prior to using this interface.
	// This should be called exactly once. Multiple calls to this function will have no effect.
	Initialize(context.Context) error

	// Create creates the given logs.
	Create(context.Context, ClusterInfo, []v1.DNSLog) (*v1.BulkResponse, error)
}

// AuditBackend defines the interface for interacting with audit logs.
type AuditBackend interface {
	// Initialize initializes the backend and must be called prior to using this interface.
	// This should be called exactly once. Multiple calls to this function will have no effect.
	Initialize(context.Context) error

	// Create creates the given logs.
	Create(context.Context, v1.AuditLogType, ClusterInfo, []audit.Event) error

	// List lists logs that match the given parameters.
	List(context.Context, ClusterInfo, v1.AuditLogParams) ([]audit.Event, error)
}
