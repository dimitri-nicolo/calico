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

type MockSecrets struct {
	SecretsData map[string][]byte
	Error       error
}

func (m *MockSecrets) Apply(ctx context.Context, secret *corev1appconfig.SecretApplyConfiguration, opts v12.ApplyOptions) (result *v1.Secret, err error) {
	panic("implement me")
}

func (*MockSecrets) Create(context.Context, *v1.Secret, v12.CreateOptions) (*v1.Secret, error) {
	panic("implement me")
}

func (*MockSecrets) Update(context.Context, *v1.Secret, v12.UpdateOptions) (*v1.Secret, error) {
	panic("implement me")
}

func (*MockSecrets) Delete(ctx context.Context, name string, options v12.DeleteOptions) error {
	panic("implement me")
}

func (*MockSecrets) DeleteCollection(ctx context.Context, options v12.DeleteOptions, listOptions v12.ListOptions) error {
	panic("implement me")
}

func (m *MockSecrets) Get(ctx context.Context, name string, options v12.GetOptions) (*v1.Secret, error) {
	return &v1.Secret{
		Data: m.SecretsData,
	}, m.Error
}

func (*MockSecrets) List(ctx context.Context, opts v12.ListOptions) (*v1.SecretList, error) {
	panic("implement me")
}

func (*MockSecrets) Watch(ctx context.Context, opts v12.ListOptions) (watch.Interface, error) {
	panic("implement me")
}

func (*MockSecrets) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, options v12.PatchOptions, subresources ...string) (result *v1.Secret, err error) {
	panic("implement me")
}
