// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Code generated by informer-gen. DO NOT EDIT.

package internalversion

import (
	"fmt"

	projectcalico "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	cache "k8s.io/client-go/tools/cache"
)

// GenericInformer is type of SharedIndexInformer which will locate and delegate to other
// sharedInformers based on type
type GenericInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() cache.GenericLister
}

type genericInformer struct {
	informer cache.SharedIndexInformer
	resource schema.GroupResource
}

// Informer returns the SharedIndexInformer.
func (f *genericInformer) Informer() cache.SharedIndexInformer {
	return f.informer
}

// Lister returns the GenericLister.
func (f *genericInformer) Lister() cache.GenericLister {
	return cache.NewGenericLister(f.Informer().GetIndexer(), f.resource)
}

// ForResource gives generic access to a shared informer of the matching type
// TODO extend this to unknown resources with a client pool
func (f *sharedInformerFactory) ForResource(resource schema.GroupVersionResource) (GenericInformer, error) {
	switch resource {
	// Group=projectcalico.org, Version=internalVersion
	case projectcalico.SchemeGroupVersion.WithResource("bgpconfigurations"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Projectcalico().InternalVersion().BGPConfigurations().Informer()}, nil
	case projectcalico.SchemeGroupVersion.WithResource("bgppeers"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Projectcalico().InternalVersion().BGPPeers().Informer()}, nil
	case projectcalico.SchemeGroupVersion.WithResource("globalnetworkpolicies"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Projectcalico().InternalVersion().GlobalNetworkPolicies().Informer()}, nil
	case projectcalico.SchemeGroupVersion.WithResource("globalnetworksets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Projectcalico().InternalVersion().GlobalNetworkSets().Informer()}, nil
	case projectcalico.SchemeGroupVersion.WithResource("globalreports"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Projectcalico().InternalVersion().GlobalReports().Informer()}, nil
	case projectcalico.SchemeGroupVersion.WithResource("globalreporttypes"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Projectcalico().InternalVersion().GlobalReportTypes().Informer()}, nil
	case projectcalico.SchemeGroupVersion.WithResource("globalthreatfeeds"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Projectcalico().InternalVersion().GlobalThreatFeeds().Informer()}, nil
	case projectcalico.SchemeGroupVersion.WithResource("hostendpoints"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Projectcalico().InternalVersion().HostEndpoints().Informer()}, nil
	case projectcalico.SchemeGroupVersion.WithResource("ippools"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Projectcalico().InternalVersion().IPPools().Informer()}, nil
	case projectcalico.SchemeGroupVersion.WithResource("licensekeys"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Projectcalico().InternalVersion().LicenseKeys().Informer()}, nil
	case projectcalico.SchemeGroupVersion.WithResource("networkpolicies"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Projectcalico().InternalVersion().NetworkPolicies().Informer()}, nil
	case projectcalico.SchemeGroupVersion.WithResource("profiles"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Projectcalico().InternalVersion().Profiles().Informer()}, nil
	case projectcalico.SchemeGroupVersion.WithResource("remoteclusterconfigurations"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Projectcalico().InternalVersion().RemoteClusterConfigurations().Informer()}, nil
	case projectcalico.SchemeGroupVersion.WithResource("tiers"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Projectcalico().InternalVersion().Tiers().Informer()}, nil

	}

	return nil, fmt.Errorf("no informer found for %v", resource)
}
