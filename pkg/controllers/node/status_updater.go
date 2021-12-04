// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package node

import (
	"context"
	"time"

	"github.com/projectcalico/libcalico-go/lib/options"
	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	tigeraapi "github.com/tigera/api/pkg/client/clientset_generated/clientset"

	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/set"

	"k8s.io/client-go/util/workqueue"
)

// NewStatusUpdateController creates a new controller responsible for cleaning the node
// specific status field in resource when corresponding node is deleted.
func NewStatusUpdateController(calicoClient client.Interface, calicoV3Client tigeraapi.Interface, nodeCache func() []string) *statusUpdateController {
	return &statusUpdateController{
		rl:               workqueue.DefaultControllerRateLimiter(),
		calicoClient:     calicoClient,
		calicoV3Client:   calicoV3Client,
		nodeCacheFn:      nodeCache,
		reconcilerPeriod: time.Hour * 12,
		kickChan:         make(chan interface{}),
	}
}

type statusUpdateController struct {
	rl               workqueue.RateLimiter
	calicoClient     client.Interface
	calicoV3Client   tigeraapi.Interface
	nodeCacheFn      func() []string
	reconcilerPeriod time.Duration
	kickChan         chan interface{}
}

func (c *statusUpdateController) Start(stop chan struct{}) {
	go c.acceptScheduledRequest(stop)
}

// acceptScheduledRequest cleans the deleted node's status in custom resource periodically.
func (c *statusUpdateController) acceptScheduledRequest(stopCh <-chan struct{}) {
	t := time.NewTicker(c.reconcilerPeriod)

	cleanup := func() {
		err := c.cleanupDPINodes()
		if err != nil {
			log.Errorf("An error occurred while cleaning DPI nodes: %v", err)
		}
		err = c.cleanupPacketCaptureNodes()
		if err != nil {
			log.Errorf("An error occurred while cleaning DPI nodes: %v", err)
		}
	}
	for {
		select {
		case <-c.kickChan:
			cleanup()
		case <-t.C:
			cleanup()
		case <-stopCh:
			return
		}
	}
}

func (c *statusUpdateController) cleanupDPINodes() error {
	possibleNodes := set.FromArray(c.nodeCacheFn())

	dpiResources, err := c.calicoClient.DeepPacketInspections().List(context.Background(), options.ListOptions{})
	if err != nil {
		return err
	}
	for _, res := range dpiResources.Items {
		var newNodes []v3.DPINode
		for _, node := range res.Status.Nodes {
			if possibleNodes.Contains(node.Node) {
				newNodes = append(newNodes, node)
			}
		}
		res.Status.Nodes = newNodes
		if len(newNodes) == len(res.Status.Nodes) {
			return nil
		}
		_, err = c.calicoClient.DeepPacketInspections().UpdateStatus(context.Background(), &res, options.SetOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *statusUpdateController) cleanupPacketCaptureNodes() error {
	possibleNodes := set.FromArray(c.nodeCacheFn())

	pcapResources, err := c.calicoClient.PacketCaptures().List(context.Background(), options.ListOptions{})
	if err != nil {
		return err
	}
	for _, res := range pcapResources.Items {
		var newFiles []v3.PacketCaptureFile
		for _, file := range res.Status.Files {
			if possibleNodes.Contains(file.Node) {
				newFiles = append(newFiles, file)
			}
		}
		res.Status.Files = newFiles
		if len(newFiles) == len(res.Status.Files) {
			return nil
		}
		_, err = c.calicoClient.PacketCaptures().Update(context.Background(), &res, options.SetOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *statusUpdateController) OnKubernetesNodeDeleted() {
	// When a Kubernetes node is deleted, trigger a sync.
	log.Debug("Kubernetes node deletion event")
	kick(c.kickChan)
}
