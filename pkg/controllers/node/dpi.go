// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package node

import (
	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	tigeraapi "github.com/tigera/api/pkg/client/clientset_generated/clientset"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

// Watcher is the interface used to watch resource type and get a list of that existing resource.
type Watcher interface {
	Run(stopCh chan struct{})
	GetExistingResources() []interface{}
}

type dpiWatcher struct {
	k8sClientset    tigeraapi.Interface
	indexerInformer cache.SharedIndexInformer
}

func NewDPIWatcher(k8sClientset tigeraapi.Interface) Watcher {
	lw := cache.NewListWatchFromClient(k8sClientset.ProjectcalicoV3().RESTClient(), "deeppacketinspections", "",
		fields.Everything())
	return &dpiWatcher{
		k8sClientset:    k8sClientset,
		indexerInformer: cache.NewSharedIndexInformer(lw, &v3.DeepPacketInspection{}, 0, cache.Indexers{}),
	}
}

// Run creates and runs a NewSharedIndexInformer for DeepPacketInspection resource.
func (w *dpiWatcher) Run(stopCh chan struct{}) {
	log.Info("Starting DPI watcher")
	go w.indexerInformer.Run(stopCh)
}

// GetExistingResources returns list of resources from Indexer's store.
func (w *dpiWatcher) GetExistingResources() []interface{} {
	return w.indexerInformer.GetIndexer().List()
}
