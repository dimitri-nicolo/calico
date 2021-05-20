// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package v1

import (
	"encoding/json"
	"fmt"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GraphEventType string

const (
	GraphEventTypeKubernetes = "kubernetes"
	GraphEventTypeAlert      = "alert"
)

type GraphEventID struct {
	Type           GraphEventType `json:"type,omitempty"`
	ID             string         `json:"id,omitempty"`
	NamespacedName `json:",inline"`
}

func (g GraphEventID) String() string {
	if g.Type == GraphEventTypeKubernetes {
		return fmt.Sprintf("%s/%s", g.Type, g.NamespacedName)
	}
	return fmt.Sprintf("%s/%s/%s", g.Type, g.NamespacedName, g.ID)
}

// Details of the event. This does not contain the full event details, but the original event may be cross referenced
// from the ID.
type GraphEventDetails struct {
	Severity    *int         `json:"severity,omitempty"`
	Description string       `json:"description,omitempty"`
	Timestamp   *metav1.Time `json:"time,omitempty"`
}

// GraphEvents is used to store event details. Stored as a map to handle deduplication, this is JSON marshaled as a
// slice.
type GraphEvents map[GraphEventID]GraphEventDetails

type graphEventWithID struct {
	ID                GraphEventID `json:"id"`
	GraphEventDetails `json:",inline"`
}

func (e GraphEvents) MarshalJSON() ([]byte, error) {
	var ids []graphEventWithID
	for id, ev := range e {
		ids = append(ids, graphEventWithID{
			ID:                id,
			GraphEventDetails: ev,
		})
	}
	sort.Slice(ids, func(i, j int) bool {
		if ids[i].Timestamp == nil && ids[j].Timestamp != nil {
			return true
		}
		return ids[i].Timestamp.Before(ids[j].Timestamp)
	})
	return json.Marshal(ids)
}
