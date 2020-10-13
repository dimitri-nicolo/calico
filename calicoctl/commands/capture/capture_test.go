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

type MockQuery struct {
	mock.Mock
}

func (m *MockQuery) Execute(cmdStr string) (string, error) {
	args := m.Called(cmdStr)
	return args.String(0), args.Error(1)
}

const any = "any"

func TestCommands_Copy(t *testing.T) {
	RegisterTestingT(t)

	var tables = []struct {
		cmdInput string
		cmdOutput string
		cmdErr error
	}{
		{fmt.Sprintf(capture.CopyCommand, any, any, any, any, any, any), "", fmt.Errorf(any)},
		{fmt.Sprintf(capture.CopyCommand, any, any, any, any, any, any), "any", nil},
	}

	for _,entry := range tables {
		// setup capture commands
		var mock = MockQuery{}
		var captureCmd = capture.NewCommands(&mock)

		// mock the execute command to return the output specified
		mock.On("Execute", entry.cmdInput).Return(entry.cmdOutput, entry.cmdErr)

		// Call Copy
		var err = captureCmd.Copy([]string{any}, any, any, any, any, any)

		// Assert results
		if entry.cmdErr != nil {
			Expect(err).To(Equal(entry.cmdErr))
		} else {
			Expect(err).To(BeNil())
		}
		mock.AssertNumberOfCalls(t, "Execute", 1)
		mock.AssertExpectations(t)
	}
}

func TestCommands_Clean(t *testing.T) {
	RegisterTestingT(t)

	var tables = []struct {
		cmdInput string
		cmdOutput string
		cmdErr error
	}{
		{fmt.Sprintf(capture.CleanCommand, any, any, any, any, any), "", fmt.Errorf(any)},
		{fmt.Sprintf(capture.CleanCommand, any, any, any, any, any), "any", nil},
	}

	for _,entry := range tables {
		// setup capture commands
		var mock = MockQuery{}
		var captureCmd = capture.NewCommands(&mock)

		// mock the execute command to return the output specified
		mock.On("Execute", entry.cmdInput).Return(entry.cmdOutput, entry.cmdErr)

		// Call Clean
		var err = captureCmd.Clean([]string{any}, any, any, any, any)

		// Assert results
		if entry.cmdErr != nil {
			Expect(err).To(Equal(entry.cmdErr))
		} else {
			Expect(err).To(BeNil())
		}
		mock.AssertNumberOfCalls(t, "Execute", 1)
		mock.AssertExpectations(t)
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
		var mock = MockQuery{}
		var captureCmd = capture.NewCommands(&mock)

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
		var output, ns = captureCmd.ResolveEntryPoints(any, any, any)
		Expect(output).To(Equal(entry.expectedEntryPods))
		Expect(ns).To(Equal("tigera-fluentd"))

		mock.AssertExpectations(t)
	}
}