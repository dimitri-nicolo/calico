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
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// scanner cli token expiration time default 100 years
var scannerCLITokenExpirationSeconds int64 = 100 * 365 * 86400

type reconciler struct {
	clusterName string
	// ownerReference is used to store the "owner" of this reconciler. If the owner has changed that signals the user
	// credential secrets should be rotated. It's valid to have an empty owner reference.
	ownerReference              string
	management                  bool
	managementK8sCLI            kubernetes.Interface
	managementOperatorNamespace string
	managedK8sCLI               kubernetes.Interface
	imageAssuranceNamespace     string
	restartChan                 chan<- string

	admissionControllerClusterRoleName string
	crAdaptorClusterRoleName           string
	intrusionDetectionClusterRoleName  string
	scannerClusterRoleName             string
	scannerCLIClusterRoleName          string
	scannerCLITokenSecretName          string
	podWatcherClusterRoleName          string
	operatorClusterRoleName            string
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

	// The management cluster already has this config map where it needs it.
	if !c.management {
		if err := c.reconcileConfigMap(); err != nil {
			reqLogger.Errorf("error reconciling admission controller config map %+v", err)
			return err
		}
	}

	if err := c.reconcileServiceAccounts(); err != nil {
		reqLogger.Errorf("error reconciling service accounts for image assurance %+v", err)
		return err
	}

	if err := c.reconcileClusterRoleBinding(); err != nil {
		reqLogger.Errorf("error reconciling cluster role bindings for image assurance %+v", err)
		return err
	}

	if err := c.reconcileCLIServiceAccountToken(); err != nil {
		reqLogger.Errorf("error reconciling cli service account token for image assurance %+v", err)
		return err
	}

	// These items are for the admission controller (which are is not in the management cluster as of right now or
	// the management cluster already has the item.
	if !c.management {
		if err := c.reconcileCASecrets(); err != nil {
			reqLogger.Errorf("error reconciling CA secrets for image assurance %+v", err)
			return err
		}

		if err := c.reconcileAdmissionControllerSecret(); err != nil {
			reqLogger.Errorf("error reconciling admission controller secrets %+v", err)
			return err
		}

		if err := c.reconcileCRAdaptorSecret(); err != nil {
			reqLogger.Errorf("error reconciling cr adaptor secrets %+v", err)
			return err
		}
	}

	reqLogger.Info("Finished reconciling ImageAssurance credentials")

	return nil
}

