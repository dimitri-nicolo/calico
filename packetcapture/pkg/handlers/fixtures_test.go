// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package handlers_test

import (
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/calico/packetcapture/pkg/capture"
)

var noFiles = []string{}
var filesOnNode1 = []string{"a"}
var filesOnNode2 = []string{"b", "c"}
var filesOnNode3 = []string{"d", "e", "f"}
var packetCaptureOneNode = &v3.PacketCapture{
	TypeMeta: metav1.TypeMeta{
		Kind:       "",
		APIVersion: "",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "name",
		Namespace: "ns",
	},
	Status: v3.PacketCaptureStatus{
		Files: []v3.PacketCaptureFile{
			{
				Node:      "node",
				Directory: "dir",
				FileNames: filesOnNode1,
			},
		},
	},
}
var packetCaptureMultipleNodes = &v3.PacketCapture{
	TypeMeta: metav1.TypeMeta{
		Kind:       "",
		APIVersion: "",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "name",
		Namespace: "ns",
	},
	Status: v3.PacketCaptureStatus{
		Files: []v3.PacketCaptureFile{
			{
				Node:      "node1",
				Directory: "dir",
				FileNames: filesOnNode1,
			},
			{
				Node:      "node2",
				Directory: "dir",
				FileNames: filesOnNode2,
			},
			{
				Node:      "node3",
				Directory: "dir",
				FileNames: filesOnNode3,
			},
		},
	},
}

var packetCaptureNoFiles = &v3.PacketCapture{
	TypeMeta: metav1.TypeMeta{
		Kind:       "",
		APIVersion: "",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "name",
		Namespace: "ns",
	},
	Status: v3.PacketCaptureStatus{
		Files: []v3.PacketCaptureFile{
			{
				Node:      "node",
				Directory: "dir",
			},
		},
	},
}
var packetCaptureEmptyStatus = &v3.PacketCapture{
	TypeMeta: metav1.TypeMeta{
		Kind:       "",
		APIVersion: "",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "name",
		Namespace: "ns",
	},
	Status: v3.PacketCaptureStatus{},
}
var packetCaptureNoStatus = &v3.PacketCapture{
	TypeMeta: metav1.TypeMeta{
		Kind:       "",
		APIVersion: "",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "name",
		Namespace: "ns",
	},
}

var finished = v3.PacketCaptureStateFinished
var capturing = v3.PacketCaptureStateCapturing
var scheduled = v3.PacketCaptureStateScheduled

var finishedPacketCaptureOneNode = &v3.PacketCapture{
	TypeMeta: metav1.TypeMeta{
		Kind:       "",
		APIVersion: "",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "name",
		Namespace: "ns",
	},
	Status: v3.PacketCaptureStatus{
		Files: []v3.PacketCaptureFile{
			{
				Node:      "node",
				Directory: "dir",
				FileNames: filesOnNode1,
				State:     (*v3.PacketCaptureState)(&finished),
			},
		},
	},
}
var finishedPacketCaptureMultipleNodes = &v3.PacketCapture{
	TypeMeta: metav1.TypeMeta{
		Kind:       "",
		APIVersion: "",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "name",
		Namespace: "ns",
	},
	Status: v3.PacketCaptureStatus{
		Files: []v3.PacketCaptureFile{
			{
				Node:      "node1",
				Directory: "dir",
				FileNames: filesOnNode1,
				State:     (*v3.PacketCaptureState)(&finished),
			},
			{
				Node:      "node2",
				Directory: "dir",
				FileNames: filesOnNode2,
				State:     (*v3.PacketCaptureState)(&finished),
			},
			{
				Node:      "node3",
				Directory: "dir",
				FileNames: filesOnNode3,
				State:     (*v3.PacketCaptureState)(&finished),
			},
		},
	},
}

var differentStatesPacketCaptureMultipleNodes = &v3.PacketCapture{
	TypeMeta: metav1.TypeMeta{
		Kind:       "",
		APIVersion: "",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "name",
		Namespace: "ns",
	},
	Status: v3.PacketCaptureStatus{
		Files: []v3.PacketCaptureFile{
			{
				Node:      "node1",
				Directory: "dir",
				FileNames: filesOnNode1,
				State:     &capturing,
			},
			{
				Node:      "node2",
				Directory: "dir",
				FileNames: filesOnNode2,
				State:     (*v3.PacketCaptureState)(&scheduled),
			},
			{
				Node:      "node3",
				Directory: "dir",
				FileNames: filesOnNode3,
				State:     (*v3.PacketCaptureState)(&finished),
			},
		},
	},
}

var oneFinishedPacketCaptureMultipleNodes = &v3.PacketCapture{
	TypeMeta: metav1.TypeMeta{
		Kind:       "",
		APIVersion: "",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "name",
		Namespace: "ns",
	},
	Status: v3.PacketCaptureStatus{
		Files: []v3.PacketCaptureFile{
			{
				Node:      "node1",
				Directory: "dir",
				FileNames: filesOnNode1,
				State:     &capturing,
			},
			{
				Node:      "node2",
				Directory: "dir",
				FileNames: filesOnNode2,
				State:     (*v3.PacketCaptureState)(&finished),
			},
			{
				Node:      "node3",
				Directory: "dir",
				FileNames: filesOnNode3,
				State:     &capturing,
			},
		},
	},
}

var point = capture.EntryPoint{
	EntryPod: capture.EntryPod{
		ContainerName: "fluentd",
		PodName:       "entryPod",
		PodNamespace:  "entryNs",
	},
	CaptureDirectory: "dir",
	CaptureNamespace: "ns",
	CaptureName:      "name",
}
var pointNode1 = capture.EntryPoint{
	EntryPod: capture.EntryPod{
		ContainerName: "fluentd",
		PodName:       "entryPod1",
		PodNamespace:  "entryNs",
	},
	CaptureDirectory: "dir",
	CaptureNamespace: "ns",
	CaptureName:      "name",
}
var pointNode2 = capture.EntryPoint{
	EntryPod: capture.EntryPod{
		ContainerName: "fluentd",
		PodName:       "entryPod2",
		PodNamespace:  "entryNs",
	},
	CaptureDirectory: "dir",
	CaptureNamespace: "ns",
	CaptureName:      "name",
}
var pointNode3 = capture.EntryPoint{
	EntryPod: capture.EntryPod{
		ContainerName: "fluentd",
		PodName:       "entryPod3",
		PodNamespace:  "entryNs",
	},
	CaptureDirectory: "dir",
	CaptureNamespace: "ns",
	CaptureName:      "name",
}
