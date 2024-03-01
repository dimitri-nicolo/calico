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

	if !isDelete(key) {
		// Add the namespace to the engine for processing. The engine holds a set of namespaces.
		r.addNamespace(key.Name)
	} else {
		// Remove the namespace from the engine processing items, and the cache.
		r.removeNamespace(key.Name)
	}

	return nil
}

// addNamespace adds a namespace for tracking, and adds it to the filtered namespaces for
// processing, if it is selected by the recommendation selector.
func (r *namespaceReconciler) addNamespace(namespace string) {
	if !r.engine.Namespaces.Contains(namespace) {
		// Keep track of every namespace.
		r.engine.Namespaces.Add(namespace)
		r.clog.WithField("namespace", namespace).Debug("Added namespace for tracking.")
		// Only add a namespace for processing if it is selected by the recommendation selector.
		selector := r.engine.GetScope().GetSelector()
		if selector.String() == "" || selector.Evaluate(map[string]string{v3.LabelName: namespace}) {
			if !r.engine.FilteredNamespaces.Contains(namespace) {
				r.engine.FilteredNamespaces.Add(namespace)
				r.clog.WithField("namespace", namespace).Info("Added namespace for processing.")
			}
		}
	}
}

// removeNamespace removes items from further processing, and any reference of the namespace in the other policy
// rules.
func (r *namespaceReconciler) removeNamespace(namespace string) {
	// Remove the namespace from the engine runnable items, and the cache.
	r.engine.Namespaces.Discard(namespace)
	if r.engine.FilteredNamespaces.Contains(namespace) {
		r.engine.FilteredNamespaces.Discard(namespace)
	}
	if _, ok := r.cache.Get(namespace); ok {
		r.cache.Delete(namespace)
	}
	r.clog.WithField("namespace", namespace).Debug("Removed namespace from further processing.")

	// Remove every rule referencing the deleted namespace from the cache items.
	for _, key := range r.cache.ListKeys() {
		if val, ok := r.cache.Get(key); ok {
			snp := val.(v3.StagedNetworkPolicy)
			r.removeRulesReferencingDeletedNamespace(&snp, namespace)
			r.cache.Set(key, snp)
		}
	}
}

// removeRulesReferencingDeletedNamespace removes every rule from the staged network policy referencing the passed in namespace.
func (r *namespaceReconciler) removeRulesReferencingDeletedNamespace(snp *v3.StagedNetworkPolicy, namespace string) {
	r.clog.Debugf("Remove all references to namespace: %s, from staged network policy: %s", namespace, snp.Name)
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
