// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package policystore

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/projectcalico/calico/app-policy/proto"

	"github.com/projectcalico/calico/felix/ip"
)

type mockWorkloadCallbacks struct {
	callStack []string
}

func (m *mockWorkloadCallbacks) Update(cidr ip.CIDR, v *proto.WorkloadEndpointUpdate) {
	m.callStack = append(m.callStack, fmt.Sprint("Update ", cidr, v))
}

func (m *mockWorkloadCallbacks) Delete(cidr ip.CIDR, v *proto.WorkloadEndpointRemove) {
	m.callStack = append(m.callStack, fmt.Sprint("Delete ", cidr))
}

func (m *mockWorkloadCallbacks) Get(k ip.CIDR) []*proto.WorkloadEndpoint {
	return nil
}

type workloadsTestCase struct {
	comment  string
	updates  []interface{}
	mock     *mockWorkloadCallbacks
	expected []string
}

func runTestCase(t *testing.T, comment string, updates []interface{}, expected []string) {
	tc := &workloadsTestCase{comment, updates, &mockWorkloadCallbacks{}, expected}
	tc.runAssertions(t)
}

func (tc *workloadsTestCase) runAssertions(t *testing.T) {
	handler := newWorkloadEndpointUpdateHandler()
	for _, upd := range tc.updates {
		handler.onResourceUpdate(upd, tc.mock)
	}
	assert.ElementsMatch(t, tc.expected, tc.mock.callStack, fmt.Sprintf("test case failed: %v", tc.comment))
}

func wepUpdate(name string, ip4s ...string) *proto.WorkloadEndpointUpdate {
	return &proto.WorkloadEndpointUpdate{
		Id: &proto.WorkloadEndpointID{
			OrchestratorId: "kubernetes",
			EndpointId:     "eth0",
			WorkloadId:     name,
		},
		Endpoint: &proto.WorkloadEndpoint{
			Name:     name,
			Ipv4Nets: ip4s,
		},
	}
}

func wepRemove(name string) *proto.WorkloadEndpointRemove {
	return &proto.WorkloadEndpointRemove{
		Id: &proto.WorkloadEndpointID{
			OrchestratorId: "kubernetes",
			EndpointId:     "eth0",
			WorkloadId:     name,
		},
	}
}

func TestWorkloads(t *testing.T) {
	runTestCase(t,
		"single workload endpoint update and remove",
		[]interface{}{
			wepUpdate("some-pod-1", "10.0.0.1/32"),
			wepRemove("some-pod-1"),
		},
		[]string{
			fmt.Sprintf("Update 10.0.0.1/32 %v", wepUpdate("some-pod-1", "10.0.0.1/32")),
			"Delete 10.0.0.1/32",
		},
	)

	runTestCase(t,
		"single workload endpoint multi-ips update and remove",
		[]interface{}{
			wepUpdate("some-pod-1", "10.0.0.1/32", "10.0.0.2/32"),
			wepRemove("some-pod-1"),
		},
		[]string{
			fmt.Sprintf("Update 10.0.0.1/32 %v", wepUpdate("some-pod-1", "10.0.0.1/32", "10.0.0.2/32")),
			fmt.Sprintf("Update 10.0.0.2/32 %v", wepUpdate("some-pod-1", "10.0.0.1/32", "10.0.0.2/32")),
			"Delete 10.0.0.1/32",
			"Delete 10.0.0.2/32",
		},
	)

	runTestCase(t,
		"multi workload endpoint update and remove",
		[]interface{}{
			wepUpdate("some-pod-1", "10.0.0.1/32"),
			wepUpdate("some-pod-2", "10.0.0.2/32"),
			wepUpdate("some-pod-2", "10.0.2.1/32"),
			wepRemove("some-pod-1"),
			wepRemove("some-pod-2"),
		},
		[]string{
			fmt.Sprintf("Update 10.0.0.1/32 %v", wepUpdate("some-pod-1", "10.0.0.1/32")),
			fmt.Sprintf("Update 10.0.0.2/32 %v", wepUpdate("some-pod-2", "10.0.0.2/32")),
			"Delete 10.0.0.2/32",
			fmt.Sprintf("Update 10.0.2.1/32 %v", wepUpdate("some-pod-2", "10.0.2.1/32")),
			"Delete 10.0.0.1/32",
			"Delete 10.0.2.1/32",
		},
	)

	runTestCase(t,
		"single workload endpoint changing ips",
		[]interface{}{
			wepUpdate("some-pod-1", "10.0.0.1/32", "10.0.0.2/32"),
			wepUpdate("some-pod-1", "10.0.0.1/32"),
			wepRemove("some-pod-1"),
		},
		[]string{
			fmt.Sprintf("Update 10.0.0.1/32 %s", wepUpdate("some-pod-1", "10.0.0.1/32", "10.0.0.2/32")),
			fmt.Sprintf("Update 10.0.0.2/32 %v", wepUpdate("some-pod-1", "10.0.0.1/32", "10.0.0.2/32")),
			"Delete 10.0.0.2/32",
			fmt.Sprintf("Update 10.0.0.1/32 %s", wepUpdate("some-pod-1", "10.0.0.1/32")),
			"Delete 10.0.0.1/32",
		},
	)
}
