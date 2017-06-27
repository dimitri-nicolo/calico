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

// Package calicoauth implements the tier check on policy operations based
// no RBAC permission as set on the Tier resource.
package authorizer

import (
	"github.com/golang/glog"
	"k8s.io/apiserver/pkg/authorization/authorizer"
)

type CalicoAuthorizer struct {
	authorizer.Authorizer
}

func NewCalicoAuthorizer(a authorizer.Authorizer) *CalicoAuthorizer {
	ca := &CalicoAuthorizer{
		a,
	}

	return ca
}

func (c *CalicoAuthorizer) Authorize(requestAttributes authorizer.Attributes) (authorized bool, reason string, err error) {
	authorized = true
	resourceType := requestAttributes.GetResource()
	if resourceType == "policies" {
		tierName := requestAttributes.GetSelectorQuery()
		attrs := authorizer.AttributesRecord{}
		attrs.APIGroup = requestAttributes.GetAPIGroup()
		attrs.APIVersion = requestAttributes.GetAPIVersion()
		attrs.Name = tierName
		attrs.Resource = "tiers"
		attrs.User = requestAttributes.GetUser()
		attrs.Verb = requestAttributes.GetVerb()
		authorized, reason, err = c.Authorizer.Authorize(attrs)
		glog.Infof("For policy %s tier %s is getting authorized first and its a %s", requestAttributes.GetName(), tierName, authorized)
	}
	if authorized == false {
		return authorized, reason, err
	}
	glog.Infof("Authorizer SelectorQuery: %s", requestAttributes.GetSelectorQuery())
	glog.Infof("Authorizer APIGroup: %s", requestAttributes.GetAPIGroup())
	glog.Infof("Authorizer APIVersion: %s", requestAttributes.GetAPIVersion())
	glog.Infof("Authorizer Name: %s", requestAttributes.GetName())
	glog.Infof("Authorizer Namespace: %s", requestAttributes.GetNamespace())
	glog.Infof("Authorizer Resource: %s", requestAttributes.GetResource())
	glog.Infof("Authorizer Subresource: %s", requestAttributes.GetSubresource())
	glog.Infof("Authorizer User: %s", requestAttributes.GetUser())
	glog.Infof("Authorizer Verb: %s", requestAttributes.GetVerb())

	return c.Authorizer.Authorize(requestAttributes)
}
