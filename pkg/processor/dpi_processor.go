// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package processor

import (
	"context"
	"fmt"

	"github.com/tigera/deep-packet-inspection/pkg/exec"
	"k8s.io/client-go/util/retry"

	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"

	"time"
)

const (
	maxErrors          = 10
	snortRetryDuration = 1000 * time.Millisecond
)

// Processor for each WEP interface added starts a snort process, and gracefully stops the snort process if
// WEP interfaces are removed.
type Processor interface {
	Add(ctx context.Context, wepKey model.WorkloadEndpointKey, iface string)
	Remove(wepKey model.WorkloadEndpointKey)
	WEPInterfaceCount() int
	Close()
}

type dpiProcessor struct {
	dpiKey                  model.ResourceKey
	nodeName                string
	calicoClient            clientv3.Interface
	snortExecFn             exec.Snort
	ch                      chan cacheReq
	snortAlertFileBasePath  string
	snortAlertFileSize      int
	snortCommunityRulesFile string
}

type requestType int

const (
	getRequest requestType = iota
	getCacheSizeRequest
	addRequest
	updateRequest
	deleteRequest
	deleteAllRequest
)

type cacheReq struct {
	wepKey        string
	snortExec     exec.Exec
	cacheResponse chan cacheResponse
	requestType   requestType
}

type cacheResponse struct {
	snortExecCh exec.Exec
	cacheSize   int
	success     bool
}

func NewProcessor(ctx context.Context,
	calicoClient clientv3.Interface,
	dpikey interface{},
	nodeName string,
	snortExecFn exec.Snort,
	snortAlertFileBasePath string,
	snortAlertFileSize int,
	snortCommunityRulesFile string,
) Processor {
	d := &dpiProcessor{
		dpiKey:                  dpikey.(model.ResourceKey),
		nodeName:                nodeName,
		calicoClient:            calicoClient,
		snortExecFn:             snortExecFn,
		snortAlertFileBasePath:  snortAlertFileBasePath,
		snortAlertFileSize:      snortAlertFileSize,
		snortCommunityRulesFile: snortCommunityRulesFile,
		ch:                      make(chan cacheReq, 100),
	}
	go d.run(ctx)
	d.retryStatusUpdate(ctx, &v3.DPIActive{
		Success:     false,
		LastUpdated: &metav1.Time{Time: time.Now()},
	}, nil)
	return d
}

// run caches all the snort processes against the WEP key by watching the channel for updates, it returns
// a cacheResponse object with success set to true if the request is fulfilled.
//
// Request types:
//  	- getRequest: returns the SnortExec for the requested WEP key
//
// 		- getCacheSizeRequest: returns the number of items in the cache
//
//		- addRequest: adds the given WEP key into cache if it doesn't already exist and set snortExec to nil.
//		  This request is sent before starting the loop that starts/restarts snort.
//
// 		- updateRequest: maps the WEP key to given snortExec and returns success response only if WEP key already exists.
//		  This request returns success as false if the WEP key doesn't already exist, the only scenario during which
//		  WEP key is deleted from the cache if either WEP or DPI resource is deleted or context is cancelled,
//		  updateRequest is no longer valid at this point.
//	      As this request is sent from within the goroutine loop that starts/restarts snort, goroutine returns if success is false.
//
//		- deleteRequest, deleteAllRequest: stops the running snort process, deletes it from the cache and update the status if needed.
func (p *dpiProcessor) run(ctx context.Context) {
	wepKeyToSnortExec := make(map[string]exec.Exec)
	for {
		select {
		case r := <-p.ch:
			switch r.requestType {
			case getRequest:
				snortExec, ok := wepKeyToSnortExec[r.wepKey]
				r.cacheResponse <- cacheResponse{snortExecCh: snortExec, success: ok}
			case getCacheSizeRequest:
				r.cacheResponse <- cacheResponse{cacheSize: len(wepKeyToSnortExec), success: true}
			case addRequest:
				_, ok := wepKeyToSnortExec[r.wepKey]
				// For addRequest, if item already exist set success as false, there is already a snort process running.
				if ok {
					r.cacheResponse <- cacheResponse{success: false}
				} else {
					wepKeyToSnortExec[r.wepKey] = nil
					r.cacheResponse <- cacheResponse{success: true}
				}
			case updateRequest:
				_, ok := wepKeyToSnortExec[r.wepKey]
				// For updateRequest, if item doesn't exist set success as false, WEP key is already removed from cache.
				if !ok {
					r.cacheResponse <- cacheResponse{success: false}
				} else {
					wepKeyToSnortExec[r.wepKey] = r.snortExec
					r.cacheResponse <- cacheResponse{success: true}
				}
			case deleteRequest:
				snortExec, ok := wepKeyToSnortExec[r.wepKey]
				if ok {
					if snortExec != nil {
						log.WithFields(log.Fields{"DPI": p.dpiKey, "WEP interface": r.wepKey}).Info("Stopping snort process")
						snortExec.Stop()
					}
					delete(wepKeyToSnortExec, r.wepKey)

					// If there is no snort running, update the DeepPacketInspection status.
					if len(wepKeyToSnortExec) == 0 {
						p.retryStatusUpdate(ctx, &v3.DPIActive{
							Success:     false,
							LastUpdated: &metav1.Time{Time: time.Now()},
						}, nil)
					}

					r.cacheResponse <- cacheResponse{success: true}
				} else {
					log.WithFields(log.Fields{"DPI": p.dpiKey, "WEP interface": r.wepKey}).Info("WEP Interface doesn't exist.")
					r.cacheResponse <- cacheResponse{success: false}
				}
			case deleteAllRequest:
				log.WithField("DPI", p.dpiKey).Debug("Stopping all snort processes")
				for _, snortExec := range wepKeyToSnortExec {
					if snortExec != nil {
						log.WithFields(log.Fields{"DPI": p.dpiKey, "WEP interface": r.wepKey}).Info("Stopping snort process")
						snortExec.Stop()
					}
				}

				// Status needs to be updated only if the request actually stopped snort process
				if len(wepKeyToSnortExec) != 0 {
					p.retryStatusUpdate(ctx, &v3.DPIActive{
						Success:     false,
						LastUpdated: &metav1.Time{Time: time.Now()},
					}, nil)
				}

				// reset all cached WEP Keys
				wepKeyToSnortExec = make(map[string]exec.Exec)

				r.cacheResponse <- cacheResponse{success: true}
			}
		case <-ctx.Done():
			return
		}
	}
}

