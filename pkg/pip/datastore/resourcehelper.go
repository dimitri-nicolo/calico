// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package datastore

import (
	"strings"

	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	"github.com/tigera/compliance/pkg/resources"
)

type resourceHelper struct {
	listFunc func(ClientSet) (resources.ResourceList, error)
}

func listCalicoNetworkPolicies(c ClientSet) (resources.ResourceList, error) {
	// List tiers.
	tiers, err := c.Tiers().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// List network policies. When going through the AAPIS we need to list on a tier by tier basis.
	var npList *v3.NetworkPolicyList
	for _, tier := range tiers.Items {
		nps, err := c.NetworkPolicies("").List(metav1.ListOptions{
			LabelSelector: apiv3.LabelTier + "=" + tier.Name,
		})
		if err != nil {
			return nil, err
		}
		if npList == nil {
			npList = nps
		} else {
			npList.Items = append(npList.Items, nps.Items...)
		}
	}

	// Filter out kubernetes network policies.
	nps := []v3.NetworkPolicy{}
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

func listCalicoGlobalNetworkPolicies(c ClientSet) (resources.ResourceList, error) {
	// List tiers.
	tiers, err := c.Tiers().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// List global network policies. When going through the AAPIS we need to list on a tier by tier basis.
	var npList *v3.GlobalNetworkPolicyList
	for _, tier := range tiers.Items {
		nps, err := c.GlobalNetworkPolicies().List(metav1.ListOptions{
			LabelSelector: apiv3.LabelTier + "=" + tier.Name,
		})
		if err != nil {
			return nil, err
		}
		if npList == nil {
			npList = nps
		} else {
			npList.Items = append(npList.Items, nps.Items...)
		}
	}
	return npList, nil
}
