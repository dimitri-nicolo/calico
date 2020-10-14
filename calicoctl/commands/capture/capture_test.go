// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package capture_test

import (
	"fmt"
	"github.com/projectcalico/calicoctl/calicoctl/commands/capture"
	"github.com/projectcalico/calicoctl/calicoctl/commands/common"
	"github.com/stretchr/testify/mock"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/projectcalico/libcalico-go/lib/logutils"

	log "github.com/sirupsen/logrus"
)

func init() {
	log.AddHook(logutils.ContextHook{})
	log.SetFormatter(&logutils.Formatter{})
}

type MockCmd struct {
	mock.Mock
}

func (m *MockCmd) Execute(cmdStr string) (string, error) {
	args := m.Called(cmdStr)
	return args.String(0), args.Error(1)
}

type MockResolver struct {
	mock.Mock
}

func (m *MockResolver) EntryPoints(captureDir, captureName, captureNs string) []string {
	args := m.Called(captureDir, captureName, captureNs)
	return args.Get(0).([]string)
}

const any = "any"

func TestCommands_Copy(t *testing.T) {
	RegisterTestingT(t)

	var tables = []struct {
		namespaces            []string
		name                  string
		entryPointsPerCapture int
		copyCmdOutput         string
		copyCmdErr            error
		expectedErrors        []error
	}{
		{[]string{"ns1"}, "capture", 1, "", fmt.Errorf(any), []error{fmt.Errorf(any)}},
		{[]string{"ns1"}, "capture", 0, "", fmt.Errorf(any), []error{fmt.Errorf("failed to find capture files for ns1/capture")}},
		{[]string{"ns1"}, "capture", 2, "", fmt.Errorf(any), []error{fmt.Errorf(any), fmt.Errorf(any)}},
		{[]string{"ns1", "ns2"}, "capture", 1, "", fmt.Errorf(any), []error{fmt.Errorf(any), fmt.Errorf(any)}},
		{[]string{"ns1", "ns2"}, "capture", 0, "", fmt.Errorf(any), []error{fmt.Errorf("failed to find capture files for ns1/capture"),fmt.Errorf("failed to find capture files for ns2/capture") }},
		{[]string{"ns1"}, "capture",1,  any, nil, nil},
		{[]string{"ns1"}, "capture",0,  any, nil, []error{fmt.Errorf("failed to find capture files for ns1/capture")}},
		{[]string{"ns1"}, "capture",2,  any, nil, nil},
		{[]string{"ns1", "ns2"}, "capture", 1, any, nil, nil},
		{[]string{"ns1", "ns2"}, "capture", 0, any, nil, []error{fmt.Errorf("failed to find capture files for ns1/capture"),fmt.Errorf("failed to find capture files for ns2/capture") }},
		{[]string{"ns1", "ns2"}, "capture", 2, any, nil, nil},
		{[]string{}, "capture", 0, any, nil, nil},
	}

	for _,entry := range tables {
		// setup capture commands
		var mock = MockCmd{}
		var resolver = MockResolver{}
		var captureCmd = capture.NewCommands(&mock, &resolver)

		for _, ns := range entry.namespaces {
			var fluentDs[]string
			for i := 1; i <= entry.entryPointsPerCapture; i++ {
				var fluentD = fmt.Sprintf("fluentd-%s-%s-%v", ns, entry.name, i)
				// mock the execute command to return the output specified
				mock.On("Execute", fmt.Sprintf(capture.CopyCommand,capture.TigeraFluentDNS, fluentD, any, ns, entry.name, any)).Return(entry.copyCmdOutput, entry.copyCmdErr)
				fluentDs = append(fluentDs, fluentD)
			}
			// mock the fluentD entry points
			resolver.On("EntryPoints", any, entry.name, ns).Return(fluentDs)
		}

		// Call Clean
		var _, err = captureCmd.Copy(entry.namespaces, entry.name, any, any)

		// Assert results
		if entry.expectedErrors != nil {
			Expect(err).To(Equal(entry.expectedErrors))
		} else {
			Expect(err).To(BeNil())
		}

		mock.AssertNumberOfCalls(t, "Execute", len(entry.namespaces) * entry.entryPointsPerCapture)
		mock.AssertExpectations(t)

		resolver.AssertNumberOfCalls(t, "EntryPoints", len(entry.namespaces))
		resolver.AssertExpectations(t)
	}
}

