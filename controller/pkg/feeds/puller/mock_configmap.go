// Copyright 2019-2020 Tigera Inc. All rights reserved.

package puller

import (
	"context"

	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	corev1appconfig "k8s.io/client-go/applyconfigurations/core/v1"
)

type MockConfigMap struct {
	ConfigMapData map[string]string
	Error         error
}

func (m *MockConfigMap) Apply(ctx context.Context, configMap *corev1appconfig.ConfigMapApplyConfiguration, opts v12.ApplyOptions) (result *v1.ConfigMap, err error) {
	panic("implement me")
}

func (*MockConfigMap) Create(context.Context, *v1.ConfigMap, v12.CreateOptions) (*v1.ConfigMap, error) {
	panic("implement me")
}

func (*MockConfigMap) Update(context.Context, *v1.ConfigMap, v12.UpdateOptions) (*v1.ConfigMap, error) {
	panic("implement me")
}

func (*MockConfigMap) Delete(ctx context.Context, name string, options v12.DeleteOptions) error {
	panic("implement me")
}

func (*MockConfigMap) DeleteCollection(ctx context.Context, options v12.DeleteOptions, listOptions v12.ListOptions) error {
	panic("implement me")
}

func (m *MockConfigMap) Get(ctx context.Context, name string, options v12.GetOptions) (*v1.ConfigMap, error) {
	return &v1.ConfigMap{
		Data: m.ConfigMapData,
	}, m.Error
}

func (*MockConfigMap) List(ctx context.Context, opts v12.ListOptions) (*v1.ConfigMapList, error) {
	panic("implement me")
}

func (*MockConfigMap) Watch(ctx context.Context, opts v12.ListOptions) (watch.Interface, error) {
	panic("implement me")
}

func (*MockConfigMap) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, options v12.PatchOptions, subresources ...string) (result *v1.ConfigMap, err error) {
	panic("implement me")
}
