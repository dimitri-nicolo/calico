// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package calc_test

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

var Wep1Key = model.WorkloadEndpointKey{WorkloadID: "wep1"}
var Wep2Key = model.WorkloadEndpointKey{WorkloadID: "wep2"}
var WepWithProfileKey = model.WorkloadEndpointKey{WorkloadID: "wep-profile"}
var Wep1Value = &model.WorkloadEndpoint{
	Name: "wep1",
	Labels: map[string]string{
		"label":                       "a",
		"projectcalico.org/namespace": "default",
	},
}
var Wep1UpdatedValue = &model.WorkloadEndpoint{
	Name: "wep1",
	Labels: map[string]string{
		"label":                       "c",
		"projectcalico.org/namespace": "default",
	},
}
var Wep2Value = &model.WorkloadEndpoint{
	Name: "wep2",
	Labels: map[string]string{
		"label":                       "b",
		"projectcalico.org/namespace": "default",
	},
}
var WepWithProfileValue = &model.WorkloadEndpoint{
	Name: "wep-profile",
	Labels: map[string]string{
		"projectcalico.org/namespace": "default",
	},
	ProfileIDs: []string{"profile-dev"},
}

var ProfileDevKey = model.ProfileLabelsKey{ProfileKey: model.ProfileKey{Name: "profile-dev"}}
var ProfileDevValue = map[string]string{"profile": "dev"}

var CaptureAllKey = model.ResourceKey{Name: "packet-capture-all", Namespace: "default", Kind: v3.KindPacketCapture}
var CaptureSelectionKey = model.ResourceKey{Name: "packet-capture-selection", Namespace: "default", Kind: v3.KindPacketCapture}
var CaptureDevKey = model.ResourceKey{Name: "packet-capture-dev", Namespace: "default", Kind: v3.KindPacketCapture}
var CaptureDifferentNamespaceKey = model.ResourceKey{Name: "packet-capture-different-namespace", Namespace: "different", Kind: v3.KindPacketCapture}
var CaptureAllValue = &v3.PacketCapture{
	TypeMeta: metav1.TypeMeta{
		Kind: v3.KindPacketCapture,
	},
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "default",
		Name:      "packet-capture-all",
	},
	Spec: v3.PacketCaptureSpec{
		Selector: "all()",
	},
}
var CaptureSelectAValue = &v3.PacketCapture{
	TypeMeta: metav1.TypeMeta{
		Kind: v3.KindPacketCapture,
	},
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "default",
		Name:      "packet-capture-selection",
	},
	Spec: v3.PacketCaptureSpec{
		Selector: "label == 'a'",
	},
}
var CaptureSelectBValue = &v3.PacketCapture{
	TypeMeta: metav1.TypeMeta{
		Kind: v3.KindPacketCapture,
	},
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "default",
		Name:      "packet-capture-selection",
	},
	Spec: v3.PacketCaptureSpec{
		Selector: "label == 'b'",
	},
}
var CaptureSelectDevValue = &v3.PacketCapture{
	TypeMeta: metav1.TypeMeta{
		Kind: v3.KindPacketCapture,
	},
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "default",
		Name:      "packet-capture-dev",
	},
	Spec: v3.PacketCaptureSpec{
		Selector: "profile == 'dev'",
	},
}
var CaptureDifferentNamespaceValue = &v3.PacketCapture{
	TypeMeta: metav1.TypeMeta{
		Kind: v3.KindPacketCapture,
	},
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "different",
		Name:      "packet-capture-different-namespace",
	},
	Spec: v3.PacketCaptureSpec{
		Selector: "all()",
	},
}
