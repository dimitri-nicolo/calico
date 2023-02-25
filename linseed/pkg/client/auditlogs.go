// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package client

import (
	"context"
	"fmt"

	"github.com/projectcalico/calico/libcalico-go/lib/json"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
)

// AuditLogsInterface has methods related to audit logs.
type AuditLogsInterface interface {
	List(context.Context, v1.Params) (*v1.List[v1.AuditLog], error)
	Create(context.Context, v1.AuditLogType, []v1.AuditLog) (*v1.BulkResponse, error)
}

// AuditLogs implements AuditLogsInterface.
type audit struct {
	restClient rest.RESTClient
	clusterID  string
}

// newAuditLogs returns a new AuditLogsInterface bound to the supplied client.
func newAuditLogs(c Client, cluster string) AuditLogsInterface {
	return &audit{restClient: c.RESTClient(), clusterID: cluster}
}

// List gets the audit for the given input params.
func (f *audit) List(ctx context.Context, params v1.Params) (*v1.List[v1.AuditLog], error) {
	logs := v1.List[v1.AuditLog]{}
	err := f.restClient.Post().
		Path("/audit/logs").
		Params(params).
		Cluster(f.clusterID).
		Do(ctx).
		Into(&logs)
	if err != nil {
		return nil, err
	}
	return &logs, nil
}

func (f *audit) Create(ctx context.Context, logType v1.AuditLogType, auditl []v1.AuditLog) (*v1.BulkResponse, error) {
	var err error
	body := []byte{}
	for _, e := range auditl {
		// Add a newline between each. Do it here so that
		// we don't have a newline after the last event.
		if len(body) != 0 {
			body = append(body, []byte("\n")...)
		}

		// Add the item.
		out, err := json.Marshal(e)
		if err != nil {
			return nil, err
		}
		body = append(body, out...)
	}

	resp := v1.BulkResponse{}
	err = f.restClient.Post().
		Path(fmt.Sprintf("/audit/logs/%s/bulk", logType)).
		Cluster(f.clusterID).
		BodyJSON(body).
		ContentType(rest.ContentTypeMultilineJSON).
		Do(ctx).
		Into(&resp)
	return &resp, err
}
