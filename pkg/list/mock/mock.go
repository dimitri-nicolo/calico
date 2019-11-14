// Copyright (c) 2019 Tigera, Inc. All rights reserved.
// pkg/list/mock package defines a mocking framework for
//  any downstream packages that uses the list interface.
//  (e.g. snapshot, replay)
package mock

import (
	"time"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/libcalico-go/lib/errors"
	"github.com/projectcalico/libcalico-go/lib/resources"

	"github.com/tigera/lma/pkg/list"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

// mockList is used by both mockSource and mockDestination.
type mockLister struct {
	data []*list.TimestampedResourceList
}

// Initialize is used by the test to fill the lister with a list for each resource type
//   Useful for replayer.
func (m *mockLister) Initialize(ts time.Time) {
	for _, rh := range resources.GetAllResourceHelpers() {
		resList := rh.NewResourceList()
		tm := rh.TypeMeta()
		resList.GetObjectKind().SetGroupVersionKind((&tm).GroupVersionKind())
		m.data = append(m.data, &list.TimestampedResourceList{resList, metav1.Time{ts}, metav1.Time{ts}})
	}
}

// mockLister implements the expected logic of the list fetcher.
func (m *mockLister) retrieveList(tm metav1.TypeMeta, from *time.Time, to *time.Time, ascending bool) (*list.TimestampedResourceList, error) {
	listToReturn := (*list.TimestampedResourceList)(nil)
	for i := 0; i < len(m.data); i++ {
		resList := m.data[i]
		typeMetaMatches := resList.GetObjectKind().GroupVersionKind() == tm.GroupVersionKind()
		fromMatches := from == nil || resList.RequestCompletedTimestamp.Time.After(*from)
		toMatches := to == nil || resList.RequestCompletedTimestamp.Time.Before(*to)
		overwriteValid := listToReturn == nil ||
			(ascending && resList.RequestCompletedTimestamp.Time.Before(listToReturn.RequestCompletedTimestamp.Time)) ||
			(!ascending && resList.RequestCompletedTimestamp.Time.After(listToReturn.RequestCompletedTimestamp.Time))
		log.WithFields(log.Fields{"i": i, "tm": typeMetaMatches, "from": fromMatches, "to": toMatches, "overwriteValid": overwriteValid}).Debug("processing list")
		if typeMetaMatches && fromMatches && toMatches && overwriteValid {
			listToReturn = resList
		}
	}
	if listToReturn == nil {
		return nil, errors.ErrorResourceDoesNotExist{}
	}
	return listToReturn, nil
}

// LoadList is used by the tester to preload lists into the mock.
func (m *mockLister) LoadList(l *list.TimestampedResourceList) {
	m.data = append(m.data, l)
}

// Source implements pkg/list.Source
type Source struct {
	mockLister
	RetrieveCalls int
}

func NewSource() *Source {
	return &Source{}
}

// RetrieveList implements pkg/list.Source.RetrieveList using mockLister.retrieveList
func (r *Source) RetrieveList(kind metav1.TypeMeta) (*list.TimestampedResourceList, error) {
	r.RetrieveCalls++
	return r.retrieveList(kind, nil, nil, false)
}

// Destination implements pkg/api.ListDestination
type Destination struct {
	mockLister
	RetrieveCalls int
	StoreCalls    int
}

// NewDestination creates a mock list.Destination struct with an optional parameter of filling it with empty lists.
func NewDestination(tm *time.Time) *Destination {
	dest := &Destination{}
	if tm != nil {
		dest.Initialize(*tm)
	}
	return dest
}

// StoreList implements pkg/api.ListDestination.StoreList using mockLister.LoadList
func (r *Destination) StoreList(_ metav1.TypeMeta, list *list.TimestampedResourceList) error {
	r.StoreCalls++
	r.mockLister.LoadList(list)
	return nil
}

// RetrieveList implements pkg/api.ListDestination.RetrieveList using mockLister.retrieveList
func (r *Destination) RetrieveList(tm metav1.TypeMeta, from *time.Time, to *time.Time, ascending bool) (*list.TimestampedResourceList, error) {
	r.RetrieveCalls++
	return r.retrieveList(tm, from, to, ascending)
}
