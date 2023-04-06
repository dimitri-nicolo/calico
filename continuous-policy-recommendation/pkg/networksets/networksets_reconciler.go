// Copyright (c) 2022 Tigera Inc. All rights reserved.
package networksets

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

type networksetReconciler struct {
	calico        calicoclient.ProjectcalicoV3Interface
	resourceCache cache.ObjectCache[*v3.NetworkSet]
}

// Reconcile listens to Kubernetes events (create, update, delete) for the all
// NetworkSets in the cluster and updates the
func (nr *networksetReconciler) Reconcile(namespacedName types.NamespacedName) error {
	ns, err := nr.calico.NetworkSets(namespacedName.Namespace).Get(
		context.Background(),
		namespacedName.Name,
		metav1.GetOptions{},
	)

	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	cachedNS := nr.resourceCache.Get(namespacedName.Name)

	// Handle caching the new NetworkSet so it can be cross referenced in the enxt tick
	// of the recommendation engine
	if cachedNS == nil || reflect.ValueOf(cachedNS).IsNil() {
		log.Debugf("Storing new NetworkSet: %s in the cache", namespacedName.Name)
		nr.resourceCache.Set(namespacedName.Name, ns)

		return nil
	}

	// Handle untracking the deleted NetworkSet
	if k8serrors.IsNotFound(err) {
		log.Infof("Removing the NetworkSet: %s from the cache", cachedNS.Name)
		nr.resourceCache.Delete(namespacedName.Name)

		return nil
	}

	// Handle update meta values if specs are the same
	if !resources.DeepEqual(ns.Spec, cachedNS.Spec) {
		log.Debugf("Storing updated NetworkSet: %s in the cache", namespacedName.Name)

		// updated meta fields update controller's cached entry
		nr.resourceCache.Set(ns.Name, ns)
		return nil
	}

	return nil
}

// Close cleans all NetworkSets in the cluster, tracked by the cache
func (nr *networksetReconciler) Close() {
	networkSets := nr.resourceCache.GetAll()
	for _, ns := range networkSets {
		err := nr.calico.NetworkSets(ns.Namespace).Delete(
			context.Background(),
			ns.Name,
			metav1.DeleteOptions{},
		)

		if err != nil {
			log.WithError(err).Warnf("Error deleting NetworkSet: %s continuing as to not block Close()", ns.Name)
		}
	}
}
