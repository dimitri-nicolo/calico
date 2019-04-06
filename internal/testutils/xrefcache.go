// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package testutils

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	"github.com/tigera/compliance/pkg/resources"
	"github.com/tigera/compliance/pkg/syncer"
	"github.com/tigera/compliance/pkg/xrefcache"
)

const (
	Label1 byte = 1 << iota
	Label2
	Label3
	Label4
	Label5
	Label6
)

const (
	NoLabels = byte(0)
)

const (
	Name1 int = iota
	Name2
	Name3
	Name4
)

const (
	Namespace1 int = iota
	Namespace2
	Namespace3
	Namespace4
)

func NewXrefCacheTester() *XRefCacheTester {
	return &XRefCacheTester{
		xrefCache: xrefcache.NewXrefCache(),
	}
}

func labelByteToLabels(l byte) map[string]string {
	labels := make(map[string]string)
	for i := uint(0); i < 8; i++ {
		if (l>>i)&1 == 1 {
			labels[fmt.Sprintf("label%d")] = ""
		}
	}
	return labels
}

func labelByteToSelector(l byte) string {
	if l == 0 {
		return "all()"
	}

	sels := []string{}
	for i := uint(0); i < 8; i++ {
		if (l>>i)&1 == 1 {
			sels = append(sels, fmt.Sprintf("has(label%d)", i))
		}
	}
	return strings.Join(sels, " && ")
}

func labelByteToK8sSelector(l byte) metav1.LabelSelector {
	sel := metav1.LabelSelector{}
	for i := uint(0); i < 8; i++ {
		if (l>>i)&1 == 1 {
			sel.MatchExpressions = append(sel.MatchExpressions, metav1.LabelSelectorRequirement{
				Key:      fmt.Sprintf("label%d", i),
				Operator: metav1.LabelSelectorOpExists,
			})

		}
	}
	return sel
}

func getResourceId(kind schema.GroupVersionKind, nameIdx, namespaceIdx int) resources.ResourceID {
	name := fmt.Sprintf("%s-%d", strings.ToLower(kind.Kind), nameIdx)
	var namespace string
	if namespaceIdx > 0 {
		namespace = fmt.Sprintf("namespace-%d")
	}
	return resources.ResourceID{
		GroupVersionKind: kind,
		NameNamespace: resources.NameNamespace{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func getTypeMeta(r resources.ResourceID) metav1.TypeMeta {
	return metav1.TypeMeta{
		Kind:       r.Kind,
		APIVersion: r.GroupVersion().String(),
	}
}

func getObjectMeta(r resources.ResourceID, labels byte) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      r.Name,
		Namespace: r.Namespace,
		Labels:    labelByteToLabels(labels),
	}
}

type XRefCacheTester struct {
	xrefCache xrefcache.XrefCache
}

//
// -- HostEndpoint access --
//

func (t *XRefCacheTester) GetHostEndpoint(nameIdx int) *xrefcache.CacheEntryCalicoNetworkSet {
	r := getResourceId(resources.ResourceTypeHostEndpoints, nameIdx, 0)
	e := t.xrefCache.Get(r)
	if e == nil {
		return nil
	}
	return e.(*xrefcache.CacheEntryCalicoNetworkSet)
}

func (t *XRefCacheTester) SetHostEndpoint(nameIdx int, labels byte, nets []string) {
	r := getResourceId(resources.ResourceTypeHostEndpoints, nameIdx, 0)
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeSet,
		ResourceID: r,
		Resource: &apiv3.HostEndpoint{
			TypeMeta:   getTypeMeta(r),
			ObjectMeta: getObjectMeta(r, labels),
			Spec:       apiv3.HostEndpointSpec{},
		},
	})
}

func (t *XRefCacheTester) DeleteHostEndpoint(nameIdx int) {
	r := getResourceId(resources.ResourceTypeHostEndpoints, nameIdx, 0)
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeDeleted,
		ResourceID: r,
	})
}

//
// -- Tier access --
//

