// Copyright (c) 2022 Tigera Inc. All rights reserved.

package stagednetworkpolicies

import (
	"context"
	"reflect"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"

	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/cache"
	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/resources"

	log "github.com/sirupsen/logrus"
)

type stagednetworkpoliciesReconciler struct {
	calico        calicoclient.ProjectcalicoV3Interface
	resourceCache cache.ObjectCache[*v3.StagedNetworkPolicy]
}

func (sr *stagednetworkpoliciesReconciler) Reconcile(namespacedName types.NamespacedName) error {
	snp, err := sr.calico.StagedNetworkPolicies(namespacedName.Namespace).Get(
		context.Background(),
		namespacedName.Name,
		metav1.GetOptions{},
	)

	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	cachedSNP := sr.resourceCache.Get(namespacedName.Name)

	if cachedSNP == nil || reflect.ValueOf(cachedSNP).IsNil() {
		log.Debugf("Ignorining untracked StagedNetworkPolicy %s from Policy Recommendation", namespacedName.Name)
		return nil
	}

	// Handle Restoring deleted StagedNetworkPolicy managed by policy recommendation
	if k8serrors.IsNotFound(err) {
		log.Infof("Restoring StagedNetworkPolicy %s", cachedSNP.Name)

		createdSNP, err := sr.calico.StagedNetworkPolicies(cachedSNP.Namespace).Create(
			context.Background(),
			cachedSNP,
			metav1.CreateOptions{},
		)

		if err != nil {
			log.WithError(err).Errorf("unable to restore StagedNetworkPolicy %s that has been deleted",
				cachedSNP.Name)
			return err
		}

		sr.resourceCache.Set(createdSNP.Name, createdSNP)

		return nil
	}

	// Handle update meta values if specs are the same
	if resources.DeepEqual(snp.Spec, cachedSNP.Spec) {
		// updated meta fields update controller's cached entry
		sr.resourceCache.Set(snp.Name, snp)
		return nil
	}

	// Handle Restoring altered StorageNetworkPolicy
	log.Infof("Updating StagedNetworkPolicy %s", cachedSNP.Name)
	updatedStagedNetworkPolicies, err := sr.calico.StagedNetworkPolicies(cachedSNP.Namespace).Update(
		context.Background(),
		cachedSNP,
		metav1.UpdateOptions{},
	)

	if err != nil {
		return err
	}

	sr.resourceCache.Set(cachedSNP.Name, updatedStagedNetworkPolicies)

	return nil
}

func (sr *stagednetworkpoliciesReconciler) Close() {
	snps := sr.resourceCache.GetAll()
	for _, snp := range snps {
		err := sr.calico.StagedNetworkPolicies(snp.Namespace).Delete(
			context.Background(),
			snp.Name,
			metav1.DeleteOptions{},
		)

		if err != nil {
			log.WithError(err).Warnf("Error deleting StagedNetworkPolicy: %s continuing as to not block close()", snp.Name)
		}
	}
}
