// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	"context"
	"sync"

	"github.com/tigera/lma/pkg/auth"
	"github.com/tigera/lma/pkg/k8s"

	lmav1 "github.com/tigera/lma/pkg/apis/v1"
	lmaelastic "github.com/tigera/lma/pkg/elastic"
)

// Sanity check the realServiceGraphBackend satisfies the ServiceGraphBackend interface.
var _ ServiceGraphBackend = &realServiceGraphBackend{}

type ServiceGraphBackend interface {
	// The underlying requests for the following methods are handled in the background and use the application context
	// which can be embedded in the backend instance, therefore these methods do not include context parameters in the
	// signature.
	GetFlowConfig(cluster string) (*FlowConfig, error)
	GetL3FlowData(cluster string, tr lmav1.TimeRange, fc *FlowConfig) ([]L3Flow, error)
	GetL7FlowData(cluster string, tr lmav1.TimeRange) ([]L7Flow, error)
	GetDNSData(cluster string, tr lmav1.TimeRange) ([]DNSLog, error)
	GetEvents(cluster string, tr lmav1.TimeRange) ([]Event, error)

	// These methods access data for a specific user request and therefore need to include the users request context.
	NewRBACFilter(ctx context.Context, rd *RequestData) (RBACFilter, error)
	NewNameHelper(ctx context.Context, rd *RequestData) (NameHelper, error)
}

type realServiceGraphBackend struct {
	ctx              context.Context
	authz            auth.RBACAuthorizer
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

func (r *realServiceGraphBackend) GetL3FlowData(cluster string, tr lmav1.TimeRange, fc *FlowConfig) ([]L3Flow, error) {
	return GetL3FlowData(r.ctx, r.elastic, cluster, tr, fc)
}

func (r *realServiceGraphBackend) GetDNSData(cluster string, tr lmav1.TimeRange) ([]DNSLog, error) {
	return GetDNSClientData(r.ctx, r.elastic, cluster, tr)
}

func (r *realServiceGraphBackend) GetL7FlowData(cluster string, tr lmav1.TimeRange) ([]L7Flow, error) {
	return GetL7FlowData(r.ctx, r.elastic, cluster, tr)
}

func (r *realServiceGraphBackend) GetEvents(cluster string, tr lmav1.TimeRange) ([]Event, error) {
	cs, err := r.clientSetFactory.NewClientSetForApplication(cluster)
	if err != nil {
		return nil, err
	}
	return GetEvents(r.ctx, r.elastic, cs, cluster, tr)
}

func (r *realServiceGraphBackend) NewRBACFilter(ctx context.Context, rd *RequestData) (RBACFilter, error) {
	return NewRBACFilter(ctx, r.authz, r.clientSetFactory, rd.HTTPRequest, rd.ServiceGraphRequest.Cluster)
}

func (r *realServiceGraphBackend) NewNameHelper(ctx context.Context, rd *RequestData) (NameHelper, error) {
	cs, err := r.clientSetFactory.NewClientSetForApplication(rd.ServiceGraphRequest.Cluster)
	if err != nil {
		return nil, err
	}
	return NewNameHelper(ctx, cs, rd.ServiceGraphRequest.SelectedView.HostAggregationSelectors)
}

// ---- Mock backend for testing ----

type MockServiceGraphBackend struct {
	FlowConfig         FlowConfig
	FlowConfigErr      error
	L3                 []L3Flow
	L3Err              error
	L7                 []L7Flow
	L7Err              error
	DNS                []DNSLog
	DNSErr             error
	Events             []Event
	EventsErr          error
	RBACFilter         RBACFilter
	RBACFilterErr      error
	NameHelper         NameHelper
	NameHelperErr      error
	lock               sync.Mutex
	numCallsFlowConfig int
	numCallsL3         int
	numCallsL7         int
	numCallsDNS        int
	numCallsEvents     int
	numCallsRBACFilter int
	numCallsNameHelper int
	wgElastic          sync.WaitGroup
	numBlockedElastic  int
}

func (m *MockServiceGraphBackend) waitElastic() {
	m.lock.Lock()
	m.numBlockedElastic++
	m.lock.Unlock()
	m.wgElastic.Wait()
	m.lock.Lock()
	m.numBlockedElastic--
	m.lock.Unlock()
}

func (m *MockServiceGraphBackend) GetFlowConfig(cluster string) (*FlowConfig, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.numCallsFlowConfig++
	if m.FlowConfigErr != nil {
		return nil, m.FlowConfigErr
	}
	return &m.FlowConfig, nil
}

func (m *MockServiceGraphBackend) GetL3FlowData(cluster string, tr lmav1.TimeRange, fc *FlowConfig) ([]L3Flow, error) {
	m.waitElastic()
	m.lock.Lock()
	defer m.lock.Unlock()
	m.numCallsL3++
	if m.L3Err != nil {
		return nil, m.L3Err
	}
	return m.L3, nil
}

func (m *MockServiceGraphBackend) GetL7FlowData(cluster string, tr lmav1.TimeRange) ([]L7Flow, error) {
	m.waitElastic()
	m.lock.Lock()
	defer m.lock.Unlock()
	m.numCallsL7++
	if m.L7Err != nil {
		return nil, m.L7Err
	}
	return m.L7, nil
}

func (m *MockServiceGraphBackend) GetDNSData(cluster string, tr lmav1.TimeRange) ([]DNSLog, error) {
	m.waitElastic()
	m.lock.Lock()
	defer m.lock.Unlock()
	m.numCallsDNS++
	if m.DNSErr != nil {
		return nil, m.DNSErr
	}
	return m.DNS, nil
}

func (m *MockServiceGraphBackend) GetEvents(cluster string, tr lmav1.TimeRange) ([]Event, error) {
	m.waitElastic()
	m.lock.Lock()
	defer m.lock.Unlock()
	m.numCallsEvents++
	if m.EventsErr != nil {
		return nil, m.EventsErr
	}
	return m.Events, nil
}

func (m *MockServiceGraphBackend) NewRBACFilter(ctx context.Context, rd *RequestData) (RBACFilter, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.numCallsRBACFilter++
	if m.RBACFilterErr != nil {
		return nil, m.RBACFilterErr
	}
	return m.RBACFilter, nil
}

func (m *MockServiceGraphBackend) NewNameHelper(ctx context.Context, rd *RequestData) (NameHelper, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.numCallsNameHelper++
	if m.NameHelperErr != nil {
		return nil, m.NameHelperErr
	}
	return m.NameHelper, nil
}

func (m *MockServiceGraphBackend) SetBlockElastic() {
	m.wgElastic.Add(1)
}

func (m *MockServiceGraphBackend) SetUnblockElastic() {
	m.wgElastic.Done()
}

func (m *MockServiceGraphBackend) GetNumCallsFlowConfig() int {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.numCallsFlowConfig
}

func (m *MockServiceGraphBackend) GetNumCallsL3() int {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.numCallsL3
}

func (m *MockServiceGraphBackend) GetNumCallsL7() int {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.numCallsL7
}

func (m *MockServiceGraphBackend) GetNumCallsDNS() int {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.numCallsDNS
}

func (m *MockServiceGraphBackend) GetNumCallsEvents() int {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.numCallsEvents
}

func (m *MockServiceGraphBackend) GetNumCallsRBACFilter() int {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.numCallsRBACFilter
}

func (m *MockServiceGraphBackend) GetNumCallsNameHelper() int {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.numCallsNameHelper
}

func (m *MockServiceGraphBackend) GetNumBlocked() int {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.numBlockedElastic
}
