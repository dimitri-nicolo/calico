// Copyright 2021 Tigera Inc. All rights reserved.

package utils

import (
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/cacher"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/errorcondition"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
)

// AddErrorToFeedStatus adds an error entry with type errType and body err to the error conditions of the
// corresponding GlobalThreatFeed CR with a retry mechanism.
// A retry only kicks off when the update failure is caused by a StatusConflict and it will retry at most cacher.MaxUpdateRetry times
func AddErrorToFeedStatus(feedCacher cacher.GlobalThreatFeedCacher, errType string, err error) {
	getCachedFeedResponse := feedCacher.GetGlobalThreatFeed()
	if getCachedFeedResponse.Err != nil {
		log.WithError(getCachedFeedResponse.Err).
			Error("abort adding error to feed status because failed to retrieve cached GlobalThreatFeed CR")
		return
	}
	if getCachedFeedResponse.GlobalThreatFeed == nil {
		log.Error("abort adding error to feed status because cached GlobalThreatFeed CR cannot be empty")
		return
	}

	toBeUpdated := getCachedFeedResponse.GlobalThreatFeed
	for i := 1; i <= cacher.MaxUpdateRetry; i++ {
		log.Debug(fmt.Sprintf("%d/%d attempt to add error to feed status", i, cacher.MaxUpdateRetry))
		errorcondition.AddError(&toBeUpdated.Status, errType, err)
		updateResponse := feedCacher.UpdateGlobalThreatFeedStatus(toBeUpdated)
		updateErr := updateResponse.Err
		if updateErr == nil {
			log.Debug("attempt to add error to feed status succeeded, exiting the loop")
			return
		}
		statusErr, ok := updateErr.(*apiErrors.StatusError)
		if !ok || statusErr.Status().Code != http.StatusConflict {
			log.WithError(updateErr).Error("abort adding error to feed status due to unrecoverable failure")
			return
		}
		log.WithError(updateErr).Error("failed adding error to feed status")
		toBeUpdated = updateResponse.GlobalThreatFeed
	}
}

// ClearErrorFromFeedStatus removes all error entries with type errType from the error conditions of the
// corresponding GlobalThreatFeed CR with a retry mechanism.
// A retry only kicks off when the update failure is caused by a StatusConflict and it will retry at most cacher.MaxUpdateRetry times
func ClearErrorFromFeedStatus(feedCacher cacher.GlobalThreatFeedCacher, errType string) {
	getCachedFeedResponse := feedCacher.GetGlobalThreatFeed()
	if getCachedFeedResponse.Err != nil {
		log.WithError(getCachedFeedResponse.Err).
			Error("abort clearing error from feed status because failed to retrieve cached GlobalThreatFeed CR")
		return
	}
	if getCachedFeedResponse.GlobalThreatFeed == nil {
		log.Error("abort clearing error from feed status because cached GlobalThreatFeed CR cannot be empty")
		return
	}

	toBeUpdated := getCachedFeedResponse.GlobalThreatFeed
	for i := 1; i <= cacher.MaxUpdateRetry; i++ {
		log.Debug(fmt.Sprintf("%d/%d attempt to clear error from feed status", i, cacher.MaxUpdateRetry))
		errorcondition.ClearError(&toBeUpdated.Status, errType)
		updateResponse := feedCacher.UpdateGlobalThreatFeedStatus(toBeUpdated)
		updateErr := updateResponse.Err
		if updateErr == nil {
			log.Debug("attempt to clear error from feed status succeeded, exiting the loop")
			return
		}
		statusErr, ok := updateErr.(*apiErrors.StatusError)
		if !ok || statusErr.Status().Code != http.StatusConflict {
			log.WithError(updateErr).Error("abort removing error from feed status due to unrecoverable failure")
			return
		}
		log.WithError(updateErr).Error("failed removing error from feed status")
		toBeUpdated = updateResponse.GlobalThreatFeed
	}
}
