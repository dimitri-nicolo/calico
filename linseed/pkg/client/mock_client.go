// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package client

import "github.com/projectcalico/calico/linseed/pkg/client/rest"

type MockClient interface {
	Client
	SetResults(results ...rest.MockResult)
}

type mockClient struct {
	restClient rest.RESTClient
	tenant     string
}

func (c *mockClient) RESTClient() rest.RESTClient {
	return c.restClient
}

// L3Flows returns an interface for managing v1.L3Flow resources.
func (c *mockClient) L3Flows(cluster string) L3FlowsInterface {
	return newL3Flows(c, cluster)
}

// L7Flows returns an interface for managing v1.L7Flow resources.
func (c *mockClient) L7Flows(cluster string) L7FlowsInterface {
	return newL7Flows(c, cluster)
}

// DNSFlows returns an interface for managing v1.DNSFlow resources.
func (c *mockClient) DNSFlows(cluster string) DNSFlowsInterface {
	return newDNSFlows(c, cluster)
}

// Events returns an interface for managing v1.Events resources.
func (c *mockClient) Events(cluster string) EventsInterface {
	return newEvents(c, cluster)
}

// FlowLogs returns an interface for managing v1.FlowLog resources.
func (c *mockClient) FlowLogs(cluster string) FlowLogsInterface {
	return newFlowLogs(c, cluster)
}

// DNSLogs returns an interface for managing v1.DNSLog resources.
func (c *mockClient) DNSLogs(cluster string) DNSLogsInterface {
	return newDNSLogs(c, cluster)
}

// L7Logs returns an interface for managing v1.L7Log resources.
func (c *mockClient) L7Logs(cluster string) L7LogsInterface {
	return newL7Logs(c, cluster)
}

// AuditLogs returns an interface for managing v1.AuditLog resources.
func (c *mockClient) AuditLogs(cluster string) AuditLogsInterface {
	return newAuditLogs(c, cluster)
}

func NewMockClient(tenantID string, results ...rest.MockResult) MockClient {
	return &mockClient{
		restClient: rest.NewMockClient(results...),
		tenant:     tenantID,
	}
}

func (m *mockClient) SetResults(results ...rest.MockResult) {
	m.restClient = rest.NewMockClient(results...)
}
