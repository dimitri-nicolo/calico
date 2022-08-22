// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.

// package copy contains functions to help copy k8s resources
package resource

import (
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigMap copies the given corev1.ConfigMap to a new instance. Check the clearSystemData method to see what values are
// retained in the ObjectMeta after the copy.
func CopyConfigMap(s *corev1.ConfigMap) *corev1.ConfigMap {
	d := s.DeepCopy()
	d.ObjectMeta = clearSystemData(d.ObjectMeta)

	return d
}

// Secret copies the given corev1.Secret to a new instance. Check the clearSystemData method to see what values are retained
// in the ObjectMeta after the copy.
func CopySecret(s *corev1.Secret) *corev1.Secret {
	d := s.DeepCopy()
	d.ObjectMeta = clearSystemData(d.ObjectMeta)

	return d
}

// CopyLicenseKey copies the given v3.LicenseKey to a new instance. Check the clearSystemData method to see what values are retained
// in the ObjectMeta after the copy.
func CopyLicenseKey(s *v3.LicenseKey) *v3.LicenseKey {
	d := s.DeepCopy()
	d.ObjectMeta = clearSystemData(d.ObjectMeta)

	return d
}

// clearSystemData creates a new ObjectMeta struct and copies all fields from the given Object meta struct not set by
// the system
func clearSystemData(meta metav1.ObjectMeta) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:        meta.Name,
		Namespace:   meta.Namespace,
		Annotations: meta.Annotations,
		Labels:      meta.Labels,
		Finalizers:  meta.Finalizers,
	}
}
