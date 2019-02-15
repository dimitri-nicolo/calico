// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Code generated by client-gen. DO NOT EDIT.

package clientset

import (
	glog "github.com/golang/glog"
	projectcalicov3 "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
	discovery "k8s.io/client-go/discovery"
	rest "k8s.io/client-go/rest"
	flowcontrol "k8s.io/client-go/util/flowcontrol"
)

type Interface interface {
	Discovery() discovery.DiscoveryInterface
	ProjectcalicoV3() projectcalicov3.ProjectcalicoV3Interface
	// Deprecated: please explicitly pick a version if possible.
	Projectcalico() projectcalicov3.ProjectcalicoV3Interface
}

// Clientset contains the clients for groups. Each group has exactly one
// version included in a Clientset.
type Clientset struct {
	*discovery.DiscoveryClient
	projectcalicoV3 *projectcalicov3.ProjectcalicoV3Client
}

// ProjectcalicoV3 retrieves the ProjectcalicoV3Client
func (c *Clientset) ProjectcalicoV3() projectcalicov3.ProjectcalicoV3Interface {
	return c.projectcalicoV3
}

// Deprecated: Projectcalico retrieves the default version of ProjectcalicoClient.
// Please explicitly pick a version.
func (c *Clientset) Projectcalico() projectcalicov3.ProjectcalicoV3Interface {
	return c.projectcalicoV3
}

// Discovery retrieves the DiscoveryClient
func (c *Clientset) Discovery() discovery.DiscoveryInterface {
	if c == nil {
		return nil
	}
	return c.DiscoveryClient
}

// NewForConfig creates a new Clientset for the given config.
func NewForConfig(c *rest.Config) (*Clientset, error) {
	configShallowCopy := *c
	if configShallowCopy.RateLimiter == nil && configShallowCopy.QPS > 0 {
		configShallowCopy.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(configShallowCopy.QPS, configShallowCopy.Burst)
	}
	var cs Clientset
	var err error
	cs.projectcalicoV3, err = projectcalicov3.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}

	cs.DiscoveryClient, err = discovery.NewDiscoveryClientForConfig(&configShallowCopy)
	if err != nil {
		glog.Errorf("failed to create the DiscoveryClient: %v", err)
		return nil, err
	}
	return &cs, nil
}

// NewForConfigOrDie creates a new Clientset for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *Clientset {
	var cs Clientset
	cs.projectcalicoV3 = projectcalicov3.NewForConfigOrDie(c)

	cs.DiscoveryClient = discovery.NewDiscoveryClientForConfigOrDie(c)
	return &cs
}

// New creates a new Clientset for the given RESTClient.
func New(c rest.Interface) *Clientset {
	var cs Clientset
	cs.projectcalicoV3 = projectcalicov3.New(c)

	cs.DiscoveryClient = discovery.NewDiscoveryClient(c)
	return &cs
}
