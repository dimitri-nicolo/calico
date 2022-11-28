// Copyright (c) 2022 Tigera Inc. All rights reserved.

package namespace

import (
	"context"

	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/cache"

	log "github.com/sirupsen/logrus"
)

type namespaceReconciler struct {
	kubernetes    kubernetes.Interface
	resourceCache cache.ObjectCache[*v1.Namespace]
}

func (nr *namespaceReconciler) Reconcile(namespacedName types.NamespacedName) error {
	namespace, err := nr.kubernetes.CoreV1().Namespaces().Get(context.Background(), namespacedName.Name, metav1.GetOptions{})

	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	if k8serrors.IsNotFound(err) { // deleted namespace
		// namespace is deleted, delete all policies associated with the namespace
		log.Infof("Removing all associated policies with the namespace: %s", namespacedName.Name)

		nr.resourceCache.Delete(namespacedName.Name)
		return nil
	}

	// handle create  or update
	nr.resourceCache.Set(namespace.Name, namespace)

	return nil
}

func (nr *namespaceReconciler) Close() {
}
