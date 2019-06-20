// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package datastore

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	clientv3 "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
	"github.com/tigera/compliance/pkg/list"
	"github.com/tigera/compliance/pkg/resources"
)

var resourceHelpersMap = map[metav1.TypeMeta]*resourceHelper{
	resources.TypeK8sPods: {
		func(c ClientSet) (resources.ResourceList, error) {
			return c.CoreV1().Pods("").List(metav1.ListOptions{})
		},
	},
	resources.TypeK8sNamespaces: {
		func(c ClientSet) (resources.ResourceList, error) {
			return c.CoreV1().Namespaces().List(metav1.ListOptions{})
		},
	},
	resources.TypeK8sServiceAccounts: {
		func(c ClientSet) (resources.ResourceList, error) {
			return c.CoreV1().ServiceAccounts("").List(metav1.ListOptions{})
		},
	},
	resources.TypeK8sEndpoints: {
		func(c ClientSet) (resources.ResourceList, error) {
			return c.CoreV1().Endpoints("").List(metav1.ListOptions{})
		},
	},
	resources.TypeK8sServices: {
		func(c ClientSet) (resources.ResourceList, error) {
			return c.CoreV1().Services("").List(metav1.ListOptions{})
		},
	},
	resources.TypeK8sNetworkPolicies: {
		func(c ClientSet) (resources.ResourceList, error) {
			return c.NetworkingV1().NetworkPolicies("").List(metav1.ListOptions{})
		},
	},
	resources.TypeK8sNetworkPoliciesExtensions: {
		func(c ClientSet) (resources.ResourceList, error) {
			return c.NetworkingV1().NetworkPolicies("").List(metav1.ListOptions{})
		},
	},
	resources.TypeCalicoTiers: {
		func(c ClientSet) (resources.ResourceList, error) {
			return c.Tiers().List(metav1.ListOptions{})
		},
	},
	resources.TypeCalicoHostEndpoints: {
		func(c ClientSet) (resources.ResourceList, error) {
			return c.HostEndpoints().List(metav1.ListOptions{})
		},
	},
	resources.TypeCalicoGlobalNetworkSets: {
		func(c ClientSet) (resources.ResourceList, error) {
			return c.GlobalNetworkSets().List(metav1.ListOptions{})
		},
	},
	resources.TypeCalicoNetworkPolicies: {
		listCalicoNetworkPolicies,
	},
	resources.TypeCalicoGlobalNetworkPolicies: {
		listCalicoGlobalNetworkPolicies,
	},
}

type clientSet struct {
	kubernetes.Interface
	clientv3.ProjectcalicoV3Interface
}

func (cs *clientSet) RetrieveList(kind metav1.TypeMeta) (*list.TimestampedResourceList, error) {
	requestStartTime := metav1.Now()
	l, err := resourceHelpersMap[kind].listFunc(cs)
	if err != nil {
		return nil, err
	}
	requestCompletedTime := metav1.Now()

	// TODO(rlb): strictly speaking the list type is NOT the same as the items it contains.
	// List func succeeded. Overwrite the group/version/kind which k8s does not correctly populate.
	l.GetObjectKind().SetGroupVersionKind(kind.GroupVersionKind())
	return &list.TimestampedResourceList{
		ResourceList:              l,
		RequestStartedTimestamp:   requestStartTime,
		RequestCompletedTimestamp: requestCompletedTime,
	}, nil
}
