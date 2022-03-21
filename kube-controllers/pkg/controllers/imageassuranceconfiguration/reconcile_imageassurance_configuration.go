// Copyright (c) 2022 Tigera, Inc. All rights reserved.

//go:build tesla
// +build tesla

package imageassuranceconfiguration

import (
	"context"
	"fmt"

	"github.com/projectcalico/calico/kube-controllers/pkg/controllers/utils"
	"github.com/projectcalico/calico/kube-controllers/pkg/resource"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

type reconciler struct {
	clusterName string
	// ownerReference is used to store the "owner" of this reconciler. If the owner has changed that signals the user
	// credential secrets should be rotated. It's valid to have an empty owner reference.
	ownerReference                 string
	management                     bool
	managementK8sCLI               kubernetes.Interface
	managementOperatorNamespace    string
	managedK8sCLI                  kubernetes.Interface
	managedImageAssuranceNamespace string
	restartChan                    chan<- string
}

// Reconcile makes sure that the managed cluster this is running for has all the configuration needed for it's components
// to access image assurance. If the managed cluster this is running for is actually a management cluster, then the secret
// for the image assurance api certificate are not copied over
func (c *reconciler) Reconcile(name types.NamespacedName) error {
	reqLogger := log.WithFields(map[string]interface{}{
		"cluster": c.clusterName,
		"key":     name,
	})
	reqLogger.Info("Reconciling ImageAssurance credentials")

	if err := c.verifyOperatorNamespaces(reqLogger); err != nil {
		return err
	}

	if err := c.reconcileServiceAccount(); err != nil {
		reqLogger.Errorf("error reconciling admission controller service account %+v", err)
		return err
	}

	if err := c.reconcileClusterRoleBinding(); err != nil {
		reqLogger.Errorf("error reconciling admission controller cluster role binding %+v", err)
		return err
	}

	if err := c.reconcileCASecrets(); err != nil {
		reqLogger.Errorf("error reconciling CA secrets for image assurance %+v", err)
		return err
	}

	if err := c.reconcileAdmissionControllerSecret(); err != nil {
		reqLogger.Errorf("error reconciling admission controller secrets %+v", err)
		return err
	}

	reqLogger.Info("Finished reconciling ImageAssurance credentials")

	return nil
}

// reconcileCASecrets takes in tigera-image-assurance-api-cert-pair from management cluster, modifies it to copy it over to
// managed cluster. It copies only the CA crt to managed cluster, it also renames it to tigera-image-assurance-api-cert.
func (c *reconciler) reconcileCASecrets() error {
	secret, err := c.managementK8sCLI.CoreV1().Secrets(c.managementOperatorNamespace).Get(context.Background(),
		resource.ImageAssuranceAPICertPairSecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	secret.ObjectMeta.Namespace = c.managedImageAssuranceNamespace
	secret.ObjectMeta.Name = resource.ImageAssuranceAPICertSecretName
	secret.Data = map[string][]byte{
		corev1.TLSCertKey: secret.Data[corev1.TLSCertKey],
	}
	if err := resource.WriteSecretToK8s(c.managedK8sCLI, resource.CopySecret(secret)); err != nil {
		return err
	}

	return nil
}

// reconcileAdmissionControllerSecret reconciles secrets for admission controller from management cluster to managed cluster
func (c *reconciler) reconcileAdmissionControllerSecret() error {
	sa, err := c.managementK8sCLI.CoreV1().ServiceAccounts(c.managementOperatorNamespace).Get(context.Background(),
		fmt.Sprintf(resource.ManagementIAAdmissionControllerResourceNameFormat, c.clusterName), metav1.GetOptions{})

	if err != nil {
		return err
	}

	saSecret, err := c.managementK8sCLI.CoreV1().Secrets(c.managementOperatorNamespace).Get(context.Background(),
		sa.Secrets[0].Name, metav1.GetOptions{})

	if err != nil {
		return err
	}

	managedSecret := c.managedAdmissionControllerSecret(saSecret)
	if err := resource.WriteSecretToK8s(c.managedK8sCLI, resource.CopySecret(managedSecret)); err != nil {
		return err
	}

	return nil
}

// reconcileServiceAccount ensures that service account exists for admission controller in management cluster. We don't
// need to check for the existence of service account, we always write, if nothing has changed it's a no-op, else it's updated.
func (c *reconciler) reconcileServiceAccount() error {
	mgmtSa, err := c.managementK8sCLI.CoreV1().ServiceAccounts(c.managementOperatorNamespace).Get(context.Background(),
		fmt.Sprintf(resource.ManagementIAAdmissionControllerResourceNameFormat, c.clusterName), metav1.GetOptions{})

	if !errors.IsNotFound(err) {
		return err
	}

	sa := c.admissionControllerServiceAccount()
	// if service account exists in management cluster, then copy the name of the secret of Service account.
	if mgmtSa != nil {
		sa.Secrets = mgmtSa.Secrets
	}
	if err := resource.WriteServiceAccountToK8s(c.managementK8sCLI, sa); err != nil {
		return err
	}

	return nil
}

// reconcileClusterRoleBinding ensures that cluster role binding exists for admission controller in management cluster. We don't
// need to check for the cluster role binding here, we always write, if nothing has changed it's a no-op, else it's updated.
func (c *reconciler) reconcileClusterRoleBinding() error {
	crb := c.admissionControllerClusterRoleBinding()
	if err := resource.WriteClusterRoleBindingToK8s(c.managementK8sCLI, crb); err != nil {
		return err
	}

	return nil
}

// verifyOperatorNamespaces makes sure that the active operator namespace has not changed in the
// managed or management cluster. If the namespace has changed then send a message to the restartChan
// so the kube-controller will restart so the new namespaces can be used.
func (c *reconciler) verifyOperatorNamespaces(reqLogger *log.Entry) error {
	if !c.management {
		m, err := utils.FetchOperatorNamespace(c.managementK8sCLI)
		if err != nil {
			return fmt.Errorf("failed to fetch the operator namespace from the management cluster: %w", err)
		}
		if m != c.managementOperatorNamespace {
			msg := fmt.Sprintf("The active operator namespace for the managed cluster %s has changed from %s to %s", c.clusterName, c.managementOperatorNamespace, m)
			reqLogger.Info(msg)
			c.restartChan <- msg
		}
	}
	return nil
}

func (c *reconciler) managedAdmissionControllerSecret(mgmtSecret *corev1.Secret) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      resource.ManagedIAAdmissionControllerResourceName,
			Namespace: c.managedImageAssuranceNamespace,
		},
		Data: mgmtSecret.Data,
	}
}

// admissionControllerServiceAccount returns a definition for service account
func (c *reconciler) admissionControllerServiceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{Kind: rbacv1.ServiceAccountKind, APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf(resource.ManagementIAAdmissionControllerResourceNameFormat, c.clusterName),
			Namespace: c.managementOperatorNamespace,
		},
	}
}

// admissionControllerClusterRoleBinding returns a definition for cluster role binding
func (c *reconciler) admissionControllerClusterRoleBinding() *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{Kind: "ClusterRoleBinding", APIVersion: "rbac.authorization.k8s.io/v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf(resource.ManagementIAAdmissionControllerResourceNameFormat, c.clusterName),
			Labels: map[string]string{},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     resource.ImageAssuranceAdmissionControllerRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      fmt.Sprintf(resource.ManagementIAAdmissionControllerResourceNameFormat, c.clusterName),
				Namespace: c.managementOperatorNamespace,
			},
		},
	}
}