func (t *XRefCacheTester) GetTier(nameIdx int) *xrefcache.CacheEntryCalicoNetworkSet {
	r := getResourceId(resources.ResourceTypeTiers, nameIdx, 0)
	e := t.xrefCache.Get(r)
	if e == nil {
		return nil
	}
	return e.(*xrefcache.CacheEntryCalicoNetworkSet)
}

func (t *XRefCacheTester) SetTier(nameIdx int, order float64) {
	r := getResourceId(resources.ResourceTypeTiers, nameIdx, 0)
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeSet,
		ResourceID: r,
		Resource: &apiv3.Tier{
			TypeMeta:   getTypeMeta(r),
			ObjectMeta: getObjectMeta(r, 0),
			Spec: apiv3.TierSpec{
				Order: &order,
			},
		},
	})
}

func (t *XRefCacheTester) DeleteTier(nameIdx int) {
	r := getResourceId(resources.ResourceTypeTiers, nameIdx, 0)
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeDeleted,
		ResourceID: r,
	})
}

//
// -- GlobalNetworkSet access --
//

func (t *XRefCacheTester) GetGlobalNetworkSet(nameIdx int) *xrefcache.CacheEntryCalicoNetworkSet {
	r := getResourceId(resources.ResourceTypeGlobalNetworkSets, nameIdx, 0)
	e := t.xrefCache.Get(r)
	if e == nil {
		return nil
	}
	return e.(*xrefcache.CacheEntryCalicoNetworkSet)
}

func (t *XRefCacheTester) SetGlobalNetworkSet(nameIdx int, labels byte, nets []string) {
	r := getResourceId(resources.ResourceTypeGlobalNetworkSets, nameIdx, 0)
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeSet,
		ResourceID: r,
		Resource: &apiv3.GlobalNetworkSet{
			TypeMeta:   getTypeMeta(r),
			ObjectMeta: getObjectMeta(r, labels),
			Spec: apiv3.GlobalNetworkSetSpec{
				Nets: nets,
			},
		},
	})
}

func (t *XRefCacheTester) DeleteGlobalNetworkSet(nameIdx int) {
	r := getResourceId(resources.ResourceTypeGlobalNetworkSets, nameIdx, 0)
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeDeleted,
		ResourceID: r,
	})
}

//
// -- Calico GlobalNetworkPolicy access --
//

func (t *XRefCacheTester) GetGlobalNetworkPolicy(nameIdx int) *xrefcache.CacheEntryNetworkPolicy {
	r := getResourceId(resources.ResourceTypeGlobalNetworkPolicies, nameIdx, 0)
	e := t.xrefCache.Get(r)
	if e == nil {
		return nil
	}
	return e.(*xrefcache.CacheEntryNetworkPolicy)
}

func (t *XRefCacheTester) SetGlobalNetworkPolicy(nameIdx int, labelSelector byte, ingress, egress []apiv3.Rule) {
	r := getResourceId(resources.ResourceTypeGlobalNetworkPolicies, nameIdx, 0)
	types := []apiv3.PolicyType{}
	if ingress != nil {
		types = append(types, apiv3.PolicyTypeIngress)
	}
	if egress != nil {
		types = append(types, apiv3.PolicyTypeEgress)
	}
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeSet,
		ResourceID: r,
		Resource: &apiv3.GlobalNetworkPolicy{
			TypeMeta:   getTypeMeta(r),
			ObjectMeta: getObjectMeta(r, labelSelector),
			Spec: apiv3.GlobalNetworkPolicySpec{
				Selector: labelByteToSelector(labelSelector),
				Ingress:  ingress,
				Egress:   egress,
				Types:    types,
			},
		},
	})
}

func (t *XRefCacheTester) DeleteGlobalNetworkPolicy(nameIdx int) {
	r := getResourceId(resources.ResourceTypeGlobalNetworkPolicies, nameIdx, 0)
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeDeleted,
		ResourceID: r,
	})
}

//
// -- Calico NetworkPolicy access --
//

