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

package endpoint

import (
	"fmt"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/names"
	"k8s.io/client-go/pkg/api"

	"github.com/tigera/calico-k8sapiserver/pkg/apis/calico"
)

type apiServerStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

// strategy is the default logic that applies when creating and updating
// Role objects.
var Strategy = apiServerStrategy{api.Scheme, names.SimpleNameGenerator}

func (apiServerStrategy) NamespaceScoped() bool {
	return true
}

func (apiServerStrategy) PrepareForCreate(ctx genericapirequest.Context, obj runtime.Object) {

}

func (apiServerStrategy) PrepareForUpdate(ctx genericapirequest.Context, obj, old runtime.Object) {
}

func (apiServerStrategy) Validate(ctx genericapirequest.Context, obj runtime.Object) field.ErrorList {
	return field.ErrorList{}
	// return validation.ValidateEndpoint(obj.(*calico.Endpoint))
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
	return field.ErrorList{}
	// return validation.ValidateEndpointUpdate(obj.(*calico.Endpoint), old.(*calico.Endpoint))
}

func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, bool, error) {
	apiserver, ok := obj.(*calico.Endpoint)
	if !ok {
		return nil, nil, false, fmt.Errorf("given object is not a Endpoint.")
	}
	return labels.Set(apiserver.ObjectMeta.Labels), EndpointToSelectableFields(apiserver), apiserver.Initializers != nil, nil
}

// MatchEndpoint is the filter used by the generic etcd backend to watch events
// from etcd to clients of the apiserver only interested in specific labels/fields.
func MatchEndpoint(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// EndpointToSelectableFields returns a field set that represents the object.
func EndpointToSelectableFields(obj *calico.Endpoint) fields.Set {
	return generic.ObjectMetaFieldsSet(&obj.ObjectMeta, true)
}
