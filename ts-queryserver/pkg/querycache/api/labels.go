package api

import (
	"sync"

	"github.com/projectcalico/calico/libcalico-go/lib/set"
)

type ResourceType string

const (
	LabelsResourceTypePods              ResourceType = "pods"
	LabelsResourceTypeNamespaces        ResourceType = "namespaces"
	LabelsResourceTypeServiceAccounts   ResourceType = "serviceAccounts"
	LabelsResourceTypeAllPolicies       ResourceType = "policies"
	LabelsResourceTypeAllNetworkSets    ResourceType = "networkSets"
	LabelsResourceTypeManagedClusters   ResourceType = "managedClusters"
	LabelsResourceTypeGlobalThreatFeeds ResourceType = "globalThreatfeeds"
)

// LabelsMap keeps a map from keys to unique values.
type LabelsMap struct {
	mu     sync.RWMutex
	labels map[string]set.Set[string]
}

type LabelsMapInterface interface {
	GetLabels() map[string]set.Set[string]
	SetLabels(labels map[string]set.Set[string]) map[string]set.Set[string]
}

func NewLabelsMap() *LabelsMap {
	return &LabelsMap{
		mu:     sync.RWMutex{},
		labels: map[string]set.Set[string]{},
	}
}

func (lm *LabelsMap) GetLabels() map[string]set.Set[string] {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	return lm.labels
}

func (lm *LabelsMap) SetLabels(key string, value ...string) map[string]set.Set[string] {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// add new key to the map
	if lm.labels == nil {
		lm.labels = map[string]set.Set[string]{}
	}

	if lm.labels[key] == nil {
		lm.labels[key] = set.New[string]()
	}

	// add new value(s) for an existing key
	lm.labels[key].AddAll(value)

	return lm.labels
}
