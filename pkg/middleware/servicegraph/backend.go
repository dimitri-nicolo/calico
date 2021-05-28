// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	"context"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
	"github.com/tigera/es-proxy/pkg/k8s"
	lmaelastic "github.com/tigera/lma/pkg/elastic"
)

// Sanity check the realServiceGraphBackend satisfies the ServiceGraphBackend interface.
var _ ServiceGraphBackend = &realServiceGraphBackend{}

type ServiceGraphBackend interface {
	// The underlying requests for the following methods are handled in the background and use the application context
	// which can be embedded in the backend instance, therefore these methods do not include context parameters in the
	// signature.
	GetFlowConfig(cluster string) (*FlowConfig, error)
	GetL3FlowData(cluster string, tr v1.TimeRange, fc *FlowConfig) ([]L3Flow, error)
	GetL7FlowData(cluster string, tr v1.TimeRange) ([]L7Flow, error)
	GetEvents(cluster string, tr v1.TimeRange) ([]Event, error)

	// These methods access data for a specific user request and therefore need to include the users request context.
	GetRBACFilter(cxt context.Context, rd *RequestData) (RBACFilter, error)
	GetNameHelper(ctx context.Context, rd *RequestData) (NameHelper, error)
}

type realServiceGraphBackend struct {
	ctx              context.Context
	elastic          lmaelastic.Client
	clientSetFactory k8s.ClientSetFactory
}

func (r *realServiceGraphBackend) GetFlowConfig(cluster string) (*FlowConfig, error) {
	cs, err := r.clientSetFactory.NewClientSetForApplication(cluster)
	if err != nil {
		return nil, err
	}
	return GetFlowConfig(r.ctx, cs)
}

func (r *realServiceGraphBackend) GetL3FlowData(cluster string, tr v1.TimeRange, fc *FlowConfig) ([]L3Flow, error) {
	return GetL3FlowData(r.ctx, r.elastic, cluster, tr, fc)
}

func (r *realServiceGraphBackend) GetL7FlowData(cluster string, tr v1.TimeRange) ([]L7Flow, error) {
	return GetL7FlowData(r.ctx, r.elastic, cluster, tr)
}

func (r *realServiceGraphBackend) GetEvents(cluster string, tr v1.TimeRange) ([]Event, error) {
	cs, err := r.clientSetFactory.NewClientSetForApplication(cluster)
	if err != nil {
		return nil, err
	}
	return GetEvents(r.ctx, r.elastic, cs, cluster, tr)
}

func (r *realServiceGraphBackend) GetRBACFilter(ctx context.Context, rd *RequestData) (RBACFilter, error) {
	cs, err := r.clientSetFactory.NewClientSetForUser(rd.HTTPRequest, rd.ServiceGraphRequest.Cluster)
	if err != nil {
		return nil, err
	}
	return GetRBACFilter(ctx, cs)
}

func (r *realServiceGraphBackend) GetNameHelper(ctx context.Context, rd *RequestData) (NameHelper, error) {
	cs, err := r.clientSetFactory.NewClientSetForApplication(rd.ServiceGraphRequest.Cluster)
	if err != nil {
		return nil, err
	}
	return GetNameHelper(ctx, cs, rd.ServiceGraphRequest.SelectedView.HostAggregationSelectors)
}
