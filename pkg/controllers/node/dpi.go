// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package node

import (
	"context"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/set"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/cache"
)

type dpiStatusUpdater struct {
	calicoClient     client.Interface
	informer         cache.SharedIndexInformer
	getCachedNodesFn func(informer cache.SharedIndexInformer) []interface{}
}

func NewDPIStatusUpdater(calicoClient client.Interface, informer cache.SharedIndexInformer, getNodeFn func(informer cache.SharedIndexInformer) []interface{}) ResourceStatusUpdater {
	return &dpiStatusUpdater{
		calicoClient:     calicoClient,
		informer:         informer,
		getCachedNodesFn: getNodeFn,
	}
}

func (u *dpiStatusUpdater) Run(stopCh chan struct{}) {
	go u.informer.Run(stopCh)
}

// CleanDeletedNodesFromStatus gets the list of cached DeepPacketInspection resources, and removes the status related to nodes
// no longer in the cached list of nodes.
func (u *dpiStatusUpdater) CleanDeletedNodesFromStatus(cachedNodes set.Set) {
	log.Debugf("Cleaning the deleted nodes status from DeepPacketInspection resources.")
	dpiResources := u.getCachedNodesFn(u.informer)
	if len(dpiResources) == 0 {
		// No dpi resource to cleanup
		return
	}

	for _, res := range dpiResources {
		if dpi, ok := res.(*v3.DeepPacketInspection); ok {
			if err := u.removeDeletedNodes(context.Background(), dpi.DeepCopy(), cachedNodes); err != nil {
				log.WithFields(log.Fields{"deepPacketInspection": dpi.Name, "namespace": dpi.Namespace}).
					WithError(err).Error("Failed to update status, will retry later.")
			}
		} else {
			log.Errorf("Failed to get DeepPacketInspection resource from #%v", res)
		}
	}
}

// removeDeletedNodes updates status of the DeepPacketInspection resource by removing fields
// from status corresponding to the node not in cached node.
func (u *dpiStatusUpdater) removeDeletedNodes(ctx context.Context, dpi *v3.DeepPacketInspection, existingNodes set.Set) error {
	var err error
	if len(dpi.Status.Nodes) == 0 {
		return nil
	}

	index := 0
	var availableNodes []v3.DPINode
	for _, dpiNode := range dpi.Status.Nodes {
		if existingNodes.Contains(dpiNode.Node) {
			availableNodes[index] = dpiNode
			index++
		}
	}
	if index != len(dpi.Status.Nodes) {
		dpi.Status.Nodes = availableNodes
		_, err = u.calicoClient.DeepPacketInspections().UpdateStatus(ctx, dpi, options.SetOptions{})
	}

	return err
}
