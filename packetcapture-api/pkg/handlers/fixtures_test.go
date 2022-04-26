// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package handlers_test

import (
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/packetcapture-api/pkg/capture"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var noFiles = []string{}
var files = []string{"a", "b"}
var otherFiles = []string{"c", "d"}
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
				FileNames: files,
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
				Node:      "nodeOne",
				Directory: "dir",
				FileNames: files,
			},
			{
				Node:      "nodeTwo",
				Directory: "dir",
				FileNames: otherFiles,
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
				FileNames: files,
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
				Node:      "nodeOne",
				Directory: "dir",
				FileNames: files,
				State:     (*v3.PacketCaptureState)(&finished),
			},
			{
				Node:      "nodeTwo",
				Directory: "dir",
				FileNames: otherFiles,
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
				Node:      "nodeOne",
				Directory: "dir",
				FileNames: files,
				State:     &capturing,
			},
			{
				Node:      "nodeTwo",
				Directory: "dir",
				FileNames: otherFiles,
				State:     (*v3.PacketCaptureState)(&scheduled),
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
				Node:      "nodeOne",
				Directory: "dir",
				FileNames: files,
				State:     &capturing,
			},
			{
				Node:      "nodeTwo",
				Directory: "dir",
				FileNames: otherFiles,
				State:     (*v3.PacketCaptureState)(&finished),
			},
		},
	},
}

var point = capture.EntryPoint{PodName: "entryPod", PodNamespace: "entryNs", CaptureDirectory: "dir",
	CaptureNamespace: "ns", CaptureName: "name"}
var pointNodeOne = capture.EntryPoint{PodName: "entryPodOne", PodNamespace: "entryNs", CaptureDirectory: "dir",
	CaptureNamespace: "ns", CaptureName: "name"}
var pointNodeTwo = capture.EntryPoint{PodName: "entryPodTwo", PodNamespace: "entryNs", CaptureDirectory: "dir",
	CaptureNamespace: "ns", CaptureName: "name"}