// reconcileConfigMap takes in tigera-image-assurance-config from management cluster and copy it over to the
// managed cluster.
func (c *reconciler) reconcileConfigMap() error {
	configMap, err := c.managementK8sCLI.CoreV1().ConfigMaps(c.managementOperatorNamespace).Get(context.Background(),
		resource.ImageAssuranceConfigMapName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	configMap.ObjectMeta.Namespace = c.imageAssuranceNamespace
	configMap.ObjectMeta.Name = resource.ImageAssuranceConfigMapName

	// Add the cluster name to the secret so Image Assurance components know what cluster they're running in.
	configMap.Data["clusterName"] = c.clusterName

	if err := resource.WriteConfigMapToK8s(c.managedK8sCLI, resource.CopyConfigMap(configMap)); err != nil {
		return err
	}

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

	secret.ObjectMeta.Namespace = c.imageAssuranceNamespace
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

// reconcileCRAdaptorSecret reconciles secrets for the CR Adaptor from management cluster to managed cluster.
func (c *reconciler) reconcileCRAdaptorSecret() error {
	sa, err := c.managementK8sCLI.CoreV1().ServiceAccounts(c.managementOperatorNamespace).Get(context.Background(),
		fmt.Sprintf(resource.ManagementIACRAdaptorResourceNameFormat, c.clusterName), metav1.GetOptions{})

	if err != nil {
		return err
	}

	saSecret, err := c.managementK8sCLI.CoreV1().Secrets(c.managementOperatorNamespace).Get(context.Background(),
		sa.Secrets[0].Name, metav1.GetOptions{})

	if err != nil {
		return err
	}

	managedSecret := c.managedCRAdaptorControllerSecret(saSecret)
	if err := resource.WriteSecretToK8s(c.managedK8sCLI, resource.CopySecret(managedSecret)); err != nil {
		return err
	}

	return nil
}

// reconcileServiceAccount ensures that service account exists in management cluster. We don't
// need to check for the existence of service account, we always write, if nothing has changed it's a no-op, else it's updated.
func (c *reconciler) reconcileServiceAccounts() error {
	// Admission controller only runs in the managed cluster.
	if !c.management {
		sa := c.admissionControllerServiceAccount()
		if err := resource.WriteServiceAccountToK8s(c.managementK8sCLI, sa); err != nil {
			return err
		}

		sa = c.crAdaptorServiceAccount()
		if err := resource.WriteServiceAccountToK8s(c.managementK8sCLI, sa); err != nil {
			return err
		}
	}

	// Intrusion detection controller, scanner, pod watcher only runs in the management cluster
	if c.management {
		isa := c.intrusionDetectionControllerServiceAccount()
		if err := resource.WriteServiceAccountToK8s(c.managementK8sCLI, isa); err != nil {
			return err
		}
		ssa := c.scannerServiceAccount()
		if err := resource.WriteServiceAccountToK8s(c.managementK8sCLI, ssa); err != nil {
			return err
		}
		psa := c.podWatcherServiceAccount()
		if err := resource.WriteServiceAccountToK8s(c.managementK8sCLI, psa); err != nil {
			return err
		}
		operatorSA := c.operatorServiceAccount()
		if err := resource.WriteServiceAccountToK8s(c.managementK8sCLI, operatorSA); err != nil {
			return err
		}
	}

	return nil
}

// reconcileClusterRoleBinding ensures that cluster role bindings exists in management cluster. We don't
// need to check for the cluster role binding here, we always write, if nothing has changed it's a no-op, else it's updated.
func (c *reconciler) reconcileClusterRoleBinding() error {
	// Admission controller only runs in the managed cluster.
	if !c.management {
		crb := c.admissionControllerClusterRoleBinding()
		if err := resource.WriteClusterRoleBindingToK8s(c.managementK8sCLI, crb); err != nil {
			return err
		}

		crb = c.crAdaptorClusterRoleBinding()
		if err := resource.WriteClusterRoleBindingToK8s(c.managementK8sCLI, crb); err != nil {
			return err
		}
	}

	// Intrusion detection controller, scanner, pod watcher only runs in the management cluster
	if c.management {
		icrb := c.idsControllerClusterRoleBinding()
		if err := resource.WriteClusterRoleBindingToK8s(c.managementK8sCLI, icrb); err != nil {
			return err
		}
		scrb := c.scannerClusterRoleBinding()
		if err := resource.WriteClusterRoleBindingToK8s(c.managementK8sCLI, scrb); err != nil {
			return err
		}
		ccrb := c.scannerCLIClusterRoleBinding()
		if err := resource.WriteClusterRoleBindingToK8s(c.managementK8sCLI, ccrb); err != nil {
			return err
		}
		pcrb := c.podWatcherClusterRoleBinding()
		if err := resource.WriteClusterRoleBindingToK8s(c.managementK8sCLI, pcrb); err != nil {
			return err
		}
		operatorCRB := c.operatorClusterRoleBinding()
		if err := resource.WriteClusterRoleBindingToK8s(c.managementK8sCLI, operatorCRB); err != nil {
			return err
		}
	}
	return nil
}

// verifyOperatorNamespaces makes sure that the active operator namespace has not changed in the
// management cluster. If the namespace has changed then send a message to the restartChan
// so the kube-controller will restart so the new namespaces can be used.
func (c *reconciler) verifyOperatorNamespaces(reqLogger *log.Entry) error {
	m, err := utils.FetchOperatorNamespace(c.managementK8sCLI)
	if err != nil {
		return fmt.Errorf("failed to fetch the operator namespace from the management cluster: %w", err)
	}
	if m != c.managementOperatorNamespace {
		msg := fmt.Sprintf("The active operator namespace for the managed cluster %s has changed from %s to %s", c.clusterName, c.managementOperatorNamespace, m)
		reqLogger.Info(msg)
		c.restartChan <- msg
	}

	return nil
}

func (c *reconciler) managedAdmissionControllerSecret(mgmtSecret *corev1.Secret) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      resource.ManagedIAAdmissionControllerResourceName,
			Namespace: c.imageAssuranceNamespace,
		},
		Data: mgmtSecret.Data,
	}
}

