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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// token expiration time for all token requests, defaults to 100 years
var defaultTokenExpirationSeconds int64 = 100 * 365 * 86400

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
	runtimeCleanerClusterRoleName      string
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

	if err := c.reconcileManagementServiceAccountSecrets(); err != nil {
		reqLogger.Errorf("error reconciling service accounts for image assurance %+v", err)
		return err
	}

	if err := c.reconcileClusterRoleBindings(); err != nil {
		reqLogger.Errorf("error reconciling cluster role bindings for image assurance %+v", err)
		return err
	}

	if err := c.reconcileCLIServiceAccountToken(); err != nil {
		reqLogger.Errorf("error reconciling cli service account token for image assurance %+v", err)
		return err
	}

	// These items are only for managed cluster components, as of right now or management cluster already has the item.
	if !c.management {
		if err := c.reconcileCASecrets(); err != nil {
			reqLogger.Errorf("error reconciling CA secrets for image assurance %+v", err)
			return err
		}

		if err := c.reconcileAdmissionControllerToken(); err != nil {
			reqLogger.Errorf("error reconciling admission controller secrets %+v", err)
			return err
		}

		if err := c.reconcileCRAdaptorToken(); err != nil {
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

// reconcileManagementServiceAccountSecrets ensures that service account and a token against it exists in management cluster.
func (c *reconciler) reconcileManagementServiceAccountSecrets() error {
	// Intrusion detection controller, scanner, pod watcher only runs in the management cluster
	if c.management {
		err := c.reconcileManagementServiceAccountSecret(resource.ImageAssuranceIDSControllerServiceAccountName, c.managementOperatorNamespace)
		if err != nil {
			return err
		}
		err = c.reconcileManagementServiceAccountSecret(resource.ImageAssuranceScannerServiceAccountName, c.managementOperatorNamespace)
		if err != nil {
			return err
		}
		err = c.reconcileManagementServiceAccountSecret(resource.ImageAssuranceOperatorServiceAccountName, c.managementOperatorNamespace)
		if err != nil {
			return err
		}
		err = c.reconcileManagementServiceAccountSecret(resource.ImageAssuranceRuntimeCleanerServiceAccountName, c.managementOperatorNamespace)
		if err != nil {
			return err
		}
	}

	return nil
}

// reconcileClusterRoleBindings ensures that cluster role bindings exists in management cluster. We don't
// need to check for the cluster role binding here, we always write, if nothing has changed it's a no-op, else it's updated.
func (c *reconciler) reconcileClusterRoleBindings() error {
	// Admission controller only runs in the managed cluster.
	if !c.management {
		mgmtResourceName := fmt.Sprintf(resource.ManagementIAAdmissionControllerResourceNameFormat, c.clusterName)
		crb := getClusterRoleBindingDefinition(mgmtResourceName, c.admissionControllerClusterRoleName, mgmtResourceName, c.managementOperatorNamespace)
		if err := resource.WriteClusterRoleBindingToK8s(c.managementK8sCLI, crb); err != nil {
			return err
		}

		mgmtResourceName = fmt.Sprintf(resource.ManagementIACRAdaptorResourceNameFormat, c.clusterName)
		crb = getClusterRoleBindingDefinition(mgmtResourceName, c.crAdaptorClusterRoleName, mgmtResourceName, c.managementOperatorNamespace)
		if err := resource.WriteClusterRoleBindingToK8s(c.managementK8sCLI, crb); err != nil {
			return err
		}
	}

	// Intrusion detection controller, scanner, pod watcher only runs in the management cluster
	if c.management {
		icrb := getClusterRoleBindingDefinition(resource.ImageAssuranceIDSControllerClusterRoleBindingName, c.intrusionDetectionClusterRoleName,
			resource.ImageAssuranceIDSControllerServiceAccountName, c.managementOperatorNamespace)
		if err := resource.WriteClusterRoleBindingToK8s(c.managementK8sCLI, icrb); err != nil {
			return err
		}
		scrb := getClusterRoleBindingDefinition(resource.ImageAssuranceScannerClusterRoleBindingName, c.scannerClusterRoleName,
			resource.ImageAssuranceScannerServiceAccountName, c.managementOperatorNamespace)
		if err := resource.WriteClusterRoleBindingToK8s(c.managementK8sCLI, scrb); err != nil {
			return err
		}
		ccrb := getClusterRoleBindingDefinition(resource.ImageAssuranceScannerCLIClusterRoleBindingName, c.scannerCLIClusterRoleName,
			resource.ImageAssuranceScannerCLIServiceAccountName, resource.ManagerNameSpaceName)
		if err := resource.WriteClusterRoleBindingToK8s(c.managementK8sCLI, ccrb); err != nil {
			return err
		}
		pcrb := getClusterRoleBindingDefinition(resource.ImageAssurancePodWatcherClusterRoleBindingName, c.podWatcherClusterRoleName,
			resource.ImageAssurancePodWatcherServiceAccountName, c.managementOperatorNamespace)
		if err := resource.WriteClusterRoleBindingToK8s(c.managementK8sCLI, pcrb); err != nil {
			return err
		}
		operatorCRB := getClusterRoleBindingDefinition(resource.ImageAssuranceOperatorClusterRoleBindingName, c.operatorClusterRoleName,
			resource.ImageAssuranceOperatorServiceAccountName, c.managementOperatorNamespace)
		if err := resource.WriteClusterRoleBindingToK8s(c.managementK8sCLI, operatorCRB); err != nil {
			return err
		}
		rcrb := getClusterRoleBindingDefinition(resource.ImageAssuranceRuntimeCleanerClusterRoleBindingName, c.runtimeCleanerClusterRoleName,
			resource.ImageAssuranceRuntimeCleanerServiceAccountName, c.managementOperatorNamespace)
		if err := resource.WriteClusterRoleBindingToK8s(c.managementK8sCLI, rcrb); err != nil {
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

//reconcileAdmissionControllerToken creates a service account and secret for the admission controller in the management cluster
// using token request API, and then copies the secret to the managed cluster with a well-known name
// (tigera-image-assurance-admission-controller-api-access) to be used by the admission controller.
func (c *reconciler) reconcileAdmissionControllerToken() error {
	mgmtClusterResourceName := fmt.Sprintf(resource.ManagementIAAdmissionControllerResourceNameFormat, c.clusterName)
	sa := getServiceAccountDefinition(mgmtClusterResourceName, c.managementOperatorNamespace)
	if err := resource.WriteServiceAccountToK8s(c.managementK8sCLI, sa); err != nil {
		return err
	}

	secret := getTokenSecretDefinition(mgmtClusterResourceName, c.managementOperatorNamespace)
	if err := resource.WriteSecretToK8s(c.managementK8sCLI, secret); err != nil {
		return err
	}

	tokenRequest := getTokenRequestDefinitionWithSecret(mgmtClusterResourceName, c.managementOperatorNamespace, secret)
	tokenResp, err := resource.WriteServiceAccountTokenRequestToK8s(c.managementK8sCLI, tokenRequest, sa.Name)
	if err != nil {
		return err
	}

	// Update the empty secret in management cluster with token data.
	secret.Data["token"] = []byte(tokenResp.Status.Token)
	if err := resource.WriteSecretToK8s(c.managementK8sCLI, secret); err != nil {
		return err
	}

	// Copy the same secret to managed cluster with well known name and token data.
	mngdSecret := c.getSecretDefinition(resource.ManagedIAAdmissionControllerResourceName, c.imageAssuranceNamespace, secret.Data)
	if err := resource.WriteSecretToK8s(c.managedK8sCLI, mngdSecret); err != nil {
		return err
	}

	return nil
}

// reconcileCRAdaptorToken creates a service account and secret for the CR adaptor in the management cluster
// using token request API, and then copies the secret to the managed cluster with a well-known name
// (tigera-image-assurance-cr-adaptor-api-access) to be used by the CR adaptor.
func (c *reconciler) reconcileCRAdaptorToken() error {
	mgmtClusterResourceName := fmt.Sprintf(resource.ManagementIACRAdaptorResourceNameFormat, c.clusterName)
	sa := getServiceAccountDefinition(mgmtClusterResourceName, c.managementOperatorNamespace)
	if err := resource.WriteServiceAccountToK8s(c.managementK8sCLI, sa); err != nil {
		return err
	}

	secret := getTokenSecretDefinition(mgmtClusterResourceName, c.managementOperatorNamespace)
	if err := resource.WriteSecretToK8s(c.managementK8sCLI, secret); err != nil {
		return err
	}

	tokenRequest := getTokenRequestDefinitionWithSecret(mgmtClusterResourceName, c.managementOperatorNamespace, secret)
	tokenResp, err := resource.WriteServiceAccountTokenRequestToK8s(c.managementK8sCLI, tokenRequest, sa.Name)
	if err != nil {
		return err
	}

	// Update the empty secret in management cluster with token data.
	secret.Data["token"] = []byte(tokenResp.Status.Token)
	if err := resource.WriteSecretToK8s(c.managementK8sCLI, secret); err != nil {
		return err
	}

	// Copy the same secret to managed cluster with well known name and token data.
	mngdSecret := c.getSecretDefinition(resource.ManagedIACRAdaptorResourceName, c.imageAssuranceNamespace, secret.Data)
	if err := resource.WriteSecretToK8s(c.managedK8sCLI, mngdSecret); err != nil {
		return err
	}

	return nil
}

// reconcileCLIServiceAccountToken creates a token for tigera-image-assurance-scanner-cli-api-access service account that
// can be used with scanner CLI.
func (c *reconciler) reconcileCLIServiceAccountToken() error {
	scsa := getServiceAccountDefinition(resource.ImageAssuranceScannerCLIServiceAccountName, resource.ManagerNameSpaceName)
	if err := resource.WriteServiceAccountToK8s(c.managementK8sCLI, scsa); err != nil {
		return err
	}

	secret := getTokenSecretDefinition(c.scannerCLITokenSecretName, resource.ManagerNameSpaceName)
	if err := resource.WriteSecretToK8s(c.managementK8sCLI, secret); err != nil {
		return err
	}

	tokenRequest := getTokenRequestDefinitionWithSecret(c.scannerCLITokenSecretName, resource.ManagerNameSpaceName, secret)
	tokenResp, err := resource.WriteServiceAccountTokenRequestToK8s(c.managementK8sCLI, tokenRequest, scsa.Name)
	if err != nil {
		return err
	}

	// Update the empty secret with token data.
	secret.Data["token"] = []byte(tokenResp.Status.Token)
	if err := resource.WriteSecretToK8s(c.managementK8sCLI, secret); err != nil {
		return err
	}

	return nil
}

// reconcileManagementServiceAccountSecret creates a service account with a provided name and namespace in management cluster.
// It also creates secret with the same name, namespace as the provided service account and populates with an API token using TokenRequest API.
func (c *reconciler) reconcileManagementServiceAccountSecret(resourceName, namespace string) error {
	sa := getServiceAccountDefinition(resourceName, namespace)
	if err := resource.WriteServiceAccountToK8s(c.managementK8sCLI, sa); err != nil {
		return err
	}

	secret := getTokenSecretDefinition(resourceName, namespace)
	if err := resource.WriteSecretToK8s(c.managementK8sCLI, secret); err != nil {
		return err
	}

	tokenRequest := getTokenRequestDefinitionWithSecret(resourceName, namespace, secret)
	tokenResp, err := resource.WriteServiceAccountTokenRequestToK8s(c.managementK8sCLI, tokenRequest, sa.Name)
	if err != nil {
		return err
	}

	// Update the empty secret with token data.
	secret.Data["token"] = []byte(tokenResp.Status.Token)
	if err := resource.WriteSecretToK8s(c.managementK8sCLI, secret); err != nil {
		return err
	}

	return nil
}
