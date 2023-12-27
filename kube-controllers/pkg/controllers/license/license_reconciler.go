// Copyright (c) 2021-2023 Tigera, Inc. All rights reserved.

package license

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	tigeraapi "github.com/tigera/api/pkg/client/clientset_generated/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/projectcalico/calico/kube-controllers/pkg/resource"
)

type reconciler struct {
	clusterName                 string
	managedCalicoCLI            tigeraapi.Interface
	managementCalicoCLI         tigeraapi.Interface
	managedLicenseUIDVersion    string
	managementLicenseUIDVersion string
}

func NewLicenseReconciler(managedCalicoCLI tigeraapi.Interface, managementCalicoCLI tigeraapi.Interface, clusterName string) *reconciler {
	return &reconciler{
		clusterName:         clusterName,
		managedCalicoCLI:    managedCalicoCLI,
		managementCalicoCLI: managementCalicoCLI,
	}
}

// Reconcile will be triggered by any chnages peformed on: the designated license to be copied over to the managed cluster
// that is created with the management cluster (this can also be the default license if configured accordingly), default
// license created within the managed cluster
func (c *reconciler) Reconcile(name types.NamespacedName) error {
	reqLogger := log.WithFields(map[string]interface{}{
		"cluster": c.clusterName,
		"key":     name,
	})
	reqLogger.Info("Reconciling License")

	if err := c.reconcileManagedLicense(); err != nil {
		return err
	}

	reqLogger.Info("Finished reconciling License")

	return nil
}

func (c *reconciler) reconcileManagedLicense() error {
	logger := log.WithField("cluster", c.clusterName)

	// Read and calculate hash for the designated license to be copied over to the managed clusters from the management clusters
	license, err := c.managementCalicoCLI.ProjectcalicoV3().LicenseKeys().Get(context.Background(), resource.LicenseName, metav1.GetOptions{})
	if err != nil {
		logger.WithError(err).Error("Failed to read license for management cluster")
		return err
	}
	managementUIDVersion := uidVersion(license)

	// Read and calculate hash for the default license rrom the managed clusters
	managedLicense, err := c.managedCalicoCLI.ProjectcalicoV3().LicenseKeys().Get(context.Background(), resource.LicenseName, metav1.GetOptions{})
	// Ignore license not found on the managed cluster (most likely license needs to be copied over)
	if err != nil && !errors.IsNotFound(err) {
		logger.WithError(err).Error("Failed to read license for managed cluster")
		return err
	}

	var managedUIDVersion string
	if managedLicense != nil {
		managedUIDVersion = uidVersion(managedLicense)
	}

	// Designated management license has changed or the license from the managed cluster has changed
	if managementUIDVersion != c.managementLicenseUIDVersion || managedUIDVersion != c.managedLicenseUIDVersion {
		logger.Info("Copy license to managed cluster")
		copy := resource.CopyLicenseKey(license)
		copy.Name = resource.LicenseName
		if err := resource.WriteLicenseKeyToK8s(c.managedCalicoCLI, copy); err != nil {
			logger.WithError(err).Error("Failed to write license to managed cluster")
			return err
		}
		c.managementLicenseUIDVersion = managementUIDVersion
		c.managedLicenseUIDVersion = managementUIDVersion
	}

	return nil
}

func uidVersion(license *v3.LicenseKey) string {
	return fmt.Sprintf("%s-%s", license.UID, license.ResourceVersion)
}
