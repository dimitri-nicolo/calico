// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This file was originally copied from confd-private:pkg/backends/calico/secret_watcher.go
package remotecluster

import (
	"context"
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type secretWatchData struct {
	// The channel that we should write to when we no longer want this watch.
	stopCh chan struct{}

	// Stale marker.
	stale bool

	// Secret value.
	secret *v1.Secret
}

type secretKey struct {
	namespace string
	name      string
}

type SecretUpdateReceiver interface {
	OnSecretUpdated(namespace, name string)
}

type secretWatcher struct {
	secretReceiver SecretUpdateReceiver
	k8sClientset   *kubernetes.Clientset
	mutex          sync.Mutex
	watches        map[secretKey]*secretWatchData
}

func NewSecretWatcher(sur SecretUpdateReceiver, k8sClient *kubernetes.Clientset) *secretWatcher {
	sw := &secretWatcher{
		secretReceiver: sur,
		watches:        make(map[secretKey]*secretWatchData),
	}
	if k8sClient != nil {
		sw.k8sClientset = k8sClient
		return sw
	}
	log.Infof("No kubernetes client available, secrets will not be available for RemoteClusterConfiguration")

	return nil
}

func (sw *secretWatcher) MarkStale() {
	sw.mutex.Lock()
	defer sw.mutex.Unlock()

	for _, watchData := range sw.watches {
		watchData.stale = true
	}
}

func (sw *secretWatcher) ensureWatchingSecret(sk secretKey) {
	if _, ok := sw.watches[sk]; ok {
		log.Debugf("Already watching secret '%v' (namespace %v)", sk.name, sk.namespace)
	} else {
		log.Debugf("Start a watch for secret '%v' (namespace %v)", sk.name, sk.namespace)
		// We're not watching this secret yet, so start a watch for it.
		watcher := cache.NewListWatchFromClient(sw.k8sClientset.CoreV1().RESTClient(), "secrets", sk.namespace, fields.OneTermEqualSelector("metadata.name", sk.name))
		_, controller := cache.NewInformer(watcher, &v1.Secret{}, 0, sw)
		sw.watches[sk] = &secretWatchData{stopCh: make(chan struct{})}
		go controller.Run(sw.watches[sk].stopCh)
		log.Debugf("Controller for secret '%v' (namespace %v) is now running", sk.name, sk.namespace)
	}
}

func (sw *secretWatcher) GetSecretData(namespace, name string) (map[string][]byte, error) {
	sw.mutex.Lock()
	defer sw.mutex.Unlock()
	log.Debugf("Get secret in namespace '%v' for name '%v'", namespace, name)

	sk := secretKey{namespace, name}

	if _, ok := sw.watches[sk]; ok {
		// Get and decode the Secret of interest.
		if sw.watches[sk].secret == nil {
			return nil, nil
		}

		// Mark it as still in use.
		sw.watches[sk].stale = false

		return sw.watches[sk].secret.Data, nil
	} else {
		// There is no watch running for the secret so directly query it to start with.
		secret, err := sw.k8sClientset.CoreV1().Secrets(namespace).Get(context.Background(), name, metav1.GetOptions{})
		// Ensure that we're watching this secret.
		sw.ensureWatchingSecret(sk)

		// Mark it as still in use.
		sw.watches[sk].stale = false

		if err != nil && !kerrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to retrieve secret %s", err)
		}

		if secret == nil {
			return nil, nil
		}

		return secret.Data, nil
	}
}

func (sw *secretWatcher) IgnoreSecret(namespace, name string) {
	sk := secretKey{namespace, name}
	sw.deleteSecretWatcher(sk)
}

func (sw *secretWatcher) SweepStale() {
	sw.mutex.Lock()
	defer sw.mutex.Unlock()

	for sk, watchData := range sw.watches {
		if watchData.stale {
			close(watchData.stopCh)
			delete(sw.watches, sk)
		}
	}
}

func (sw *secretWatcher) OnAdd(obj interface{}) {
	log.Debug("Secret added")
	s := obj.(*v1.Secret)
	sw.updateSecret(s)
	sw.secretReceiver.OnSecretUpdated(s.Namespace, s.Name)
}

func (sw *secretWatcher) OnUpdate(oldObj, newObj interface{}) {
	log.Debug("Secret updated")
	s := newObj.(*v1.Secret)
	sw.updateSecret(s)
	sw.secretReceiver.OnSecretUpdated(s.Namespace, s.Name)
}

func (sw *secretWatcher) OnDelete(obj interface{}) {
	log.Debug("Secret deleted")
	s := obj.(*v1.Secret)
	sk := secretKey{s.Namespace, s.Name}
	sw.deleteSecret(sk)
	sw.secretReceiver.OnSecretUpdated(s.Namespace, s.Name)
}

func (sw *secretWatcher) updateSecret(secret *v1.Secret) {
	sk := secretKey{secret.Namespace, secret.Name}
	sw.mutex.Lock()
	defer sw.mutex.Unlock()
	if _, ok := sw.watches[sk]; ok {
		sw.watches[sk].secret = secret
	}
}

func (sw *secretWatcher) deleteSecret(sk secretKey) {
	sw.mutex.Lock()
	defer sw.mutex.Unlock()
	if _, ok := sw.watches[sk]; ok {
		sw.watches[sk].secret = nil
	}
}

func (sw *secretWatcher) deleteSecretWatcher(sk secretKey) {
	sw.mutex.Lock()
	defer sw.mutex.Unlock()
	if _, ok := sw.watches[sk]; ok {
		if sw.watches[sk].stopCh != nil {
			close(sw.watches[sk].stopCh)
			sw.watches[sk].stopCh = nil
		}
		delete(sw.watches, sk)
	}
}
