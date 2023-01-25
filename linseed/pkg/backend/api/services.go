// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package api

import (
	"context"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

// FlowLogBackend is the service definition used to make queries for flow logs
type FlowLogBackend interface {
	Initialize(ctx context.Context) error
	Create(ctx context.Context, i ClusterInfo, f FlowLog) error
}

// FlowBackend is the service definition used to make queries for flows
type FlowBackend interface {
	Initialize(ctx context.Context) error
	List(ctx context.Context, i ClusterInfo, opts v1.L3FlowParams) ([]v1.L3Flow, error)
}
