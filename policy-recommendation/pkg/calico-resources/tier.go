// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package calicoresources

import (
	"context"

	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rectypes "github.com/projectcalico/calico/policy-recommendation/pkg/types"
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
func DeleteTier(ctx context.Context, calico calicoclient.ProjectcalicoV3Interface, name string, clog *log.Entry) error {
	if _, err := calico.Tiers().Get(ctx, name, metav1.GetOptions{}); err != nil {
		if !kerrors.IsNotFound(err) {
			clog.WithError(err).Warnf("Error getting tier: %s, cannot delete tier", name)
			return err
		}
		clog.Debugf("No tier found with name %s to delete.", name)
		return nil
	}

	if err := calico.Tiers().Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		clog.WithError(err).Warnf("Error deleting tier: %s", name)
		return err
	}

	return nil
}

// MaybeCreateTier attempts to create a new tier if does not already exist.
func MaybeCreateTier(ctx context.Context, calico calicoclient.ProjectcalicoV3Interface, name string, order float64, clog *log.Entry) error {
	if _, err := calico.Tiers().Get(ctx, name, metav1.GetOptions{}); err != nil {
		if kerrors.IsNotFound(err) {
			// Tier doesn't exist, create a new one
			tier := newTier(name, &order)

			if _, err = calico.Tiers().Create(ctx, tier, metav1.CreateOptions{}); err != nil {
				clog.WithField("key", name).WithError(err).Error("failed to create tier")
				return err
			}
		} else {
			clog.WithField("key", name).WithError(err).Error("failed to get tier")
			return err
		}
		clog.WithField("tier", rectypes.PolicyRecommendationTierName).Info("Created recommendation tier")
	}

	return nil
}
