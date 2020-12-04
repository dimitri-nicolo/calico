// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.

// file write contains methods to help write k8s resources to the k8s cluster
package resource

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ConfigMap creates or updates the given corev1.ConfigMap using the given kuberenetes.Clientset depending on whether or
// not the given ConfigMap exists in the k8s cluster
func WriteConfigMapToK8s(cli kubernetes.Interface, configMap *corev1.ConfigMap) error {
	ctx := context.Background()

	if _, err := cli.CoreV1().ConfigMaps(configMap.Namespace).Get(ctx, configMap.Name, metav1.GetOptions{}); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		if _, err := cli.CoreV1().ConfigMaps(configMap.Namespace).Create(ctx, configMap, metav1.CreateOptions{}); err != nil {
			return err
		}
	} else {
		if _, err := cli.CoreV1().ConfigMaps(configMap.Namespace).Update(ctx, configMap, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}
	return nil
}

// Secret creates or updates the given corev1.Secret using the given kuberenetes.Clientset depending on whether or
// not the given Secret exists in the k8s cluster
func WriteSecretToK8s(cli kubernetes.Interface, secret *corev1.Secret) error {
	ctx := context.Background()

	if _, err := cli.CoreV1().Secrets(secret.Namespace).Get(ctx, secret.Name, metav1.GetOptions{}); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		if _, err := cli.CoreV1().Secrets(secret.Namespace).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
			return err
		}
	} else {
		if _, err := cli.CoreV1().Secrets(secret.Namespace).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}
	return nil
}
