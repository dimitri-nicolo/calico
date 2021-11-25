// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package node

import (
	"context"

	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"k8s.io/client-go/tools/cache"

	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/set"
)

type packetCaptureStatusUpdater struct {
	calicoClient client.Interface
	informer     cache.SharedIndexInformer
	getNodeFn    func(informer cache.SharedIndexInformer) []interface{}
}

func NewPacketCaptureStatusUpdater(calicoClient client.Interface, informer cache.SharedIndexInformer, getNodeFn func(informer cache.SharedIndexInformer) []interface{}) ResourceStatusUpdater {
	return &packetCaptureStatusUpdater{
		calicoClient: calicoClient,
		informer:     informer,
		getNodeFn:    getNodeFn,
	}
}

func (u *packetCaptureStatusUpdater) Run(stopCh chan struct{}) {
	go u.informer.Run(stopCh)
}

// CleanDeletedNodesFromStatus gets the list of cached PacketCapture resources, and removes the status related to nodes
// no longer in the cached list of nodes.
func (u *packetCaptureStatusUpdater) CleanDeletedNodesFromStatus(cachedNodes set.Set) {
	log.Debugf("Cleaning the deleted nodes status from PacketCapture resources.")
	pcapResources := u.getNodeFn(u.informer)
	if len(pcapResources) == 0 {
		// No pcap resource to cleanup
		return
	}

	for _, res := range pcapResources {
		if pcap, ok := res.(*v3.PacketCapture); ok {
			if err := u.removeDeletedNodes(context.Background(), pcap.DeepCopy(), cachedNodes); err != nil {
				log.WithFields(log.Fields{"packetCapture": pcap.Name, "namespace": pcap.Namespace}).
					WithError(err).Error("Failed to update status, will retry later.")
			}
		} else {
			log.Errorf("Failed to get PacketCapture resource from #%v", res)
		}
	}
}

// removeDeletedNodes updates status of the PacketCapture resource by removing fields
// from status corresponding to the node not in cached node.
func (u *packetCaptureStatusUpdater) removeDeletedNodes(ctx context.Context, pcap *v3.PacketCapture, existingNodes set.Set) error {
	var err error
	if len(pcap.Status.Files) == 0 {
		return nil
	}

	index := 0
	var availableNodes []v3.PacketCaptureFile
	for _, pcapNode := range pcap.Status.Files {
		if existingNodes.Contains(pcapNode.Node) {
			availableNodes[index] = pcapNode
			index++
		}
	}
	if index != len(pcap.Status.Files) {
		pcap.Status.Files = availableNodes
		_, err = u.calicoClient.PacketCaptures().Update(ctx, pcap, options.SetOptions{})
	}

	return err
}
