// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package globalreport

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

	calico "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
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
func (apiServerStrategy) PrepareForCreate(ctx genericapirequest.Context, obj runtime.Object) {
	globalReport := obj.(*calico.GlobalReport)
	globalReport.Status = v3.ReportStatus{}
}

// PrepareForUpdate copies the Status from old to obj
func (apiServerStrategy) PrepareForUpdate(ctx genericapirequest.Context, obj, old runtime.Object) {
	newGlobalReport := obj.(*calico.GlobalReport)
	oldGlobalReport := old.(*calico.GlobalReport)
	newGlobalReport.Status = oldGlobalReport.Status
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
	return ValidateGlobalReportUpdate(obj.(*calico.GlobalReport), old.(*calico.GlobalReport))
}

type apiServerStatusStrategy struct {
	apiServerStrategy
}

func NewStatusStrategy(strategy apiServerStrategy) apiServerStatusStrategy {
	return apiServerStatusStrategy{strategy}
}

func (apiServerStatusStrategy) PrepareForUpdate(ctx genericapirequest.Context, obj, old runtime.Object) {
	newGlobalReport := obj.(*calico.GlobalReport)
	oldGlobalReport := old.(*calico.GlobalReport)
	newGlobalReport.Spec = oldGlobalReport.Spec
	newGlobalReport.Labels = oldGlobalReport.Labels
}

// ValidateUpdate is the default update validation for an end user updating status
func (apiServerStatusStrategy) ValidateUpdate(ctx genericapirequest.Context, obj, old runtime.Object) field.ErrorList {
	return ValidateGlobalReportUpdate(obj.(*calico.GlobalReport), old.(*calico.GlobalReport))
}

func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, bool, error) {
	apiserver, ok := obj.(*calico.GlobalReport)
	if !ok {
		return nil, nil, false, fmt.Errorf("given object (type: %v) is not a Global Report", reflect.TypeOf(obj))
	}
	return labels.Set(apiserver.ObjectMeta.Labels), GlobalReportToSelectableFields(apiserver), apiserver.Initializers != nil, nil
}

// MatchGlobalReport is the filter used by the generic etcd backend to watch events
// from etcd to clients of the apiserver only interested in specific labels/fields.
func MatchGlobalReport(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// GlobalReportToSelectableFields returns a field set that represents the object.
func GlobalReportToSelectableFields(obj *calico.GlobalReport) fields.Set {
	return generic.ObjectMetaFieldsSet(&obj.ObjectMeta, false)
}

func ValidateGlobalReportUpdate(update, old *calico.GlobalReport) field.ErrorList {
	return apivalidation.ValidateObjectMetaUpdate(&update.ObjectMeta, &old.ObjectMeta, field.NewPath("metadata"))
}
