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

// GlobalThreatFeedsGetter has a method to return a GlobalThreatFeedInterface.
// A group's client should implement this interface.
type GlobalThreatFeedsGetter interface {
	GlobalThreatFeeds() GlobalThreatFeedInterface
}

// GlobalThreatFeedInterface has methods to work with GlobalThreatFeed resources.
type GlobalThreatFeedInterface interface {
	Create(ctx context.Context, globalThreatFeed *v3.GlobalThreatFeed, opts v1.CreateOptions) (*v3.GlobalThreatFeed, error)
	Update(ctx context.Context, globalThreatFeed *v3.GlobalThreatFeed, opts v1.UpdateOptions) (*v3.GlobalThreatFeed, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, globalThreatFeed *v3.GlobalThreatFeed, opts v1.UpdateOptions) (*v3.GlobalThreatFeed, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v3.GlobalThreatFeed, error)
	List(ctx context.Context, opts v1.ListOptions) (*v3.GlobalThreatFeedList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.GlobalThreatFeed, err error)
	GlobalThreatFeedExpansion
}

// globalThreatFeeds implements GlobalThreatFeedInterface
type globalThreatFeeds struct {
	*gentype.ClientWithList[*v3.GlobalThreatFeed, *v3.GlobalThreatFeedList]
}

// newGlobalThreatFeeds returns a GlobalThreatFeeds
func newGlobalThreatFeeds(c *ProjectcalicoV3Client) *globalThreatFeeds {
	return &globalThreatFeeds{
		gentype.NewClientWithList[*v3.GlobalThreatFeed, *v3.GlobalThreatFeedList](
			"globalthreatfeeds",
			c.RESTClient(),
			scheme.ParameterCodec,
			"",
			func() *v3.GlobalThreatFeed { return &v3.GlobalThreatFeed{} },
			func() *v3.GlobalThreatFeedList { return &v3.GlobalThreatFeedList{} }),
	}
}
