// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package client

import "github.com/projectcalico/calico/linseed/pkg/client/rest"

type mockCient struct {
	restClient *rest.RESTClient
	tenant     string
}

func (c *mockCient) RESTClient() *rest.RESTClient {
	return nil
}

// L3Flows returns an interface for managing v1.L3Flow resources.
func (c *mockCient) L3Flows(cluster string) L3FlowsInterface {
	return newL3Flows(c, cluster)
}

// L7Flows returns an interface for managing v1.L7Flow resources.
func (c *mockCient) L7Flows(cluster string) L7FlowsInterface {
	return newL7Flows(c, cluster)
}

// DNSFlows returns an interface for managing v1.DNSFlow resources.
func (c *mockCient) DNSFlows(cluster string) DNSFlowsInterface {
	return newDNSFlows(c, cluster)
}

// Events returns an interface for managing v1.Events resources.
func (c *mockCient) Events(cluster string) EventsInterface {
	return newEvents(c, cluster)
}

// FlowLogs returns an interface for managing v1.FlowLog resources.
func (c *mockCient) FlowLogs(cluster string) FlowLogsInterface {
	return newFlowLogs(c, cluster)
}

// DNSLogs returns an interface for managing v1.DNSLog resources.
func (c *mockCient) DNSLogs(cluster string) DNSLogsInterface {
	return newDNSLogs(c, cluster)
}

// L7Logs returns an interface for managing v1.L7Log resources.
func (c *mockCient) L7Logs(cluster string) L7LogsInterface {
	return newL7Logs(c, cluster)
}

// AuditLogs returns an interface for managing v1.AuditLog resources.
func (c *mockCient) AuditLogs(cluster string) AuditLogsInterface {
	return newAuditLogs(c, cluster)
}

func NewMockClient(tenantID string) Client {
	return &mockCient{
		restClient: nil, // TODO: Right now, this isn't a functional client.
		tenant:     tenantID,
	}
}
