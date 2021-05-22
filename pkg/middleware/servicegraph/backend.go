package servicegraph

import (
	"context"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
	"github.com/tigera/es-proxy/pkg/k8s"
	lmaelastic "github.com/tigera/lma/pkg/elastic"
)

type ServiceGraphBackend interface {
	GetFlowConfig(cluster string) (*FlowConfig, error)
	GetL3FlowData(cluster string, tr v1.TimeRange, fc *FlowConfig) ([]L3Flow, error)
	GetL7FlowData(cluster string, tr v1.TimeRange) ([]L7Flow, error)
	GetEvents(cluster string, tr v1.TimeRange) ([]Event, error)
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
