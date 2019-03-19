// Copyright 2019 Tigera Inc. All rights reserved.

package mock

import (
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

type GlobalThreatFeedInterface struct {
	GlobalThreatFeedList *v3.GlobalThreatFeedList
	Error                error
}

func (m *GlobalThreatFeedInterface) Create(gtf *v3.GlobalThreatFeed) (*v3.GlobalThreatFeed, error) {
	return gtf, m.Error
}

func (m *GlobalThreatFeedInterface) Update(gtf *v3.GlobalThreatFeed) (*v3.GlobalThreatFeed, error) {
	return gtf, m.Error
}

func (m *GlobalThreatFeedInterface) Delete(name string, options *v1.DeleteOptions) error {
	return m.Error
}

func (m *GlobalThreatFeedInterface) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return m.Error
}

func (m *GlobalThreatFeedInterface) Get(name string, options v1.GetOptions) (*v3.GlobalThreatFeed, error) {
	return nil, m.Error
}

func (m *GlobalThreatFeedInterface) List(opts v1.ListOptions) (*v3.GlobalThreatFeedList, error) {
	return m.GlobalThreatFeedList, m.Error
}

func (m *GlobalThreatFeedInterface) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return nil, m.Error
}

func (m *GlobalThreatFeedInterface) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v3.GlobalThreatFeed, err error) {
	return nil, m.Error
}
