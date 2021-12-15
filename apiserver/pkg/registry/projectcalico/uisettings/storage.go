// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package uisettings

import (
	"context"
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/api/meta"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/generic/registry"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"

	calico "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/apiserver/pkg/registry/projectcalico/authorizer"
	"github.com/projectcalico/calico/apiserver/pkg/registry/projectcalico/server"
	"github.com/projectcalico/calico/apiserver/pkg/registry/projectcalico/util"
)

// rest implements a RESTStorage for API services against etcd
type REST struct {
	*genericregistry.Store
	authorizer authorizer.UISettingsAuthorizer
	shortNames []string
}

func (r *REST) ShortNames() []string {
	return r.shortNames
}

func (r *REST) Categories() []string {
	return []string{""}
}

// EmptyObject returns an empty instance
func EmptyObject() runtime.Object {
	return &calico.UISettings{}
}

// NewList returns a new shell of a binding list
func NewList() runtime.Object {
	return &calico.UISettingsList{}
}

// NewREST returns a RESTStorage object that will work against API services.
func NewREST(scheme *runtime.Scheme, opts server.Options) (*REST, error) {
	strategy := NewStrategy(scheme)

	prefix := "/" + opts.ResourcePrefix()
	// We adapt the store's keyFunc so that we can use it with the StorageDecorator
	// without making any assumptions about where objects are stored in etcd
	keyFunc := func(obj runtime.Object) (string, error) {
		accessor, err := meta.Accessor(obj)
		if err != nil {
			return "", err
		}
		return registry.NoNamespaceKeyFunc(
			genericapirequest.NewContext(),
			prefix,
			accessor.GetName(),
		)
	}
	storageInterface, dFunc, err := opts.GetStorage(
		prefix,
		keyFunc,
		strategy,
		func() runtime.Object { return &calico.UISettings{} },
		func() runtime.Object { return &calico.UISettingsList{} },
		GetAttrs,
		nil,
		nil,
	)
	if err != nil {
		return nil, err
	}
	store := &genericregistry.Store{
		NewFunc:     func() runtime.Object { return &calico.UISettings{} },
		NewListFunc: func() runtime.Object { return &calico.UISettingsList{} },
		KeyRootFunc: opts.KeyRootFunc(false),
		KeyFunc:     opts.KeyFunc(false),
		ObjectNameFunc: func(obj runtime.Object) (string, error) {
			return obj.(*calico.UISettings).Name, nil
		},
		PredicateFunc:            MatchUISettings,
		DefaultQualifiedResource: calico.Resource("uisettings"),

		CreateStrategy:          strategy,
		UpdateStrategy:          strategy,
		DeleteStrategy:          strategy,
		EnableGarbageCollection: true,

		Storage:     storageInterface,
		DestroyFunc: dFunc,
	}

	return &REST{store, authorizer.NewUISettingsAuthorizer(opts.Authorizer), opts.ShortNames}, nil
}

func (r *REST) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	groupName, err := util.GetUISettingsGroupNameFromSelector(options)
	if err != nil {
		return nil, err
	}
	err = r.authorizer.AuthorizeUISettingsOperation(ctx, "", groupName)
	if err != nil {
		return nil, err
	}

	return r.Store.List(ctx, options)
}

func (r *REST) Create(ctx context.Context, obj runtime.Object, val rest.ValidateObjectFunc, createOpt *metav1.CreateOptions) (runtime.Object, error) {
	uiSettings := obj.(*calico.UISettings)
	group := uiSettings.Spec.Group
	if group == "" {
		return nil, fmt.Errorf("UISettings Spec.Group is not specified")
	}

	err := r.authorizer.AuthorizeUISettingsOperation(ctx, uiSettings.Name, group)
	if err != nil {
		return nil, err
	}

	return r.Store.Create(ctx, uiSettings, val, createOpt)
}

func (r *REST) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc,
	updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	groupName, _ := util.GetUISettingsGroupFromUISettingsName(name)
	err := r.authorizer.AuthorizeUISettingsOperation(ctx, name, groupName)
	if err != nil {
		return nil, false, err
	}

	// Modify the update validation to check that the owner reference is not being updated to remove or change the
	// group.
	updatedUpdateValidation := func(ctx context.Context, obj, old runtime.Object) error {
		oldUISettings := old.(*calico.UISettings)
		newUISettings := obj.(*calico.UISettings)
		if !reflect.DeepEqual(oldUISettings.OwnerReferences, newUISettings.OwnerReferences) {
			return fmt.Errorf("Not permitted to change UISettingsGroup owner reference")
		}
		oldGroup := oldUISettings.Spec.Group
		newGroup := newUISettings.Spec.Group
		if oldGroup != newGroup {
			return fmt.Errorf("Not permitted to change Spec.Group")
		}
		return updateValidation(ctx, obj, old)
	}

	return r.Store.Update(ctx, name, objInfo, createValidation, updatedUpdateValidation, forceAllowCreate, options)
}

// Get retrieves the item from storage.
func (r *REST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	groupName, _ := util.GetUISettingsGroupFromUISettingsName(name)
	err := r.authorizer.AuthorizeUISettingsOperation(ctx, name, groupName)
	if err != nil {
		return nil, err
	}

	return r.Store.Get(ctx, name, options)
}

func (r *REST) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	groupName, err := util.GetUISettingsGroupFromUISettingsName(name)
	if err != nil {
		return nil, false, err
	}
	err = r.authorizer.AuthorizeUISettingsOperation(ctx, name, groupName)
	if err != nil {
		return nil, false, err
	}

	return r.Store.Delete(ctx, name, deleteValidation, options)
}

func (r *REST) Watch(ctx context.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
	groupName, err := util.GetUISettingsGroupNameFromSelector(options)
	if err != nil {
		return nil, err
	}
	err = r.authorizer.AuthorizeUISettingsOperation(ctx, "", groupName)
	if err != nil {
		return nil, err
	}

	return r.Store.Watch(ctx, options)
}
