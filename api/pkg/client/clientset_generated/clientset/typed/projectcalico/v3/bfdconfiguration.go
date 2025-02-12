// Copyright (c) 2025 Tigera, Inc. All rights reserved.

// Code generated by client-gen. DO NOT EDIT.

package v3

import (
	"context"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	scheme "github.com/tigera/api/pkg/client/clientset_generated/clientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	gentype "k8s.io/client-go/gentype"
)

// BFDConfigurationsGetter has a method to return a BFDConfigurationInterface.
// A group's client should implement this interface.
type BFDConfigurationsGetter interface {
	BFDConfigurations() BFDConfigurationInterface
}

// BFDConfigurationInterface has methods to work with BFDConfiguration resources.
type BFDConfigurationInterface interface {
	Create(ctx context.Context, bFDConfiguration *v3.BFDConfiguration, opts v1.CreateOptions) (*v3.BFDConfiguration, error)
	Update(ctx context.Context, bFDConfiguration *v3.BFDConfiguration, opts v1.UpdateOptions) (*v3.BFDConfiguration, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v3.BFDConfiguration, error)
	List(ctx context.Context, opts v1.ListOptions) (*v3.BFDConfigurationList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.BFDConfiguration, err error)
	BFDConfigurationExpansion
}

// bFDConfigurations implements BFDConfigurationInterface
type bFDConfigurations struct {
	*gentype.ClientWithList[*v3.BFDConfiguration, *v3.BFDConfigurationList]
}

// newBFDConfigurations returns a BFDConfigurations
func newBFDConfigurations(c *ProjectcalicoV3Client) *bFDConfigurations {
	return &bFDConfigurations{
		gentype.NewClientWithList[*v3.BFDConfiguration, *v3.BFDConfigurationList](
			"bfdconfigurations",
			c.RESTClient(),
			scheme.ParameterCodec,
			"",
			func() *v3.BFDConfiguration { return &v3.BFDConfiguration{} },
			func() *v3.BFDConfigurationList { return &v3.BFDConfigurationList{} }),
	}
}