// Add if the WEP interface doesn't have a corresponding snort process running, start one and
// update the status of DeepPacketInspection resource.
// This kicks a goroutine that starts/restarts snort on failure.
func (p *dpiProcessor) Add(ctx context.Context, wepKey model.WorkloadEndpointKey, iface string) {
	log.WithFields(log.Fields{"DPI": p.dpiKey.String(), "WEP": wepKey.String()}).Debugf("Adding new WEP interface.")

	resCh := make(chan cacheResponse)

	p.ch <- cacheReq{requestType: addRequest, wepKey: wepKey.String(), cacheResponse: resCh}
	res := <-resCh
	close(resCh)

	if res.snortExecCh != nil {
		log.WithFields(log.Fields{"DPI": p.dpiKey.String(), "WEP": wepKey.String()}).Debugf("WEP Interface already exist.")
		return
	}

	// start snort
	go p.runSnort(ctx, wepKey, iface)

	log.WithFields(log.Fields{"DPI": p.dpiKey.String(), "WEP": wepKey.String()}).Info("Snort process has started and running")
	p.retryStatusUpdate(ctx, &v3.DPIActive{
		Success:     true,
		LastUpdated: &metav1.Time{Time: time.Now()},
	}, nil)
}

// Remove stops snort process running on the WEP interface, if there are no more snort processes running for
// the DeepPacketInspection resource sets its active status to false.
func (p *dpiProcessor) Remove(wepKey model.WorkloadEndpointKey) {
	log.WithFields(log.Fields{"DPI": p.dpiKey.String(), "WEP": wepKey.String()}).Debugf("Removing WEP interface")

	resCh := make(chan cacheResponse)
	defer close(resCh)

	p.ch <- cacheReq{requestType: deleteRequest, wepKey: wepKey.String(), cacheResponse: resCh}
	<-resCh

}

// WEPInterfaceCount returns the number of WEP interfaces selected by the DPI selector.
func (p *dpiProcessor) WEPInterfaceCount() int {
	resCh := make(chan cacheResponse)
	defer close(resCh)

	p.ch <- cacheReq{requestType: getCacheSizeRequest, cacheResponse: resCh}
	res := <-resCh
	return res.cacheSize
}

// Close terminates all snort process and sets the active status of DeepPacketInspection resource to false.
func (p *dpiProcessor) Close() {
	log.WithField("DPI", p.dpiKey).Info("Terminating all snort process")
	resCh := make(chan cacheResponse)
	defer close(resCh)
	p.ch <- cacheReq{requestType: deleteAllRequest, cacheResponse: resCh}
	<-resCh
}

// retryStatusUpdate retries setting the status of DeepPacketInspection resource if there is a conflict.
func (p *dpiProcessor) retryStatusUpdate(ctx context.Context, statusActive *v3.DPIActive, statusErr *v3.DPIErrorCondition) {
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return p.statusUpdate(ctx, statusActive, statusErr)
	}); err != nil {
		log.WithField("DPI", p.dpiKey).WithError(err).Error("failed to update status after retries")
	}
}

