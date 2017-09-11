/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package fake

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
	v1beta1 "k8s.io/kubernetes/pkg/apis/apps/v1beta1"
)

// FakeControllerRevisions implements ControllerRevisionInterface
type FakeControllerRevisions struct {
	Fake *FakeAppsV1beta1
	ns   string
}

var controllerrevisionsResource = schema.GroupVersionResource{Group: "apps", Version: "v1beta1", Resource: "controllerrevisions"}

var controllerrevisionsKind = schema.GroupVersionKind{Group: "apps", Version: "v1beta1", Kind: "ControllerRevision"}

func (c *FakeControllerRevisions) Create(controllerRevision *v1beta1.ControllerRevision) (result *v1beta1.ControllerRevision, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(controllerrevisionsResource, c.ns, controllerRevision), &v1beta1.ControllerRevision{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.ControllerRevision), err
}

func (c *FakeControllerRevisions) Update(controllerRevision *v1beta1.ControllerRevision) (result *v1beta1.ControllerRevision, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(controllerrevisionsResource, c.ns, controllerRevision), &v1beta1.ControllerRevision{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.ControllerRevision), err
}

func (c *FakeControllerRevisions) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(controllerrevisionsResource, c.ns, name), &v1beta1.ControllerRevision{})

	return err
}

func (c *FakeControllerRevisions) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(controllerrevisionsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1beta1.ControllerRevisionList{})
	return err
}

func (c *FakeControllerRevisions) Get(name string, options v1.GetOptions) (result *v1beta1.ControllerRevision, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(controllerrevisionsResource, c.ns, name), &v1beta1.ControllerRevision{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.ControllerRevision), err
}

func (c *FakeControllerRevisions) List(opts v1.ListOptions) (result *v1beta1.ControllerRevisionList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(controllerrevisionsResource, controllerrevisionsKind, c.ns, opts), &v1beta1.ControllerRevisionList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1beta1.ControllerRevisionList{}
	for _, item := range obj.(*v1beta1.ControllerRevisionList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested controllerRevisions.
func (c *FakeControllerRevisions) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(controllerrevisionsResource, c.ns, opts))

}

// Patch applies the patch and returns the patched controllerRevision.
func (c *FakeControllerRevisions) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1beta1.ControllerRevision, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(controllerrevisionsResource, c.ns, name, data, subresources...), &v1beta1.ControllerRevision{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.ControllerRevision), err
}
