/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	calico "github.com/projectcalico/libcalico-go/lib/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PolicyList is a list of Policy objects.
type PolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Items []Policy `json:"items" protobuf:"bytes,2,rep,name=items"`
}

type PolicyStatus struct {
}

// +genclient=true

type Policy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   calico.PolicySpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status PolicyStatus      `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// TierList is a list of Policy objects.
type TierList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Items []Tier `json:"items" protobuf:"bytes,2,rep,name=items"`
}

//  +genclient=true
//  +nonNamespaced=true

type Tier struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec calico.TierSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

// NodeList is a list of Policy objects.
type NodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Items []Node `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// +genclient=true
// +nonNamespaced=true

type Node struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec calico.NodeSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}
