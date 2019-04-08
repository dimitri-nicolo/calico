// Copyright (c) 2019 Tigera, Inc. SelectAll rights reserved.
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

func NewXrefCacheTester() *XRefCacheTester {
	return &XRefCacheTester{
		xrefCache: xrefcache.NewXrefCache(),
	}
}

func ipByteToIPString(ip IP) string {
	switch ip {
	case IP1:
		return "192.168.0.0"
	case IP2:
		return "192.168.0.1"
	case IP3:
		return "192.168.10.0"
	case IP4:
		return "192.168.10.1"
	}
	return ""
}

func ipByteToIPStringSlice(ip IP) []string {
	var ips []string
	if ip&IP1 != 0 {
		ips = append(ips, ipByteToIPString(IP1))
	}
	if ip&IP2 != 0 {
		ips = append(ips, ipByteToIPString(IP2))
	}
	if ip&IP3 != 0 {
		ips = append(ips, ipByteToIPString(IP3))
	}
	if ip&IP4 != 0 {
		ips = append(ips, ipByteToIPString(IP4))
	}
	return ips
}

func labelByteToLabels(l Label) map[string]string {
	labels := make(map[string]string)
	for i := uint(0); i < 8; i++ {
		if l&(1<<i) != 0 {
			labels[fmt.Sprintf("label%d", i+1)] = ""
		}
	}
	return labels
}

func selectorByteToSelector(s Selector) string {
	if s == SelectAll {
		return "all()"
	}
	if s == NoSelector {
		return ""
	}
	sels := []string{}
	for i := uint(0); i < 8; i++ {
		if s&(1<<i) != 0 {
			sels = append(sels, fmt.Sprintf("has(label%d)", i+1))
		}
	}
	return strings.Join(sels, " && ")
}

func selectorByteToNamespaceSelector(s Selector) string {
	if s == SelectAll {
		return "all()"
	}
	if s == NoSelector {
		return ""
	}
	sels := []string{}
	for i := uint(0); i < 8; i++ {
		if s&(1<<i) != 0 {
			sels = append(sels, fmt.Sprintf("has(pcns.label%d)", i+1))
		}
	}
	return strings.Join(sels, " && ")
}

func selectorByteToK8sSelector(s Selector) *metav1.LabelSelector {
	if s == NoSelector {
		return nil
	}
	sel := &metav1.LabelSelector{}
	if s == SelectAll {
		return sel
	}
	for i := uint(0); i < 8; i++ {
		if s&(1<<i) != 0 {
			sel.MatchExpressions = append(sel.MatchExpressions, metav1.LabelSelectorRequirement{
				Key:      fmt.Sprintf("label%d", i+1),
				Operator: metav1.LabelSelectorOpExists,
			})

		}
	}
	return sel
}

func getResourceId(kind schema.GroupVersionKind, nameIdx Name, namespaceIdx Namespace) resources.ResourceID {
	name := fmt.Sprintf("%s-%d", strings.ToLower(kind.Kind), nameIdx)
	var namespace string
	if namespaceIdx > 0 {
		namespace = fmt.Sprintf("namespace-%d", namespaceIdx)
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

func getObjectMeta(r resources.ResourceID, labels Label) metav1.ObjectMeta {
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

func (t *XRefCacheTester) GetHostEndpoint(nameIdx Name) *xrefcache.CacheEntryEndpoint {
	r := getResourceId(resources.ResourceTypeHostEndpoints, nameIdx, 0)
	e := t.xrefCache.Get(r)
	if e == nil {
		return nil
	}
	return e.(*xrefcache.CacheEntryEndpoint)
}

func (t *XRefCacheTester) SetHostEndpoint(nameIdx Name, labels Label, ips IP) {
	r := getResourceId(resources.ResourceTypeHostEndpoints, nameIdx, 0)
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeSet,
		ResourceID: r,
		Resource: &apiv3.HostEndpoint{
			TypeMeta:   getTypeMeta(r),
			ObjectMeta: getObjectMeta(r, labels),
			Spec: apiv3.HostEndpointSpec{
				Node:        "node1",
				ExpectedIPs: ipByteToIPStringSlice(ips),
			},
		},
	})
}

func (t *XRefCacheTester) DeleteHostEndpoint(nameIdx Name) {
	r := getResourceId(resources.ResourceTypeHostEndpoints, nameIdx, 0)
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeDeleted,
		ResourceID: r,
	})
}

