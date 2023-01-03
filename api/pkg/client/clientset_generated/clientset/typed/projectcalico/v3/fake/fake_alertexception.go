// Copyright (c) 2023 Tigera, Inc. All rights reserved.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeAlertExceptions implements AlertExceptionInterface
type FakeAlertExceptions struct {
	Fake *FakeProjectcalicoV3
}

var alertexceptionsResource = schema.GroupVersionResource{Group: "projectcalico.org", Version: "v3", Resource: "alertexceptions"}

var alertexceptionsKind = schema.GroupVersionKind{Group: "projectcalico.org", Version: "v3", Kind: "AlertException"}

// Get takes name of the alertException, and returns the corresponding alertException object, and an error if there is any.
func (c *FakeAlertExceptions) Get(ctx context.Context, name string, options v1.GetOptions) (result *v3.AlertException, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(alertexceptionsResource, name), &v3.AlertException{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.AlertException), err
}

// List takes label and field selectors, and returns the list of AlertExceptions that match those selectors.
func (c *FakeAlertExceptions) List(ctx context.Context, opts v1.ListOptions) (result *v3.AlertExceptionList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(alertexceptionsResource, alertexceptionsKind, opts), &v3.AlertExceptionList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v3.AlertExceptionList{ListMeta: obj.(*v3.AlertExceptionList).ListMeta}
	for _, item := range obj.(*v3.AlertExceptionList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested alertExceptions.
func (c *FakeAlertExceptions) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(alertexceptionsResource, opts))
}

// Create takes the representation of a alertException and creates it.  Returns the server's representation of the alertException, and an error, if there is any.
func (c *FakeAlertExceptions) Create(ctx context.Context, alertException *v3.AlertException, opts v1.CreateOptions) (result *v3.AlertException, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(alertexceptionsResource, alertException), &v3.AlertException{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.AlertException), err
}

// Update takes the representation of a alertException and updates it. Returns the server's representation of the alertException, and an error, if there is any.
func (c *FakeAlertExceptions) Update(ctx context.Context, alertException *v3.AlertException, opts v1.UpdateOptions) (result *v3.AlertException, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(alertexceptionsResource, alertException), &v3.AlertException{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.AlertException), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeAlertExceptions) UpdateStatus(ctx context.Context, alertException *v3.AlertException, opts v1.UpdateOptions) (*v3.AlertException, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(alertexceptionsResource, "status", alertException), &v3.AlertException{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.AlertException), err
}

// Delete takes name of the alertException and deletes it. Returns an error if one occurs.
func (c *FakeAlertExceptions) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteActionWithOptions(alertexceptionsResource, name, opts), &v3.AlertException{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeAlertExceptions) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(alertexceptionsResource, listOpts)

	_, err := c.Fake.Invokes(action, &v3.AlertExceptionList{})
	return err
}

// Patch applies the patch and returns the patched alertException.
func (c *FakeAlertExceptions) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.AlertException, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(alertexceptionsResource, name, pt, data, subresources...), &v3.AlertException{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.AlertException), err
}
