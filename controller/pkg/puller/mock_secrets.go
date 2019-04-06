// Copyright 2019 Tigera Inc. All rights reserved.

package puller

import (
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

type MockSecrets struct {
	SecretsData map[string][]byte
	Error       error
}

func (*MockSecrets) Create(*v1.Secret) (*v1.Secret, error) {
	panic("implement me")
}

func (*MockSecrets) Update(*v1.Secret) (*v1.Secret, error) {
	panic("implement me")
}

func (*MockSecrets) Delete(name string, options *v12.DeleteOptions) error {
	panic("implement me")
}

func (*MockSecrets) DeleteCollection(options *v12.DeleteOptions, listOptions v12.ListOptions) error {
	panic("implement me")
}

func (m *MockSecrets) Get(name string, options v12.GetOptions) (*v1.Secret, error) {
	return &v1.Secret{
		Data: m.SecretsData,
	}, m.Error
}

func (*MockSecrets) List(opts v12.ListOptions) (*v1.SecretList, error) {
	panic("implement me")
}

func (*MockSecrets) Watch(opts v12.ListOptions) (watch.Interface, error) {
	panic("implement me")
}

func (*MockSecrets) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.Secret, err error) {
	panic("implement me")
}