func (c *reconciler) managedCRAdaptorControllerSecret(mgmtSecret *corev1.Secret) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      resource.ManagedIACRAdaptorResourceName,
			Namespace: c.imageAssuranceNamespace,
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

// crAdaptorServiceAccount returns a definition for service account
func (c *reconciler) crAdaptorServiceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{Kind: rbacv1.ServiceAccountKind, APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf(resource.ManagementIACRAdaptorResourceNameFormat, c.clusterName),
			Namespace: c.managementOperatorNamespace,
		},
	}
}

// intrusionDetectionControllerServiceAccount returns a definition for service account
func (c *reconciler) intrusionDetectionControllerServiceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{Kind: rbacv1.ServiceAccountKind, APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      resource.ImageAssuranceIDSControllerServiceAccountName,
			Namespace: c.managementOperatorNamespace,
		},
	}
}

// scannerServiceAccount returns a definition for service account
func (c *reconciler) scannerServiceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{Kind: rbacv1.ServiceAccountKind, APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      resource.ImageAssuranceScannerServiceAccountName,
			Namespace: c.managementOperatorNamespace,
		},
	}
}

// podWatcherServiceAccount returns a definition for service account
func (c *reconciler) podWatcherServiceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{Kind: rbacv1.ServiceAccountKind, APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      resource.ImageAssurancePodWatcherServiceAccountName,
			Namespace: c.managementOperatorNamespace,
		},
	}
}

// operatorServiceAccount returns a definition for service account.
func (c *reconciler) operatorServiceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{Kind: rbacv1.ServiceAccountKind, APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      resource.ImageAssuranceOperatorServiceAccountName,
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
			Name:     c.admissionControllerClusterRoleName,
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

// crAdaptorClusterRoleBinding returns a definition for cluster role binding
func (c *reconciler) crAdaptorClusterRoleBinding() *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{Kind: "ClusterRoleBinding", APIVersion: "rbac.authorization.k8s.io/v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf(resource.ManagementIACRAdaptorResourceNameFormat, c.clusterName),
			Labels: map[string]string{},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     c.crAdaptorClusterRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      fmt.Sprintf(resource.ManagementIACRAdaptorResourceNameFormat, c.clusterName),
				Namespace: c.managementOperatorNamespace,
			},
		},
	}
}

// idsControllerClusterRoleBinding returns a definition for cluster role binding
func (c *reconciler) idsControllerClusterRoleBinding() *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{Kind: "ClusterRoleBinding", APIVersion: "rbac.authorization.k8s.io/v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:   resource.ImageAssuranceIDSControllerClusterRoleBindingName,
			Labels: map[string]string{},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     c.intrusionDetectionClusterRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      resource.ImageAssuranceIDSControllerServiceAccountName,
				Namespace: c.managementOperatorNamespace,
			},
		},
	}
}

// scannerClusterRoleBinding returns a definition for cluster role binding
func (c *reconciler) scannerClusterRoleBinding() *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{Kind: "ClusterRoleBinding", APIVersion: "rbac.authorization.k8s.io/v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:   resource.ImageAssuranceScannerClusterRoleBindingName,
			Labels: map[string]string{},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     c.scannerClusterRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      resource.ImageAssuranceScannerServiceAccountName,
				Namespace: c.managementOperatorNamespace,
			},
		},
	}
}

