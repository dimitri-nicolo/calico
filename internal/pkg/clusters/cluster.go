// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package clusters

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/tigera/voltron/internal/pkg/targets"
)

// Clusters is a mapping of Cluster objects keyed by their IDs
type Clusters struct {
	clusterMap map[string]*Cluster
	targets    *targets.Targets
}

// Cluster contains metadata used to track a specific cluster
type Cluster struct {
	ID          string  `json:"id"`
	DisplayName string  `json:"displayName"`
	TargetURL   url.URL `json:"targetURL,omitempty"`
}

// New creates a new cluster list
func New() *Clusters {
	c := make(map[string]*Cluster)
	t := targets.NewEmpty()

	return &Clusters{
		clusterMap: c,
		targets:    t,
	}
}

// Add inserts a new Cluster into the map
func (clusters *Clusters) Add(id string, cluster *Cluster) {
	clusters.clusterMap[id] = cluster
	clusters.targets.Add(id, cluster.TargetURL.String())
}

// List returns the full listing of Cluster entries
func (clusters *Clusters) List() map[string]*Cluster {
	return clusters.clusterMap
}

// ListTargets returns the full listing of targets from the Cluster entries
func (clusters *Clusters) ListTargets() map[string]*url.URL {
	return clusters.targets.List()
}

// GetTargets returns the Targets object
func (clusters *Clusters) GetTargets() *targets.Targets {
	return clusters.targets
}

// UnmarshalJSON implements the Unmarshaler interface, which allows us to handle
// unmarshalling a cluster object from JSON:
// https://golang.org/pkg/encoding/json/#Unmarshaler
func (c *Cluster) UnmarshalJSON(j []byte) error {
	var rawStrings map[string]string

	err := json.Unmarshal(j, &rawStrings)
	if err != nil {
		return err
	}

	for k, v := range rawStrings {
		if strings.ToLower(k) == "id" {
			c.ID = v
		}
		if strings.ToLower(k) == "displayname" {
			c.DisplayName = v
		}
		if strings.ToLower(k) == "targeturl" {
			u, err := url.Parse(v)
			if err != nil {
				return err
			}
			c.TargetURL = *u
		}
	}

	return nil
}

// MarshalJSON implements the Marshaler interface, which allows us to handle
// nmarshalling a cluster object into JSON:
// https://golang.org/pkg/encoding/json/#Marshaler
func (c Cluster) MarshalJSON() ([]byte, error) {
	clusterObj := struct {
		ID          string `json:"id"`
		DisplayName string `json:"displayName"`
		TargetURL   string `json:"targetURL,omitempty"`
	}{
		ID:          c.ID,
		DisplayName: c.DisplayName,
		TargetURL:   c.TargetURL.String(),
	}

	return json.Marshal(clusterObj)
}
