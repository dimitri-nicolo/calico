// Copyright 2019-2020 Tigera Inc. All rights reserved.

package calico

import (
	"context"

	v3 "github.com/projectcalico/apiserver/pkg/apis/projectcalico/v3"
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

func (m *MockGlobalAlertInterface) UpdateStatus(ctx context.Context, gtf *v3.GlobalAlert, options v1.UpdateOptions) (*v3.GlobalAlert, error) {
	m.GlobalAlert = gtf
	return gtf, m.Error
}

func (m *MockGlobalAlertInterface) Create(ctx context.Context, gtf *v3.GlobalAlert, options v1.CreateOptions) (*v3.GlobalAlert, error) {
	return gtf, m.Error
}

func (m *MockGlobalAlertInterface) Update(ctx context.Context, gtf *v3.GlobalAlert, options v1.UpdateOptions) (*v3.GlobalAlert, error) {
	return gtf, m.Error
}

func (m *MockGlobalAlertInterface) Delete(ctx context.Context, name string, options v1.DeleteOptions) error {
	return m.Error
}

func (m *MockGlobalAlertInterface) DeleteCollection(ctx context.Context, options v1.DeleteOptions, listOptions v1.ListOptions) error {
	return m.Error
}

func (m *MockGlobalAlertInterface) Get(ctx context.Context, name string, options v1.GetOptions) (*v3.GlobalAlert, error) {
	return m.GlobalAlert, m.Error
}

func (m *MockGlobalAlertInterface) List(ctx context.Context, opts v1.ListOptions) (*v3.GlobalAlertList, error) {
	return m.GlobalAlertList, m.Error
}

func (m *MockGlobalAlertInterface) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	if m.WatchError == nil {
		if m.W == nil {
			m.W = &MockWatch{make(chan watch.Event)}
		}
		return m.W, nil
	} else {
		return nil, m.WatchError
	}
}

func (m *MockGlobalAlertInterface) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, options v1.PatchOptions, subresources ...string) (result *v3.GlobalAlert, err error) {
	return nil, m.Error
}
