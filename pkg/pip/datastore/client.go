// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package datastore

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/tigera/compliance/pkg/list"

	calicoclient "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset"
	clientv3 "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
)

// ClientSet is a combined Calico/Kubernetes client set interface,
// code.
type ClientSet interface {
	kubernetes.Interface
	clientv3.ProjectcalicoV3Interface
	list.Source
}

// GetClientSet returns a client set (k8s and Calico)
func GetClientSet(config *rest.Config) (ClientSet, error) {
	k, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	c, err := calicoclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &clientSet{
		k,
		c.ProjectcalicoV3(),
	}, nil
}
