// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package managedcluster

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/names"
	apivalidation "k8s.io/kubernetes/pkg/apis/core/validation"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	calico "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
)

type apiServerStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

// NewStrategy returns a new NamespaceScopedStrategy for instances
func NewStrategy(typer runtime.ObjectTyper) apiServerStrategy {
	return apiServerStrategy{typer, names.SimpleNameGenerator}
}

func (apiServerStrategy) NamespaceScoped() bool {
	return false
}

// PrepareForCreate clears the Status
// TODO: This is where we will generate the manifest for Guardian (https://tigera.atlassian.net/browse/SAAS-168)
func (apiServerStrategy) PrepareForCreate(ctx genericapirequest.Context, obj runtime.Object) {
	managedCluster := obj.(*calico.ManagedCluster)
	managedCluster.Status = v3.ManagedClusterStatus{}
}

// TODO: This is where we will copy the Status from old to obj when we add Status (https://tigera.atlassian.net/browse/SAAS-182)
func (apiServerStrategy) PrepareForUpdate(ctx genericapirequest.Context, obj, old runtime.Object) {
	newManagedCluster := obj.(*calico.ManagedCluster)
	oldManagedCluster := old.(*calico.ManagedCluster)
	newManagedCluster.Status = oldManagedCluster.Status
}

func (apiServerStrategy) Validate(ctx genericapirequest.Context, obj runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

func (apiServerStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (apiServerStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (apiServerStrategy) Canonicalize(obj runtime.Object) {
}

func (apiServerStrategy) ValidateUpdate(ctx genericapirequest.Context, obj, old runtime.Object) field.ErrorList {
	return ValidateManagedClusterUpdate(obj.(*calico.ManagedCluster), old.(*calico.ManagedCluster))
}

type apiServerStatusStrategy struct {
	apiServerStrategy
}

func NewStatusStrategy(strategy apiServerStrategy) apiServerStatusStrategy {
	return apiServerStatusStrategy{strategy}
}

func (apiServerStatusStrategy) PrepareForUpdate(ctx genericapirequest.Context, obj, old runtime.Object) {
	newManagedCluster := obj.(*calico.ManagedCluster)
	oldManagedCluster := old.(*calico.ManagedCluster)
	newManagedCluster.Spec = oldManagedCluster.Spec
	newManagedCluster.Labels = oldManagedCluster.Labels
}

// ValidateUpdate is the default update validation for an end user updating status
func (apiServerStatusStrategy) ValidateUpdate(ctx genericapirequest.Context, obj, old runtime.Object) field.ErrorList {
	return ValidateManagedClusterUpdate(obj.(*calico.ManagedCluster), old.(*calico.ManagedCluster))
}

func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, bool, error) {
	apiserver, ok := obj.(*calico.ManagedCluster)
	if !ok {
		return nil, nil, false, fmt.Errorf("given object (type %v) is not a Managed Cluster", reflect.TypeOf(obj))
	}
	return labels.Set(apiserver.ObjectMeta.Labels), ManagedClusterToSelectableFields(apiserver), apiserver.Initializers != nil, nil
}

// MatchManagedCluster is the filter used by the generic etcd backend to watch events
// from etcd to clients of the apiserver only interested in specific labels/fields.
func MatchManagedCluster(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// ManagedClusterToSelectableFields returns a field set that represents the object.
func ManagedClusterToSelectableFields(obj *calico.ManagedCluster) fields.Set {
	return generic.ObjectMetaFieldsSet(&obj.ObjectMeta, false)
}

func ValidateManagedClusterUpdate(update, old *calico.ManagedCluster) field.ErrorList {
	return apivalidation.ValidateObjectMetaUpdate(&update.ObjectMeta, &old.ObjectMeta, field.NewPath("metadata"))
}
