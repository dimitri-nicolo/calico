package api

import (
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

// labelsMap keeps a map from keys to unique values.
type labelsMap struct {
	labels map[string]set.Set[string]
}

type LabelsMapInterface interface {
	GetLabels() map[string]set.Set[string]
	SetLabels(key string, values ...string) map[string]set.Set[string]
}

func NewLabelsMap() LabelsMapInterface {
	return &labelsMap{
		labels: make(map[string]set.Set[string]),
	}
}

func (lm *labelsMap) GetLabels() map[string]set.Set[string] {
	return lm.labels
}

func (lm *labelsMap) SetLabels(key string, values ...string) map[string]set.Set[string] {
	// add new key to the map
	if lm.labels[key] == nil {
		lm.labels[key] = set.New[string]()
	}

	// add new value(s) for an existing key
	lm.labels[key].AddAll(values)

	return lm.labels
}
