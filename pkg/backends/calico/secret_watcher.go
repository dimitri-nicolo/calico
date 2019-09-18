// Copyright (c) 2019 Tigera, Inc. All rights reserved.
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
package calico

import (
	"encoding/base64"
	"fmt"
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

type secretWatchData struct {
	// The channel that we should write to when we no longer want this watch.
	stopCh chan struct{}

	// Stale marker.
	stale bool

	// Secret value.
	secret *v1.Secret
}

type secretWatcher struct {
	client       *client
	k8sClientset *kubernetes.Clientset
	mutex        sync.Mutex
	watches      map[string]*secretWatchData
}

func NewSecretWatcher(c *client) (*secretWatcher, error) {
	sw := &secretWatcher{
		client:  c,
		watches: make(map[string]*secretWatchData),
	}

	// set up k8s client
	// attempt 1: KUBECONFIG env var
	cfgFile := os.Getenv("KUBECONFIG")
	cfg, err := clientcmd.BuildConfigFromFlags("", cfgFile)
	if err != nil {
		log.WithError(err).Info("KUBECONFIG environment variable not found, attempting in-cluster")
		// attempt 2: in cluster config
		if cfg, err = rest.InClusterConfig(); err != nil {
			return nil, err
		}
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	sw.k8sClientset = clientset

	return sw, nil
}

func (sw *secretWatcher) MarkStale() {
	sw.mutex.Lock()
	defer sw.mutex.Unlock()

	for _, watchData := range sw.watches {
		watchData.stale = true
	}
}

func (sw *secretWatcher) ensureWatchingSecret(name string) {
	if _, ok := sw.watches[name]; !ok {
		// We're not watching this secret yet, so start a watch for it.
		watcher := cache.NewListWatchFromClient(sw.k8sClientset.Core().RESTClient(), "secrets", "kube-system", fields.OneTermEqualSelector("metadata.name", name))
		_, controller := cache.NewInformer(watcher, &v1.Secret{}, 0, sw)
		sw.watches[name] = &secretWatchData{stopCh: make(chan struct{})}
		go controller.Run(sw.watches[name].stopCh)

		// Block until the controller has synced.
		for !controller.HasSynced() {
			sw.snoozeWithoutMutex()
		}
	}
}

func (sw *secretWatcher) snoozeWithoutMutex() {
	sw.mutex.Unlock()
	defer sw.mutex.Lock()
	time.Sleep(100 * time.Millisecond)
}

func (sw *secretWatcher) GetSecret(name, key string) (string, error) {
	sw.mutex.Lock()
	defer sw.mutex.Unlock()

	// Ensure that we're watching this secret.
	sw.ensureWatchingSecret(name)

	// Mark it as still in use.
	sw.watches[name].stale = false

	// Get and decode the key of interest.
	if sw.watches[name].secret == nil {
		return "", fmt.Errorf("No data available for secret %v", name)
	}
	if data, ok := sw.watches[name].secret.Data[key]; ok {
		if s, err := base64.StdEncoding.DecodeString(string(data)); err != nil {
			return "", fmt.Errorf("Error decoding value for secret %v key %v: %v", name, key, err)
		} else {
			return string(s), nil
		}
	} else {
		return "", fmt.Errorf("Secret %v does not have key %v", name, key)
	}
}

func (sw *secretWatcher) SweepStale() {
	sw.mutex.Lock()
	defer sw.mutex.Unlock()

	for name, watchData := range sw.watches {
		if watchData.stale {
			var q struct{}
			watchData.stopCh <- q
			delete(sw.watches, name)
		}
	}
}

func (sw *secretWatcher) OnAdd(obj interface{}) {
	sw.updateSecret(obj.(*v1.Secret))
	sw.client.recheckPeerConfig()
}

func (sw *secretWatcher) OnUpdate(oldObj, newObj interface{}) {
	sw.updateSecret(newObj.(*v1.Secret))
	sw.client.recheckPeerConfig()
}

func (sw *secretWatcher) OnDelete(obj interface{}) {
	sw.deleteSecret(obj.(*v1.Secret))
	sw.client.recheckPeerConfig()
}

func (sw *secretWatcher) updateSecret(secret *v1.Secret) {
	sw.mutex.Lock()
	defer sw.mutex.Unlock()
	sw.watches[secret.Name].secret = secret
}

func (sw *secretWatcher) deleteSecret(secret *v1.Secret) {
	sw.mutex.Lock()
	defer sw.mutex.Unlock()
	delete(sw.watches, secret.Name)
}
