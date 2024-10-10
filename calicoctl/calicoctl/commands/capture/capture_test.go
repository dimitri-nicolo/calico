// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package capture_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/projectcalico/calico/calicoctl/calicoctl/commands/capture"

	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
)

func init() {
	logutils.ConfigureFormatter("test")
}

type MockCmd struct {
	mock.Mock
}

func (m *MockCmd) Execute(cmdStr string) (string, error) {
	args := m.Called(cmdStr)
	return args.String(0), args.Error(1)
}

const any = "any"
const multipleNodes = `fluentD-node1
fluentD-node2`
const oneNode = `fluentD-node1
`
const zeroNodes = ``

const ns1 = "ns1"
const ns2 = "ns2"
const defaultNs = "default"
const pod1 = "fluentD-node1"
const pod2 = "fluentD-node2"

func TestCommands_Copy(t *testing.T) {
	RegisterTestingT(t)

	var tables = []struct {
		locations      []capture.Location
		copyCmdErr     map[string]error
		expectedErrors []error
	}{
		// Given a single location, copy fails for pod 1
		{
			[]capture.Location{
				{Name: any, CaptureNamespace: ns1, Pod: pod1, Dir: any, Namespace: any, Container: any},
			}, map[string]error{pod1: errors.New(any)}, []error{errors.New(any)},
		},
		// Given a single location, copy does not fail
		{
			[]capture.Location{
				{Name: any, CaptureNamespace: ns1, Pod: pod1, Dir: any, Namespace: any, Container: any},
			}, map[string]error{}, nil,
		},
		// Given multiple locations, copy does not fail
		{
			[]capture.Location{
				{Name: any, CaptureNamespace: ns1, Pod: pod1, Dir: any, Namespace: any, Container: any},
				{Name: any, CaptureNamespace: ns2, Pod: pod1, Dir: any, Namespace: any, Container: any},
				{Name: any, CaptureNamespace: ns1, Pod: pod2, Dir: any, Namespace: any, Container: any},
				{Name: any, CaptureNamespace: ns2, Pod: pod2, Dir: any, Namespace: any, Container: any},
			}, map[string]error{}, nil,
		},
		// Given multiple location, copy fails for pod1
		{
			[]capture.Location{
				{Name: any, CaptureNamespace: ns1, Pod: pod1, Dir: any, Namespace: any, Container: any},
				{Name: any, CaptureNamespace: ns2, Pod: pod1, Dir: any, Namespace: any, Container: any},
				{Name: any, CaptureNamespace: ns1, Pod: pod2, Dir: any, Namespace: any, Container: any},
				{Name: any, CaptureNamespace: ns2, Pod: pod2, Dir: any, Namespace: any, Container: any},
			}, map[string]error{pod1: errors.New(any)}, []error{errors.New(any), errors.New(any)},
		},
		// Given multiple location, copy fails for pod1 and pod2
		{
			[]capture.Location{
				{Name: any, CaptureNamespace: ns1, Pod: pod1, Dir: any, Namespace: any, Container: any},
				{Name: any, CaptureNamespace: ns2, Pod: pod1, Dir: any, Namespace: any, Container: any},
				{Name: any, CaptureNamespace: ns1, Pod: pod2, Dir: any, Namespace: any, Container: any},
				{Name: any, CaptureNamespace: ns2, Pod: pod2, Dir: any, Namespace: any, Container: any},
			}, map[string]error{pod1: errors.New(any), pod2: errors.New(any)}, []error{errors.New(any), errors.New(any), errors.New(any), errors.New(any)},
		},
		{
			// No locations given
			[]capture.Location{}, map[string]error{}, nil,
		},
	}

	for _, entry := range tables {
		// setup capture commands
		var mock = MockCmd{}
		var captureCmd = capture.NewCommands(&mock)

		for _, loc := range entry.locations {
			// mock the execute command to return the output specified
			mock.On("Execute", fmt.Sprintf(capture.CopyCommand, any, loc.Pod, any, loc.CaptureNamespace, loc.Name, any, any)).Return(any, entry.copyCmdErr[loc.Pod])
		}

		// Call Copy
		var _, err = captureCmd.Copy(entry.locations, any)

		// Assert results
		if entry.expectedErrors != nil {
			Expect(err).To(Equal(entry.expectedErrors))
		} else {
			Expect(err).To(BeNil())
		}

		mock.AssertNumberOfCalls(t, "Execute", len(entry.locations))
		mock.AssertExpectations(t)
	}
}

