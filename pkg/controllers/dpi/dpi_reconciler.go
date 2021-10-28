// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package dpi

import (
	"context"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	log "github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	tigeraapi "github.com/tigera/api/pkg/client/clientset_generated/clientset"
)

type reconciler struct {
	cacheDPI     map[string]*v3.DeepPacketInspection
	k8sClientset tigeraapi.Interface
}

func NewReconciler(k8sClientset tigeraapi.Interface) *reconciler {
	return &reconciler{
		k8sClientset: k8sClientset,
		cacheDPI:     make(map[string]*v3.DeepPacketInspection),
	}
}

// Reconcile will be triggered when DeepPacketInspection resources is either created or deleted.
// It caches the DeepPacketInspection resource.
func (c *reconciler) Reconcile(name types.NamespacedName) error {
	log.Infof("Reconciling DPI resource %s", name.String())
	dpi, err := c.k8sClientset.ProjectcalicoV3().DeepPacketInspections(name.Namespace).Get(context.Background(), name.Name, metav1.GetOptions{})
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return err
		}
		delete(c.cacheDPI, name.String())
		return nil
	}
	c.cacheDPI[name.String()] = dpi
	return nil
}

// GetCachedDPIs returns list of cached DeepPacketInspection resources.
func (c *reconciler) GetCachedDPIs() []*v3.DeepPacketInspection {
	res := make([]*v3.DeepPacketInspection, 0, len(c.cacheDPI))
	for _, v := range c.cacheDPI {
		res = append(res, v)
	}
	return res
}
