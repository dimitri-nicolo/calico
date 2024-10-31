// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package datastore

import (
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/projectcalico/calico/libcalico-go/lib/resources"
	lmaL "github.com/projectcalico/calico/lma/pkg/list"
)

func (cs *clientSet) RetrieveList(kind metav1.TypeMeta) (*lmaL.TimestampedResourceList, error) {
	log.WithField("type", kind).Debug("Listing resource")

	// Use the resource helper to list the appropriate resource kind.
	requestStartTime := metav1.Now()
	l, err := resourceHelpersMap[kind].listFunc(cs)
	if err != nil {
		return nil, err
	}
	requestCompletedTime := metav1.Now()

	// Set the type meta of both the list and the entries in the list - we need to do this because it is not
	// filled in on the API server response.
	if err = SetListTypeMeta(l, kind); err != nil {
		return nil, err
	}

	return &lmaL.TimestampedResourceList{
		ResourceList:              l,
		RequestStartedTimestamp:   requestStartTime,
		RequestCompletedTimestamp: requestCompletedTime,
	}, nil
}

// SetListTypeMeta sets the TypeMeta of the list and each of the items in the list.
func SetListTypeMeta(list resources.ResourceList, kind metav1.TypeMeta) error {
	// TODO(rlb): strictly speaking the list type is NOT the same as the items it contains.
	// Overwrite the group/version/kind of the list.
	list.GetObjectKind().SetGroupVersionKind(kind.GroupVersionKind())

	// Also update the items in the list to have the expected type (which is also left blank).
	if err := meta.EachListItem(list, func(obj runtime.Object) error {
		obj.GetObjectKind().SetGroupVersionKind(kind.GroupVersionKind())
		return nil
	}); err != nil {
		return err
	}

	return nil
}
