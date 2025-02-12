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

// SecurityEventWebhooksGetter has a method to return a SecurityEventWebhookInterface.
// A group's client should implement this interface.
type SecurityEventWebhooksGetter interface {
	SecurityEventWebhooks() SecurityEventWebhookInterface
}

// SecurityEventWebhookInterface has methods to work with SecurityEventWebhook resources.
type SecurityEventWebhookInterface interface {
	Create(ctx context.Context, securityEventWebhook *v3.SecurityEventWebhook, opts v1.CreateOptions) (*v3.SecurityEventWebhook, error)
	Update(ctx context.Context, securityEventWebhook *v3.SecurityEventWebhook, opts v1.UpdateOptions) (*v3.SecurityEventWebhook, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, securityEventWebhook *v3.SecurityEventWebhook, opts v1.UpdateOptions) (*v3.SecurityEventWebhook, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v3.SecurityEventWebhook, error)
	List(ctx context.Context, opts v1.ListOptions) (*v3.SecurityEventWebhookList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.SecurityEventWebhook, err error)
	SecurityEventWebhookExpansion
}

// securityEventWebhooks implements SecurityEventWebhookInterface
type securityEventWebhooks struct {
	*gentype.ClientWithList[*v3.SecurityEventWebhook, *v3.SecurityEventWebhookList]
}

// newSecurityEventWebhooks returns a SecurityEventWebhooks
func newSecurityEventWebhooks(c *ProjectcalicoV3Client) *securityEventWebhooks {
	return &securityEventWebhooks{
		gentype.NewClientWithList[*v3.SecurityEventWebhook, *v3.SecurityEventWebhookList](
			"securityeventwebhooks",
			c.RESTClient(),
			scheme.ParameterCodec,
			"",
			func() *v3.SecurityEventWebhook { return &v3.SecurityEventWebhook{} },
			func() *v3.SecurityEventWebhookList { return &v3.SecurityEventWebhookList{} }),
	}
}
