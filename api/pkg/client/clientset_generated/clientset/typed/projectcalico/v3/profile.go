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

// ProfilesGetter has a method to return a ProfileInterface.
// A group's client should implement this interface.
type ProfilesGetter interface {
	Profiles() ProfileInterface
}

// ProfileInterface has methods to work with Profile resources.
type ProfileInterface interface {
	Create(ctx context.Context, profile *v3.Profile, opts v1.CreateOptions) (*v3.Profile, error)
	Update(ctx context.Context, profile *v3.Profile, opts v1.UpdateOptions) (*v3.Profile, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v3.Profile, error)
	List(ctx context.Context, opts v1.ListOptions) (*v3.ProfileList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.Profile, err error)
	ProfileExpansion
}

// profiles implements ProfileInterface
type profiles struct {
	*gentype.ClientWithList[*v3.Profile, *v3.ProfileList]
}

// newProfiles returns a Profiles
func newProfiles(c *ProjectcalicoV3Client) *profiles {
	return &profiles{
		gentype.NewClientWithList[*v3.Profile, *v3.ProfileList](
			"profiles",
			c.RESTClient(),
			scheme.ParameterCodec,
			"",
			func() *v3.Profile { return &v3.Profile{} },
			func() *v3.ProfileList { return &v3.ProfileList{} }),
	}
}