func TestCommands_Clean(t *testing.T) {
	RegisterTestingT(t)

	var tables = []struct {
		namespaces            []string
		name                  string
		entryPointsPerCapture int
		cleanCmdOutput        string
		cleanCmdErr           error
		expectedErrors        []error
	}{
		{[]string{"ns1"}, "capture", 1, "", fmt.Errorf(any), []error{fmt.Errorf(any)}},
		{[]string{"ns1"}, "capture", 0, "", fmt.Errorf(any), []error{fmt.Errorf("failed to find capture files for ns1/capture")}},
		{[]string{"ns1"}, "capture", 2, "", fmt.Errorf(any), []error{fmt.Errorf(any), fmt.Errorf(any)}},
		{[]string{"ns1", "ns2"}, "capture", 1, "", fmt.Errorf(any), []error{fmt.Errorf(any), fmt.Errorf(any)}},
		{[]string{"ns1", "ns2"}, "capture", 0, "", fmt.Errorf(any), []error{fmt.Errorf("failed to find capture files for ns1/capture"),fmt.Errorf("failed to find capture files for ns2/capture") }},
		{[]string{"ns1"}, "capture",1,  any, nil, nil},
		{[]string{"ns1"}, "capture",0,  any, nil, []error{fmt.Errorf("failed to find capture files for ns1/capture")}},
		{[]string{"ns1"}, "capture",2,  any, nil, nil},
		{[]string{"ns1", "ns2"}, "capture", 1, any, nil, nil},
		{[]string{"ns1", "ns2"}, "capture", 0, any, nil, []error{fmt.Errorf("failed to find capture files for ns1/capture"),fmt.Errorf("failed to find capture files for ns2/capture") }},
		{[]string{"ns1", "ns2"}, "capture", 2, any, nil, nil},
		{[]string{}, "capture", 0, any, nil, nil},
	}

	for _,entry := range tables {
		// setup capture commands
		var mock = MockCmd{}
		var resolver = MockResolver{}
		var captureCmd = capture.NewCommands(&mock, &resolver)

		for _, ns := range entry.namespaces {
			var fluentDs[]string
			for i := 1; i <= entry.entryPointsPerCapture; i++ {
				var fluentD = fmt.Sprintf("fluentd-%s-%s-%v", ns, entry.name, i)
				// mock the execute command to return the output specified
				mock.On("Execute", fmt.Sprintf(capture.CleanCommand,capture.TigeraFluentDNS, fluentD, any, ns, entry.name)).Return(entry.cleanCmdOutput, entry.cleanCmdErr)
				fluentDs = append(fluentDs, fluentD)
			}
			// mock the fluentD entry points
			resolver.On("EntryPoints", any, entry.name, ns).Return(fluentDs)
		}

		// Call Clean
		var _, err = captureCmd.Clean(entry.namespaces, entry.name, any)

		// Assert results
		if entry.expectedErrors != nil {
			Expect(err).To(Equal(entry.expectedErrors))
		} else {
			Expect(err).To(BeNil())
		}

		mock.AssertNumberOfCalls(t, "Execute", len(entry.namespaces) * entry.entryPointsPerCapture)
		mock.AssertExpectations(t)

		resolver.AssertNumberOfCalls(t, "EntryPoints", len(entry.namespaces))
		resolver.AssertExpectations(t)
	}
}

