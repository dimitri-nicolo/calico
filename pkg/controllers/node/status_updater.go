// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package node

import (
	"time"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"

	tigeraapi "github.com/tigera/api/pkg/client/clientset_generated/clientset"

	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/set"

	"k8s.io/client-go/util/workqueue"
)

type ResourceStatusUpdater interface {
	Run(stopCh chan struct{})
	CleanDeletedNodesFromStatus(cachedNodes set.Set)
}

func GetCachedNodes(informer cache.SharedIndexInformer) []interface{} {
	return informer.GetIndexer().List()
}

// NewStatusUpdateController creates a new controller responsible for cleaning the node
// specific status field in resource when corresponding node is deleted.
func NewStatusUpdateController(calicoClient client.Interface, calicoV3Client tigeraapi.Interface, nodeCache func() []string) *statusUpdateController {
	dpiListWatch := cache.NewListWatchFromClient(calicoV3Client.ProjectcalicoV3().RESTClient(), "packetcaptures", "",
		fields.Everything())
	pcapListWatch := cache.NewListWatchFromClient(calicoV3Client.ProjectcalicoV3().RESTClient(), "packetcaptures", "",
		fields.Everything())

	dpiInformer := cache.NewSharedIndexInformer(dpiListWatch, &v3.PacketCapture{}, 0, cache.Indexers{})
	pcapInformer := cache.NewSharedIndexInformer(pcapListWatch, &v3.PacketCapture{}, 0, cache.Indexers{})

	dpiStatusUpdater := NewDPIStatusUpdater(calicoClient, dpiInformer, GetCachedNodes)
	pcapStatusUpdater := NewPacketCaptureStatusUpdater(calicoClient, pcapInformer, GetCachedNodes)
	statusUpdaters := []ResourceStatusUpdater{dpiStatusUpdater, pcapStatusUpdater}
	return &statusUpdateController{
		rl:               workqueue.DefaultControllerRateLimiter(),
		calicoClient:     calicoClient,
		calicoV3Client:   calicoV3Client,
		statusUpdaters:   statusUpdaters,
		nodeCacheFn:      nodeCache,
		reconcilerPeriod: time.Minute * 5,
	}
}

type statusUpdateController struct {
	rl               workqueue.RateLimiter
	calicoClient     client.Interface
	calicoV3Client   tigeraapi.Interface
	statusUpdaters   []ResourceStatusUpdater
	nodeCacheFn      func() []string
	reconcilerPeriod time.Duration
}

func (c *statusUpdateController) Start(stop chan struct{}) {
	for _, s := range c.statusUpdaters {
		s.Run(stop)
	}
	go c.acceptScheduledRequest(stop)
}

// acceptScheduledRequest cleans the deleted node's status in custom resource periodically.
func (c *statusUpdateController) acceptScheduledRequest(stopCh <-chan struct{}) {
	t := time.NewTicker(c.reconcilerPeriod)
	for {
		select {
		case <-t.C:
			for _, s := range c.statusUpdaters {
				s.CleanDeletedNodesFromStatus(set.FromArray(c.nodeCacheFn()))
			}
		case <-stopCh:
			return
		}
	}
}
