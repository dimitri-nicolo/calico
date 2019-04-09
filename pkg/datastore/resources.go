// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package datastore

import (
	"context"
	"strings"

	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/options"

	"github.com/tigera/compliance/pkg/resources"
)

type resourceHelper struct {
	listFunc func(ClientSet) (resources.ResourceList, error)
}

var (
	resourceHelpersMap = map[metav1.TypeMeta]*resourceHelper{
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
				return c.Tiers().List(context.Background(), options.ListOptions{})
			},
		},
		resources.TypeCalicoHostEndpoints: {
			func(c ClientSet) (resources.ResourceList, error) {
				return c.HostEndpoints().List(context.Background(), options.ListOptions{})
			},
		},
		resources.TypeCalicoGlobalNetworkSets: {
			func(c ClientSet) (resources.ResourceList, error) {
				return c.GlobalNetworkSets().List(context.Background(), options.ListOptions{})
			},
		},
		resources.TypeCalicoNetworkPolicies: {
			listNetworkPolicies,
		},
		resources.TypeCalicoGlobalNetworkPolicies: {
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
