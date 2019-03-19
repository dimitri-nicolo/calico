// Copyright 2019 Tigera Inc. All rights reserved.

package mock

import (
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

type Secrets struct {
	SecretsData map[string][]byte
	Error       error
}

func (*Secrets) Create(*v1.Secret) (*v1.Secret, error) {
	panic("implement me")
}

func (*Secrets) Update(*v1.Secret) (*v1.Secret, error) {
	panic("implement me")
}

func (*Secrets) Delete(name string, options *v12.DeleteOptions) error {
	panic("implement me")
}

func (*Secrets) DeleteCollection(options *v12.DeleteOptions, listOptions v12.ListOptions) error {
	panic("implement me")
}

func (m *Secrets) Get(name string, options v12.GetOptions) (*v1.Secret, error) {
	return &v1.Secret{
		Data: m.SecretsData,
	}, m.Error
}

func (*Secrets) List(opts v12.ListOptions) (*v1.SecretList, error) {
	panic("implement me")
}

func (*Secrets) Watch(opts v12.ListOptions) (watch.Interface, error) {
	panic("implement me")
}

func (*Secrets) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.Secret, err error) {
	panic("implement me")
}
