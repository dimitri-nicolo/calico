// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package api

import (
	"context"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

// FlowBackend defines the interface for interacting with L3 flows
type FlowBackend interface {
	List(context.Context, ClusterInfo, v1.L3FlowParams) ([]v1.L3Flow, error)
}

// FlowLogBackend defines the interface for interacting with L3 flow logs
type FlowLogBackend interface {
	// Initialize initializes the backend and must be called prior to using this interface.
	// This should be called exactly once. Multiple calls to this function will have no effect.
	Initialize(context.Context) error

	// Create creates the given L3 logs.
	Create(context.Context, ClusterInfo, []FlowLog) error
}

// L7FlowBackend defines the interface for interacting with L7 flows.
type L7FlowBackend interface {
	List(context.Context, ClusterInfo, v1.L7FlowParams) ([]v1.L7Flow, error)
}

// L7FlowBackend defines the interface for interacting with L7 flow logs.
type L7LogBackend interface {
	// Initialize initializes the backend and must be called prior to using this interface.
	// This should be called exactly once. Multiple calls to this function will have no effect.
	Initialize(context.Context) error

	// Create creates the given L7 logs.
	Create(context.Context, ClusterInfo, []L7Log) error
}
