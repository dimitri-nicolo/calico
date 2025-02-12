// Copyright (c) 2025 Tigera, Inc. All rights reserved.

// Code generated by lister-gen. DO NOT EDIT.

package v3

import (
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/listers"
	"k8s.io/client-go/tools/cache"
)

// GlobalAlertTemplateLister helps list GlobalAlertTemplates.
// All objects returned here must be treated as read-only.
type GlobalAlertTemplateLister interface {
	// List lists all GlobalAlertTemplates in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v3.GlobalAlertTemplate, err error)
	// Get retrieves the GlobalAlertTemplate from the index for a given name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v3.GlobalAlertTemplate, error)
	GlobalAlertTemplateListerExpansion
}

// globalAlertTemplateLister implements the GlobalAlertTemplateLister interface.
type globalAlertTemplateLister struct {
	listers.ResourceIndexer[*v3.GlobalAlertTemplate]
}

// NewGlobalAlertTemplateLister returns a new GlobalAlertTemplateLister.
func NewGlobalAlertTemplateLister(indexer cache.Indexer) GlobalAlertTemplateLister {
	return &globalAlertTemplateLister{listers.New[*v3.GlobalAlertTemplate](indexer, v3.Resource("globalalerttemplate"))}
}
