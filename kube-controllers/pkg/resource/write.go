// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.

// file write contains methods to help write k8s resources to the k8s cluster
package resource

import (
	"context"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	tigeraapi "github.com/tigera/api/pkg/client/clientset_generated/clientset"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
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
	var created *corev1.Secret
	var err error

	ctx := context.Background()

	if _, err = cli.CoreV1().Secrets(secret.Namespace).Get(ctx, secret.Name, metav1.GetOptions{}); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		if created, err = cli.CoreV1().Secrets(secret.Namespace).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
			return err
		}
	} else {
		if created, err = cli.CoreV1().Secrets(secret.Namespace).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}

	// Assign the created secret to the given one.
	*secret = *created
	// If this is a new secret, default Data to an empty map for backwards compatability.
	if secret.Data == nil {
		secret.Data = map[string][]byte{}
	}
	return nil
}

// WriteLicenseKeyToK8s creates or updates the give v3.LicenseKey using the give tigeraapi.Interfaces depending on whether or
// not the given Secret exists in the k8s cluster
func WriteLicenseKeyToK8s(cli tigeraapi.Interface, licenseKey *v3.LicenseKey) error {
	ctx := context.Background()

	if license, err := cli.ProjectcalicoV3().LicenseKeys().Get(ctx, licenseKey.Name, metav1.GetOptions{}); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		if _, err := cli.ProjectcalicoV3().LicenseKeys().Create(ctx, licenseKey, metav1.CreateOptions{}); err != nil {
			return err
		}
	} else {
		licenseKey.ResourceVersion = license.ResourceVersion
		if _, err := cli.ProjectcalicoV3().LicenseKeys().Update(ctx, licenseKey, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}
	return nil
}

// WriteServiceAccountToK8s Secret creates or updates the given corev1.ServiceAccount using the given kuberenetes.Clientset
// depending on whether not the given ServiceAccount exists in the k8s cluster
func WriteServiceAccountToK8s(cli kubernetes.Interface, serviceAccount *corev1.ServiceAccount) error {
	ctx := context.Background()

	if sa, err := cli.CoreV1().ServiceAccounts(serviceAccount.Namespace).Get(ctx, serviceAccount.Name, metav1.GetOptions{}); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		if _, err := cli.CoreV1().ServiceAccounts(serviceAccount.Namespace).Create(ctx, serviceAccount, metav1.CreateOptions{}); err != nil {
			return err
		}
	} else {
		serviceAccount.Secrets = sa.Secrets
		if _, err := cli.CoreV1().ServiceAccounts(serviceAccount.Namespace).Update(ctx, serviceAccount, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}
	return nil
}

// WriteClusterRoleBindingToK8s creates or updates the given rbacv1.ClusterRoleBinding using the given kuberenetes.Clientset
// depending on whether not the given ClusterRoleBinding exists in the k8s cluster
func WriteClusterRoleBindingToK8s(cli kubernetes.Interface, crb *rbacv1.ClusterRoleBinding) error {
	ctx := context.Background()

	if _, err := cli.RbacV1().ClusterRoleBindings().Get(ctx, crb.Name, metav1.GetOptions{}); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		if _, err := cli.RbacV1().ClusterRoleBindings().Create(ctx, crb, metav1.CreateOptions{}); err != nil {
			return err
		}
	} else {
		if _, err := cli.RbacV1().ClusterRoleBindings().Update(ctx, crb, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}
	return nil
}

// WriteServiceAccountTokenRequestToK8s creates token against the token requests that is sent for a given service account name.
func WriteServiceAccountTokenRequestToK8s(cli kubernetes.Interface, treq *authenticationv1.TokenRequest, saName string) (*authenticationv1.TokenRequest, error) {
	ctx := context.Background()

	response, err := cli.CoreV1().ServiceAccounts(treq.Namespace).CreateToken(ctx, saName, treq, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return response, nil
}
