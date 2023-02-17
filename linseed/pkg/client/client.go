// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package client

import (
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
)

type Client interface {
	RESTClient() *rest.RESTClient
	L3Flows(string) L3FlowsInterface
	FlowLogs(string) FlowLogsInterface
	L7Flows(string) L7FlowsInterface
	L7Logs(string) L7LogsInterface
	DNSFlows(string) DNSFlowsInterface
	DNSLogs(string) DNSLogsInterface
	Events(string) EventsInterface
	AuditLogs(string) AuditLogsInterface
}

type client struct {
	restClient *rest.RESTClient
}

// L3Flows returns an interface for managing v1.L3Flow resources.
func (c *client) RESTClient() *rest.RESTClient {
	return c.restClient
}

// L3Flows returns an interface for managing v1.L3Flow resources.
func (c *client) L3Flows(cluster string) L3FlowsInterface {
	return newL3Flows(c, cluster)
}

// L7Flows returns an interface for managing v1.L7Flow resources.
func (c *client) L7Flows(cluster string) L7FlowsInterface {
	return newL7Flows(c, cluster)
}

// DNSFlows returns an interface for managing v1.DNSFlow resources.
func (c *client) DNSFlows(cluster string) DNSFlowsInterface {
	return newDNSFlows(c, cluster)
}

// Events returns an interface for managing v1.Events resources.
func (c *client) Events(cluster string) EventsInterface {
	return newEvents(c, cluster)
}

// FlowLogs returns an interface for managing v1.FlowLog resources.
func (c *client) FlowLogs(cluster string) FlowLogsInterface {
	return newFlowLogs(c, cluster)
}

// DNSLogs returns an interface for managing v1.DNSLog resources.
func (c *client) DNSLogs(cluster string) DNSLogsInterface {
	return newDNSLogs(c, cluster)
}

// L7Logs returns an interface for managing v1.L7Log resources.
func (c *client) L7Logs(cluster string) L7LogsInterface {
	return newL7Logs(c, cluster)
}

// AuditLogs returns an interface for managing v1.AuditLog resources.
func (c *client) AuditLogs(cluster string) AuditLogsInterface {
	return newAuditLogs(c, cluster)
}

func NewClient(tenantID string, cfg rest.Config) (Client, error) {
	rc, err := rest.NewClient(tenantID, cfg)
	if err != nil {
		return nil, err
	}
	return &client{
		restClient: rc,
	}, nil
}
