// Copyright (c) 2025 Tigera, Inc. All rights reserved.

package types

import (
	"testing"

	googleproto "google.golang.org/protobuf/proto"

	"github.com/projectcalico/calico/felix/proto"
)

func TestProtoToExternalNetworkID(t *testing.T) {
	tests := []struct {
		name string
		s    *proto.ExternalNetworkID
		want ExternalNetworkID
	}{
		{"empty", &proto.ExternalNetworkID{}, ExternalNetworkID{}},
		{"non-empty",
			&proto.ExternalNetworkID{Name: "foo"},
			ExternalNetworkID{Name: "foo"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ProtoToExternalNetworkID(tt.s); got != tt.want {
				t.Errorf("ProtoToExternalNetworkID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExternalNetworkIDToProto(t *testing.T) {
	tests := []struct {
		name string
		s    ExternalNetworkID
		want *proto.ExternalNetworkID
	}{
		{"empty", ExternalNetworkID{}, &proto.ExternalNetworkID{}},
		{"non-empty",
			ExternalNetworkID{Name: "foo"},
			&proto.ExternalNetworkID{Name: "foo"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExternalNetworkIDToProto(tt.s); !googleproto.Equal(got, tt.want) {
				t.Errorf("ExternalNetworkIDToProto() = %v, want %v", got, tt.want)
			}
		})
	}
}
