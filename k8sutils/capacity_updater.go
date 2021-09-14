// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package k8sutils

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	"github.com/projectcalico/felix/aws"
)

type CapacityUpdater struct {
	nodeName           string
	k8sClient          *kubernetes.Clientset
	updateC            chan aws.SecondaryIfaceCapacities
	lastCapacityUpdate *aws.SecondaryIfaceCapacities
	resyncNeeded       bool
	clock              clock.Clock
	timeout            time.Duration
}

const (
	ResourceSecondaryIPv4 = "projectcalico.org/aws-secondary-ipv4"
	defaultTimeout        = 30 * time.Second
)

func NewCapacityUpdater(nodeName string, k8sClient *kubernetes.Clientset) *CapacityUpdater {
	return &CapacityUpdater{
		nodeName:  nodeName,
		k8sClient: k8sClient,
		updateC:   make(chan aws.SecondaryIfaceCapacities, 1),
		clock:     clock.RealClock{},
		timeout:   defaultTimeout,
	}
}

func (u *CapacityUpdater) OnCapacityChange(capacities aws.SecondaryIfaceCapacities) {
	// Discard any queued update by doing a non-blocking read.
	select {
	case <-u.updateC:
	default:
	}

	// This write should never block.
	u.updateC <- capacities
}

func (u *CapacityUpdater) Start(ctx context.Context) {
	go u.loopUpdatingK8s(ctx)
}

func (u *CapacityUpdater) loopUpdatingK8s(ctx context.Context) {
	logrus.WithField("nodeName", u.nodeName).Info("Kubernetes capacity updater running in background")

	// Set ourselves up for exponential backoff after a failure.  backoffMgr.Backoff() returns the same Timer
	// on each call so we need to stop it properly when cancelling it.
	var backoffTimer clock.Timer
	var backoffC <-chan time.Time
	backoffMgr := u.newBackoffManager()
	stopBackoffTimer := func() {
		if backoffTimer != nil {
			// Reset the timer before calling Backoff() again for correct behaviour. This is the standard
			// time.Timer.Stop() dance...
			if !backoffTimer.Stop() {
				<-backoffTimer.C()
			}
			backoffTimer = nil
			backoffC = nil
		}
	}
	defer stopBackoffTimer()

	var caps aws.SecondaryIfaceCapacities
	var resyncNeeded = false
	for {
		select {
		case <-ctx.Done():
			logrus.Info("CapacityUpdater shutting down; context closed.")
			return
		case caps = <-u.updateC:
			resyncNeeded = true
		case <-backoffC:
			// Important: nil out the timer so that stopBackoffTimer() won't try to stop it again (and deadlock).
			backoffC = nil
			backoffTimer = nil
			logrus.Warn("Retrying k8s resync after backoff.")
		}

		stopBackoffTimer()

		if resyncNeeded {
			err := u.handleCapacityChange(caps)
			if err != nil {
				logrus.WithError(err).Error("Failed to resync with AWS. Will retry after backoff.")
				backoffTimer = backoffMgr.Backoff()
				backoffC = backoffTimer.C()
			} else {
				resyncNeeded = false
			}
		}
	}
}

func (u *CapacityUpdater) handleCapacityChange(caps aws.SecondaryIfaceCapacities) error {
	if u.lastCapacityUpdate != nil && u.lastCapacityUpdate.Equals(caps) {
		logrus.Debug("Capacity update made no changes")
		return nil
	}

	ctx, cancel := u.newContext()
	defer cancel()

	nodeClient := u.k8sClient.CoreV1().Nodes()
	node, err := nodeClient.Get(ctx, u.nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to look up our kubernetes node by name (%s): %w", u.nodeName, err)
	}

	res := node.Status.Capacity.Name(ResourceSecondaryIPv4, resource.DecimalSI)
	if res.Value() == int64(caps.MaxCalicoSecondaryIPs) {
		logrus.WithField("capacity", res.Value()).Debug("Kubernetes secondary IP capacity already correct")
		return nil
	}

	var capResource interface{}
	if caps.MaxCalicoSecondaryIPs > 0 {
		capResource = fmt.Sprint(caps.MaxCalicoSecondaryIPs)
	}
	patch := map[string]interface{}{
		"status": map[string]interface{}{
			"capacity": map[string]interface{}{
				ResourceSecondaryIPv4: capResource,
			},
		},
	}

	patchData, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("BUG: failed to marshall JSON patch: %w", err)
	}

	// Capacity updates must be done as a PATCH.
	_, err = nodeClient.PatchStatus(ctx, u.nodeName, patchData)
	if err != nil {
		return fmt.Errorf("failed to patch kubernetes Node resource: %w", err)
	}

	u.lastCapacityUpdate = &caps
	return nil
}

func (u *CapacityUpdater) newBackoffManager() wait.BackoffManager {
	const (
		initBackoff   = 1 * time.Second
		maxBackoff    = 1 * time.Minute
		resetDuration = 10 * time.Minute
		backoffFactor = 2.0
		jitter        = 0.1
	)
	backoffMgr := wait.NewExponentialBackoffManager(initBackoff, maxBackoff, resetDuration, backoffFactor, jitter, u.clock)
	return backoffMgr
}

func (u *CapacityUpdater) newContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), u.timeout)
}
