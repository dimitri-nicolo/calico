// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package client

import (
	"context"
	"encoding/json"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
)

// FlowLogsInterface has methods related to flowLogs.
type FlowLogsInterface interface {
	List(context.Context, v1.Params) (*v1.List[v1.FlowLog], error)
	Create(context.Context, []v1.FlowLog) (*v1.BulkResponse, error)
}

// FlowLogs implements FlowLogsInterface.
type flowLogs struct {
	restClient *rest.RESTClient
	clusterID  string
}

// newFlowLogs returns a new FlowLogsInterface bound to the supplied client.
func newFlowLogs(c *client, cluster string) FlowLogsInterface {
	return &flowLogs{restClient: c.restClient, clusterID: cluster}
}

// List gets the flowLogs for the given input params.
func (f *flowLogs) List(ctx context.Context, params v1.Params) (*v1.List[v1.FlowLog], error) {
	flowLogs := v1.List[v1.FlowLog]{}
	err := f.restClient.Post().
		Path("/api/v1/flows/network/logs").
		Params(params).
		Cluster(f.clusterID).
		Do(ctx).
		Into(&flowLogs)
	if err != nil {
		return nil, err
	}
	return &flowLogs, nil
}

func (f *flowLogs) Create(ctx context.Context, flowLogs []v1.FlowLog) (*v1.BulkResponse, error) {
	var err error
	body := []byte{}
	for _, e := range flowLogs {
		// Add each item, separated by a newline.
		out, err := json.Marshal(e)
		if err != nil {
			return nil, err
		}
		body = append(body, out...)
		body = append(body, []byte("\n")...)
	}

	resp := v1.BulkResponse{}
	err = f.restClient.Post().
		Path("/api/v1/bulk/flows/network/logs").
		Cluster(f.clusterID).
		BodyJSON(body).
		ContentType("application/x-ndjson").
		Do(ctx).
		Into(&resp)
	return &resp, err
}
