// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package dpiupdater

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/calico/libcalico-go/lib/options"
)

const (
	maxErrors = 10
)

type DPIStatusUpdater interface {
	UpdateStatus(ctx context.Context, dpiKey model.ResourceKey, isActive bool)
	UpdateStatusWithError(ctx context.Context, dpiKey model.ResourceKey, isActive bool, errMsg string)
}

func NewDPIStatusUpdater(calicoClient clientv3.Interface, nodeName string) DPIStatusUpdater {
	return &dpiStatusUpdater{
		calicoClient: calicoClient,
		nodeName:     nodeName,
	}
}

type dpiStatusUpdater struct {
	calicoClient clientv3.Interface
	nodeName     string
}

// UpdateStatusWithError sets the status of DeepPacketInspection with given error message,
// it retries setting the status on conflict.
func (d *dpiStatusUpdater) UpdateStatusWithError(ctx context.Context, dpiKey model.ResourceKey, isActive bool, errMsg string) {
	activeStatus := v3.DPIActive{
		Success:     isActive,
		LastUpdated: &metav1.Time{Time: time.Now()},
	}
	statusErr := &v3.DPIErrorCondition{
		Message:     errMsg,
		LastUpdated: &metav1.Time{Time: time.Now()},
	}
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return d.update(ctx, dpiKey, activeStatus, statusErr)
	}); err != nil {
		log.WithField("DPI", dpiKey).WithError(err).Error("failed to update status after retries")
	}
}

// UpdateStatus sets the status of DeepPacketInspection and retries setting the status if there is a conflict.
func (d *dpiStatusUpdater) UpdateStatus(ctx context.Context, dpiKey model.ResourceKey, isActive bool) {
	activeStatus := v3.DPIActive{
		Success:     isActive,
		LastUpdated: &metav1.Time{Time: time.Now()},
	}
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return d.update(ctx, dpiKey, activeStatus, nil)
	}); err != nil {
		log.WithField("DPI", dpiKey).WithError(err).Error("failed to update status after retries")
	}
}

// update gets the DeepPacketInspection resource and updates the status field with the values for status activity and
// status error for the current node.
func (d *dpiStatusUpdater) update(ctx context.Context, dpiKey model.ResourceKey, statusActive v3.DPIActive, statusErr *v3.DPIErrorCondition) error {
	res, err := d.calicoClient.DeepPacketInspections().Get(ctx, dpiKey.Namespace, dpiKey.Name, options.GetOptions{})
	if err != nil {
		log.WithError(err).Errorf("could not get resource %s", dpiKey.String())
		return err
	}

	if res.Status.Nodes == nil {
		res.Status.Nodes = []v3.DPINode{}
	}

	currentNodeIndex := -1
	currentNode := v3.DPINode{Node: d.nodeName}
	// get the status of the current node if it already exists
	for i, s := range res.Status.Nodes {
		if s.Node == d.nodeName {
			currentNode = s
			currentNodeIndex = i
			break
		}
	}

	currentNode.Active = statusActive
	if currentNodeIndex == -1 {
		// There is no status for the current node, append the current node's status
		if statusErr != nil {
			currentNode.ErrorConditions = []v3.DPIErrorCondition{*statusErr}
		}
		res.Status.Nodes = append(res.Status.Nodes, currentNode)
	} else {
		// update the existing status of current node
		if statusErr != nil {
			errorConditions := currentNode.ErrorConditions
			errorConditions = append(errorConditions, *statusErr)
			if len(errorConditions) > maxErrors {
				errorConditions = errorConditions[1:]
			}
			currentNode.ErrorConditions = errorConditions
		}
		res.Status.Nodes[currentNodeIndex] = currentNode
	}

	_, err = d.calicoClient.DeepPacketInspections().UpdateStatus(ctx, res, options.SetOptions{})
	return err
}
