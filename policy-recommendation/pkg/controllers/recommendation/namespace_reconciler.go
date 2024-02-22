// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package recommendation_controller

import (
	"context"
	"sync"

	log "github.com/sirupsen/logrus"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	rcache "github.com/projectcalico/calico/kube-controllers/pkg/cache"
	lmak8s "github.com/projectcalico/calico/lma/pkg/k8s"
	"github.com/projectcalico/calico/policy-recommendation/pkg/controllers/controller"
	recengine "github.com/projectcalico/calico/policy-recommendation/pkg/engine"
)

type namespaceReconciler struct {
	// ctx is the context.
	ctx context.Context

	// cache is the cache that is used to store the recommendations.
	cache rcache.ResourceCache

	// clientSet is the client set that is used to interact with the Calico or Kubernetes API.
	clientSet lmak8s.ClientSet

	// engine is the recommendation engine.
	engine *recengine.RecommendationEngine

	// clog is the logger for the controller.
	clog *log.Entry

	// mutex is used to synchronize access to the cache.
	mutex sync.Mutex
}

func newNamespaceReconciler(
	ctx context.Context, clientSet lmak8s.ClientSet, cache rcache.ResourceCache, engine *recengine.RecommendationEngine, clog *log.Entry,
) controller.Reconciler {
	return &namespaceReconciler{
		ctx:       ctx,
		cache:     cache,
		clientSet: clientSet,
		engine:    engine,
		clog:      clog,
	}
}

// Reconcile will be triggered by any changes performed on the designated namespace.
func (r *namespaceReconciler) Reconcile(key types.NamespacedName) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// isDelete returns true if the namespace was deleted from the datastore.
	isDelete := func(key types.NamespacedName) bool {
		_, err := r.clientSet.CoreV1().Namespaces().Get(r.ctx, key.Name, metav1.GetOptions{})
		if err != nil && kerrors.IsNotFound(err) {
			return true
		}

		return false
	}

	// Adds a namespace to the set for further processing by the engine. Only add selected namespaces.
	addNamespace := func(namespace string) {
		if _, ok := r.cache.Get(namespace); !ok {
			if r.engine == nil {
				r.clog.Debug("Engine is empty, will not add namespace for processing")
				return
			}
			if !r.engine.ProcessedNamespaces.Contains(namespace) {
				selector := r.engine.GetScope().GetSelector()
				if selector.String() == "" || selector.Evaluate(map[string]string{v3.LabelName: namespace}) {
					// Add the namespace to the engine for processing, if the recommendation selector
					// evaluates to true or the selector is empty.
					r.clog.WithField("namespace", namespace).Info("Adding namespace for recommendation processing")
					r.engine.ProcessedNamespaces.Add(namespace)
				}
			}
		}
	}

	// Removes every rule from the staged network policy referencing the passed in namespace.
	removeRulesReferencingDeletedNamespace := func(snp *v3.StagedNetworkPolicy, namespace string) {
		r.clog.Debugf("Remoe all references to namespace: %s, from staged network policy: %s", namespace, snp.Name)
		ingress := []v3.Rule{}
		for i, rule := range snp.Spec.Ingress {
			if rule.Source.NamespaceSelector != namespace {
				ingress = append(ingress, snp.Spec.Ingress[i])
			}
		}
		snp.Spec.Ingress = ingress

		egress := []v3.Rule{}
		for i, rule := range snp.Spec.Egress {
			if rule.Destination.NamespaceSelector != namespace {
				egress = append(egress, snp.Spec.Egress[i])
			}
		}
		snp.Spec.Egress = egress
	}

	// Removes items from further processing, and any reference of the namespace in the other policy
	// rules.
	removeNamespace := func(namespace string) {
		if _, ok := r.cache.Get(namespace); ok {
			// Remove the namespace from the engine runnable items, and the cache.
			r.clog.WithField("namespace", key).Debug("Removing namespace from cache")
			r.engine.ProcessedNamespaces.Discard(namespace)
			r.cache.Delete(namespace)

			// Remove every rule referencing the deleted namespace from the cache items.
			for _, key := range r.cache.ListKeys() {
				if val, ok := r.cache.Get(key); ok {
					snp := val.(v3.StagedNetworkPolicy)
					removeRulesReferencingDeletedNamespace(&snp, namespace)
					r.cache.Set(key, snp)
				}
			}
		}
	}

	if !isDelete(key) {
		// Add the namespace to the engine for processing. The engine holds a set of namespaces.
		addNamespace(key.Name)
	} else {
		// Remove the namespace from the engine processing items, and the cache.
		removeNamespace(key.Name)
	}

	return nil
}