func TestCommands_Clean(t *testing.T) {
	RegisterTestingT(t)

	var tables = []struct {
		locations      []capture.Location
		cleanCmdErr    map[string]error
		expectedErrors []error
	}{
		// Given a single location, clean fails for pod 1
		{
			[]capture.Location{
				{Name: any, CaptureNamespace: ns1, Pod: pod1, Dir: any, Namespace: any, Container: any},
			}, map[string]error{pod1: errors.New(any)}, []error{errors.New(any)},
		},
		// Given a single location, clean does not fail
		{
			[]capture.Location{
				{Name: any, CaptureNamespace: ns1, Pod: pod1, Dir: any, Namespace: any, Container: any},
			}, map[string]error{}, nil,
		},
		// Given multiple locations, clean does not fail
		{
			[]capture.Location{
				{Name: any, CaptureNamespace: ns1, Pod: pod1, Dir: any, Namespace: any, Container: any},
				{Name: any, CaptureNamespace: ns2, Pod: pod1, Dir: any, Namespace: any, Container: any},
				{Name: any, CaptureNamespace: ns1, Pod: pod2, Dir: any, Namespace: any, Container: any},
				{Name: any, CaptureNamespace: ns2, Pod: pod2, Dir: any, Namespace: any, Container: any},
			}, map[string]error{}, nil,
		},
		// Given multiple location, clean fails for pod1
		{
			[]capture.Location{
				{Name: any, CaptureNamespace: ns1, Pod: pod1, Dir: any, Namespace: any, Container: any},
				{Name: any, CaptureNamespace: ns2, Pod: pod1, Dir: any, Namespace: any, Container: any},
				{Name: any, CaptureNamespace: ns1, Pod: pod2, Dir: any, Namespace: any, Container: any},
				{Name: any, CaptureNamespace: ns2, Pod: pod2, Dir: any, Namespace: any, Container: any},
			}, map[string]error{pod1: errors.New(any)}, []error{errors.New(any), errors.New(any)},
		},
		// Given multiple location, clean fails for pod1 and pod2
		{
			[]capture.Location{
				{Name: any, CaptureNamespace: ns1, Pod: pod1, Dir: any, Namespace: any, Container: any},
				{Name: any, CaptureNamespace: ns2, Pod: pod1, Dir: any, Namespace: any, Container: any},
				{Name: any, CaptureNamespace: ns1, Pod: pod2, Dir: any, Namespace: any, Container: any},
				{Name: any, CaptureNamespace: ns2, Pod: pod2, Dir: any, Namespace: any, Container: any},
			}, map[string]error{pod1: errors.New(any), pod2: errors.New(any)}, []error{errors.New(any), errors.New(any), errors.New(any), errors.New(any)},
		},
		{
			// No locations given
			[]capture.Location{}, map[string]error{}, nil,
		},
	}

	for _, entry := range tables {
		// setup capture commands
		var mock = MockCmd{}
		var captureCmd = capture.NewCommands(&mock)

		for _, loc := range entry.locations {
			// mock the execute command to return the output specified
			mock.On("Execute", fmt.Sprintf(capture.CleanCommand, any, loc.Pod, any, any, loc.CaptureNamespace, loc.Name)).Return(any, entry.cleanCmdErr[loc.Pod])
		}

		// Call Clean
		var _, err = captureCmd.Clean(entry.locations)

		// Assert results
		if entry.expectedErrors != nil {
			Expect(err).To(Equal(entry.expectedErrors))
		} else {
			Expect(err).To(BeNil())
		}

		mock.AssertNumberOfCalls(t, "Execute", len(entry.locations))
		mock.AssertExpectations(t)
	}
}