//
// -- Tier access --
//

func (t *XRefCacheTester) GetTier(nameIdx Name) *xrefcache.CacheEntryCalicoNetworkSet {
	r := getResourceId(resources.ResourceTypeTiers, nameIdx, 0)
	e := t.xrefCache.Get(r)
	if e == nil {
		return nil
	}
	return e.(*xrefcache.CacheEntryCalicoNetworkSet)
}

func (t *XRefCacheTester) SetTier(nameIdx Name, order float64) {
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

func (t *XRefCacheTester) DeleteTier(nameIdx Name) {
	r := getResourceId(resources.ResourceTypeTiers, nameIdx, 0)
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeDeleted,
		ResourceID: r,
	})
}

//
// -- GlobalNetworkSet access --
//

func (t *XRefCacheTester) GetGlobalNetworkSet(nameIdx Name) *xrefcache.CacheEntryCalicoNetworkSet {
	r := getResourceId(resources.ResourceTypeGlobalNetworkSets, nameIdx, 0)
	e := t.xrefCache.Get(r)
	if e == nil {
		return nil
	}
	return e.(*xrefcache.CacheEntryCalicoNetworkSet)
}

func (t *XRefCacheTester) SetGlobalNetworkSet(nameIdx Name, labels Label, nets Net) {
	r := getResourceId(resources.ResourceTypeGlobalNetworkSets, nameIdx, 0)
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeSet,
		ResourceID: r,
		Resource: &apiv3.GlobalNetworkSet{
			TypeMeta:   getTypeMeta(r),
			ObjectMeta: getObjectMeta(r, labels),
			Spec: apiv3.GlobalNetworkSetSpec{
				Nets: getCalicoNets(nets),
			},
		},
	})
}

func (t *XRefCacheTester) DeleteGlobalNetworkSet(nameIdx Name) {
	r := getResourceId(resources.ResourceTypeGlobalNetworkSets, nameIdx, 0)
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeDeleted,
		ResourceID: r,
	})
}

//
// -- Calico GlobalNetworkPolicy access --
//

func (t *XRefCacheTester) GetGlobalNetworkPolicy(nameIdx Name) *xrefcache.CacheEntryNetworkPolicy {
	r := getResourceId(resources.ResourceTypeGlobalNetworkPolicies, nameIdx, 0)
	e := t.xrefCache.Get(r)
	if e == nil {
		return nil
	}
	return e.(*xrefcache.CacheEntryNetworkPolicy)
}

func (t *XRefCacheTester) SetGlobalNetworkPolicy(nameIdx Name, s Selector, ingress, egress []apiv3.Rule) {
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
			ObjectMeta: getObjectMeta(r, NoLabels),
			Spec: apiv3.GlobalNetworkPolicySpec{
				Selector: selectorByteToSelector(s),
				Ingress:  ingress,
				Egress:   egress,
				Types:    types,
			},
		},
	})
}

func (t *XRefCacheTester) DeleteGlobalNetworkPolicy(nameIdx Name) {
	r := getResourceId(resources.ResourceTypeGlobalNetworkPolicies, nameIdx, 0)
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeDeleted,
		ResourceID: r,
	})
}

//
// -- Calico NetworkPolicy access --
//

func (t *XRefCacheTester) GetNetworkPolicy(nameIdx Name, namespaceIdx Namespace) *xrefcache.CacheEntryNetworkPolicy {
	r := getResourceId(resources.ResourceTypeNetworkPolicies, nameIdx, namespaceIdx)
	e := t.xrefCache.Get(r)
	if e == nil {
		return nil
	}
	return e.(*xrefcache.CacheEntryNetworkPolicy)
}

func (t *XRefCacheTester) SetNetworkPolicy(nameIdx Name, namespaceIdx Namespace, s Selector, ingress, egress []apiv3.Rule) {
	r := getResourceId(resources.ResourceTypeNetworkPolicies, nameIdx, namespaceIdx)
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
			ObjectMeta: getObjectMeta(r, NoLabels),
			Spec: apiv3.NetworkPolicySpec{
				Selector: selectorByteToSelector(s),
				Ingress:  ingress,
				Egress:   egress,
				Types:    types,
			},
		},
	})
}

