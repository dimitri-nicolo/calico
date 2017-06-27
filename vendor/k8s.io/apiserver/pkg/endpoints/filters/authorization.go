/*
Copyright 2016 The Kubernetes Authors.

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

package filters

import (
	"errors"
	"net/http"
	"strings"

	"github.com/golang/glog"

	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"k8s.io/apiserver/pkg/endpoints/request"
)

// WithAuthorizationCheck passes all authorized requests on to handler, and returns a forbidden error otherwise.
func WithAuthorization(handler http.Handler, requestContextMapper request.RequestContextMapper, a authorizer.Authorizer) http.Handler {
	if a == nil {
		glog.Warningf("Authorization is disabled")
		return handler
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx, ok := requestContextMapper.Get(req)
		if !ok {
			responsewriters.InternalError(w, req, errors.New("no context found for request"))
			return
		}

		attributes, err := GetAuthorizerAttributes(ctx)
		if err != nil {
			responsewriters.InternalError(w, req, err)
			return
		}
		attributes = WithSelectorQuery(attributes, req)
		glog.Infof("Authorizer Path: %s", attributes.GetPath())
		glog.Infof("Authorizer APIGroup: %s", attributes.GetAPIGroup())
		glog.Infof("Authorizer APIVersion: %s", attributes.GetAPIVersion())
		glog.Infof("Authorizer Name: %s", attributes.GetName())
		glog.Infof("Authorizer Namespace: %s", attributes.GetNamespace())
		glog.Infof("Authorizer Resource: %s", attributes.GetResource())
		glog.Infof("Authorizer Subresource: %s", attributes.GetSubresource())
		glog.Infof("Authorizer User: %s", attributes.GetUser())
		glog.Infof("Authorizer Verb: %s", attributes.GetVerb())
		glog.Infof("Authotizer Selector Query: %s", attributes.GetSelectorQuery())
		authorized, reason, err := a.Authorize(attributes)
		if authorized {
			handler.ServeHTTP(w, req)
			return
		}
		if err != nil {
			responsewriters.InternalError(w, req, err)
			return
		}

		glog.V(4).Infof("Forbidden: %#v, Reason: %q", req.RequestURI, reason)
		responsewriters.Forbidden(attributes, w, req, reason)
	})
}

func GetAuthorizerAttributes(ctx request.Context) (authorizer.Attributes, error) {
	attribs := authorizer.AttributesRecord{}

	user, ok := request.UserFrom(ctx)
	if ok {
		attribs.User = user
	}

	requestInfo, found := request.RequestInfoFrom(ctx)
	if !found {
		return nil, errors.New("no RequestInfo found in the context")
	}

	// Start with common attributes that apply to resource and non-resource requests
	attribs.ResourceRequest = requestInfo.IsResourceRequest
	attribs.Path = requestInfo.Path
	attribs.Verb = requestInfo.Verb

	attribs.APIGroup = requestInfo.APIGroup
	attribs.APIVersion = requestInfo.APIVersion
	attribs.Resource = requestInfo.Resource
	attribs.Subresource = requestInfo.Subresource
	attribs.Namespace = requestInfo.Namespace
	attribs.Name = requestInfo.Name

	return &attribs, nil
}

func WithSelectorQuery(a authorizer.Attributes, req *http.Request) authorizer.Attributes {
	attribs := authorizer.AttributesRecord{}
	attribs.User = a.GetUser()
	attribs.ResourceRequest = a.IsResourceRequest()
	attribs.Path = a.GetPath()
	attribs.Verb = a.GetVerb()

	attribs.APIGroup = a.GetAPIGroup()
	attribs.APIVersion = a.GetAPIVersion()
	attribs.Resource = a.GetResource()
	attribs.Subresource = a.GetSubresource()
	attribs.Namespace = a.GetNamespace()
	attribs.Name = a.GetName()

	tierName := "default"
	labelSelector := req.URL.Query()["labelSelector"]
	if len(labelSelector) > 0 {
		// TODO: Check for if its a Tier selector
		tierSelector := labelSelector[0]
		tierSplice := strings.Split(tierSelector, "==")
		if len(tierSplice) > 1 {
			tierName = tierSplice[1]
		}
	}
	attribs.SelectorQuery = tierName
	return &attribs
}
