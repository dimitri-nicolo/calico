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

package licensekey

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

	calico "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
	licClient "github.com/tigera/licensing/client"

	libcalicoapi "github.com/projectcalico/libcalico-go/lib/apis/v3"
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

func (apiServerStrategy) PrepareForCreate(ctx genericapirequest.Context, obj runtime.Object) {
}

func (apiServerStrategy) PrepareForUpdate(ctx genericapirequest.Context, obj, old runtime.Object) {
}

func (apiServerStrategy) Validate(ctx genericapirequest.Context, obj runtime.Object) field.ErrorList {
	return validateLicenseKey(obj)
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
	return validateLicenseKey(obj)
}

func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, bool, error) {
	apiserver, ok := obj.(*calico.LicenseKey)
	if !ok {
		return nil, nil, false, fmt.Errorf("given object is not a License Key")
	}
	return labels.Set(apiserver.ObjectMeta.Labels), LicenseKeyToSelectableFields(apiserver), apiserver.Initializers != nil, nil
}

// MatchLicenseKey is the filter used by the generic etcd backend to watch events
// from etcd to clients of the apiserver only interested in specific labels/fields.
func MatchLicenseKey(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// LicenseKeyToSelectableFields returns a field set that represents the object.
func LicenseKeyToSelectableFields(obj *calico.LicenseKey) fields.Set {
	return generic.ObjectMetaFieldsSet(&obj.ObjectMeta, false)
}

// Convert from aggregated api server runtime object to libcalico-go's licensekey structure
func convertToLibcalico(aapiObj runtime.Object) *libcalicoapi.LicenseKey {
	aapiLicenseKey := aapiObj.(*calico.LicenseKey)
	lcgLicenseKey := &libcalicoapi.LicenseKey{}
	lcgLicenseKey.TypeMeta = aapiLicenseKey.TypeMeta
	lcgLicenseKey.ObjectMeta = aapiLicenseKey.ObjectMeta
	lcgLicenseKey.Spec = aapiLicenseKey.Spec
	return lcgLicenseKey
}

// Ensure licenseKey is decodable and valid (not expired)
func validateLicenseKey(aapiObj runtime.Object) field.ErrorList {
	allErrs := field.ErrorList{}
	lcgLicenseKey := convertToLibcalico(aapiObj)

	// Decode the license to make sure it's not corrupt.
	licClaims, err := licClient.Decode(*lcgLicenseKey)
	if err != nil {
		allErrs = append(allErrs, field.InternalError(field.NewPath("LicenseKeySpec").Child("license"),
			fmt.Errorf("license is corrupted: %s", err)))
	} else {
		// Check if the license is expired
		if licClaims.Validate() != licClient.Valid {
			allErrs = append(allErrs, field.InternalError(field.NewPath("LicenseKeySpec").Child("token"),
				fmt.Errorf("the license you're trying to create expired on %s", licClaims.Expiry.Time().Local())))
		}
	}

	return allErrs
}
