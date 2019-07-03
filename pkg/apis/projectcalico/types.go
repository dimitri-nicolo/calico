// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package projectcalico

import (
	calico "github.com/projectcalico/libcalico-go/lib/apis/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NetworkPolicyList is a list of Policy objects.
type NetworkPolicyList struct {
	metav1.TypeMeta
	metav1.ListMeta

	Items []NetworkPolicy
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type NetworkPolicy struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec calico.NetworkPolicySpec
}

// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TierList is a list of Tier objects.
type TierList struct {
	metav1.TypeMeta
	metav1.ListMeta

	Items []Tier
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Tier struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec calico.TierSpec
}

// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GlobalNetworkPolicyList is a list of Policy objects.
type GlobalNetworkPolicyList struct {
	metav1.TypeMeta
	metav1.ListMeta

	Items []GlobalNetworkPolicy
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type GlobalNetworkPolicy struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec calico.GlobalNetworkPolicySpec
}

// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GlobalNetworkPolicyList is a list of Policy objects.
type GlobalNetworkSetList struct {
	metav1.TypeMeta
	metav1.ListMeta

	Items []GlobalNetworkSet
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type GlobalNetworkSet struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec calico.GlobalNetworkSetSpec
}

// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LicenseKeyList is a list of LicenseKey objects.
type LicenseKeyList struct {
	metav1.TypeMeta
	metav1.ListMeta

	Items []LicenseKey
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type LicenseKey struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec calico.LicenseKeySpec
}

// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GlobalThreatFeedList is a list of Policy objects.
type GlobalThreatFeedList struct {
	metav1.TypeMeta
	metav1.ListMeta

	Items []GlobalThreatFeed
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status

type GlobalThreatFeed struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec   calico.GlobalThreatFeedSpec
	Status calico.GlobalThreatFeedStatus
}

// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HostEndpointList is a list of Policy objects.
type HostEndpointList struct {
	metav1.TypeMeta
	metav1.ListMeta

	Items []HostEndpoint
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type HostEndpoint struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec calico.HostEndpointSpec
}

// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GlobalReportList is a list of objects to generate compliance reports.
type GlobalReportList struct {
	metav1.TypeMeta
	metav1.ListMeta

	Items []GlobalReport
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status

type GlobalReport struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec   calico.ReportSpec
	Status calico.ReportStatus
}

// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GlobalReportTypeList is a list of objects used by GlobalReports to define report template.
type GlobalReportTypeList struct {
	metav1.TypeMeta
	metav1.ListMeta

	Items []GlobalReportType
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type GlobalReportType struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec calico.ReportTypeSpec
}

// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IPPoolList contains a list of IPPool resources.
type IPPoolList struct {
	metav1.TypeMeta
	metav1.ListMeta

	Items []IPPool
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type IPPool struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec calico.IPPoolSpec
}
