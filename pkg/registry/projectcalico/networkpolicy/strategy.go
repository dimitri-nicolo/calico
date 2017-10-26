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

package networkpolicy

import (
	"fmt"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/names"

	calico "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
)

type policyStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

// NewStrategy returns a new NamespaceScopedStrategy for instances
func NewStrategy(typer runtime.ObjectTyper) policyStrategy {
	return policyStrategy{typer, names.SimpleNameGenerator}
}

func (policyStrategy) NamespaceScoped() bool {
	return true
}

func (policyStrategy) PrepareForCreate(ctx genericapirequest.Context, obj runtime.Object) {
	policy := obj.(*calico.NetworkPolicy)
	tier, _ := getTierPolicy(policy.Name)
	policy.SetLabels(map[string]string{"projectcalico.org/tier": tier})
}

func (policyStrategy) PrepareForUpdate(ctx genericapirequest.Context, obj, old runtime.Object) {
}

func (policyStrategy) Validate(ctx genericapirequest.Context, obj runtime.Object) field.ErrorList {
	return field.ErrorList{}
	//return validation.ValidatePolicy(obj.(*calico.Policy))
}

func (policyStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (policyStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (policyStrategy) Canonicalize(obj runtime.Object) {
}

func (policyStrategy) ValidateUpdate(ctx genericapirequest.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
	// return validation.ValidatePolicyUpdate(obj.(*calico.Policy), old.(*calico.Policy))
}

func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, bool, error) {
	policy, ok := obj.(*calico.NetworkPolicy)
	if !ok {
		return nil, nil, false, fmt.Errorf("given object is not a Policy.")
	}
	return labels.Set(policy.ObjectMeta.Labels), PolicyToSelectableFields(policy), policy.Initializers != nil, nil
}

// MatchPolicy is the filter used by the generic etcd backend to watch events
// from etcd to clients of the apiserver only interested in specific labels/fields.
func MatchPolicy(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// PolicyToSelectableFields returns a field set that represents the object.
func PolicyToSelectableFields(obj *calico.NetworkPolicy) fields.Set {
	return fields.Set{
		"metadata.name":      obj.Name,
		"metadata.namespace": obj.Namespace,
		"spec.tier":          obj.Spec.Tier,
	}
}
