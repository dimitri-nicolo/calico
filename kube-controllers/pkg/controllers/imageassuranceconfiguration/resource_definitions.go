// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build tesla
// +build tesla

package imageassuranceconfiguration

import (
	authnv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *reconciler) getSecretDefinition(secretName, namespace string, data map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Data: data,
	}
}

// getTokenSecretDefinition returns definition for a secret with given name and namespace.
func getTokenSecretDefinition(secretName, namespace string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{},
	}
}

// getTokenRequestDefinitionWithSecret returns authv1.TokenRequest definition which is bound to the secret that is passed.
func getTokenRequestDefinitionWithSecret(tokenRequestName, namespace string, secret *corev1.Secret) *authnv1.TokenRequest {
	return &authnv1.TokenRequest{
		TypeMeta: metav1.TypeMeta{Kind: "TokenRequest", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      tokenRequestName,
			Namespace: namespace,
		},
		Spec: authnv1.TokenRequestSpec{
			ExpirationSeconds: &defaultTokenExpirationSeconds,
			BoundObjectRef: &authnv1.BoundObjectReference{
				Kind:       "Secret",
				APIVersion: secret.APIVersion,
				Name:       secret.Name,
				UID:        secret.UID,
			},
		},
	}
}

// getServiceAccountDefinition returns a definition for a service account with given name and namespace.
func getServiceAccountDefinition(serviceAccountName, namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{Kind: rbacv1.ServiceAccountKind, APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: namespace,
		},
	}
}

// getClusterRoleBindingDefinition returns a definition for cluster role binding with given parameters.
func getClusterRoleBindingDefinition(resourceName, roleName, subjectName, subjectNameSpace string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{Kind: "ClusterRoleBinding", APIVersion: "rbac.authorization.k8s.io/v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:   resourceName,
			Labels: map[string]string{},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     roleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      subjectName,
				Namespace: subjectNameSpace,
			},
		},
	}
}
