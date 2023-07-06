// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package calicoresources

import (
	"context"

	log "github.com/sirupsen/logrus"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
)

// NewTier returns a pointer to a calico v3 tier resource.
func newTier(name string, order *float64) *v3.Tier {
	return &v3.Tier{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v3.TierSpec{
			Order: order,
		},
	}
}

// DeleteTier deletes a tier in the datasource.
func DeleteTier(ctx context.Context, calico calicoclient.ProjectcalicoV3Interface, name string) error {
	if _, err := calico.Tiers().Get(ctx, name, metav1.GetOptions{}); err != nil {
		if !kerrors.IsNotFound(err) {
			log.WithError(err).Warnf("Error getting tier: %s, cannot delete tier", name)
			return err
		}
		log.Debugf("No tier found with name %s to delete.", name)
		return nil
	}

	if err := calico.Tiers().Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		log.WithError(err).Warnf("Error deleting tier: %s", name)
		return err
	}

	return nil
}

// MaybeCreateTier attempts to create a new tier if does not already exist.
func MaybeCreateTier(ctx context.Context, calico calicoclient.ProjectcalicoV3Interface, name string, order *float64) error {
	if _, err := calico.Tiers().Get(ctx, name, metav1.GetOptions{}); err != nil {
		if kerrors.IsNotFound(err) {
			// Tier doesn't exist, create a new one
			tier := newTier(name, order)

			log.WithField("key", name).Info("Creating new tier")
			if _, err = calico.Tiers().Create(ctx, tier, metav1.CreateOptions{}); err != nil {
				log.WithField("key", name).WithError(err).Error("failed to create tier")
				return err
			}
		} else {
			log.WithField("key", name).WithError(err).Error("failed to get tier")
			return err
		}
	}

	return nil
}
