// Copyright 2019 Tigera Inc. All rights reserved.

package calico

import (
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

type MockGlobalThreatFeedInterface struct {
	GlobalThreatFeedList *v3.GlobalThreatFeedList
	GlobalThreatFeed     *v3.GlobalThreatFeed
	Error                error
	WatchError           error
	W                    *MockWatch
}

func (m *MockGlobalThreatFeedInterface) UpdateStatus(gtf *v3.GlobalThreatFeed) (*v3.GlobalThreatFeed, error) {
	m.GlobalThreatFeed = gtf
	return gtf, m.Error
}

func (m *MockGlobalThreatFeedInterface) Create(gtf *v3.GlobalThreatFeed) (*v3.GlobalThreatFeed, error) {
	return gtf, m.Error
}

func (m *MockGlobalThreatFeedInterface) Update(gtf *v3.GlobalThreatFeed) (*v3.GlobalThreatFeed, error) {
	return gtf, m.Error
}

func (m *MockGlobalThreatFeedInterface) Delete(name string, options *v1.DeleteOptions) error {
	return m.Error
}

func (m *MockGlobalThreatFeedInterface) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return m.Error
}

func (m *MockGlobalThreatFeedInterface) Get(name string, options v1.GetOptions) (*v3.GlobalThreatFeed, error) {
	return m.GlobalThreatFeed, m.Error
}

func (m *MockGlobalThreatFeedInterface) List(opts v1.ListOptions) (*v3.GlobalThreatFeedList, error) {
	return m.GlobalThreatFeedList, m.Error
}

func (m *MockGlobalThreatFeedInterface) Watch(opts v1.ListOptions) (watch.Interface, error) {
	if m.WatchError == nil {
		if m.W == nil {
			m.W = &MockWatch{make(chan watch.Event)}
		}
		return m.W, nil
	} else {
		return nil, m.WatchError
	}
}

func (m *MockGlobalThreatFeedInterface) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v3.GlobalThreatFeed, err error) {
	return nil, m.Error
}
