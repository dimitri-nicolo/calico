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

// NewXrefCacheTester returns a new XrefCacheTester. This can be used to send in syncer events for the different
// resource types, and to query current state of the cache.
func NewXrefCacheTester() *XrefCacheTester {
	return &XrefCacheTester{
		XrefCache: xrefcache.NewXrefCache(),
	}
}

// ipByteToIPString converts the IP byte value to an IP string.
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

// ipByteToIPStringSlice converts the IP byte value to an IP string slice. Note that the ip value is actually a bit-mask
// so may encapsulate multiple addresses in one.
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

// labelByteToLabels converts the label bitmask to a set of labels with keys named label<bit> and an enpty string value.
func labelByteToLabels(l Label) map[string]string {
	labels := make(map[string]string)
	for i := uint(0); i < 8; i++ {
		if l&(1<<i) != 0 {
			labels[fmt.Sprintf("label%d", i+1)] = ""
		}
	}
	return labels
}

// selectorByteToSelector converts the selector bitmask to an ANDed set of has(label<bit>) selector string.
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

// selectorByteToNamespaceSelector converts the selector bitmask to an ANDed set of has(label<bit>) selector string.
// This specific method is used by the rule selector testing, where we need to encode a namespace label.
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

// selectorByteToSelector converts the selector bitmask to a Kubernetes selector containing the set of label<bit>) keys
// with the "Exists" operator. This is effectively the k8s equivalent of the selectorByteToSelector method.
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

// getResourceId converts index values to an actual resource ID.
func getResourceId(kind schema.GroupVersionKind, nameIdx Name, namespaceIdx Namespace) resources.ResourceID {
	name := fmt.Sprintf("%s-%d", strings.ToLower(kind.Kind), nameIdx)
	var namespace string
	if namespaceIdx > 0 {
		namespace = fmt.Sprintf("namespace-%d", namespaceIdx)
	}
	if kind == resources.ResourceTypeNamespaces {
		name = namespace
		namespace = ""
	}
	return resources.ResourceID{
		GroupVersionKind: kind,
		NameNamespace: resources.NameNamespace{
			Name:      name,
			Namespace: namespace,
		},
	}
}

// getTypeMeta returns a TypeMeta for a given resource ID.
func getTypeMeta(r resources.ResourceID) metav1.TypeMeta {
	return metav1.TypeMeta{
		Kind:       r.Kind,
		APIVersion: r.GroupVersion().String(),
	}
}

// getObjectMeta returns a ObjectMeta for a given resource ID and set of labels.
func getObjectMeta(r resources.ResourceID, labels Label) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      r.Name,
		Namespace: r.Namespace,
		Labels:    labelByteToLabels(labels),
	}
}

// XrefCacheTester is the XrefCache tester.
type XrefCacheTester struct {
	xrefcache.XrefCache
}

// GetSelector returns the selector for a given selector bitmask value.
func (t *XrefCacheTester) GetSelector(sel Selector) string {
	return selectorByteToSelector(sel)
}

//
// -- HostEndpoint access --
//

func (t *XrefCacheTester) GetHostEndpoint(nameIdx Name) *xrefcache.CacheEntryEndpoint {
	r := getResourceId(resources.ResourceTypeHostEndpoints, nameIdx, 0)
	e := t.Get(r)
	if e == nil {
		return nil
	}
	return e.(*xrefcache.CacheEntryEndpoint)
}

