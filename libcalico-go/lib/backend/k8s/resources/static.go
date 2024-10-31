// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package resources

import (
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
)

// This file contains helpers for creating hard-coded resources used in the Kuberenetes backend.

// DefaultProfile returns a single profile kvp with default allow rules.
// Since KDD doesn't support creation of arbitrary profiles, this profile can be used
// for non-kubernetes endpoints (e.g. host endpoints) to allow traffic.
func DefaultProfile() *model.KVPair {
	// Create the profile
	profile := v3.NewProfile()
	profile.ObjectMeta = metav1.ObjectMeta{
		Name: "default",
	}
	profile.Spec = v3.ProfileSpec{
		Ingress: []v3.Rule{{Action: v3.Allow}},
		Egress:  []v3.Rule{{Action: v3.Allow}},
	}

	// Embed the profile in a KVPair.
	return &model.KVPair{
		Key: model.ResourceKey{
			Name: "default",
			Kind: v3.KindProfile,
		},
		Value:    profile,
		Revision: "",
	}
}