func (t *XRefCacheTester) DeleteNetworkPolicy(nameIdx Name, namespaceIdx Namespace) {
	r := getResourceId(resources.ResourceTypeNetworkPolicies, nameIdx, namespaceIdx)
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeDeleted,
		ResourceID: r,
	})
}

//
// -- K8s NetworkPolicy access --
//

func (t *XRefCacheTester) GetK8sNetworkPolicy(nameIdx Name, namespaceIdx Namespace) *xrefcache.CacheEntryNetworkPolicy {
	r := getResourceId(resources.ResourceTypeK8sNetworkPolicies, nameIdx, namespaceIdx)
	e := t.xrefCache.Get(r)
	if e == nil {
		return nil
	}
	return e.(*xrefcache.CacheEntryNetworkPolicy)
}

func (t *XRefCacheTester) SetK8sNetworkPolicy(
	nameIdx Name, namespaceIdx Namespace, s Selector,
	ingress []networkingv1.NetworkPolicyIngressRule,
	egress []networkingv1.NetworkPolicyEgressRule,
) {
	r := getResourceId(resources.ResourceTypeK8sNetworkPolicies, nameIdx, namespaceIdx)
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
			ObjectMeta: getObjectMeta(r, NoLabels),
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: *selectorByteToK8sSelector(s),
				PolicyTypes: types,
				Ingress:     ingress,
				Egress:      egress,
			},
		},
	})
}

func (t *XRefCacheTester) DeleteK8sNetworkPolicy(nameIdx Name, namespaceIdx Namespace) {
	r := getResourceId(resources.ResourceTypeK8sNetworkPolicies, nameIdx, namespaceIdx)
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeDeleted,
		ResourceID: r,
	})
}

//
// -- K8s Pod access --
//

func (t *XRefCacheTester) GetPod(nameIdx Name, namespaceIdx Namespace) *xrefcache.CacheEntryEndpoint {
	r := getResourceId(resources.ResourceTypePods, nameIdx, namespaceIdx)
	e := t.xrefCache.Get(r)
	if e == nil {
		return nil
	}
	return e.(*xrefcache.CacheEntryEndpoint)
}

func (t *XRefCacheTester) SetPod(nameIdx Name, namespaceIdx Namespace, labels Label, ip IP, serviceAccount Name, opts PodOpt) {
	r := getResourceId(resources.ResourceTypePods, nameIdx, namespaceIdx)
	var sa string
	if serviceAccount != 0 {
		sr := getResourceId(resources.ResourceTypeServiceAccounts, serviceAccount, namespaceIdx)
		sa = sr.Name
	}
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeSet,
		ResourceID: r,
		Resource: &corev1.Pod{
			TypeMeta:   getTypeMeta(r),
			ObjectMeta: getObjectMeta(r, labels),
			Spec: corev1.PodSpec{
				NodeName:           "node1",
				ServiceAccountName: sa,
			},
			Status: corev1.PodStatus{
				PodIP: ipByteToIPString(ip),
			},
		},
	})
}

func (t *XRefCacheTester) DeletePod(nameIdx Name, namespaceIdx Namespace) {
	r := getResourceId(resources.ResourceTypePods, nameIdx, namespaceIdx)
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeDeleted,
		ResourceID: r,
	})
}

//
// -- K8s Endpoints access --
//

func (t *XRefCacheTester) GetEndpoints(nameIdx Name, namespaceIdx Namespace) *xrefcache.CacheEntryK8sServiceEndpoints {
	r := getResourceId(resources.ResourceTypeEndpoints, nameIdx, namespaceIdx)
	e := t.xrefCache.Get(r)
	if e == nil {
		return nil
	}
	return e.(*xrefcache.CacheEntryK8sServiceEndpoints)
}

