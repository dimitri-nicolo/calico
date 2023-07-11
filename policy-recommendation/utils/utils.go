// Copyright (c) 2023 Tigera, Inc. All rights reserved
package utils

import (
	"fmt"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"
)

// copyStagedNetworkPolicy copies the StagedNetworkPolicy context that may be altered by the engine,
// from a source to a destination.
// Copy:
// - egress, ingress rules, and policy types
// - Name, and namespace
// - Labels, and annotations
func CopyStagedNetworkPolicy(dest *v3.StagedNetworkPolicy, src v3.StagedNetworkPolicy) {
	// Copy egress, ingres and policy type over to the destination
	dest.Spec.Egress = make([]v3.Rule, len(src.Spec.Egress))
	copy(dest.Spec.Egress, src.Spec.Egress)
	dest.Spec.Ingress = make([]v3.Rule, len(src.Spec.Ingress))
	copy(dest.Spec.Ingress, src.Spec.Ingress)
	dest.Spec.Types = make([]v3.PolicyType, len(src.Spec.Types))
	copy(dest.Spec.Types, src.Spec.Types)

	// Copy ObjectMeta context. Context relevant to this controller is name, labels and annotation
	dest.ObjectMeta.Name = src.GetObjectMeta().GetName()
	dest.ObjectMeta.Namespace = src.GetObjectMeta().GetNamespace()

	dest.ObjectMeta.Labels = make(map[string]string)
	for key, label := range src.GetObjectMeta().GetLabels() {
		dest.ObjectMeta.Labels[key] = label
	}
	dest.ObjectMeta.Annotations = make(map[string]string)
	for key, annotation := range src.GetObjectMeta().GetAnnotations() {
		dest.ObjectMeta.Annotations[key] = annotation
	}

	dest.Spec.Selector = src.Spec.Selector

	dest.OwnerReferences = src.OwnerReferences
}

// GetPolicyName returns a policy name with tier prefix and 5 char hash suffix.
func GetPolicyName(tier, name string, suffixGenerator func() string) string {
	return fmt.Sprintf("%s.%s-%s", tier, name, suffixGenerator())
}

// SuffixGenerator returns a random 5 char string, typically used as a suffix to a name in
// Kubernetes.
func SuffixGenerator() string {
	return testutils.RandStringRunes(5)
}