// podWatcherClusterRoleBinding returns a definition for cluster role binding
func (c *reconciler) podWatcherClusterRoleBinding() *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{Kind: "ClusterRoleBinding", APIVersion: "rbac.authorization.k8s.io/v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:   resource.ImageAssurancePodWatcherClusterRoleBindingName,
			Labels: map[string]string{},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     c.podWatcherClusterRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      resource.ImageAssurancePodWatcherServiceAccountName,
				Namespace: c.managementOperatorNamespace,
			},
		},
	}
}

// operatorClusterRoleBinding returns a definition for cluster role binding
func (c *reconciler) operatorClusterRoleBinding() *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{Kind: "ClusterRoleBinding", APIVersion: "rbac.authorization.k8s.io/v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:   resource.ImageAssuranceOperatorClusterRoleBindingName,
			Labels: map[string]string{},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     c.operatorClusterRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      resource.ImageAssuranceOperatorServiceAccountName,
				Namespace: c.managementOperatorNamespace,
			},
		},
	}
}

// scannerCLIClusterRoleBinding returns a definition for cluster role binding
func (c *reconciler) scannerCLIClusterRoleBinding() *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{Kind: "ClusterRoleBinding", APIVersion: "rbac.authorization.k8s.io/v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:   resource.ImageAssuranceScannerCLIClusterRoleBindingName,
			Labels: map[string]string{},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     c.scannerCLIClusterRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      resource.ImageAssuranceScannerCLIServiceAccountName,
				Namespace: resource.ManagerNameSpaceName,
			},
		},
	}
}

// reconcileCLIServiceAccountToken creates a token for tigera-image-assurance-scanner-cli-api-access service account that
// can be used with scanner CLI.
func (c *reconciler) reconcileCLIServiceAccountToken() error {
	scsa := c.scannerCLITokenServiceAccount()
	if err := resource.WriteServiceAccountToK8s(c.managementK8sCLI, scsa); err != nil {
		return err
	}

	secret := c.scannerCLITokenSecret()
	if err := resource.WriteSecretToK8s(c.managementK8sCLI, secret); err != nil {
		return err
	}

	tokenRequest := c.scannerCLITokenTokenRequest(secret)
	tokenResp, err := resource.WriteServiceAccountTokenRequestToK8s(c.managementK8sCLI, tokenRequest, scsa.Name)
	if err != nil {
		return err
	}

	secret.Data["token"] = []byte(tokenResp.Status.Token)
	if err := resource.WriteSecretToK8s(c.managementK8sCLI, secret); err != nil {
		return err
	}

	return nil
}

//scannerCLITokenSecret returns definition for scanner CLI API token secret with a well known name in manager namespace.
func (c *reconciler) scannerCLITokenSecret() *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.scannerCLITokenSecretName,
			Namespace: resource.ManagerNameSpaceName,
		},
		Data: map[string][]byte{},
	}
}

func (c *reconciler) scannerCLITokenTokenRequest(secret *corev1.Secret) *authv1.TokenRequest {
	return &authv1.TokenRequest{
		TypeMeta: metav1.TypeMeta{Kind: "TokenRequest", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.scannerCLITokenSecretName,
			Namespace: resource.ManagerNameSpaceName,
		},
		Spec: authv1.TokenRequestSpec{
			ExpirationSeconds: &scannerCLITokenExpirationSeconds,
			BoundObjectRef: &authv1.BoundObjectReference{
				Kind:       "Secret",
				APIVersion: secret.APIVersion,
				Name:       secret.Name,
				UID:        secret.UID,
			},
		},
	}
}

// scannerCLITokenServiceAccount returns a definition for service account to be created in manager namespace for cli api access.
func (c *reconciler) scannerCLITokenServiceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{Kind: rbacv1.ServiceAccountKind, APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      resource.ImageAssuranceScannerCLIServiceAccountName,
			Namespace: resource.ManagerNameSpaceName,
		},
	}
}
