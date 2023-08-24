// Copyright (c) 2023 Tigera, Inc. All rights reserved
package utils

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	k8svalidation "k8s.io/apimachinery/pkg/util/validation"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"
)

// copyStagedNetworkPolicy copies the StagedNetworkPolicy context relevant to the recommendation
// engine from a source to a destination.
// Copy:
// - Metadata:
// -   Name, Namespace, OwnerReference, Annotations, Labels
// - Spec:
// -   Selector, StagedAction, Tier
// -   Egress, Ingress rules, and PolicyTypes
func CopyStagedNetworkPolicy(dest *v3.StagedNetworkPolicy, src v3.StagedNetworkPolicy) {
	// Metadata
	dest.ObjectMeta.Name = src.GetObjectMeta().GetName()
	dest.ObjectMeta.Namespace = src.GetObjectMeta().GetNamespace()
	dest.ObjectMeta.OwnerReferences = src.GetObjectMeta().GetOwnerReferences()
	dest.ObjectMeta.Annotations = make(map[string]string)
	for key, annotation := range src.GetObjectMeta().GetAnnotations() {
		dest.ObjectMeta.Annotations[key] = annotation
	}
	dest.ObjectMeta.Labels = make(map[string]string)
	for key, label := range src.GetObjectMeta().GetLabels() {
		dest.ObjectMeta.Labels[key] = label
	}

	// Spec
	dest.Spec.Selector = src.Spec.Selector
	dest.Spec.StagedAction = src.Spec.StagedAction
	dest.Spec.Tier = src.Spec.Tier
	dest.Spec.Egress = make([]v3.Rule, len(src.Spec.Egress))
	copy(dest.Spec.Egress, src.Spec.Egress)
	dest.Spec.Ingress = make([]v3.Rule, len(src.Spec.Ingress))
	copy(dest.Spec.Ingress, src.Spec.Ingress)
	dest.Spec.Types = make([]v3.PolicyType, len(src.Spec.Types))
	copy(dest.Spec.Types, src.Spec.Types)
}

// GetPolicyName returns a policy name with tier prefix and 5 char hash suffix. If there is an
// error in generating the policy name, it returns an empty string.
func GetPolicyName(tier, name string, suffixGenerator func() string) string {
	policy, err := getRFC1123PolicyName(tier, name, suffixGenerator())
	if err != nil {
		log.WithError(err).Errorf("Failed to generate policy name for tier %s and name %s", tier, name)
	}

	return policy
}

// SuffixGenerator returns a random 5 char string, typically used as a suffix to a name in
// Kubernetes.
func SuffixGenerator() string {
	return testutils.RandStringRunes(5)
}

// getRFC1123PolicyName returns an RFC 1123 compliant policy name.
// Returns a policy name with the following format: <TIER>.<NAME>-<SUFFIX>, if the length is valid.
// Otherwise, it cuts the <TIER>.<NAME> down to size for validity, and returns the adapted policy
// name followed by the suffix.
// The tier name and the name are assumed to be valid RFC 1123 compliant names. The suffix is a
// random 5 char string.
func getRFC1123PolicyName(tier, name, suffix string) (string, error) {
	if tier == "" || name == "" {
		return "", fmt.Errorf("either tier name '%s' or policy name '%s' is empty", tier, name)
	}

	max := k8svalidation.DNS1123LabelMaxLength - (len(suffix) + 1)
	if len(tier)+2 > max {
		return "", fmt.Errorf("tier name %s is too long to be used in a policy name", tier)
	}

	policy := fmt.Sprintf("%s.%s", tier, name)
	if len(policy) > max {
		// Truncate policy name to max length
		policy = policy[:max]
	}

	return fmt.Sprintf("%s-%s", policy, suffix), nil
}