func TestCommands_ResolveEntryPoints(t *testing.T) {
	RegisterTestingT(t)

	const multipleNodes =
`calico-node1   node1
calico-node2   node2`
	const oneNode =
`calico-node1   node1
`
	const zeroNodes = ``

	var tables = []struct {
		getCalicoNodesOutput      string
		calicoNodesWithCapture    []string
		calicoNodesWithoutCapture []string
		nodesWithFluentD []string
		nodesWithoutFluentD []string
		expectedEntryPods []string
	}{
		// two calico nodes with capture files and matching fluentD pods
		{multipleNodes, []string{"calico-node1", "calico-node2"}, []string{}, []string{"node1", "node2"}, []string{}, []string{"pod-node1", "pod-node2"}},
		// two calico nodes with capture files and fluentD pods on node1
		{multipleNodes, []string{"calico-node1", "calico-node2"}, []string{}, []string{"node1"}, []string{"node2"}, []string{"pod-node1"}},
		// two calico nodes with capture files on node1 and fluentD pods on node1
		{multipleNodes, []string{"calico-node1"}, []string{"calico-node2"}, []string{"node1"}, []string{}, []string{"pod-node1"}},
		// two calico nodes with capture files on node1 and no fluentD pods on node1
		{multipleNodes, []string{"calico-node1"}, []string{"calico-node2"}, []string{}, []string{"node1"}, nil},
		// two calico nodes with no capture files
		{multipleNodes, []string{}, []string{"calico-node1", "calico-node2"}, []string{}, []string{}, nil},
		// one calico node with capture files and fluentD pod on node1
		{oneNode, []string{"calico-node1"}, []string{}, []string{"node1"}, []string{}, []string{"pod-node1"}},
		// one calico node with capture files and no fluentD pod on node1
		{oneNode, []string{"calico-node1"}, []string{}, []string{}, []string{"node1"}, nil},
		// one calico node with no capture files
		{oneNode, []string{}, []string{"calico-node1"}, []string{}, []string{}, nil},
		// no nodes returned
		{zeroNodes, []string{}, []string{}, []string{}, []string{}, nil},
	}

	for _,entry := range tables {
		// setup capture commands
		var mock = MockCmd{}
		var fluentDResolver = capture.NewFluentDResolver(&mock)

		// mock the execute command to return the output specified
		mock.On("Execute", capture.GetCalicoNodesCommand).Return(entry.getCalicoNodesOutput, nil)
		// mock stat command so that it returns no error for nodes marked to have a capture
		for _, node := range entry.calicoNodesWithCapture {
			mock.On("Execute", fmt.Sprintf(capture.FindCaptureFileCommand, common.CalicoNamespace, node, any, any, any)).Return(any, nil)
		}
		// mock stat command so that it returns an error for nodes marked not to have a capture
		for _, node := range entry.calicoNodesWithoutCapture {
			mock.On("Execute", fmt.Sprintf(capture.FindCaptureFileCommand, common.CalicoNamespace, node, any, any, any)).Return("", fmt.Errorf(any))
		}
		// mock get fluentD pods so that is returns pod-node{index} for any fluentD pod that matches the nodes
		for _, node := range entry.nodesWithFluentD {
			mock.On("Execute", fmt.Sprintf(capture.GetPodByNodeName, "tigera-fluentd", node)).Return(fmt.Sprintf("pod-%s", node), nil)
		}
		// mock get fluentD pods so that is returns an error for any fluentD pod that does matches the nodes
		for _, node := range entry.nodesWithoutFluentD {
			mock.On("Execute", fmt.Sprintf(capture.GetPodByNodeName, "tigera-fluentd", node)).Return("", fmt.Errorf(any))
		}

		// Call ResolveEntryPoints
		var output = fluentDResolver.EntryPoints(any, any, any)
		Expect(output).To(Equal(entry.expectedEntryPods))

		mock.AssertExpectations(t)
	}
}