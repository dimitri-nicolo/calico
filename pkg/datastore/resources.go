// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package datastore

import (
	"context"
	"strings"

	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/options"

	"github.com/tigera/compliance/pkg/resources"
)

type resourceHelper struct {
	listFunc func(ClientSet) (resources.ResourceList, error)
}

var (
	resourceHelpersMap = map[schema.GroupVersionKind]*resourceHelper{
		resources.ResourceTypePods: {
			func(c ClientSet) (resources.ResourceList, error) {
				return c.CoreV1().Pods("").List(metav1.ListOptions{})
			},
		},
		resources.ResourceTypeNamespaces: {
			func(c ClientSet) (resources.ResourceList, error) {
				return c.CoreV1().Namespaces().List(metav1.ListOptions{})
			},
		},
		resources.ResourceTypeServiceAccounts: {
			func(c ClientSet) (resources.ResourceList, error) {
				return c.CoreV1().ServiceAccounts("").List(metav1.ListOptions{})
			},
		},
		resources.ResourceTypeEndpoints: {
			func(c ClientSet) (resources.ResourceList, error) {
				return c.CoreV1().Endpoints("").List(metav1.ListOptions{})
			},
		},
		resources.ResourceTypeServices: {
			func(c ClientSet) (resources.ResourceList, error) {
				return c.CoreV1().Services("").List(metav1.ListOptions{})
			},
		},
		resources.ResourceTypeK8sNetworkPolicies: {
			func(c ClientSet) (resources.ResourceList, error) {
				return c.NetworkingV1().NetworkPolicies("").List(metav1.ListOptions{})
			},
		},
		resources.ResourceTypeHostEndpoints: {
			func(c ClientSet) (resources.ResourceList, error) {
				return c.HostEndpoints().List(context.Background(), options.ListOptions{})
			},
		},
		resources.ResourceTypeGlobalNetworkSets: {
			func(c ClientSet) (resources.ResourceList, error) {
				return c.GlobalNetworkSets().List(context.Background(), options.ListOptions{})
			},
		},
		resources.ResourceTypeNetworkPolicies: {
			listNetworkPolicies,
		},
		resources.ResourceTypeGlobalNetworkPolicies: {
			func(c ClientSet) (resources.ResourceList, error) {
				return c.GlobalNetworkPolicies().List(context.Background(), options.ListOptions{})
			},
		},
	}
)

func listNetworkPolicies(c ClientSet) (resources.ResourceList, error) {
	// List network policies.
	npList, err := c.NetworkPolicies().List(context.Background(), options.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Filter out kubernetes network policies.
	nps := []apiv3.NetworkPolicy{}
	for _, np := range npList.Items {
		if strings.HasPrefix(np.Name, "knp.") {
			log.WithField("np", np.Name).Debug("passing on kubernetes network policy")
			continue
		}
		nps = append(nps, np)
	}

	// Over the network policy items with the filtered list.
	npList.Items = nps
	return npList, nil
}
