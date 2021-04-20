// Copyright (c) 2021 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"

	projectcalico "github.com/projectcalico/apiserver/pkg/apis/projectcalico"
)

// FakeGlobalAlertTemplates implements GlobalAlertTemplateInterface
type FakeGlobalAlertTemplates struct {
	Fake *FakeProjectcalico
}

var globalalerttemplatesResource = schema.GroupVersionResource{Group: "projectcalico.org", Version: "", Resource: "globalalerttemplates"}

var globalalerttemplatesKind = schema.GroupVersionKind{Group: "projectcalico.org", Version: "", Kind: "GlobalAlertTemplate"}

// Get takes name of the globalAlertTemplate, and returns the corresponding globalAlertTemplate object, and an error if there is any.
func (c *FakeGlobalAlertTemplates) Get(ctx context.Context, name string, options v1.GetOptions) (result *projectcalico.GlobalAlertTemplate, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(globalalerttemplatesResource, name), &projectcalico.GlobalAlertTemplate{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.GlobalAlertTemplate), err
}

// List takes label and field selectors, and returns the list of GlobalAlertTemplates that match those selectors.
func (c *FakeGlobalAlertTemplates) List(ctx context.Context, opts v1.ListOptions) (result *projectcalico.GlobalAlertTemplateList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(globalalerttemplatesResource, globalalerttemplatesKind, opts), &projectcalico.GlobalAlertTemplateList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &projectcalico.GlobalAlertTemplateList{ListMeta: obj.(*projectcalico.GlobalAlertTemplateList).ListMeta}
	for _, item := range obj.(*projectcalico.GlobalAlertTemplateList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested globalAlertTemplates.
func (c *FakeGlobalAlertTemplates) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(globalalerttemplatesResource, opts))
}

// Create takes the representation of a globalAlertTemplate and creates it.  Returns the server's representation of the globalAlertTemplate, and an error, if there is any.
func (c *FakeGlobalAlertTemplates) Create(ctx context.Context, globalAlertTemplate *projectcalico.GlobalAlertTemplate, opts v1.CreateOptions) (result *projectcalico.GlobalAlertTemplate, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(globalalerttemplatesResource, globalAlertTemplate), &projectcalico.GlobalAlertTemplate{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.GlobalAlertTemplate), err
}

// Update takes the representation of a globalAlertTemplate and updates it. Returns the server's representation of the globalAlertTemplate, and an error, if there is any.
func (c *FakeGlobalAlertTemplates) Update(ctx context.Context, globalAlertTemplate *projectcalico.GlobalAlertTemplate, opts v1.UpdateOptions) (result *projectcalico.GlobalAlertTemplate, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(globalalerttemplatesResource, globalAlertTemplate), &projectcalico.GlobalAlertTemplate{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.GlobalAlertTemplate), err
}

// Delete takes name of the globalAlertTemplate and deletes it. Returns an error if one occurs.
func (c *FakeGlobalAlertTemplates) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(globalalerttemplatesResource, name), &projectcalico.GlobalAlertTemplate{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeGlobalAlertTemplates) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(globalalerttemplatesResource, listOpts)

	_, err := c.Fake.Invokes(action, &projectcalico.GlobalAlertTemplateList{})
	return err
}

// Patch applies the patch and returns the patched globalAlertTemplate.
func (c *FakeGlobalAlertTemplates) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *projectcalico.GlobalAlertTemplate, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(globalalerttemplatesResource, name, pt, data, subresources...), &projectcalico.GlobalAlertTemplate{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.GlobalAlertTemplate), err
}