// statusUpdate gets the DeepPacketInspection resource and updates the status field with the values for status activity and
// status error for the current node.
func (p *dpiProcessor) statusUpdate(ctx context.Context, statusActive *v3.DPIActive, statusErr *v3.DPIErrorCondition) error {
	res, err := p.calicoClient.DeepPacketInspections().Get(ctx, p.dpiKey.Namespace, p.dpiKey.Name, options.GetOptions{})
	if err != nil {
		log.WithError(err).Errorf("could not get resource %s", p.dpiKey.String())
		return err
	}

	var currentNode v3.DPINode
	currentNode = v3.DPINode{Node: p.nodeName}
	if res.Status.Nodes == nil {
		res.Status.Nodes = []v3.DPINode{}
	}

	nodeIndex := -1
	// get the status of the current node if it already exists
	for i, s := range res.Status.Nodes {
		if s.Node == p.nodeName {
			currentNode = s
			nodeIndex = i
			break
		}
	}

	currentNode.Active = *statusActive
	// If status error exists, retain the latest maxErrors
	if statusErr != nil {
		errorConditions := currentNode.ErrorConditions
		errorConditions = append(errorConditions, *statusErr)
		if len(errorConditions) > maxErrors {
			errorConditions = errorConditions[1:]
		}
		currentNode.ErrorConditions = errorConditions
	}

	if nodeIndex == -1 {
		// There is no status for the current node, append the current node's status
		res.Status.Nodes = append(res.Status.Nodes, currentNode)
	} else {
		// update the existing status of current node
		res.Status.Nodes[nodeIndex] = currentNode
	}

	_, err = p.calicoClient.DeepPacketInspections().UpdateStatus(ctx, res, options.SetOptions{})
	if err != nil {
		log.WithError(err).Errorf("could not update status of resource %s in cluster", p.dpiKey.String())
		return err
	}
	return nil
}

// runSnort starts snort and waits for the process to complete.
// Only case where snort process terminates without error is when syscall.SIGTERM signal is sent during dpiProcessor.Close,
// if snort fails with error, restart the snort process after some interval and update the cache with the new snortExec,
// if request to update the cache fails, it implies that WEP interface is no longer valid so return.
func (p *dpiProcessor) runSnort(ctx context.Context, wepKey model.WorkloadEndpointKey, iface string) {
	for {
		snortExec, err := p.snortExecFn(wepKey.WorkloadID, iface, p.dpiKey.Namespace, p.dpiKey.Name, p.snortAlertFileBasePath,
			p.snortAlertFileSize, p.snortCommunityRulesFile)
		if err != nil {
			log.WithFields(log.Fields{"DPI": p.dpiKey.String(), "WEP": wepKey.String()}).WithError(err).
				Errorf("failed to set snort command line")
			p.retryStatusUpdate(ctx, &v3.DPIActive{
				Success:     false,
				LastUpdated: &metav1.Time{Time: time.Now()},
			}, &v3.DPIErrorCondition{
				Message:     err.Error(),
				LastUpdated: &metav1.Time{Time: time.Now()},
			})
			// If there is an error, retry after snortRetryDuration
			<-time.After(snortRetryDuration)
			continue
		}

		resCh := make(chan cacheResponse)
		p.ch <- cacheReq{requestType: updateRequest, snortExec: snortExec, wepKey: wepKey.String(), cacheResponse: resCh}
		res := <-resCh

		// Request to update the cache failed, implying that cache no longer tracks the wepKey, so return.
		if !res.success {
			log.WithFields(log.Fields{"DPI": p.dpiKey.String(), "WEP": wepKey.String()}).
				Debugf("terminating loop that runs snort")
			return
		}

		err = snortExec.Start()
		if err != nil {
			log.WithFields(log.Fields{"DPI": p.dpiKey.String(), "WEP": wepKey.String()}).WithError(err).
				Errorf("snort failed to start")
			p.retryStatusUpdate(ctx, &v3.DPIActive{
				Success:     false,
				LastUpdated: &metav1.Time{Time: time.Now()},
			}, &v3.DPIErrorCondition{
				Message:     fmt.Sprintf("failed to start snort for resource %s on WEP interface %s", p.dpiKey.String(), iface),
				LastUpdated: &metav1.Time{Time: time.Now()},
			})
			// If there is an error, retry after snortRetryDuration
			<-time.After(snortRetryDuration)
			continue
		}

		err = snortExec.Wait()
		if err != nil {
			log.WithFields(log.Fields{"DPI": p.dpiKey.String(), "WEP": wepKey.String()}).WithError(err).Errorf("snort failed")
			p.retryStatusUpdate(ctx, &v3.DPIActive{
				Success:     false,
				LastUpdated: &metav1.Time{Time: time.Now()},
			}, &v3.DPIErrorCondition{
				Message:     fmt.Sprintf("snort process failed for resource %s on WEP interface %s with error '%s'", p.dpiKey.String(), iface, err.Error()),
				LastUpdated: &metav1.Time{Time: time.Now()},
			})
			// If there is an error in starting snort restart snort with de
			<-time.After(snortRetryDuration)
		} else {
			log.WithField("DPI", p.dpiKey).Debugf("Snort terminated gracefully")
			return
		}
	}
}
