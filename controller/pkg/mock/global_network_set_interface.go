// Copyright 2019 Tigera Inc. All rights reserved.

package mock

import (
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

type GlobalNetworkSetInterface struct {
	GlobalNetworkSet *v3.GlobalNetworkSet
	Error            error
	CreateError      error
	GetError         error
	UpdateError      error
}

func (m *GlobalNetworkSetInterface) Create(gns *v3.GlobalNetworkSet) (*v3.GlobalNetworkSet, error) {
	if m.CreateError != nil {
		return nil, m.CreateError
	}
	m.GlobalNetworkSet = gns
	return gns, m.Error
}

func (m *GlobalNetworkSetInterface) Update(gns *v3.GlobalNetworkSet) (*v3.GlobalNetworkSet, error) {
	if m.UpdateError != nil {
		return nil, m.UpdateError
	}
	m.GlobalNetworkSet = gns
	return gns, m.Error
}

func (m *GlobalNetworkSetInterface) Delete(name string, options *v1.DeleteOptions) error {
	return m.Error
}

func (m *GlobalNetworkSetInterface) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return m.Error
}

func (m *GlobalNetworkSetInterface) Get(name string, options v1.GetOptions) (*v3.GlobalNetworkSet, error) {
	if m.GetError != nil {
		return nil, m.GetError
	}
	return m.GlobalNetworkSet, m.Error
}

func (m *GlobalNetworkSetInterface) List(opts v1.ListOptions) (*v3.GlobalNetworkSetList, error) {
	return nil, m.Error
}

func (m *GlobalNetworkSetInterface) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return nil, m.Error
}

func (m *GlobalNetworkSetInterface) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v3.GlobalNetworkSet, err error) {
	return nil, m.Error
}
