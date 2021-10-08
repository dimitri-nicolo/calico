// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package node

import (
	"context"
	"time"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	tigeraapi "github.com/tigera/api/pkg/client/clientset_generated/clientset"

	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/set"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/util/workqueue"
)

// NewStatusUpdateController creates a new controller responsible for cleaning the node
// specific status field in resource when corresponding node is deleted.
func NewStatusUpdateController(calicoClient client.Interface, calicoV3Client tigeraapi.Interface, nodeCache func() []string) *statusUpdateController {
	dpiWch := NewDPIWatcher(calicoV3Client)
	return &statusUpdateController{
		rl:               workqueue.DefaultControllerRateLimiter(),
		calicoClient:     calicoClient,
		calicoV3Client:   calicoV3Client,
		dpiWch:           dpiWch,
		nodeCacheFn:      nodeCache,
		reconcilerPeriod: time.Minute * 5,
	}
}

type statusUpdateController struct {
	rl               workqueue.RateLimiter
	calicoClient     client.Interface
	calicoV3Client   tigeraapi.Interface
	dpiWch           Watcher
	nodeCacheFn      func() []string
	reconcilerPeriod time.Duration
}

func (c *statusUpdateController) Start(stop chan struct{}) {
	go c.dpiWch.Run(stop)
	go c.acceptScheduledRequest(stop)
}

// acceptScheduledRequest cleans the deleted node's status in custom resource periodically.
func (c *statusUpdateController) acceptScheduledRequest(stopCh <-chan struct{}) {
	t := time.NewTicker(c.reconcilerPeriod)
	for {
		select {
		case <-t.C:
			c.cleanDeletedNodeStatus()
		case <-stopCh:
			return
		}
	}
}

// cleanDeletedNodeStatus gets the list of cached DeepPacketInspection resources, and removes the status related to nodes
// no longer in the cached list of nodes.
func (c *statusUpdateController) cleanDeletedNodeStatus() {
	log.Debugf("Cleaning the deleted nodes status from DeepPacketInspection resources.")
	dpiResources := c.dpiWch.GetExistingResources()
	if len(dpiResources) == 0 {
		// No dpi resource to cleanup
		return
	}

	existingNodes := set.FromArray(c.nodeCacheFn())
	for _, res := range dpiResources {
		if dpi, ok := res.(*v3.DeepPacketInspection); ok {
			if err := c.updateDPIStatus(context.Background(), dpi, existingNodes); err != nil {
				log.WithFields(log.Fields{"deepPacketInspection": dpi.Name, "namespace": dpi.Namespace}).
					WithError(err).Error("Failed to update status, will retry later.")
			}
		} else {
			log.Errorf("Failed to get DeepPacketInspection resource from #%v", res)
		}
	}
}

// updateDPIStatus updates status of the DeepPacketInspection resource by removing fields
// from status corresponding to the node not in cached node.
func (c *statusUpdateController) updateDPIStatus(ctx context.Context, dpi *v3.DeepPacketInspection, existingNodes set.Set) error {
	var err error
	if len(dpi.Status.Nodes) == 0 {
		return nil
	}
	if existingNodes.Len() == 0 {
		dpi.Status.Nodes = nil
		_, err = c.calicoClient.DeepPacketInspections().UpdateStatus(ctx, dpi, options.SetOptions{})
		return err
	}

	index := 0
	for _, dpiNode := range dpi.Status.Nodes {
		if existingNodes.Contains(dpiNode.Node) {
			dpi.Status.Nodes[index] = dpiNode
			index++
		}
	}
	if index != len(dpi.Status.Nodes) {
		dpi.Status.Nodes = dpi.Status.Nodes[:index]
		_, err = c.calicoClient.DeepPacketInspections().UpdateStatus(ctx, dpi, options.SetOptions{})
	}

	return err
}