func (t *XRefCacheTester) GetNetworkPolicy(nameIdx, namespaceIdx int) *xrefcache.CacheEntryNetworkPolicy {
	r := getResourceId(resources.ResourceTypeNetworkPolicies, nameIdx, namespaceIdx)
	e := t.xrefCache.Get(r)
	if e == nil {
		return nil
	}
	return e.(*xrefcache.CacheEntryNetworkPolicy)
}

func (t *XRefCacheTester) SetNetworkPolicy(nameIdx int, labelSelector byte, ingress, egress []apiv3.Rule) {
	r := getResourceId(resources.ResourceTypeNetworkPolicies, nameIdx, 0)
	types := []apiv3.PolicyType{}
	if ingress != nil {
		types = append(types, apiv3.PolicyTypeIngress)
	}
	if egress != nil {
		types = append(types, apiv3.PolicyTypeEgress)
	}
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeSet,
		ResourceID: r,
		Resource: &apiv3.NetworkPolicy{
			TypeMeta:   getTypeMeta(r),
			ObjectMeta: getObjectMeta(r, labelSelector),
			Spec: apiv3.NetworkPolicySpec{
				Selector: labelByteToSelector(labelSelector),
				Ingress:  ingress,
				Egress:   egress,
				Types:    types,
			},
		},
	})
}

func (t *XRefCacheTester) DeleteNetworkPolicy(nameIdx, namespaceIdx int) {
	r := getResourceId(resources.ResourceTypeNetworkPolicies, nameIdx, namespaceIdx)
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeDeleted,
		ResourceID: r,
	})
}

//
// -- K8s NetworkPolicy access --
//

func (t *XRefCacheTester) GetK8sNetworkPolicy(nameIdx, namespaceIdx int) *xrefcache.CacheEntryNetworkPolicy {
	r := getResourceId(resources.ResourceTypeK8sNetworkPolicies, nameIdx, namespaceIdx)
	e := t.xrefCache.Get(r)
	if e == nil {
		return nil
	}
	return e.(*xrefcache.CacheEntryNetworkPolicy)
}

func (t *XRefCacheTester) SetK8sNetworkPolicy(
	nameIdx int, labelSelector byte,
	ingress []networkingv1.NetworkPolicyIngressRule,
	egress []networkingv1.NetworkPolicyEgressRule,
) {
	r := getResourceId(resources.ResourceTypeK8sNetworkPolicies, nameIdx, 0)
	types := []networkingv1.PolicyType{}
	if ingress != nil {
		types = append(types, networkingv1.PolicyTypeIngress)
	}
	if egress != nil {
		types = append(types, networkingv1.PolicyTypeEgress)
	}
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeSet,
		ResourceID: r,
		Resource: &networkingv1.NetworkPolicy{
			TypeMeta:   getTypeMeta(r),
			ObjectMeta: getObjectMeta(r, labelSelector),
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: labelByteToK8sSelector(labelSelector),
				PolicyTypes: types,
				Ingress:     ingress,
				Egress:      egress,
			},
		},
	})
}

func (t *XRefCacheTester) DeleteK8sNetworkPolicy(nameIdx, namespaceIdx int) {
	r := getResourceId(resources.ResourceTypeK8sNetworkPolicies, nameIdx, namespaceIdx)
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeDeleted,
		ResourceID: r,
	})
}

//
// -- K8s Pod access --
//

func (t *XRefCacheTester) GetPod(nameIdx, namespaceIdx int) *xrefcache.CacheEntryEndpoint {
	r := getResourceId(resources.ResourceTypePods, nameIdx, namespaceIdx)
	e := t.xrefCache.Get(r)
	if e == nil {
		return nil
	}
	return e.(*xrefcache.CacheEntryEndpoint)
}