func (t *XRefCacheTester) SetEndpoints(nameIdx Name, namespaceIdx Namespace, ips IP) resources.ResourceID {
	r := getResourceId(resources.ResourceTypeEndpoints, nameIdx, namespaceIdx)
	ipAddrs := ipByteToIPStringSlice(ips)

	// Convert the IP addresses to endpoint subsets, splitting over multiple if there is more than a single address.
	ss := []corev1.EndpointSubset{}
	if len(ipAddrs) > 1 {
		ss = append(ss, corev1.EndpointSubset{
			Addresses: []corev1.EndpointAddress{{
				IP: ipAddrs[0],
			}},
		})
		ipAddrs = ipAddrs[1:]
	}
	addrs := []corev1.EndpointAddress{}
	for _, ip := range ipAddrs {
		addrs = append(addrs, corev1.EndpointAddress{
			IP: ip,
		})
	}
	ss = append(ss, corev1.EndpointSubset{
		Addresses: addrs,
	})

	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeSet,
		ResourceID: r,
		Resource: &corev1.Endpoints{
			TypeMeta:   getTypeMeta(r),
			ObjectMeta: getObjectMeta(r, NoLabels),
			Subsets:    ss,
		},
	})
	return r
}

func (t *XRefCacheTester) DeleteEndpoints(nameIdx Name, namespaceIdx Namespace) {
	r := getResourceId(resources.ResourceTypeEndpoints, nameIdx, namespaceIdx)
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeDeleted,
		ResourceID: r,
	})
	metav1.Now()
}

//
// -- K8s ServiceAccounts access --
//

func (t *XRefCacheTester) GetServiceAccount(nameIdx Name, namespaceIdx Namespace) *xrefcache.CacheEntryK8sServiceAccount {
	r := getResourceId(resources.ResourceTypeServiceAccounts, nameIdx, namespaceIdx)
	e := t.xrefCache.Get(r)
	if e == nil {
		return nil
	}
	return e.(*xrefcache.CacheEntryK8sServiceAccount)
}

func (t *XRefCacheTester) SetServiceAccount(nameIdx Name, namespaceIdx Namespace, labels Label, ip string) {
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

func (t *XRefCacheTester) DeleteServiceAccount(nameIdx Name, namespaceIdx Namespace) {
	r := getResourceId(resources.ResourceTypeServiceAccounts, nameIdx, 0)
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeDeleted,
		ResourceID: r,
	})
}

//
// -- K8s Namespaces access --
//

func (t *XRefCacheTester) GetNamespace(nameIdx Name, namespaceIdx Namespace) *xrefcache.CacheEntryK8sNamespace {
	r := getResourceId(resources.ResourceTypeNamespaces, nameIdx, namespaceIdx)
	e := t.xrefCache.Get(r)
	if e == nil {
		return nil
	}
	return e.(*xrefcache.CacheEntryK8sNamespace)
}

func (t *XRefCacheTester) SetNamespace(nameIdx Name, namespaceIdx Namespace, labels Label, ip string) {
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

func (t *XRefCacheTester) DeleteNamespace(nameIdx Name, namespaceIdx Namespace) {
	r := getResourceId(resources.ResourceTypeNamespaces, nameIdx, 0)
	t.xrefCache.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeDeleted,
		ResourceID: r,
	})
}

//
// -- K8s rule selector pseudo resource access --
//

func (t *XRefCacheTester) GetCachedRuleSelectors() []string {
	ids := t.xrefCache.GetCachedResourceIDs(xrefcache.KindSelector)
	selectors := make([]string, len(ids))
	for i := range ids {
		selectors[i] = ids[i].Name
	}
	return selectors
}

func (t *XRefCacheTester) GetGNPRuleSelectorCacheEntry(sel Selector, nsSel Selector) *xrefcache.CacheEntryNetworkPolicyRuleSelector {
	s := selectorByteToSelector(sel)
	if nsSel != NoNamespaceSelector {
		s = fmt.Sprintf("(%s) && (%s)", selectorByteToNamespaceSelector(nsSel), s)
	}
	entry := t.xrefCache.Get(resources.ResourceID{
		GroupVersionKind: xrefcache.KindSelector,
		NameNamespace: resources.NameNamespace{
			Name: s,
		},
	})
	if entry == nil {
		return nil
	}
	return entry.(*xrefcache.CacheEntryNetworkPolicyRuleSelector)
}
