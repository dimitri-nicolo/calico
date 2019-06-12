// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package clusters

import (
	"encoding/json"
	"net/url"
	"strings"
)

// Cluster contains metadata used to track a specific cluster
type Cluster struct {
	ID          string  `json:"id"`
	DisplayName string  `json:"displayName"`
	TargetURL   url.URL `json:"targetURL,omitempty"`
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
