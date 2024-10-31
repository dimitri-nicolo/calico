// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package recommendation_controller

import (
	"context"
	"sync"

	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	rcache "github.com/projectcalico/calico/kube-controllers/pkg/cache"
	lmak8s "github.com/projectcalico/calico/lma/pkg/k8s"
	calres "github.com/projectcalico/calico/policy-recommendation/pkg/calico-resources"
	"github.com/projectcalico/calico/policy-recommendation/pkg/controllers/controller"
)

type networkPolicyReconciler struct {
	// ctx is the context that is used to cancel the internal goroutines.
	ctx context.Context

	// cache is the cache that is used to store the recommendations.
	cache rcache.ResourceCache

	// clientSet is the calico V3 clientSet that is used to interact with the API.
	clientSet lmak8s.ClientSet

	// clog is the logger for the controller.
	clog *log.Entry

	// mutex is used to synchronize access to the cache.
	mutex sync.Mutex
}

func newNetworkPolicyReconciler(
	ctx context.Context, clientSet lmak8s.ClientSet, cache rcache.ResourceCache, clog *log.Entry,
) controller.Reconciler {
	return &networkPolicyReconciler{
		ctx:       ctx,
		cache:     cache,
		clientSet: clientSet,
		clog:      clog,
	}
}

func (r *networkPolicyReconciler) Reconcile(key types.NamespacedName) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	ignoreRecommendation := func(snp *v3.StagedNetworkPolicy, namespace string) {
		// Ignoring the recommendation is done by setting the StagedAction and label to ignore.
		snp.Labels[calres.StagedActionKey] = string(v3.StagedActionIgnore)
		snp.Spec.StagedAction = v3.StagedActionIgnore
	}

	np, err := r.clientSet.ProjectcalicoV3().NetworkPolicies(key.Namespace).Get(r.ctx, key.Name, metav1.GetOptions{})
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return err
		} else {
			return nil
		}
	}

	// When a recommendation is enforced and subsequently replaced with a NetworkPolicy,
	// ignore the recommendation from further processing. That way the user has the option to
	// reset the recommendation to learn at a later time.
	ns := np.Namespace
	if item, ok := r.cache.Get(ns); ok {
		snp := item.(v3.StagedNetworkPolicy)
		ignoreRecommendation(&snp, ns)
		r.cache.Set(ns, snp)
		r.clog.WithField("namespace", snp.Namespace).WithField("recommendation", snp.Name).Info("ignoring recommendation from further processing")
	}

	return nil
}