func (t *XRefCacheTester) SetPod(nameIdx, namespaceIdx int, labels byte, ip string) {
	r := getResourceId(resources.ResourceTypePods, nameIdx, namespaceIdx)
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeSet,
		ResourceID: r,
		Resource: &corev1.Pod{
			TypeMeta:   getTypeMeta(r),
			ObjectMeta: getObjectMeta(r, labels),
			Spec:       corev1.PodSpec{},
		},
	})
}

func (t *XRefCacheTester) DeletePod(nameIdx, namespaceIdx int) {
	r := getResourceId(resources.ResourceTypePods, nameIdx, namespaceIdx)
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeDeleted,
		ResourceID: r,
	})
}

//
// -- K8s Endpoints access --
//

func (t *XRefCacheTester) GetEndpoint(nameIdx, namespaceIdx int) *xrefcache.CacheEntryK8sServiceEndpoints {
	r := getResourceId(resources.ResourceTypeEndpoints, nameIdx, namespaceIdx)
	e := t.xrefCache.Get(r)
	if e == nil {
		return nil
	}
	return e.(*xrefcache.CacheEntryK8sServiceEndpoints)
}

func (t *XRefCacheTester) SetEndpoint(nameIdx, namespaceIdx int, labels byte, ip string) {
	r := getResourceId(resources.ResourceTypeEndpoints, nameIdx, namespaceIdx)
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeSet,
		ResourceID: r,
		Resource: &corev1.Endpoints{
			TypeMeta:   getTypeMeta(r),
			ObjectMeta: getObjectMeta(r, labels),
			Subsets:    []corev1.EndpointSubset{},
		},
	})
}

func (t *XRefCacheTester) DeleteEndpoint(nameIdx, namespaceIdx int) {
	r := getResourceId(resources.ResourceTypeEndpoints, nameIdx, 0)
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeDeleted,
		ResourceID: r,
	})
	metav1.Now()
}

//
// -- K8s ServiceAccounts access --
//

func (t *XRefCacheTester) GetServiceAccount(nameIdx, namespaceIdx int) *xrefcache.CacheEntryK8sServiceAccount {
	r := getResourceId(resources.ResourceTypeServiceAccounts, nameIdx, namespaceIdx)
	e := t.xrefCache.Get(r)
	if e == nil {
		return nil
	}
	return e.(*xrefcache.CacheEntryK8sServiceAccount)
}

func (t *XRefCacheTester) SetServiceAccount(nameIdx, namespaceIdx int, labels byte, ip string) {
	r := getResourceId(resources.ResourceTypeServiceAccounts, nameIdx, namespaceIdx)
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeSet,
		ResourceID: r,
		Resource: &corev1.ServiceAccount{
			TypeMeta:   getTypeMeta(r),
			ObjectMeta: getObjectMeta(r, labels),
		},
	})
}

func (t *XRefCacheTester) DeleteServiceAccount(nameIdx, namespaceIdx int) {
	r := getResourceId(resources.ResourceTypeServiceAccounts, nameIdx, 0)
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeDeleted,
		ResourceID: r,
	})
}

//
// -- K8s Namespaces access --
//

func (t *XRefCacheTester) GetNamespace(nameIdx, namespaceIdx int) *xrefcache.CacheEntryK8sNamespace {
	r := getResourceId(resources.ResourceTypeNamespaces, nameIdx, namespaceIdx)
	e := t.xrefCache.Get(r)
	if e == nil {
		return nil
	}
	return e.(*xrefcache.CacheEntryK8sNamespace)
}

func (t *XRefCacheTester) SetNamespace(nameIdx, namespaceIdx int, labels byte, ip string) {
	r := getResourceId(resources.ResourceTypeNamespaces, nameIdx, namespaceIdx)
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeSet,
		ResourceID: r,
		Resource: &corev1.Namespace{
			TypeMeta:   getTypeMeta(r),
			ObjectMeta: getObjectMeta(r, labels),
		},
	})
}

func (t *XRefCacheTester) DeleteNamespace(nameIdx, namespaceIdx int) {
	r := getResourceId(resources.ResourceTypeNamespaces, nameIdx, 0)
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeDeleted,
		ResourceID: r,
	})
}
