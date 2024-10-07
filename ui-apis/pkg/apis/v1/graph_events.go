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
	// The type of event.
	// - kubernetes event type corresponds to a kubernetes generated warning event.
	// - alert is an enterprise generated global alert
	// all other event types will be generated through intrusion detection and threat defense.
	Type GraphEventType `json:"type,omitempty"`

	// The ID of the event. Only valid for non-Kubernetes events, this corresponds to the elasticsearch ID of the
	// event document.
	ID string `json:"id,omitempty"`

	// The namespaced name of the event.
	// - For kubernetes event types this refers directly to the namespace and name of the Event resource.
	// - For global alerts the name will be set to the name of the alert.
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
	// The severity of the event. This is not set for Kubernetes events.
	Severity *int `json:"severity,omitempty"`

	// A summary of the event.
	Description string `json:"description,omitempty"`

	// The timestamp that the event (last) occurred at.
	Timestamp *metav1.Time `json:"time,omitempty"`
}

// GraphEvents is used to store event details. Stored as a map to handle deduplication, this is JSON marshaled as a
// slice.
//
//	[
//	  {
//	    "id": {
//	      "type": "kubernetes"
//	      "name": "n2",
//	      "namespace": "n",
//	    },
//	    "description": "A k8s thing occurred",
//	    "time": "1973-03-14T00:00:00Z"
//	  },
//	  {
//	    "id": {
//	      "type": "alert"
//	      "id": "aifn93hrbv_Ds",
//	      "name": "policy.pod",
//	    },
//	    "description": "A pod was modified occurred",
//	    "time": "1973-03-14T00:00:00Z"
//	  }
//	]
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

func (e *GraphEvents) UnmarshalJSON(b []byte) error {
	var events []graphEventWithID
	err := json.Unmarshal(b, &events)
	if err != nil {
		return err
	}

	*e = make(map[GraphEventID]GraphEventDetails)
	for _, event := range events {
		(*e)[event.ID] = event.GraphEventDetails
	}
	return nil
}
