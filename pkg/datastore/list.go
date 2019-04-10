// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package datastore

import (
	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tigera/compliance/pkg/list"
)

func (cs *clientSet) RetrieveList(kind metav1.TypeMeta) (*list.TimestampedResourceList, error) {
	log.WithField("type", kind).Debug("Listing resource")

	requestStartTime := metav1.Now()
	l, err := resourceHelpersMap[kind].listFunc(cs)
	if err != nil {
		return nil, err
	}
	requestCompletedTime := metav1.Now()

	// TODO(rlb): strictly speaking the list type is NOT the same as the items it contains.
	// List func succeeded. Overwrite the group/version/kind which k8s does not correctly populate.
	l.GetObjectKind().SetGroupVersionKind(kind.GroupVersionKind())
	return &list.TimestampedResourceList{
		ResourceList:              l,
		RequestStartedTimestamp:   requestStartTime,
		RequestCompletedTimestamp: requestCompletedTime,
	}, nil
}