func TestCommands_List(t *testing.T) {
	RegisterTestingT(t)

	var captureInDefaultNs = fmt.Sprintf("%s/%s/%s", any, defaultNs, any)
	var multipleCaptures = fmt.Sprintf("%s/%s/%s\n%s/%s/%s", any, defaultNs, "other", any, defaultNs, any)
	var otherCapture = fmt.Sprintf("%s/%s/%s", any, defaultNs, "other")
	var captureWithMultipleNs = fmt.Sprintf("%s/%s/%s\n%s/%s/%s", any, ns1, any, any, ns2, any)

	var tables = []struct {
		namespace             string
		getFluentDNodesOutput string
		findOutputPerPods     map[string]string
		errOutputForPods      []string
		expected              []capture.Location
	}{
		// two fluentD nodes with capture files with a single capture on each in namespace default
		{
			"",
			multipleNodes, map[string]string{pod1: captureInDefaultNs, pod2: captureInDefaultNs}, []string{},
			[]capture.Location{
				{Name: any, CaptureNamespace: defaultNs, Pod: pod1, Dir: any, Namespace: capture.TigeraFluentDNS, Container: capture.TigeraFluentD},
				{Name: any, CaptureNamespace: defaultNs, Pod: pod2, Dir: any, Namespace: capture.TigeraFluentDNS, Container: capture.TigeraFluentD},
			},
		},
		// two fluentD nodes with capture files with capture files in multiple namespaces
		{
			"",
			multipleNodes, map[string]string{pod1: captureWithMultipleNs, pod2: captureWithMultipleNs}, []string{},
			[]capture.Location{
				{Name: any, CaptureNamespace: ns1, Pod: pod1, Dir: any, Namespace: capture.TigeraFluentDNS, Container: capture.TigeraFluentD},
				{Name: any, CaptureNamespace: ns2, Pod: pod1, Dir: any, Namespace: capture.TigeraFluentDNS, Container: capture.TigeraFluentD},
				{Name: any, CaptureNamespace: ns1, Pod: pod2, Dir: any, Namespace: capture.TigeraFluentDNS, Container: capture.TigeraFluentD},
				{Name: any, CaptureNamespace: ns2, Pod: pod2, Dir: any, Namespace: capture.TigeraFluentDNS, Container: capture.TigeraFluentD},
			},
		},
		// two fluentD nodes with capture files with multiple capture files
		{
			"",
			multipleNodes, map[string]string{pod1: multipleCaptures, pod2: multipleCaptures}, []string{},
			[]capture.Location{
				{Name: any, CaptureNamespace: defaultNs, Pod: pod1, Dir: any, Namespace: capture.TigeraFluentDNS, Container: capture.TigeraFluentD},
				{Name: any, CaptureNamespace: defaultNs, Pod: pod2, Dir: any, Namespace: capture.TigeraFluentDNS, Container: capture.TigeraFluentD},
			},
		},
		// two fluentD nodes with capture files with another captures on each in namespace default
		{
			"",
			multipleNodes, map[string]string{pod1: otherCapture, pod2: otherCapture}, []string{},
			nil,
		},
		// two fluentD nodes with capture files with no captures
		{
			"",
			multipleNodes, map[string]string{pod1: any, pod2: any}, []string{},
			nil,
		},
		// two fluentD nodes that will error out
		{
			"",
			multipleNodes, map[string]string{}, []string{pod1, pod2},
			nil,
		},
		// two fluentD nodes with capture files with a single capture on node1 in namespace default
		{
			"",
			multipleNodes, map[string]string{pod1: captureInDefaultNs, pod2: any}, []string{},
			[]capture.Location{
				{Name: any, CaptureNamespace: defaultNs, Pod: pod1, Dir: any, Namespace: capture.TigeraFluentDNS, Container: capture.TigeraFluentD},
			},
		},
		// one fluentD nodes with capture files with a single capture on node1 in namespace default
		{
			"",
			oneNode, map[string]string{pod1: captureInDefaultNs}, []string{},
			[]capture.Location{
				{Name: any, CaptureNamespace: defaultNs, Pod: pod1, Dir: any, Namespace: capture.TigeraFluentDNS, Container: capture.TigeraFluentD},
			},
		},
		// no fluentD nodes
		{
			"",
			zeroNodes, map[string]string{}, []string{},
			nil,
		},
		// two fluentD nodes with capture files with capture files in multiple namespaces and namespace ns1
		{
			ns1,
			multipleNodes, map[string]string{pod1: captureWithMultipleNs, pod2: captureWithMultipleNs}, []string{},
			[]capture.Location{
				{Name: any, CaptureNamespace: ns1, Pod: pod1, Dir: any, Namespace: capture.TigeraFluentDNS, Container: capture.TigeraFluentD},
				{Name: any, CaptureNamespace: ns1, Pod: pod2, Dir: any, Namespace: capture.TigeraFluentDNS, Container: capture.TigeraFluentD},
			},
		},
		// two fluentD nodes with capture files with capture files in multiple namespaces and namespace default
		{
			defaultNs,
			multipleNodes, map[string]string{pod1: captureWithMultipleNs, pod2: captureWithMultipleNs}, []string{},
			nil,
		},
		// two fluentD nodes with capture files with capture files in multiple namespaces and select all namespaces
		{
			"",
			multipleNodes, map[string]string{pod1: captureWithMultipleNs, pod2: captureWithMultipleNs}, []string{},
			[]capture.Location{
				{Name: any, CaptureNamespace: ns1, Pod: pod1, Dir: any, Namespace: capture.TigeraFluentDNS, Container: capture.TigeraFluentD},
				{Name: any, CaptureNamespace: ns2, Pod: pod1, Dir: any, Namespace: capture.TigeraFluentDNS, Container: capture.TigeraFluentD},
				{Name: any, CaptureNamespace: ns1, Pod: pod2, Dir: any, Namespace: capture.TigeraFluentDNS, Container: capture.TigeraFluentD},
				{Name: any, CaptureNamespace: ns2, Pod: pod2, Dir: any, Namespace: capture.TigeraFluentDNS, Container: capture.TigeraFluentD},
			},
		},
	}

	for _, entry := range tables {
		// setup capture commands
		var mock = MockCmd{}
		var captureCmd = capture.NewCommands(&mock)

		// mock the execute command to return the output specified
		mock.On("Execute", capture.GetFluentDNodesCommand).Return(entry.getFluentDNodesOutput, nil)
		// mock stat command so that it returns no error for nodes marked to have a capture
		for node, output := range entry.findOutputPerPods {
			mock.On("Execute", fmt.Sprintf(capture.FindCaptureFileCommand, capture.TigeraFluentDNS, node, capture.TigeraFluentD, any)).Return(output, nil)
		}
		// mock stat command so that it returns a error for nodes marked to fail resolving
		for _, node := range entry.errOutputForPods {
			mock.On("Execute", fmt.Sprintf(capture.FindCaptureFileCommand, capture.TigeraFluentDNS, node, capture.TigeraFluentD, any)).Return("", errors.New(any))
		}

		// Call List
		var locations, err = captureCmd.List(any, any, entry.namespace)
		Expect(err).NotTo(HaveOccurred())
		Expect(locations).To(Equal(entry.expected))

		mock.AssertExpectations(t)
	}
}