func (t *XrefCacheTester) SetHostEndpoint(nameIdx Name, labels Label, ips IP) {
	r := getResourceId(resources.ResourceTypeHostEndpoints, nameIdx, 0)
	t.OnUpdate(syncer.Update{
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

func (t *XrefCacheTester) DeleteHostEndpoint(nameIdx Name) {
	r := getResourceId(resources.ResourceTypeHostEndpoints, nameIdx, 0)
	t.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeDeleted,
		ResourceID: r,
	})
}

//
// -- Tier access --
//

func (t *XrefCacheTester) GetTier(nameIdx Name) *xrefcache.CacheEntryCalicoNetworkSet {
	r := getResourceId(resources.ResourceTypeTiers, nameIdx, 0)
	e := t.Get(r)
	if e == nil {
		return nil
	}
	return e.(*xrefcache.CacheEntryCalicoNetworkSet)
}

func (t *XrefCacheTester) SetTier(nameIdx Name, order float64) {
	r := getResourceId(resources.ResourceTypeTiers, nameIdx, 0)
	t.OnUpdate(syncer.Update{
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

func (t *XrefCacheTester) DeleteTier(nameIdx Name) {
	r := getResourceId(resources.ResourceTypeTiers, nameIdx, 0)
	t.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeDeleted,
		ResourceID: r,
	})
}

//
// -- GlobalNetworkSet access --
//

func (t *XrefCacheTester) GetGlobalNetworkSet(nameIdx Name) *xrefcache.CacheEntryCalicoNetworkSet {
	r := getResourceId(resources.ResourceTypeGlobalNetworkSets, nameIdx, 0)
	e := t.Get(r)
	if e == nil {
		return nil
	}
	return e.(*xrefcache.CacheEntryCalicoNetworkSet)
}

func (t *XrefCacheTester) SetGlobalNetworkSet(nameIdx Name, labels Label, nets Net) {
	r := getResourceId(resources.ResourceTypeGlobalNetworkSets, nameIdx, 0)
	t.OnUpdate(syncer.Update{
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

func (t *XrefCacheTester) DeleteGlobalNetworkSet(nameIdx Name) {
	r := getResourceId(resources.ResourceTypeGlobalNetworkSets, nameIdx, 0)
	t.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeDeleted,
		ResourceID: r,
	})
}

//
// -- Calico GlobalNetworkPolicy access --
//

func (t *XrefCacheTester) GetGlobalNetworkPolicy(nameIdx Name) *xrefcache.CacheEntryNetworkPolicy {
	r := getResourceId(resources.ResourceTypeGlobalNetworkPolicies, nameIdx, 0)
	e := t.Get(r)
	if e == nil {
		return nil
	}
	return e.(*xrefcache.CacheEntryNetworkPolicy)
}

func (t *XrefCacheTester) SetGlobalNetworkPolicy(nameIdx Name, s Selector, ingress, egress []apiv3.Rule) {
	r := getResourceId(resources.ResourceTypeGlobalNetworkPolicies, nameIdx, 0)
	types := []apiv3.PolicyType{}
	if ingress != nil {
		types = append(types, apiv3.PolicyTypeIngress)
	}
	if egress != nil {
		types = append(types, apiv3.PolicyTypeEgress)
	}
	t.OnUpdate(syncer.Update{
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

func (t *XrefCacheTester) DeleteGlobalNetworkPolicy(nameIdx Name) {
	r := getResourceId(resources.ResourceTypeGlobalNetworkPolicies, nameIdx, 0)
	t.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeDeleted,
		ResourceID: r,
	})
}

//
// -- Calico NetworkPolicy access --
//

func (t *XrefCacheTester) GetNetworkPolicy(nameIdx Name, namespaceIdx Namespace) *xrefcache.CacheEntryNetworkPolicy {
	r := getResourceId(resources.ResourceTypeNetworkPolicies, nameIdx, namespaceIdx)
	e := t.Get(r)
	if e == nil {
		return nil
	}
	return e.(*xrefcache.CacheEntryNetworkPolicy)
}

func (t *XrefCacheTester) SetNetworkPolicy(nameIdx Name, namespaceIdx Namespace, s Selector, ingress, egress []apiv3.Rule) {
	r := getResourceId(resources.ResourceTypeNetworkPolicies, nameIdx, namespaceIdx)
	types := []apiv3.PolicyType{}
	if ingress != nil {
		types = append(types, apiv3.PolicyTypeIngress)
	}
	if egress != nil {
		types = append(types, apiv3.PolicyTypeEgress)
	}
	t.OnUpdate(syncer.Update{
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

func (t *XrefCacheTester) DeleteNetworkPolicy(nameIdx Name, namespaceIdx Namespace) {
	r := getResourceId(resources.ResourceTypeNetworkPolicies, nameIdx, namespaceIdx)
	t.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeDeleted,
		ResourceID: r,
	})
}

//
// -- K8s NetworkPolicy access --
//

func (t *XrefCacheTester) GetK8sNetworkPolicy(nameIdx Name, namespaceIdx Namespace) *xrefcache.CacheEntryNetworkPolicy {
	r := getResourceId(resources.ResourceTypeK8sNetworkPolicies, nameIdx, namespaceIdx)
	e := t.Get(r)
	if e == nil {
		return nil
	}
	return e.(*xrefcache.CacheEntryNetworkPolicy)
}

func (t *XrefCacheTester) SetK8sNetworkPolicy(
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
	t.OnUpdate(syncer.Update{
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

func (t *XrefCacheTester) DeleteK8sNetworkPolicy(nameIdx Name, namespaceIdx Namespace) {
	r := getResourceId(resources.ResourceTypeK8sNetworkPolicies, nameIdx, namespaceIdx)
	t.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeDeleted,
		ResourceID: r,
	})
}

//
// -- K8s Pod access --
//

func (t *XrefCacheTester) GetPod(nameIdx Name, namespaceIdx Namespace) *xrefcache.CacheEntryEndpoint {
	r := getResourceId(resources.ResourceTypePods, nameIdx, namespaceIdx)
	e := t.Get(r)
	if e == nil {
		return nil
	}
	return e.(*xrefcache.CacheEntryEndpoint)
}

func (t *XrefCacheTester) SetPod(nameIdx Name, namespaceIdx Namespace, labels Label, ip IP, serviceAccount Name, opts PodOpt) resources.ResourceID {
	r := getResourceId(resources.ResourceTypePods, nameIdx, namespaceIdx)
	var sa string
	if serviceAccount != 0 {
		sr := getResourceId(resources.ResourceTypeServiceAccounts, serviceAccount, namespaceIdx)
		sa = sr.Name
	}
	t.OnUpdate(syncer.Update{
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
	return r
}

func (t *XrefCacheTester) DeletePod(nameIdx Name, namespaceIdx Namespace) {
	r := getResourceId(resources.ResourceTypePods, nameIdx, namespaceIdx)
	t.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeDeleted,
		ResourceID: r,
	})
}

//
// -- K8s Endpoints access --
//

func (t *XrefCacheTester) GetEndpoints(nameIdx Name, namespaceIdx Namespace) *xrefcache.CacheEntryK8sServiceEndpoints {
	r := getResourceId(resources.ResourceTypeEndpoints, nameIdx, namespaceIdx)
	e := t.Get(r)
	if e == nil {
		return nil
	}
	return e.(*xrefcache.CacheEntryK8sServiceEndpoints)
}

func (t *XrefCacheTester) SetEndpoints(nameIdx Name, namespaceIdx Namespace, ips IP) resources.ResourceID {
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

	t.OnUpdate(syncer.Update{
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

func (t *XrefCacheTester) DeleteEndpoints(nameIdx Name, namespaceIdx Namespace) {
	r := getResourceId(resources.ResourceTypeEndpoints, nameIdx, namespaceIdx)
	t.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeDeleted,
		ResourceID: r,
	})
	metav1.Now()
}

//
// -- K8s ServiceAccounts access --
//

func (t *XrefCacheTester) GetServiceAccount(nameIdx Name, namespaceIdx Namespace) *xrefcache.CacheEntryK8sServiceAccount {
	r := getResourceId(resources.ResourceTypeServiceAccounts, nameIdx, namespaceIdx)
	e := t.Get(r)
	if e == nil {
		return nil
	}
	return e.(*xrefcache.CacheEntryK8sServiceAccount)
}

func (t *XrefCacheTester) SetServiceAccount(nameIdx Name, namespaceIdx Namespace, labels Label) resources.ResourceID {
	r := getResourceId(resources.ResourceTypeServiceAccounts, nameIdx, namespaceIdx)
	t.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeSet,
		ResourceID: r,
		Resource: &corev1.ServiceAccount{
			TypeMeta:   getTypeMeta(r),
			ObjectMeta: getObjectMeta(r, labels),
		},
	})
	return r
}

func (t *XrefCacheTester) DeleteServiceAccount(nameIdx Name, namespaceIdx Namespace) {
	r := getResourceId(resources.ResourceTypeServiceAccounts, nameIdx, namespaceIdx)
	t.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeDeleted,
		ResourceID: r,
	})
}

//
// -- K8s Namespaces access --
//

func (t *XrefCacheTester) GetNamespace(namespaceIdx Namespace) *xrefcache.CacheEntryK8sNamespace {
	r := getResourceId(resources.ResourceTypeNamespaces, 0, namespaceIdx)
	e := t.Get(r)
	if e == nil {
		return nil
	}
	return e.(*xrefcache.CacheEntryK8sNamespace)
}

func (t *XrefCacheTester) SetNamespace(namespaceIdx Namespace, labels Label) resources.ResourceID {
	r := getResourceId(resources.ResourceTypeNamespaces, 0, namespaceIdx)
	t.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeSet,
		ResourceID: r,
		Resource: &corev1.Namespace{
			TypeMeta:   getTypeMeta(r),
			ObjectMeta: getObjectMeta(r, labels),
		},
	})
	return r
}

func (t *XrefCacheTester) DeleteNamespace(namespaceIdx Namespace) {
	r := getResourceId(resources.ResourceTypeNamespaces, 0, namespaceIdx)
	t.OnUpdate(syncer.Update{
		Type:       syncer.UpdateTypeDeleted,
		ResourceID: r,
	})
}

//
// -- K8s rule selector pseudo resource access --
//

func (t *XrefCacheTester) GetCachedRuleSelectors() []string {
	ids := t.GetCachedResourceIDs(xrefcache.KindSelector)
	selectors := make([]string, len(ids))
	for i := range ids {
		selectors[i] = ids[i].Name
	}
	return selectors
}

func (t *XrefCacheTester) GetGNPRuleSelectorCacheEntry(sel Selector, nsSel Selector) *xrefcache.CacheEntryNetworkPolicyRuleSelector {
	s := selectorByteToSelector(sel)
	if nsSel != NoNamespaceSelector {
		s = fmt.Sprintf("(%s) && (%s)", selectorByteToNamespaceSelector(nsSel), s)
	}
	entry := t.Get(resources.ResourceID{
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
