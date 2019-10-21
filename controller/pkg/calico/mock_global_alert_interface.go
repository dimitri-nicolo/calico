// Copyright 2019 Tigera Inc. All rights reserved.

package calico

import (
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

type MockGlobalAlertInterface struct {
	GlobalAlertList *v3.GlobalAlertList
	GlobalAlert     *v3.GlobalAlert
	Error           error
	WatchError      error
	W               *MockWatch
}

func (m *MockGlobalAlertInterface) UpdateStatus(gtf *v3.GlobalAlert) (*v3.GlobalAlert, error) {
	m.GlobalAlert = gtf
	return gtf, m.Error
}

func (m *MockGlobalAlertInterface) Create(gtf *v3.GlobalAlert) (*v3.GlobalAlert, error) {
	return gtf, m.Error
}

func (m *MockGlobalAlertInterface) Update(gtf *v3.GlobalAlert) (*v3.GlobalAlert, error) {
	return gtf, m.Error
}

func (m *MockGlobalAlertInterface) Delete(name string, options *v1.DeleteOptions) error {
	return m.Error
}

func (m *MockGlobalAlertInterface) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return m.Error
}

func (m *MockGlobalAlertInterface) Get(name string, options v1.GetOptions) (*v3.GlobalAlert, error) {
	return m.GlobalAlert, m.Error
}

func (m *MockGlobalAlertInterface) List(opts v1.ListOptions) (*v3.GlobalAlertList, error) {
	return m.GlobalAlertList, m.Error
}

func (m *MockGlobalAlertInterface) Watch(opts v1.ListOptions) (watch.Interface, error) {
	if m.WatchError == nil {
		if m.W == nil {
			m.W = &MockWatch{make(chan watch.Event)}
		}
		return m.W, nil
	} else {
		return nil, m.WatchError
	}
}

func (m *MockGlobalAlertInterface) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v3.GlobalAlert, err error) {
	return nil, m.Error
}
