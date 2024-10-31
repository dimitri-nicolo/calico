// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package recommendation_controller

import (
	"context"
	"reflect"
	"sync"

	"github.com/google/go-cmp/cmp"
	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	rcache "github.com/projectcalico/calico/kube-controllers/pkg/cache"
	lmak8s "github.com/projectcalico/calico/lma/pkg/k8s"
)

type stagedNetworkPolicyReconciler struct {
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

func newStagedNetworkPolicyReconciler(
	ctx context.Context, clientSet lmak8s.ClientSet, cache rcache.ResourceCache, clog *log.Entry,
) *stagedNetworkPolicyReconciler {
	return &stagedNetworkPolicyReconciler{
		ctx:       ctx,
		cache:     cache,
		clientSet: clientSet,
		clog:      clog,
	}
}

// Reconcile will be triggered by any changes performed on StagedNetworkPolicy resources. If there
// is an update to the store then update the cache.
func (r *stagedNetworkPolicyReconciler) Reconcile(key types.NamespacedName) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	addKindAndAPIVersion := func(snp *v3.StagedNetworkPolicy) {
		// Add the kind and API version to the resource.
		snp.APIVersion = "projectcalico.org/v3"
		snp.Kind = v3.KindStagedNetworkPolicy
	}

	storeSnp, err := r.clientSet.ProjectcalicoV3().StagedNetworkPolicies(key.Namespace).Get(r.ctx, key.Name, metav1.GetOptions{})
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return err
		} else {
			return nil
		}
	}

	addKindAndAPIVersion(storeSnp)
	ns := storeSnp.Namespace
	if item, ok := r.cache.Get(ns); ok {
		cacheSnp := item.(v3.StagedNetworkPolicy)
		if !reflect.DeepEqual(*storeSnp, cacheSnp) {
			r.cache.Set(ns, *storeSnp)
			r.clog.WithField("namespace", ns).WithField("recommendation", storeSnp.Name).Info("Updated cache item, and queued it for update")
			r.clog.Debug(cmp.Diff(*storeSnp, cacheSnp))
		}
	}

	return nil
}
