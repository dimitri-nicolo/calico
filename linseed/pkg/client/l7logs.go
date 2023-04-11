// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/olivere/elastic/v7"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
)

// L7LogsInterface has methods related to L7 logs.
type L7LogsInterface interface {
	List(context.Context, v1.Params) (*v1.List[v1.L7Log], error)
	ListInto(context.Context, v1.Params, v1.Listable) error
	Create(context.Context, []v1.L7Log) (*v1.BulkResponse, error)
	Aggregations(context.Context, v1.Params) (elastic.Aggregations, error)
}

// L7Logs implements L7LogsInterface.
type l7Logs struct {
	restClient rest.RESTClient
	clusterID  string
}

// newL7Logs returns a new L7LogsInterface bound to the supplied client.
func newL7Logs(c Client, cluster string) L7LogsInterface {
	return &l7Logs{restClient: c.RESTClient(), clusterID: cluster}
}

// List gets the l7Logs for the given input params.
func (f *l7Logs) List(ctx context.Context, params v1.Params) (*v1.List[v1.L7Log], error) {
	l7Logs := v1.List[v1.L7Log]{}
	err := f.ListInto(ctx, params, &l7Logs)
	return &l7Logs, err
}

// ListInto gets the L7 Logs for the given input params.
func (f *l7Logs) ListInto(ctx context.Context, params v1.Params, l v1.Listable) error {
	if l == nil {
		return fmt.Errorf("list cannot be nil")
	}

	err := f.restClient.Post().
		Path("/l7/logs").
		Params(params).
		Cluster(f.clusterID).
		Do(ctx).
		Into(l)
	if err != nil {
		return err
	}

	return nil
}

func (f *l7Logs) Aggregations(ctx context.Context, params v1.Params) (elastic.Aggregations, error) {
	aggs := elastic.Aggregations{}
	err := f.restClient.Post().
		Path("/l7/logs/aggregation").
		Params(params).
		Cluster(f.clusterID).
		Do(ctx).
		Into(&aggs)
	if err != nil {
		return nil, err
	}
	return aggs, nil
}

func (f *l7Logs) Create(ctx context.Context, l7Logs []v1.L7Log) (*v1.BulkResponse, error) {
	var err error
	body := []byte{}
	for _, e := range l7Logs {
		if len(body) != 0 {
			// Include a separator between logs.
			body = append(body, []byte("\n")...)
		}

		// Add each item.
		out, err := json.Marshal(e)
		if err != nil {
			return nil, err
		}
		body = append(body, out...)
	}

	resp := v1.BulkResponse{}
	err = f.restClient.Post().
		Path("/l7/logs/bulk").
		Cluster(f.clusterID).
		BodyJSON(body).
		ContentType(rest.ContentTypeMultilineJSON).
		Do(ctx).
		Into(&resp)
	return &resp, err
}
