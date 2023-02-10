// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package client

import (
	"context"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
)

// EventsInterface has methods related to events.
type EventsInterface interface {
	List(ctx context.Context, params v1.Params) (*v1.List[v1.Event], error)
}

// Events implements EventsInterface.
type events struct {
	restClient *rest.RESTClient
	clusterID  string
}

// newEvents returns a new EventsInterface bound to the supplied client.
func newEvents(c *client, cluster string) EventsInterface {
	return &events{restClient: c.restClient, clusterID: cluster}
}

// List gets the events for the given input params.
func (f *events) List(ctx context.Context, params v1.Params) (*v1.List[v1.Event], error) {
	events := v1.List[v1.Event]{}
	err := f.restClient.Post().
		Path("/api/v1/events").
		Params(params).
		Cluster(f.clusterID).
		Do(ctx).
		Into(&events)
	if err != nil {
		return nil, err
	}
	return &events, nil
}
