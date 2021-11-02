// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	"context"
	"sync"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tigera/lma/pkg/auth"
	"github.com/tigera/lma/pkg/k8s"

	log "github.com/sirupsen/logrus"
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
	GetServiceLabels(cluster string) (map[v1.NamespacedName]LabelSelectors, error)
	GetReplicaSetLabels(cluster string) (map[v1.NamespacedName]LabelSelectors, error)
	GetStatefulSetLabels(cluster string) (map[v1.NamespacedName]LabelSelectors, error)
	GetDaemonSetLabels(cluster string) (map[v1.NamespacedName]LabelSelectors, error)
	GetPodsLabels(cluster string) (map[v1.NamespacedName]LabelSelectors, error)

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

func (r *realServiceGraphBackend) GetPodsLabels(cluster string) (map[v1.NamespacedName]LabelSelectors, error) {
	cs, err := r.clientSetFactory.NewClientSetForApplication(cluster)
	if err != nil {
		return nil, err
	}

	var pods = make(map[v1.NamespacedName]LabelSelectors)
	podsList, err := cs.CoreV1().Pods("").List(r.ctx, metav1.ListOptions{})
	if err != nil {
		log.WithError(err).Errorf("Failed to list pods")
	}
	for _, pod := range podsList.Items {
		if len(pod.OwnerReferences) == 0 {
			var key = v1.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}
			pods[key] = AppendLabels(pods[key], pod.Labels)
		}
	}

	return pods, nil
}

func (r *realServiceGraphBackend) GetStatefulSetLabels(cluster string) (map[v1.NamespacedName]LabelSelectors, error) {
	cs, err := r.clientSetFactory.NewClientSetForApplication(cluster)
	if err != nil {
		return nil, err
	}

	var statefulSets = make(map[v1.NamespacedName]LabelSelectors)
	stsList, err := cs.AppsV1().StatefulSets("").List(r.ctx, metav1.ListOptions{})
	if err != nil {
		log.WithError(err).Errorf("Failed to list statefulSets")
	}
	for _, sts := range stsList.Items {
		var key = v1.NamespacedName{Name: sts.Name, Namespace: sts.Namespace}
		statefulSets[key] = AppendLabelSelectors(statefulSets[key], sts.Spec.Selector)
	}

	return statefulSets, nil
}

func (r *realServiceGraphBackend) GetDaemonSetLabels(cluster string) (map[v1.NamespacedName]LabelSelectors, error) {
	cs, err := r.clientSetFactory.NewClientSetForApplication(cluster)
	if err != nil {
		return nil, err
	}

	var daemonSets = make(map[v1.NamespacedName]LabelSelectors)
	dsList, err := cs.AppsV1().DaemonSets("").List(r.ctx, metav1.ListOptions{})
	if err != nil {
		log.WithError(err).Errorf("Failed to list daemonSets")
	}
	for _, ds := range dsList.Items {
		var key = v1.NamespacedName{Name: ds.Name, Namespace: ds.Namespace}
		daemonSets[key] = AppendLabelSelectors(daemonSets[key], ds.Spec.Selector)
	}

	return daemonSets, nil
}

func (r *realServiceGraphBackend) GetReplicaSetLabels(cluster string) (map[v1.NamespacedName]LabelSelectors, error) {
	cs, err := r.clientSetFactory.NewClientSetForApplication(cluster)
	if err != nil {
		return nil, err
	}

	var replicaSets = make(map[v1.NamespacedName]LabelSelectors)
	rsList, err := cs.AppsV1().ReplicaSets("").List(r.ctx, metav1.ListOptions{})
	if err != nil {
		log.WithError(err).Errorf("Failed to list replicaSets")
	}
	for _, rs := range rsList.Items {
		var key = v1.NamespacedName{Name: rs.Name, Namespace: rs.Namespace}
		replicaSets[key] = AppendLabelSelectors(replicaSets[key], rs.Spec.Selector)
	}

	return replicaSets, nil
}

func (r *realServiceGraphBackend) GetServiceLabels(cluster string) (map[v1.NamespacedName]LabelSelectors, error) {
	cs, err := r.clientSetFactory.NewClientSetForApplication(cluster)
	if err != nil {
		return nil, err
	}

	var services = make(map[v1.NamespacedName]LabelSelectors)
	svList, err := cs.CoreV1().Services("").List(r.ctx, metav1.ListOptions{})
	if err != nil {
		log.WithError(err).Errorf("Failed to list services")
	}
	for _, sv := range svList.Items {
		var key = v1.NamespacedName{Name: sv.Name, Namespace: sv.Namespace}
		services[key] = AppendLabels(services[key], sv.Spec.Selector)
	}

	return services, nil
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
	return NewRBACFilter(ctx, r.authz, r.clientSetFactory, rd.ServiceGraphRequest.Cluster)
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
	FlowConfig                   FlowConfig
	FlowConfigErr                error
	L3                           []L3Flow
	L3Err                        error
	L7                           []L7Flow
	L7Err                        error
	DNS                          []DNSLog
	DNSErr                       error
	Events                       []Event
	EventsErr                    error
	RBACFilter                   RBACFilter
	RBACFilterErr                error
	NameHelper                   NameHelper
	NameHelperErr                error
	ServiceLabels                map[v1.NamespacedName]LabelSelectors
	ServiceLabelsErr             error
	ReplicaSetLabels             map[v1.NamespacedName]LabelSelectors
	ReplicaSetLabelsErr          error
	StatefulSetLabels            map[v1.NamespacedName]LabelSelectors
	StatefulSetLabelsErr         error
	DaemonSetLabels              map[v1.NamespacedName]LabelSelectors
	DaemonSetLabelsErr           error
	PodsLabels                   map[v1.NamespacedName]LabelSelectors
	PodsLabelsErr                error
	lock                         sync.Mutex
	numCallsFlowConfig           int
	numCallsL3                   int
	numCallsL7                   int
	numCallsDNS                  int
	numCallsEvents               int
	numCallsRBACFilter           int
	numCallsNameHelper           int
	numCallsGetServiceLabels     int
	numCallsGetReplicaSetLabels  int
	numCallsGetStatefulSetLabels int
	numCallsGetDaemonSetLabels   int
	numCallsGetPodsLabels        int
	wgElastic                    sync.WaitGroup
	numBlockedElastic            int
}

func (m *MockServiceGraphBackend) GetServiceLabels(cluster string) (map[v1.NamespacedName]LabelSelectors, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.numCallsGetServiceLabels++
	return m.ServiceLabels, m.ServiceLabelsErr
}

func (m *MockServiceGraphBackend) GetPodsLabels(cluster string) (map[v1.NamespacedName]LabelSelectors, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.numCallsGetPodsLabels++
	return m.PodsLabels, m.PodsLabelsErr
}

func (m *MockServiceGraphBackend) GetReplicaSetLabels(cluster string) (map[v1.NamespacedName]LabelSelectors, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.numCallsGetReplicaSetLabels++
	return m.ReplicaSetLabels, m.ReplicaSetLabelsErr
}

func (m *MockServiceGraphBackend) GetStatefulSetLabels(cluster string) (map[v1.NamespacedName]LabelSelectors, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.numCallsGetStatefulSetLabels++
	return m.StatefulSetLabels, m.StatefulSetLabelsErr
}

func (m *MockServiceGraphBackend) GetDaemonSetLabels(cluster string) (map[v1.NamespacedName]LabelSelectors, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.numCallsGetDaemonSetLabels++
	return m.DaemonSetLabels, m.DaemonSetLabelsErr
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
