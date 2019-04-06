// Copyright 2019 Tigera Inc. All rights reserved.

package puller

import (
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

type MockConfigMap struct {
	ConfigMapData map[string]string
	Error         error
}

func (*MockConfigMap) Create(*v1.ConfigMap) (*v1.ConfigMap, error) {
	panic("implement me")
}

func (*MockConfigMap) Update(*v1.ConfigMap) (*v1.ConfigMap, error) {
	panic("implement me")
}

func (*MockConfigMap) Delete(name string, options *v12.DeleteOptions) error {
	panic("implement me")
}

func (*MockConfigMap) DeleteCollection(options *v12.DeleteOptions, listOptions v12.ListOptions) error {
	panic("implement me")
}

func (m *MockConfigMap) Get(name string, options v12.GetOptions) (*v1.ConfigMap, error) {
	return &v1.ConfigMap{
		Data: m.ConfigMapData,
	}, m.Error
}

func (*MockConfigMap) List(opts v12.ListOptions) (*v1.ConfigMapList, error) {
	panic("implement me")
}

func (*MockConfigMap) Watch(opts v12.ListOptions) (watch.Interface, error) {
	panic("implement me")
}

func (*MockConfigMap) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.ConfigMap, err error) {
	panic("implement me")
}
