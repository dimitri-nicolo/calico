// Copyright (c) 2020 Tigera, Inc. All rights reserved.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	projectcalico "github.com/tigera/apiserver/pkg/apis/projectcalico"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeManagedClusters implements ManagedClusterInterface
type FakeManagedClusters struct {
	Fake *FakeProjectcalico
}

var managedclustersResource = schema.GroupVersionResource{Group: "projectcalico.org", Version: "", Resource: "managedclusters"}

var managedclustersKind = schema.GroupVersionKind{Group: "projectcalico.org", Version: "", Kind: "ManagedCluster"}

// Get takes name of the managedCluster, and returns the corresponding managedCluster object, and an error if there is any.
func (c *FakeManagedClusters) Get(name string, options v1.GetOptions) (result *projectcalico.ManagedCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(managedclustersResource, name), &projectcalico.ManagedCluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.ManagedCluster), err
}

// List takes label and field selectors, and returns the list of ManagedClusters that match those selectors.
func (c *FakeManagedClusters) List(opts v1.ListOptions) (result *projectcalico.ManagedClusterList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(managedclustersResource, managedclustersKind, opts), &projectcalico.ManagedClusterList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &projectcalico.ManagedClusterList{ListMeta: obj.(*projectcalico.ManagedClusterList).ListMeta}
	for _, item := range obj.(*projectcalico.ManagedClusterList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested managedClusters.
func (c *FakeManagedClusters) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(managedclustersResource, opts))
}

// Create takes the representation of a managedCluster and creates it.  Returns the server's representation of the managedCluster, and an error, if there is any.
func (c *FakeManagedClusters) Create(managedCluster *projectcalico.ManagedCluster) (result *projectcalico.ManagedCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(managedclustersResource, managedCluster), &projectcalico.ManagedCluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.ManagedCluster), err
}

// Update takes the representation of a managedCluster and updates it. Returns the server's representation of the managedCluster, and an error, if there is any.
func (c *FakeManagedClusters) Update(managedCluster *projectcalico.ManagedCluster) (result *projectcalico.ManagedCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(managedclustersResource, managedCluster), &projectcalico.ManagedCluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.ManagedCluster), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeManagedClusters) UpdateStatus(managedCluster *projectcalico.ManagedCluster) (*projectcalico.ManagedCluster, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(managedclustersResource, "status", managedCluster), &projectcalico.ManagedCluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.ManagedCluster), err
}

// Delete takes name of the managedCluster and deletes it. Returns an error if one occurs.
func (c *FakeManagedClusters) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(managedclustersResource, name), &projectcalico.ManagedCluster{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeManagedClusters) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(managedclustersResource, listOptions)

	_, err := c.Fake.Invokes(action, &projectcalico.ManagedClusterList{})
	return err
}

// Patch applies the patch and returns the patched managedCluster.
func (c *FakeManagedClusters) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *projectcalico.ManagedCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(managedclustersResource, name, pt, data, subresources...), &projectcalico.ManagedCluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.ManagedCluster), err
}
